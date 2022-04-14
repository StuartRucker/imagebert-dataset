package main

import (
	"os"
)

func main() {

	// Read Command line arguments to detect if this is a worker or leader
	if os.Args[1] == "leader" {
		BeLeader()
	} else {
		BeWorker()
	}
}
