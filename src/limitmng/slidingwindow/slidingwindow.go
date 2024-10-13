package slidingwindow

import (
	"sync"
	"time"

	"github.com/iv-menshenin/lm/types"
)

type SlidingWindow struct {
	mux       sync.RWMutex
	threshold float64
	counter   []float64
	windowSz  time.Duration
	last      time.Time
}

func New(rph types.RPT, sz time.Duration, div int) *SlidingWindow {
	var sl = SlidingWindow{
		threshold: rph.PerTime(sz),
		counter:   make([]float64, div),
		windowSz:  sz / time.Duration(div),
		last:      time.Now().UTC(),
	}
	return &sl
}

func (b *SlidingWindow) Go() func() {
	var (
		once sync.Once
		stop = make(chan struct{})
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		for {
			select {
			case <-time.After(b.windowSz):
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

func (b *SlidingWindow) tick() {
	b.mux.Lock()
	copy(b.counter[1:], b.counter[:len(b.counter)-1])
	b.counter[0] = 0
	b.mux.Unlock()
}

func (b *SlidingWindow) GetLimit() (fit bool) {
	b.mux.Lock()
	var counter float64
	for _, count := range b.counter {
		counter += count
	}
	if fit = counter < b.threshold; fit {
		b.counter[0]++
	}
	b.last = time.Now()
	b.mux.Unlock()
	return
}
