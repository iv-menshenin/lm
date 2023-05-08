package coordinator

import (
	"bytes"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var defaultAwaitTimeout = 50 * time.Millisecond

func (m *Manager) awaitResponses(cmd, data []byte) error {
	var wg sync.WaitGroup
	var cancel = make(chan struct{})
	var done = make(chan struct{})
	var timeout = make(chan struct{})

	go func() {
		select {
		case <-done:
		case <-time.After(defaultAwaitTimeout):
		}
		close(timeout)
		close(cancel)
	}()
	for _, v := range m.ins.getAllID() {
		wg.Add(1)
		var ch = make(chan struct{})
		go func(id int64) {
			select {
			case <-ch:
			case <-cancel:
			}
			m.awr.del(id)
			wg.Done()
		}(m.awr.add(append(append(append(make([]byte, 0, 20+len(data)), cmd...), v[:]...), data...), ch))
	}
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-timeout:
		return errors.New("timeout")
	}
}

func (m *Manager) awaitMostOf(cmd, data []byte) error {
	var kvorum = int64(m.ins.getCount() / 2)
	var wg sync.WaitGroup
	var cancel = make(chan struct{})
	var done = make(chan struct{})
	var kvo = make(chan struct{})
	var timeout = make(chan struct{})

	go func() {
		select {
		case <-done:
		case <-kvo:
		case <-time.After(defaultAwaitTimeout):
		}
		close(timeout)
		close(cancel)
	}()
	for _, v := range m.ins.getAllID() {
		wg.Add(1)
		var ch = make(chan struct{})
		go func(id int64) {
			select {
			case <-ch:
				if cnt := atomic.AddInt64(&kvorum, -1); cnt == -1 {
					close(kvo)
				}
			case <-cancel:
			}
			m.awr.del(id)
			wg.Done()
		}(m.awr.add(append(append(append(make([]byte, 0, 20+len(data)), cmd...), v[:]...), data...), ch))
	}
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-kvo:
		return nil
	case <-timeout:
		return errors.New("timeout")
	}
}

type (
	Awaiter struct {
		mux sync.RWMutex
		num int64
		wai []wait
	}
	wait struct {
		num  int64
		trg  int64
		data []byte
		done chan<- struct{}
	}
)

func newAwaiter() *Awaiter {
	return &Awaiter{}
}

func (a *Awaiter) trig(data []byte) {
	a.mux.RLock()
	for n := range a.wai {
		if atomic.LoadInt64(&a.wai[n].trg) > 0 {
			continue
		}
		if bytes.Equal(data, a.wai[n].data) {
			if atomic.CompareAndSwapInt64(&a.wai[n].trg, 0, 1) {
				close(a.wai[n].done)
			}
		}
	}
	a.mux.RUnlock()
}

func (a *Awaiter) add(data []byte, done chan<- struct{}) int64 {
	w := wait{
		num:  atomic.AddInt64(&a.num, 1),
		data: data,
		done: done,
	}
	a.mux.Lock()
	a.wai = append(a.wai, w)
	a.mux.Unlock()
	return w.num
}

func (a *Awaiter) del(num int64) {
	a.mux.Lock()
	var n int
	for n = 0; n < len(a.wai) && a.wai[n].num != num; {
		n++
	}
	if n < len(a.wai) {
		a.wai = append(a.wai[:n], a.wai[n+1:]...)
	}
	a.mux.Unlock()
}
