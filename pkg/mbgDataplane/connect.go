package mbgDataplane

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var clog = logrus.WithField("component", "mbgDataplane/Connect")

const TCP_TYPE = "tcp"
const MTLS_TYPE = "mtls"

func Connect(c protocol.ConnectRequest, mbgIP string) (string, string, string) {
	//Update MBG state
	state.UpdateState()
	connectionID := c.Id + ":" + c.IdDest
	if state.IsServiceLocal(c.IdDest) {
		clog.Infof("[MBG %v] Received Incoming Connect request from service: %v to service: %v", state.GetMyId(), c.Id, c.IdDest)
		dataplane := state.GetDataplane()
		switch dataplane {
		case TCP_TYPE:
			// Get a free local/external port
			// Send the external port as reply to the MBG
			localSvc := state.GetLocalService(c.IdDest)

			myConnectionPorts, err := state.GetFreePorts(connectionID)
			if err != nil {
				clog.Infof("[MBG %v] Error getting free ports %s", state.GetMyId(), err.Error())
				return "failure", "", ""
			}
			clog.Infof("[MBG %v] Using ConnectionPorts : %v", state.GetMyId(), myConnectionPorts)
			clusterIpPort := localSvc.Service.Ip
			// TODO Need to check Policy before accepting connections
			//ApplyGlobalPolicies
			//ApplyServicePolicies
			go ConnectService(myConnectionPorts.Local, clusterIpPort, c.Policy, connectionID)
			log.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, myConnectionPorts.External)
			return "Success", dataplane, myConnectionPorts.External
		case MTLS_TYPE:
			localSvc := state.GetLocalService(c.IdDest)
			// destSvc := state.GetRemoteService(c.Id)
			uid := ksuid.New()
			remoteEndPoint := connectionID + "-" + uid.String()
			mbgTarget := "https://" + mbgIP + ":8443/mbgData"
			certFile, keyFile := state.GetMbgCertsFromIp(mbgIP)
			clog.Infof("[MBG %v] Starting a Receiver service for %s Using RemoteEndpoint : %s/%s Certs(%s,%s)", state.GetMyId(),
				localSvc.Service.Ip, mbgTarget, remoteEndPoint, certFile, keyFile)

			go StartReceiverService(localSvc.Service.Ip, mbgTarget, remoteEndPoint, certFile, keyFile)
			return "Success", dataplane, remoteEndPoint
		default:
			return "failure", "", ""
		}
	} else { //For Remote service
		// This condition is applicable only for explicit connection request from a cluster.
		// Moving on, this condition would be deprecated since we would start a Cluster Service for every remote service
		// to initiate connect requests.
		log.Infof("[MBG %v] Received Outgoing Connect request from service: %v to service: %v", state.GetMyId(), c.Id, c.IdDest)
		destSvc := state.GetRemoteService(c.IdDest)
		mbgIP := state.GetServiceMbgIp(destSvc.Service.Ip)
		//Send connection request to other MBG
		connectType, connectDest, err := ConnectReq(c.Id, c.IdDest, c.Policy, mbgIP)
		if err != nil && err.Error() != "Connection already setup!" {
			clog.Infof("[MBG %v] Send connect failure to Cluster =%v ", state.GetMyId(), err.Error())
			return "Failure", "tcp", connectDest
		}
		clog.Infof("[MBG %v] Using %v:%v to connect IP-%v", state.GetMyId(), connectType, connectDest, destSvc.Service.Ip)

		//Randomize listen ports for return
		myConnectionPorts, err := state.GetFreePorts(connectionID)
		if err != nil {
			clog.Infof("[MBG %v] Error getting free ports %s", state.GetMyId(), err.Error())
			return err.Error(), "tcp", myConnectionPorts.External

		}
		clog.Infof("[MBG %v] Using ConnectionPorts : %v", state.GetMyId(), myConnectionPorts)
		//Create data connection
		destIp := destSvc.Service.Ip + ":" + connectDest
		go ConnectService(myConnectionPorts.Local, destIp, c.Policy, connectionID)
		//Return a reply with to connect request
		clog.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, myConnectionPorts.External)
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
		var forward MbgTcpForwarder

		forward.InitTcpForwarder(srcIp, destIp, connName)
		forward.RunTcpForwarder()
	} else {
		var ingress MbgTcpForwarder
		var egress MbgTcpForwarder

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
		clog.Fatal(err)
	}
	if r.Message == "Success" {
		clog.Printf("Successfully Connected : Using Connection:Port - %s:%s", r.ConnectType, r.ConnectDest)
		return r.ConnectType, r.ConnectDest, nil
	}
	clog.Printf("[MBG %v] Failed to Connect : %s", state.GetMyId(), r.Message)
	if "Connection already setup!" == r.Message {
		return r.ConnectType, r.ConnectDest, fmt.Errorf("Connection already setup!")
	} else {
		return "", "", fmt.Errorf("Connect Request Failed")
	}

}

// Start a Cluster Service which is a proxy for remote service
// It receives connections from local service and performs Connect API
// and sets up an mTLS forwarding to the remote service upon accepted (policy checks, etc)
func StartClusterService(serviceId, clusterServicePort, targetMbg, certificate, key string) error {
	acceptor, err := net.Listen("tcp", clusterServicePort)
	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		state.UpdateState()
		mlog.Infof("Receiving Outgoing connection %s->%s ", ac.RemoteAddr().String(), ac.LocalAddr().String())
		if err != nil {
			return err
		}

		// Ideally do a control plane connect API, Policy checks, and then create a mTLS forwarder
		// RemoteEndPoint has to be in the connect Request/Response

		localSvc, err := state.LookupLocalService(ac.RemoteAddr().String())
		if err != nil {
			log.Infof("Denying Outgoing connection%v", err)
			ac.Close()
			continue
		}
		log.Infof("[MBG %v] Accepting Outgoing Connect request from service: %v to service: %v", state.GetMyId(), localSvc.Service.Id, serviceId)

		destSvc := state.GetRemoteService(serviceId)
		mbgIP := state.GetServiceMbgIp(destSvc.Service.Ip)
		//Send connection request to other MBG
		connectType, connectDest, err := ConnectReq(localSvc.Service.Id, serviceId, "forward", mbgIP)
		if err != nil && err.Error() != "Connection already setup!" {
			log.Infof("[MBG %v] Send connect failure to Cluster = %v ", state.GetMyId(), err.Error())
			ac.Close()
			continue
		}
		log.Infof("[MBG %v] Using %s for  %s/%s to connect to Service-%v", state.GetMyId(), connectType, targetMbg, connectDest, destSvc.Service.Id)

		var mtlsForward MbgMtlsForwarder
		mtlsForward.InitmTlsForwarder(targetMbg, connectDest, certificate, key)

		mtlsForward.setSocketConnection(ac)
		go mtlsForward.dispatch(ac)
	}
}

// Receiver service is run at the cluster of the server service which receives connection from a remote service
func StartReceiverService(clusterServicePort, targetMbg, remoteEndPoint, certificate, key string) error {
	conn, err := net.Dial("tcp", clusterServicePort)
	if err != nil {
		return err
	}
	mlog.Infof("Receiver Connection at %s, %s", conn.LocalAddr().String(), remoteEndPoint)
	var mtlsForward MbgMtlsForwarder
	mtlsForward.InitmTlsForwarder(targetMbg, remoteEndPoint, certificate, key)
	mtlsForward.setSocketConnection(conn)
	go mtlsForward.dispatch(conn)
	return nil
}
