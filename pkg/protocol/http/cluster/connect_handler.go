package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/cmd/cluster/state"
	"github.ibm.com/mbg-agent/pkg/clusterProxy"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func ConnectReq(svcId, svcIdDest, svcPolicy, mbgIP string) (string, string, error) {
	log.Printf("Start connect Request to MBG %v for service %v", svcIdDest, mbgIP)

	address := "http://" + mbgIP + "/connect"
	j, err := json.Marshal(protocol.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy})
	if err != nil {
		log.Fatal(err)
	}
	//send connect
	resp := httpAux.HttpPost(address, j)
	var r protocol.ConnectReply
	err = json.Unmarshal(resp, &r)
	if err != nil {
		log.Fatal(err)
	}
	if r.Message == "Success" {
		log.Printf("Successfully Connected : Using Connection:Port - %s:%s", r.ConnectType, r.ConnectDest)
		return r.ConnectType, r.ConnectDest, nil
	}

	log.Printf("[Cluster %v] Failed to Connect : %s port %s", state.GetId(), r.Message, r.ConnectDest)
	if "Connection already setup!" == r.Message {
		return r.ConnectType, r.ConnectDest, fmt.Errorf(r.Message)
	} else {
		return "", "", fmt.Errorf("Connect Request Failed")
	}

}

func ConnectClient(svcId, svcIdDest, sourceIp, destIp, connName string) {
	var c clusterProxy.ProxyClient
	var stopChan = make(chan os.Signal, 2) //creating channel for interrupt
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	state.AddOpenConnection(svcId, svcIdDest, os.Getpid())

	c.InitClient(sourceIp, destIp, connName)
	done := &sync.WaitGroup{}
	done.Add(1)
	go c.RunClient(done)

	<-stopChan // wait for SIGINT
	log.Infof("Receive SIGINT for connection from %v to %v \n", svcId, svcIdDest)
	c.CloseConnection()
	done.Wait()
	log.Infof("Connection from %v to %v is close\n", svcId, svcIdDest)

}
