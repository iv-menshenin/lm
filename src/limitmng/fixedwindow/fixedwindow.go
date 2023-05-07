package fixedwindow

import (
	"sync"
	"time"

	"github.com/iv-menshenin/lm/src/types"
)

type FixedWindow struct {
	mux       sync.RWMutex
	threshold float64
	counter   float64
	windowSz  time.Duration
	last      time.Time
}

func New(rph types.RPT, wsz time.Duration) *FixedWindow {
	fw := FixedWindow{
		threshold: rph.PerTime(wsz),
		windowSz:  wsz,
		last:      time.Now(),
	}
	return &fw
}

func (b *FixedWindow) Go() func() {
	var (
		once sync.Once
		stop = make(chan struct{})
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		for {
			select {
			case <-time.After(10 * time.Millisecond):
				if time.Since(b.last) > b.windowSz {
					b.tick()
				}
			case <-stop:
				return
			}
		}
	}()
	return func() {
		once.Do(func() {
			close(stop)
			<-done
		})
	}
}

func (b *FixedWindow) tick() {
	b.mux.Lock()
	b.counter = 0
	b.last = time.Now().UTC()
	b.mux.Unlock()
}

func (b *FixedWindow) GetLimit() bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	if b.counter > b.threshold {
		return false
	}
	b.counter += 1
	return true
}
