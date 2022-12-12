package mbgControlplane

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/mbgDataplane"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

func Connect(c protocol.ConnectRequest) (string, string, string) {
	//Update MBG state
	state.UpdateState()
	connectionID := c.Id + ":" + c.IdDest
	if state.IsServiceLocal(c.IdDest) {
		log.Infof("[MBG %v] Received Incoming Connect request from service: %v to service: %v", state.GetMyId(), c.Id, c.IdDest)
		// Get a free local/external port
		// Send the external port as reply to the MBG
		localSvc := state.GetLocalService(c.IdDest)

		myConnectionPorts, err := state.GetFreePorts(connectionID)
		if err != nil {
			log.Infof("[MBG %v] Error getting free ports %s", state.GetMyId(), err.Error())
			return err.Error(), "tcp", myConnectionPorts.External

		}
		log.Infof("[MBG %v] Using ConnectionPorts : %v", state.GetMyId(), myConnectionPorts)
		clusterIpPort := localSvc.Service.Ip
		// TODO Need to check Policy before accepting connections
		//ApplyGlobalPolicies
		//ApplyServicePolicies
		go ConnectService(myConnectionPorts.Local, clusterIpPort, c.Policy, connectionID)
		log.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, myConnectionPorts.External)
		return "Success", "tcp", myConnectionPorts.External

	} else { //For Remote service
		log.Infof("[MBG %v] Received Outgoing Connect request from service: %v to service: %v", state.GetMyId(), c.Id, c.IdDest)
		destSvc := state.GetRemoteService(c.IdDest)
		mbgIP := state.GetServiceMbgIp(destSvc.Service.Ip)
		//Send connection request to other MBG
		connectType, connectDest, err := ConnectReq(c.Id, c.IdDest, c.Policy, mbgIP)
		if err != nil && err.Error() != "Connection already setup!" {
			log.Infof("[MBG %v] Send connect failure to Cluster =%v ", state.GetMyId(), err.Error())
			return "Failure", "tcp", connectDest
		}
		log.Infof("[MBG %v] Using %v:%v to connect IP-%v", state.GetMyId(), connectType, connectDest, destSvc.Service.Ip)

		//Randomize listen ports for return
		myConnectionPorts, err := state.GetFreePorts(connectionID)
		if err != nil {
			log.Infof("[MBG %v] Error getting free ports %s", state.GetMyId(), err.Error())
			return err.Error(), "tcp", myConnectionPorts.External

		}
		log.Infof("[MBG %v] Using ConnectionPorts : %v", state.GetMyId(), myConnectionPorts)
		//Create data connection
		destIp := destSvc.Service.Ip + ":" + connectDest
		go ConnectService(myConnectionPorts.Local, destIp, c.Policy, connectionID)
		//Return a reply with to connect request
		log.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, myConnectionPorts.External)
		return "Success", "tcp", myConnectionPorts.External
	}
}

//Run server for Data connection - we have one server and client that we can add some network functions e.g: TCP-split
//By default we just forward the data
func ConnectService(svcListenPort, svcIp, policy, connName string) {

	srcIp := ":" + svcListenPort
	destIp := svcIp

	policyTarget := policyEngine.GetPolicyTarget(policy)
	if policyTarget == "" {
		// No Policy to be applied
		var forward mbgDataplane.MbgTcpForwarder

		forward.InitTcpForwarder(srcIp, destIp, connName)
		forward.RunTcpForwarder()
	} else {
		var ingress mbgDataplane.MbgTcpForwarder
		var egress mbgDataplane.MbgTcpForwarder

		ingress.InitTcpForwarder(srcIp, policyTarget, connName)
		egress.InitTcpForwarder(policyTarget, destIp, connName)
		ingress.RunTcpForwarder()
		egress.RunTcpForwarder()
	}

}

//Send control request to connect
func ConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) (string, string, error) {
	log.Printf("Start connect Request to MBG %v for service %v", mbgIp, svcIdDest)
	address := "http://" + mbgIp + "/connect"

	j, err := json.Marshal(protocol.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy})
	if err != nil {
		log.Fatal(err)
	}
	//Send connect
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
	log.Printf("[MBG %v] Failed to Connect : %s", state.GetMyId(), r.Message)
	if "Connection already setup!" == r.Message {
		return r.ConnectType, r.ConnectDest, fmt.Errorf("Connection already setup!")
	} else {
		return "", "", fmt.Errorf("Connect Request Failed")
	}

}
