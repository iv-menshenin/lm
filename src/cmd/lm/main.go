package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/iv-menshenin/lm/udp"
)

func main() {
	l, err := udp.New(7666)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			r, err := l.Listen()
			if err != nil {
				log.Printf("ERROR: %+v\n", err)
				os.Exit(1)
			}
			log.Printf("RECEIVED: %s %s\n", r.Addr.String(), r.Addr.Network())
			log.Printf("DATA: %s\n", string(r.Data))
		}
	}()
	var idX [16]byte
	_, err = rand.Read(idX[:])
	if err != nil {
		panic(err)
	}
	var ID = hex.EncodeToString(idX[:])
	log.Printf("MYID: %s\n", ID)
	for {
		<-time.After(time.Second)
		data := fmt.Sprintf("MSGFROM:%s:PING", ID)
		if err = l.Send([]byte(data)); err != nil {
			log.Printf("ERROR: %+v\n", err)
			os.Exit(1)
		}
	}
}
