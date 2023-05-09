package coordinator

import (
	"bytes"
	"crypto/rand"
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iv-menshenin/lm/transport"
)

type Manager struct {
	ID   [16]byte
	port uint16

	ls Transport

	state int64
	armed int64

	err  error
	once sync.Once
	done chan struct{}

	ins *Instances
	awr *Awaiter
}

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
		ID:    idX,
		ls:    transport,
		state: StateCreated,
		armed: UnArmed,
		done:  make(chan struct{}),
		ins:   newInstances(),
		awr:   newAwaiter(),
	}
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
	var cycles int64
	for {
		time.Sleep(10 * time.Millisecond)
		switch atomic.LoadInt64(&m.state) {
		case StateCreated:
			continue

		case StateReady:
			if del := m.ins.cleanup(); del > 0 {
				log.Printf("UNLINKED: %d\n", del)
			}
			if time.Since(lastTimeDiscovered).Seconds() >= 1 {
				if cycles > 3 {
					var (
						err  error
						cnt  = 3
						hash = m.ins.hashAllID(m.ID)
					)
					for {
						errCh := m.awaitMostOf(cmdCompared, hash)
						if err = m.sendCompareInstances(hash); err != nil {
							m.setErr(err)
							return
						}
						err = <-errCh
						if cnt--; err == nil || cnt < 0 {
							break
						}
					}
					if err != nil {
						if atomic.CompareAndSwapInt64(&m.armed, Armed, UnArmed) {
							// TODO
							log.Printf("UNARMED: %+v\n", err)
						}
					} else {
						if atomic.CompareAndSwapInt64(&m.armed, UnArmed, Armed) {
							// TODO
							log.Printf("ARMED\n")
						}
					}
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
			cycles++
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

func (m *Manager) setErr(err error) {
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
			// TODO
			log.Printf("ERROR: %+v", err)
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
	if bytes.Equal(msg.sender, m.ID[:]) {
		// skip self owned messages
		return nil
	}
	var err error
	// tell them all that we are online
	if bytes.Equal(msg.cmd, cmdBroadKnock) {
		m.ins.add(msg.sender, msg.addr)
		err = m.sendWelcome(msg.addr)
	}
	// register everyone who said hello
	if bytes.Equal(msg.cmd, cmdWelcome) {
		m.ins.add(msg.sender, msg.addr)
	}
	// confirm the on-line instance list
	if bytes.Equal(msg.cmd, cmdBroadCompare) {
		if bytes.Equal(m.ins.hashAllID(m.ID), msg.data) {
			err = m.sendCompared(msg.addr, msg.data)
		}
	}
	// someone bragged about a captured key
	if bytes.Equal(msg.cmd, cmdBroadMine) {
		if err = m.ins.save(msg.sender, string(msg.data), msg.addr); err != nil {
			err = m.sendReset(msg.data)
		} else {
			err = m.sendSaved(msg.addr, msg.data)
		}
	}
	return err
}
