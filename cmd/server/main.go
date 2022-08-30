package main

import (
	"flag"
	"fmt"
	"os"

	"github.ibm.com/ei-agent/pkg/server"
)

var (
	listener    = flag.String("listen", ":5001", "listen host:port for client")
	servicenode = flag.String("sn", "", "listen host:port of server side service node")
)

func main() {
	var s server.SnServer

	flag.Parse()
	if *listener == "" || *servicenode == "" {
		fmt.Println("missing listener or service")
		os.Exit(-1)
	}
	s.SrverInit(*listener, *servicenode, false, nil)
	s.RunSrver()
}
