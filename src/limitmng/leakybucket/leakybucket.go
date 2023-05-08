package leakybucket

import (
	"sync"
	"time"

	"github.com/iv-menshenin/lm/types"
)

type LeakyBucket struct {
	mux       sync.RWMutex
	threshold float64
	counter   float64
	windowSz  time.Duration
	bandwidth float64
	last      time.Time
}

func New(rph types.RPT, wsz time.Duration) *LeakyBucket {
	if wsz == 0 {
		wsz = adjustWindowSz(rph)
	}
	bwh := rph.PerTime(wsz)
	lb := LeakyBucket{
		threshold: bwh,
		windowSz:  wsz,
		bandwidth: bwh,
		last:      time.Now(),
	}
	return &lb
}

func adjustWindowSz(rph types.RPT) time.Duration {
	switch {
	case rph.PerSecond() > 60:
		return time.Second

	case rph.PerMinute() > 60:
		return 15 * time.Second

	default:
		return time.Minute
	}
}

func (b *LeakyBucket) Go() func() {
	var (
		once sync.Once
		stop = make(chan struct{})
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		for {
			select {
			case <-time.After(b.next()):
				b.tick()
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

func (b *LeakyBucket) next() time.Duration {
	if wait := b.windowSz - time.Since(b.last); wait > 0 {
		return wait
	}
	return 0
}

func (b *LeakyBucket) tick() {
	b.mux.Lock()
	if b.counter > 0 {
		b.counter -= b.bandwidth
	}
	b.last = time.Now().UTC()
	b.mux.Unlock()
}

func (b *LeakyBucket) GetLimit() bool {
	b.mux.Lock()
	defer b.mux.Unlock()
	if b.counter > b.threshold {
		return false
	}
	b.counter += 1
	return true
}
