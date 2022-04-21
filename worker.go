package main

import (
	"fmt"
	"log"
	"net/rpc"
	"sync"
	"time"
)

type Worker struct {
	client              *rpc.Client
	currentlyActive     int
	currentlyActiveLock sync.Mutex
}

func (w *Worker) run() {
	for {
		if w.currentlyActive > 15 {
			time.Sleep(time.Second * 1)
			continue
		}

		args := GetURLArgs{}
		reply := GetURLReply{}
		err := w.client.Call("Leader.GetURL", &args, &reply)
		if err != nil {
			log.Fatal("RPC Error:", err)
		}
		url := reply.URL
		if url == "" {

			time.Sleep(time.Second)
			continue
		}

		fmt.Printf("Worker processing url: %s\n", url)

		go func() {
			// process the url
			w.currentlyActiveLock.Lock()
			w.currentlyActive++
			w.currentlyActiveLock.Unlock()

			processWebsite(url)

			w.currentlyActiveLock.Lock()
			w.currentlyActive--
			w.currentlyActiveLock.Unlock()

		}()

		time.Sleep(time.Millisecond * 10)

	}

}
