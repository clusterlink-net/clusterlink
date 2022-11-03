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
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

var (
	listener          = flag.String("listen", ":5001", "ip:port listen for host")
	target            = flag.String("target", ":5003", "ip:port for destination")
	policy            = flag.String("policy", "Forward", "policy for traffic pass in Mbg")
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
	var serverTarget string
	if *policy == "Forward" {
		serverTarget = cListener
	} else if *policy == "TCP-split" {
		serverTarget = service.GetPolicyIp(*policy)
	} else {
		fmt.Println(*policy, "- Policy  not exist use Forward")
		serverTarget = cListener
	}
	s.InitMbgSwitch(*listener, serverTarget)
	c.InitClient(cListener, *target)

	go c.RunClient()
	s.RunMbgSwitch()

	// if s.MbgMode { //For mbg update the target
	//

}
