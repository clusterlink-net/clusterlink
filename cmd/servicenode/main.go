package main

import (
	"flag"
	"fmt"
	"os"

	"github.ibm.com/ei-agent/pkg/client"
	"github.ibm.com/ei-agent/pkg/server"
)

var (
	listener          = flag.String("listen", ":5001", "listen host:port for client")
	maxDataBufferSize = 64 * 1024
)

func main() {
	var s server.SnServer
	var c client.SnClient

	flag.Parse()
	if *listener == "" {
		fmt.Println("missing listener")
		os.Exit(-1)
	}
	//init
	cListener := "localhost:5200"
	c.InitClient(cListener, "", false, "", "", "")
	s.SrverInit(*listener, "", true, &c)

	go c.RunClient()
	s.RunSrver()
}
