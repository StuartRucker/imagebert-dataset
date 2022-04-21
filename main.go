package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

func main() {
	maskify_init()

	// Read Command line arguments to detect if this is a worker or leader
	if os.Args[1] == "leader" {
		leader := new(Leader)

		leader.urls = []string{"https://lukesmith.xyz/"}
		// append the same url to the list 100x
		// for i := 0; i < 100; i++ {
		// 	leader.urls = append(leader.urls, "https://lukesmith.xyz/")
		// }
		leader.used = make(map[string]bool)

		go leader.run()

		err := rpc.Register(leader)
		if err != nil {
			log.Fatal("Format of service Task isn't correct. ", err)
		}
		// Register a HTTP handler
		rpc.HandleHTTP()
		// Listen to TPC connections on port 1234
		listener, e := net.Listen("tcp", ":1234")
		if e != nil {
			log.Fatal("Listen error: ", e)
		}
		log.Printf("Serving RPC server on port %d", 1234)
		// Start accept incoming HTTP connections
		err = http.Serve(listener, nil)
		if err != nil {
			log.Fatal("Error serving: ", err)
		}

	} else {

		// Create a TCP connection to localhost on port 1234
		client, err := rpc.DialHTTP("tcp", "localhost:1234")
		if err != nil {
			log.Fatal("Connection error: ", err)
		}
		worker := Worker{client: client}
		worker.run()

	}
}
