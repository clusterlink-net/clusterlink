/**********************************************************/
/* Package Main to run multi-cloud border gateway
/* by tunning mbg-switch and gateway
/**********************************************************/
package main

import (
	"flag"
	"fmt"
	"os"

	"github.ibm.com/mbg-agent/pkg/client"
	mbgSwitch "github.ibm.com/mbg-agent/pkg/mbg-switch"
)

var (
	listener          = flag.String("listen", ":5001", "listen host:port for client")
	maxDataBufferSize = 64 * 1024
)

func main() {
	var s mbgSwitch.MbgSwitch
	var c client.MbgClient

	flag.Parse()
	if *listener == "" {
		fmt.Println("missing listener")
		os.Exit(-1)
	}
	//init
	cListener := ":5000"
	c.InitClient(cListener, "", false, "", "", "")
	s.InitMbgSwitch(*listener, "", true, &c)

	go c.RunClient()
	s.RunMbgSwitch()
}
