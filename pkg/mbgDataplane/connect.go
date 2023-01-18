package mbgDataplane

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var clog = logrus.WithField("component", "mbgDataplane/Connect")

const TCP_TYPE = "tcp"
const MTLS_TYPE = "mtls"

/***************** Local Service function **********************************/

func Connect(c protocol.ConnectRequest, targetMbgIP string, conn net.Conn) (string, string, string) {
	//Update MBG state
	state.UpdateState()
	if state.IsServiceLocal(c.IdDest) {
		return StartProxyLocalService(c, targetMbgIP, conn)
	} else { //For Remote service
		clog.Errorf("Serivce %s is not exist", c.IdDest)
		return "failure", "", ""
	}
}

//ConnectLocalService waiting for connection from host and do two things:
//1. Create tcp connection to destination (Not Secure)- TODO support also secure connection
//2. Register new handle function and hijack the connection
func StartProxyLocalService(c protocol.ConnectRequest, targetMbgIP string, conn net.Conn) (string, string, string) {
	clog.Infof("[MBG %v] Received Incoming Connect request from service: %v to service: %v", state.GetMyId(), c.Id, c.IdDest)
	connectionID := c.Id + ":" + c.IdDest
	dataplane := state.GetDataplane()
	localSvc := state.GetLocalService(c.IdDest)
	mbgTarget := state.GetMbgTarget(c.MbgID)
	policyResp, err := state.GetEventManager().RaiseNewConnectionRequestEvent(eventManager.ConnectionRequestAttr{SrcService: c.Id, DstService: c.IdDest, Direction: eventManager.Incoming, OtherMbg: c.MbgID})
	if err != nil {
		clog.Errorf("[MBG %v] Unable to raise connection request event", state.GetMyId())
		return "failure", "", ""
	}
	if policyResp.Action == eventManager.Deny {
		return "failure", "", ""
	}

	switch dataplane {
	case TCP_TYPE:
		clog.Infof("[MBG %v] Sending Connect reply to Connection(%v) to use Dest:%v", state.GetMyId(), connectionID, "use connect hijack")
		go StartTcpProxyService("use connect mode", localSvc.Service.Ip, c.Policy, connectionID, conn, nil)
		return "Success", dataplane, "use connect mode"
	case MTLS_TYPE:
		uid := ksuid.New()
		remoteEndPoint := connectionID + "-" + uid.String()
		clog.Infof("[MBG %v] Starting a Receiver service for %s Using RemoteEndpoint : %s/%s", state.GetMyId(),
			localSvc.Service.Ip, mbgTarget, remoteEndPoint)

		go StartMtlsProxyLocalService(localSvc.Service.Ip, mbgTarget, remoteEndPoint)
		return "Success", dataplane, remoteEndPoint
	default:
		return "failure", "", ""
	}
}

// Receiver service is run at the mbg which receives connection from a remote service
func StartMtlsProxyLocalService(localServicePort, targetMbgIPPort, remoteEndPoint string) error {
	clog.Info("Start dial to service port ", localServicePort)
	conn, err := net.Dial("tcp", localServicePort) //Todo - support destination with secure connection
	if err != nil {
		return err
	}
	clog.Infof("[MBG %v] Receiver Connection at %s, %s", state.GetMyId(), conn.LocalAddr().String(), remoteEndPoint)
	mtlsForward := MbgMtlsForwarder{ChiRouter: state.GetChiRouter()}
	mtlsForward.StartMtlsForwarderServer(targetMbgIPPort, remoteEndPoint, "", "", "", conn)
	return nil
}

//Run server for Data connection - we have one server and client that we can add some network functions e.g: TCP-split
//By default we just forward the data
func StartTcpProxyService(svcListenPort, svcIp, policy, connName string, serverConn, clientConn net.Conn) {

	srcIp := svcListenPort
	destIp := svcIp

	policyTarget := policyEngine.GetPolicyTarget(policy)
	if policyTarget == "" {
		// No Policy to be applied
		var forward MbgTcpForwarder
		forward.InitTcpForwarder(srcIp, destIp, connName)
		if serverConn != nil {
			forward.SetServerConnection(serverConn)
		}
		if clientConn != nil {
			forward.SetClientConnection(clientConn)
		}
		forward.RunTcpForwarder()
	} else {
		var ingress MbgTcpForwarder
		var egress MbgTcpForwarder

		ingress.InitTcpForwarder(srcIp, policyTarget, connName)
		egress.InitTcpForwarder(policyTarget, destIp, connName)
		if serverConn != nil {
			ingress.SetServerConnection(serverConn)
		}
		if clientConn != nil {
			egress.SetServerConnection(clientConn)
		}
		ingress.RunTcpForwarder()
		egress.RunTcpForwarder()
	}

}

/***************** Remote Service function **********************************/

// Start a Local Service which is a proxy for remote service
// It receives connections from local service and performs Connect API
// and sets up an mTLS forwarding to the remote service upon accepted (policy checks, etc)
func StartProxyRemoteService(serviceId, localServicePort, targetMbgIPPort, rootCA, certificate, key string) error {
	clog.Infof("Start to listen to %v ", localServicePort)
	var err error
	var acceptor net.Listener
	dataplane := state.GetDataplane()
	acceptor, err = net.Listen("tcp", localServicePort) //TODO- need to support secure endpoint

	if err != nil {
		return err
	}
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		state.UpdateState()
		clog.Infof("Receiving Outgoing connection %s->%s ", ac.RemoteAddr().String(), ac.LocalAddr().String())
		if err != nil {
			return err
		}

		// Ideally do a control plane connect API, Policy checks, and then create a mTLS forwarder
		// RemoteEndPoint has to be in the connect Request/Response

		localSvc, err := state.LookupLocalService(ac.RemoteAddr().String())
		if err != nil {
			clog.Infof("Denying Outgoing connection from: %v ,Error: %v", ac.RemoteAddr().String(), err)
			ac.Close()
			continue
		}
		clog.Infof("[MBG %v] Accepting Outgoing Connect request from service: %v to service: %v", state.GetMyId(), localSvc.Service.Id, serviceId)

		destSvc := state.GetRemoteService(serviceId)
		mbgIP := state.GetServiceMbgIp(destSvc.Service.Ip)

		switch dataplane {
		case TCP_TYPE:
			connDest, err := tcpConnectReq(localSvc.Service.Id, serviceId, "forward", mbgIP)

			if err != nil && err.Error() != "Connection already setup!" {
				clog.Infof("[MBG %v] Send connect failure to mbgctl = %v ", state.GetMyId(), err.Error())
				ac.Close()
				continue
			}
			connectDest := "Use open connect socket" //not needed ehr we use connect - destSvc.Service.Ip + ":" + connectDest
			clog.Infof("[MBG %v] Using %s for  %s/%s to connect to Service-%v", state.GetMyId(), dataplane, targetMbgIPPort, connectDest, destSvc.Service.Id)
			connectionID := localSvc.Service.Id + ":" + destSvc.Service.Id
			go StartTcpProxyService(localServicePort, connectDest, "forward", connectionID, ac, connDest)

		case MTLS_TYPE:
			mtlsForward := MbgMtlsForwarder{ChiRouter: state.GetChiRouter()}

			//Send connection request to other MBG
			connectType, connectDest, err := mtlsConnectReq(localSvc.Service.Id, serviceId, "forward", mbgIP)

			if err != nil && err.Error() != "Connection already setup!" {
				clog.Infof("[MBG %v] Key Send connect failure to mbgctl = %v ", state.GetMyId(), err.Error())
				ac.Close()
				continue
			}
			clog.Infof("[MBG %v] Using %s for  %s/%s to connect to Service-%v", state.GetMyId(), connectType, targetMbgIPPort, connectDest, destSvc.Service.Id)
			mtlsForward.StartMtlsForwarderClient(targetMbgIPPort, connectDest, rootCA, certificate, key, ac)
		default:
			clog.Errorf("%v -Not supported", dataplane)

		}
	}
}

//Send control request to connect
func mtlsConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) (string, string, error) {
	clog.Infof("Start connect Request to MBG %v for service %v", mbgIp, svcIdDest)
	address := state.GetAddrStart() + mbgIp + "/connect"

	j, err := json.Marshal(protocol.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy, MbgID: state.GetMyId()})
	if err != nil {
		clog.Error(err)
		return "", "", err
	}
	//Send connect
	resp := httpAux.HttpPost(address, j, state.GetHttpClient())
	var r protocol.ConnectReply
	err = json.Unmarshal(resp, &r)
	if err != nil {
		clog.Error(err)
		return "", "", err
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

func tcpConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) (net.Conn, error) {
	clog.Printf("Start connect Request to MBG %v for service %v", mbgIp, svcIdDest)
	url := state.GetAddrStart() + mbgIp + "/connect"

	jsonData, err := json.Marshal(protocol.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy, MbgID: state.GetMyId()})
	if err != nil {
		clog.Error(err)
		return nil, err
	}
	c, resp := httpAux.HttpConnect(mbgIp, url, string(jsonData))
	if resp == nil {
		clog.Printf("Successfully Connected using connect method")
		return c, nil
	}

	if "Connection already setup!" == resp.Error() {
		return c, fmt.Errorf("Connection already setup!")
	} else {
		return nil, fmt.Errorf("Connect Request Failed")
	}
}
