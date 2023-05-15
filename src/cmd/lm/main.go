package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/iv-menshenin/choreo/fleetctrl"
	"github.com/iv-menshenin/choreo/transport"
)

func main() {
	udp, err := transport.NewUDP(7999)
	if err != nil {
		panic(err)
	}
	defer udp.Close()
	c := fleetctrl.New(udp)
	c.SetLogLevel(fleetctrl.LogLevelDebug)
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
			if serv == fleetctrl.Mine {
				var current int64
				writer.Header().Set("Keep-Alive", "300")
				log.Printf("PROCESS: %s\n", c.Key())
				switch request.URL.Path {
				case "/keys/increment":
					keyMux.Lock()
					current = keyStore[key]
					current++
					keyStore[key] = current
					keyMux.Unlock()
					writer.WriteHeader(http.StatusOK)

				case "/keys/decrement":
					keyMux.Lock()
					current = keyStore[key]
					current--
					keyStore[key] = current
					keyMux.Unlock()
					writer.WriteHeader(http.StatusOK)

				case "/keys/value":
					keyMux.Lock()
					current = keyStore[key]
					keyMux.Unlock()
					writer.WriteHeader(http.StatusOK)

				default:
					writer.WriteHeader(http.StatusNotFound)
					log.Printf("NOT FOUND: %s\n", request.URL.Path)
					return
				}
				if _, err = writer.Write([]byte(fmt.Sprintf("CURRENT FOR [%s]: %d", key, current))); err != nil {
					log.Printf("ERROR %s: %+v\n", c.Key(), err)
				}
				return
			}
			req := request.Clone(context.Background())
			hostPort := strings.Split(serv, ":")
			req.Header.Add("Keep-Alive", "300")
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
			writer.Header().Set("Keep-Alive", "300")
			writer.WriteHeader(resp.StatusCode)
			if _, err = io.Copy(writer, resp.Body); err != nil {
				log.Printf("ERROR %s: %+v\n", c.Key(), err)
			}
		}),
	}
	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("%+v\n", err)
			os.Exit(1)
		}
	}()
	go func() {
		if err := c.Manage(); err != nil {
			panic(err)
		}
	}()
	var sig = make(chan os.Signal, 16)
	signal.Notify(sig, os.Interrupt)
	<-sig
	log.Println("INTERRUPTED")
	c.Stop()
	if err = server.Close(); err != nil {
		panic(err)
	}
}
