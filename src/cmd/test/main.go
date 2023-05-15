package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	const keysCount = 6
	var wg sync.WaitGroup
	var ticker int64
	var errors int64
	var mux sync.Mutex
	var max [keysCount]int
	defer func(tm time.Time) {
		spent := time.Since(tm)
		fmt.Printf("DONE %+v WITH %v (RPS %0.2f); ERRORS %d", max, spent, float64(ticker)/spent.Seconds(), errors)
	}(time.Now())
	for n := 0; n < 64; n++ {
		wg.Add(1)
		go func(sy int) {
			defer wg.Done()
			var cnt = 1000
			for {
				<-time.After(100 * time.Millisecond)
				u, _ := url.Parse("http://192.168.1.204:8080/keys/increment")
				u.Query().Set("key", fmt.Sprintf("XCC-%d", sy))
				req, _ := http.NewRequest(http.MethodGet, u.String(), nil)
				req.Header.Set("Keep-Alive", "timeout=600")
				req.Header.Set("Connection", "Keep-Alive")
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					atomic.AddInt64(&errors, 1)
					continue
				}
				if resp.StatusCode != 200 {
					resp.Body.Close()
					atomic.AddInt64(&ticker, 1)
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
		}(n % keysCount)
	}
	wg.Wait()
}
