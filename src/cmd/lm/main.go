package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/iv-menshenin/lm/coordinator"
	"github.com/iv-menshenin/lm/transport"
)

func main() {
	udp, err := transport.NewUDP(7999)
	if err != nil {
		panic(err)
	}
	defer udp.Close()
	c := coordinator.New(udp)
	c.SetLogLevel(coordinator.LogLevelDebug)
	var (
		keyStore = make(map[string]int64)
		keyMux   sync.Mutex
	)
	server := http.Server{
		Addr: ":8080",
		Handler: http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			var (
				err  error
				serv string
				key  = request.URL.Query().Get("key")
			)
			serv, err = c.CheckKey(request.Context(), key)
			if err != nil {
				log.Printf("ERROR %s: %+v\n", c.Key(), err)
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			if serv == coordinator.Mine {
				writer.Header().Set("Keep-Alive", "True")
				log.Printf("PROCESS: %s\n", c.Key())
				switch request.URL.Path {
				case "/keys/increment":
					keyMux.Lock()
					current := keyStore[key]
					current++
					keyStore[key] = current
					keyMux.Unlock()
					writer.WriteHeader(http.StatusOK)
					writer.Write([]byte(fmt.Sprintf("CURRENT FOR [%s]: %d", key, current)))
				case "/keys/decrement":
					keyMux.Lock()
					current := keyStore[key]
					current--
					keyStore[key] = current
					keyMux.Unlock()
					writer.WriteHeader(http.StatusOK)
					writer.Write([]byte(fmt.Sprintf("CURRENT FOR [%s]: %d", key, current)))
				case "/keys/value":
					keyMux.Lock()
					current := keyStore[key]
					keyMux.Unlock()
					writer.WriteHeader(http.StatusOK)
					writer.Write([]byte(fmt.Sprintf("CURRENT FOR [%s]: %d", key, current)))
				}
				return
			}
			req := request.Clone(context.Background())
			hostPort := strings.Split(serv, ":")
			req.Header.Add("Keep-Alive", "True")
			req.URL.Host = hostPort[0] + ":8080"
			req.URL.Scheme = "http"
			log.Printf("REDIRECT: %s\n", req.URL.String())
			resp, err := http.Get(req.URL.String())
			if err != nil {
				log.Printf("ERROR %s: %+v\n", c.Key(), err)
				writer.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()
			writer.WriteHeader(resp.StatusCode)
			io.Copy(writer, resp.Body)
			return
		}),
	}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			fmt.Printf("%+v\n", err)
			os.Exit(1)
		}
	}()
	if err := c.Manage(); err != nil {
		panic(err)
	}
}
