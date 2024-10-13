package limitmng

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/iv-menshenin/lm/limitmng/fixedwindow"
	"github.com/iv-menshenin/lm/limitmng/leakybucket"
	"github.com/iv-menshenin/lm/limitmng/slidingwindow"
	"github.com/iv-menshenin/lm/limitmng/tokenbucket"
	"github.com/iv-menshenin/lm/types"
)

func Test_LimitPrimitives(t *testing.T) {
	t.Parallel()
	const (
		wsz = 500 * time.Millisecond
		tst = 10 * time.Second
		cnt = 10000
	)
	rph := types.PerSecond(1000)
	t.Run("leakybucket", func(t *testing.T) {
		t.Parallel()
		var lb = leakybucket.New(rph, wsz)
		defer lb.Go()()
		testLimit(t, lb, tst, cnt, 2)
	})
	t.Run("tokenbucket", func(t *testing.T) {
		t.Parallel()
		var lb = tokenbucket.New(rph, wsz)
		defer lb.Go()()
		testLimit(t, lb, tst, cnt, 2)
	})
	t.Run("fixedwindow", func(t *testing.T) {
		t.Parallel()
		var lb = fixedwindow.New(rph, wsz)
		defer lb.Go()()
		testLimit(t, lb, tst, cnt, 2)
	})
	t.Run("slidingwindow", func(t *testing.T) {
		t.Parallel()
		var lb = slidingwindow.New(rph, wsz, 5)
		defer lb.Go()()
		testLimit(t, lb, tst, cnt, 2)
	})
}

type Limit interface {
	GetLimit() bool
}

func testLimit(t *testing.T, obj Limit, dur time.Duration, count, allowedDiffPerc float64) {
	t.Helper()
	var (
		testTimeout = make(chan struct{})
		testCounter int64
	)
	go func() {
		<-time.After(dur)
		close(testTimeout)
	}()
	var wg sync.WaitGroup
	for n := 0; n < 10; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-time.After(50 * time.Microsecond):
					if obj.GetLimit() {
						atomic.AddInt64(&testCounter, 1)
					}
					continue
				case <-testTimeout:
				}
				break
			}
		}()
	}
	wg.Wait()
	if perc := 100 * math.Abs(1-(float64(testCounter)/count)); perc > allowedDiffPerc {
		t.Errorf("counter: %d (%0.1f %%)", testCounter, perc)
	}
}
