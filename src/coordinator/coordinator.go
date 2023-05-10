package coordinator

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iv-menshenin/lm/transport"
)

type (
	Manager struct {
		loglevel LogLevel

		id   [16]byte
		port uint16

		ls Transport

		state int64
		armed int64

		err  error
		once sync.Once
		done chan struct{}

		ins *Instances
		awr *Awaiter
		own *Ownership
	}
	LogLevel uint8
)

const (
	LogLevelError LogLevel = iota
	LogLevelWarning
	LogLevelDebug
)

type Transport interface {
	SendAll([]byte) error
	Send([]byte, net.Addr) error
	Listen([]byte) (*transport.Received, error)
}

const (
	StateCreated int64 = iota
	StateReady
	StateDiscovery
	StateDeactivated
	StateClosed
	StateBroken
)

const (
	UnArmed int64 = iota
	Armed
)

func New(transport Transport) *Manager {
	var idX [16]byte
	_, err := rand.Read(idX[:])
	if err != nil {
		panic(err)
	}
	return &Manager{
		loglevel: LogLevelError,
		id:       idX,
		ls:       transport,
		state:    StateCreated,
		armed:    UnArmed,
		done:     make(chan struct{}),
		ins:      newInstances(),
		awr:      newAwaiter(),
		own:      newOwnership(),
	}
}

func (m *Manager) SetLogLevel(l LogLevel) {
	m.loglevel = l
}

func (m *Manager) debug(format string, args ...any) {
	if m.loglevel >= LogLevelDebug {
		log.Printf(format, args...)
	}
}

func (m *Manager) warning(format string, args ...any) {
	if m.loglevel >= LogLevelWarning {
		log.Printf(format, args...)
	}
}

func (m *Manager) error(format string, args ...any) {
	log.Printf(format, args...)
}

func (m *Manager) Key() string {
	return fmt.Sprintf("%x", m.id[:])
}

func (m *Manager) Manage() error {
	if !atomic.CompareAndSwapInt64(&m.state, StateCreated, StateDiscovery) {
		return errors.New("wrong state")
	}
	go m.stateLoop()
	go m.readLoop()
	<-m.done
	return m.err
}

func (m *Manager) stateLoop() {
	var lastTimeDiscovered time.Time
	for {
		time.Sleep(10 * time.Millisecond)
		switch atomic.LoadInt64(&m.state) {
		case StateCreated:
			continue

		case StateReady:
			if del := m.ins.cleanup(); del > 0 {
				m.warning("UNLINKED: %d", del)
			}
			if time.Since(lastTimeDiscovered).Seconds() >= 1 {
				if err := m.checkArmedStatus(); err != nil {
					m.setErr(err)
					return
				}
				atomic.CompareAndSwapInt64(&m.state, StateReady, StateDiscovery)
			}
			continue

		case StateDiscovery:
			if err := m.sendKnockKnock(); err != nil {
				m.setErr(err)
				return
			}
			atomic.CompareAndSwapInt64(&m.state, StateDiscovery, StateReady)
			lastTimeDiscovered = time.Now()
			continue

		case StateDeactivated:
			atomic.StoreInt64(&m.state, StateClosed)
			fallthrough
		case StateClosed:
			return
		case StateBroken:
			return
		}
	}
}

func (m *Manager) checkArmedStatus() (err error) {
	m.debug("CHECK STATUS")
	var (
		cnt  = 3
		hash = m.ins.hashAllID(m.id)
	)
	for {
		errCh := m.awaitMostOf(cmdCompared, hash)
		if err = m.sendCompareInstances(hash); err != nil {
			return err
		}
		err = <-errCh
		if cnt--; err == nil || cnt < 0 {
			break
		}
	}
	if err != nil {
		if atomic.CompareAndSwapInt64(&m.armed, Armed, UnArmed) {
			m.warning("UNARMED: %+v", err)
		}
		return nil
	}
	if atomic.CompareAndSwapInt64(&m.armed, UnArmed, Armed) {
		m.warning("ARMED")
	}
	return nil
}

func (m *Manager) setErr(err error) {
	m.error("ERROR: %+v", err)
	m.once.Do(func() {
		m.err = err
		atomic.StoreInt64(&m.state, StateBroken)
	})
}

func (m *Manager) readLoop() {
	var buf [1024]byte
	for {
		received, err := m.ls.Listen(buf[:])
		if err != nil {
			m.setErr(err)
			return
		}
		parsed, err := parse(received)
		if err != nil {
			m.error("ERROR: %+v", err)
			continue
		}
		if err = m.process(parsed); err != nil {
			m.setErr(err)
			return
		}
		m.awr.trig(received.Data)
	}
}

func (m *Manager) process(msg Msg) error {
	if bytes.Equal(msg.sender, m.id[:]) {
		// skip self owned messages
		return nil
	}
	var err error
	// tell them all that we are online
	if bytes.Equal(msg.cmd, cmdBroadKnock) {
		if m.ins.add(msg.sender, msg.addr) {
			m.debug("REGISTERED: %x %s", msg.sender, msg.addr.String())
		}
		err = m.sendWelcome(msg.addr)
	}
	// register everyone who said hello
	if bytes.Equal(msg.cmd, cmdWelcome) {
		if m.ins.add(msg.sender, msg.addr) {
			m.debug("REGISTERED: %x %s", msg.sender, msg.addr.String())
		}
	}
	// confirm the on-line instance list
	if bytes.Equal(msg.cmd, cmdBroadCompare) {
		if bytes.Equal(m.ins.hashAllID(m.id), msg.data) {
			err = m.sendCompared(msg.addr, msg.data)
		}
	}
	// someone bragged about a captured key
	if bytes.Equal(msg.cmd, cmdBroadMine) {
		if err = m.ins.save(msg.sender, string(msg.data), msg.addr); err != nil {
			// err = m.sendReset(msg.data) not yours
		} else {
			log.Printf("OWNERSHIP APPROVED %x: %s", msg.sender, string(msg.data))
			err = m.sendSaved(msg.addr, msg.sender, msg.data)
		}
	}
	if bytes.Equal(msg.cmd, cmdBroadWantKey) {
		switch m.ins.search(string(msg.data)) {
		case Mine:
			err = m.sendRegistered(msg.addr, msg.data)
		case "":
			var id [16]byte
			copy(id[:], msg.sender)
			if m.own.add(id, string(msg.data)) {
				log.Printf("OWNERSHIP CANDIDATE %x: %s", id, string(msg.data))
				err = m.sendCandidate(msg.addr, msg.sender, msg.data)
			}
		}
	}
	if bytes.Equal(msg.cmd, cmdRegistered) {
		if err = m.ins.save(msg.sender, string(msg.data), msg.addr); err != nil {
			// TODO insecured
			err = m.sendReset(msg.data) // not yours
		}
	}
	// someone wants to revoke possession of a key because of a conflict
	if bytes.Equal(msg.cmd, cmdBroadReset) {
		m.ins.reset(string(msg.data))
	}

	return err
}
