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
		rph = 3600000
		wsz = 50 * time.Millisecond
	)
	t.Run("leakybucket", func(t *testing.T) {
		t.Parallel()
		var lb = leakybucket.New(rph, wsz)
		defer lb.Go()()
		testLimit(t, lb, time.Second, 1000, 1)
	})
	t.Run("tokenbucket", func(t *testing.T) {
		t.Parallel()
		var lb = tokenbucket.New(rph, wsz)
		defer lb.Go()()
		testLimit(t, lb, time.Second, 1000, 1)
	})
	t.Run("fixedwindow", func(t *testing.T) {
		t.Parallel()
		// used a longer testing time because of a boundary issue
		var thisTestTime = time.Second + 10*time.Millisecond
		var lb = fixedwindow.New(types.PerTime(1000, thisTestTime), wsz)
		defer lb.Go()()
		testLimit(t, lb, thisTestTime, 1000, 5.1)
	})
	t.Run("slidingwindow", func(t *testing.T) {
		t.Parallel()
		// there is less of a boundary problem for a sliding window than for a fixed window, but it is also present
		var thisTestTime = time.Second + 10*time.Millisecond
		var lb = slidingwindow.New(types.PerTime(1000, time.Second), wsz, 5)
		defer lb.Go()()
		testLimit(t, lb, thisTestTime, 1000, 5.1)
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
