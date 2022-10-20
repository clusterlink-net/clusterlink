/**********************************************************/
/* Package Main to run client gateway in the host side
/**********************************************************/
package main

import (
	"flag"
	"fmt"
	"os"

	client "github.ibm.com/mbg-agent/pkg/client"
	service "github.ibm.com/mbg-agent/pkg/serviceMap"
)

var (
	listener    = flag.String("listen", ":5001", "listen host:port for client")
	target      = flag.String("mbg", "", "listen host:port of server side mbg")
	appDestPort = flag.String("destPort", "5003", "Destination IP")
	appDestIp   = flag.String("destIp", "127.0.0.1", "Destination port")
	serviceType = flag.String("service", "Forward", "choose service type")
)

func main() {
	var c client.MbgClient
	flag.Parse()
	fmt.Println("********** Start Gateway ***********")
	if *listener == "" || *target == "" {
		fmt.Println("missing listener or service")
		os.Exit(-1)
	}

	if !service.CheckServiceExist(*serviceType) {
		fmt.Println("[Error]: Service not exist:", *serviceType)
		os.Exit(1)
	}

	c.InitClient(*listener, *target, true, *appDestPort, *appDestIp, *serviceType)
	c.RunClient()
}
