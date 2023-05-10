package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var wg sync.WaitGroup
	var ticker int64
	var errors int64
	var mux sync.Mutex
	var max [3]int
	defer func(tm time.Time) {
		spent := time.Since(tm)
		fmt.Printf("DONE %+v WITH %v (RPS %0.2f); ERRORS %d", max, spent, float64(ticker)/spent.Seconds(), errors)
	}(time.Now())
	for n := 0; n < 10; n++ {
		wg.Add(1)
		go func(sy int) {
			defer wg.Done()
			var cnt = 1000
			for {
				<-time.After(50 * time.Millisecond)
				resp, err := http.Get("http://192.168.1.204:8080/keys/increment?key=X" + strconv.Itoa(sy))
				if err != nil {
					atomic.AddInt64(&errors, 1)
					continue
				}
				if resp.StatusCode != 200 {
					resp.Body.Close()
					atomic.AddInt64(&errors, 1)
					continue
				}
				data, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					panic(err)
				}
				i, err := strconv.Atoi(strings.TrimSpace(strings.Split(string(data), ":")[1]))
				if err == nil {
					mux.Lock()
					if max[sy] < i {
						max[sy] = i
					}
					mux.Unlock()
				}
				if cnt--; cnt < 0 {
					break
				}
				atomic.AddInt64(&ticker, 1)
			}
		}(n % 3)
	}
	wg.Wait()
}
