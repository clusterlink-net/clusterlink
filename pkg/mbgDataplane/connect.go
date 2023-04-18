package mbgDataplane

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	"github.ibm.com/mbg-agent/pkg/deployment/kubernetes"
	"github.ibm.com/mbg-agent/pkg/eventManager"
	"github.ibm.com/mbg-agent/pkg/protocol"
	httpAux "github.ibm.com/mbg-agent/pkg/protocol/http/aux_func"
)

var clog *logrus.Entry

const TCP_TYPE = "tcp"
const MTLS_TYPE = "mtls"

/***************** Local Service function **********************************/

func Connect(c protocol.ConnectRequest, targetMbgIP string, w http.ResponseWriter) (bool, string, string) {
	//Update MBG state
	state.UpdateState()
	clog = logrus.WithFields(logrus.Fields{
		"component": state.GetMyId() + "-Dataplane",
	})
	if state.IsServiceLocal(c.IdDest) {
		return StartProxyLocalService(c, targetMbgIP, w)
	} else { //For Remote service
		clog.Errorf("Service %s does not exist", c.IdDest)
		return false, "", ""
	}
}

// ConnectLocalService waiting for connection from host and do two things:
// 1. Create tcp connection to destination (Not Secure)- TODO support also secure connection
// 2. Register new handle function and hijack the connection
func StartProxyLocalService(c protocol.ConnectRequest, targetMbgIP string, w http.ResponseWriter) (bool, string, string) {
	clog.Infof("Received Incoming Connect request from service: %v to service: %v", c.Id, c.IdDest)
	connectionID := c.Id + ":" + c.IdDest
	dataplane := state.GetDataplane()
	localSvc := state.GetLocalService(c.IdDest)
	policyResp, err := state.GetEventManager().RaiseNewConnectionRequestEvent(eventManager.ConnectionRequestAttr{SrcService: c.Id, DstService: c.IdDest, Direction: eventManager.Incoming, OtherMbg: c.MbgID})
	if err != nil {
		clog.Error("Unable to raise connection request event ", state.GetMyId())
		return false, "", ""
	}
	if policyResp.Action == eventManager.Deny {
		clog.Infof("Denying incoming connect request (%s,%s) due to policy", c.Id, c.IdDest)
		return false, "", ""
	}

	mbgTarget := state.GetMbgTarget(c.MbgID)

	switch dataplane {
	case TCP_TYPE:
		clog.Infof("Sending Connect reply to Connection(%v) to use Dest:%v", connectionID, "use connect hijack")
		conn := hijackConn(w)
		if conn == nil {
			clog.Error("Hijack Failure")
			return false, "", ""
		}
		go StartTcpProxyService("httpconnect", localSvc.GetIpAndPort(), c.Policy, connectionID, conn, nil)
		return true, dataplane, "httpconnect"
	case MTLS_TYPE:
		uid := ksuid.New()
		remoteEndPoint := connectionID + "-" + uid.String()
		clog.Infof("Starting a Receiver service for %s Using RemoteEndpoint : %s/%s",
			localSvc.Ip, mbgTarget, remoteEndPoint)

		go StartMtlsProxyLocalService(localSvc.GetIpAndPort(), mbgTarget, remoteEndPoint)
		return true, dataplane, remoteEndPoint
	default:
		return false, "", ""
	}
}

// Receiver service is run at the mbg which receives connection from a remote service
func StartMtlsProxyLocalService(localServicePort, targetMbgIPPort, remoteEndPoint string) error {
	conn, err := net.Dial("tcp", localServicePort) //Todo - support destination with secure connection
	if err != nil {
		return err
	}
	clog.Infof("Received new Connection at %s, %s", conn.LocalAddr().String(), remoteEndPoint)
	mtlsForward := MbgMtlsForwarder{ChiRouter: state.GetChiRouter()}
	mtlsForward.StartMtlsForwarderServer(targetMbgIPPort, remoteEndPoint, "", "", "", conn)
	return nil
}

// Run server for Data connection - we have one server and client that we can add some network functions e.g: TCP-split
// By default we just forward the data
func StartTcpProxyService(svcListenPort, svcIp, policy, connName string, serverConn, clientConn net.Conn) {

	srcIp := svcListenPort
	destIp := svcIp

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
}

/***************** Remote Service function **********************************/

// Start a Local Service which is a proxy for remote service
// It receives connections from local service and performs Connect API
// and sets up an mTLS forwarding to the remote service upon accepted (policy checks, etc)
func CreateProxyRemoteService(serviceId, servicePort, rootCA, certificate, key string) {
	acceptor, err := net.Listen("tcp", servicePort) //TODO- need to support secure endpoint
	if err != nil {
		clog.Infof("Error Listen: to port  %v", err)

	}
	go StartProxyRemoteService(serviceId, acceptor, servicePort, rootCA, certificate, key)
	state.WaitServiceStopCh(serviceId, servicePort)
	acceptor.Close()
}
func StartProxyRemoteService(serviceId string, acceptor net.Listener, servicePort, rootCA, certificate, key string) error {
	dataplane := state.GetDataplane()
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		state.UpdateState()
		clog = logrus.WithFields(logrus.Fields{
			"component": state.GetMyId() + "-Dataplane",
		})
		if err != nil {
			clog.Infof("Accept() returned error: %v", err)
			return err
		}
		appIp := ac.RemoteAddr().String()
		clog.Infof("Receiving Outgoing connection %s->%s ", ac.RemoteAddr().String(), ac.LocalAddr().String())

		// Ideally do a control plane connect API, Policy checks, and then create a mTLS forwarder
		// RemoteEndPoint has to be in the connect Request/Response
		appLabel, err := kubernetes.Data.GetLabel(strings.Split(ac.RemoteAddr().String(), ":")[0], kubernetes.APP_LABEL)
		if err != nil {
			clog.Errorf("Unable to get App Info :%+v", err)
		}
		clog.Infof("App Label:%s", appLabel)
		// Need to look up the label to find local service
		// If label isnt found, Check for IP.
		// If we cant find the service, we get the "service id" as a wildcard
		// which is sent to the policy engine to decide.
		localSvc, err := state.LookupLocalService(appLabel, appIp)

		policyResp, err := state.GetEventManager().RaiseNewConnectionRequestEvent(eventManager.ConnectionRequestAttr{SrcService: localSvc.Id, DstService: serviceId, Direction: eventManager.Outgoing, OtherMbg: eventManager.Wildcard})
		if err != nil {
			clog.Errorf("Unable to raise connection request event")
			ac.Close()
			continue
		}
		if policyResp.Action == eventManager.Deny {
			clog.Infof("Denying Outgoing connection due to policy")
			ac.Close()
			continue
		}

		clog.Infof("Accepting Outgoing Connect request from service: %v to service: %v", localSvc.Id, serviceId)

		destSvc := state.GetRemoteService(serviceId)[0]
		var mbgIP string
		if policyResp.TargetMbg == "" {
			// Policy Agent hasnt suggested anything any target MBG, hence we fall back to our defaults
			mbgIP = destSvc.MbgIp
		} else {
			mbgIP = state.GetMbgTarget(policyResp.TargetMbg)
		}
		switch dataplane {
		case TCP_TYPE:
			connDest, err := tcpConnectReq(localSvc.Id, serviceId, "forward", mbgIP)

			if err != nil {
				clog.Infof("Unable to connect(tcp): %v ", err.Error())
				ac.Close()
				continue
			}
			connectDest := "Use open connect socket" //not needed ehr we use connect - destSvc.Service.Ip + ":" + connectDest
			clog.Infof("Using %s for  %s/%s to connect to Service-%v", dataplane, mbgIP, connectDest, destSvc.Id)
			connectionID := localSvc.Id + ":" + destSvc.Id
			go StartTcpProxyService(servicePort, connectDest, "forward", connectionID, ac, connDest)

		case MTLS_TYPE:
			mtlsForward := MbgMtlsForwarder{ChiRouter: state.GetChiRouter()}

			//Send connection request to other MBG
			connectType, connectDest, err := mtlsConnectReq(localSvc.Id, serviceId, "forward", mbgIP)

			if err != nil {
				clog.Infof("Unable to connect(mtls): %v ", err.Error())
				ac.Close()
				continue
			}
			clog.Infof("Using %s for  %s/%s to connect to Service-%v", connectType, mbgIP, connectDest, destSvc.Id)
			serverName := state.GetMyId()
			mtlsForward.StartMtlsForwarderClient(mbgIP, connectDest, rootCA, certificate, key, serverName, ac)
		default:
			clog.Errorf("%v -Not supported", dataplane)

		}
	}
}

// Send control request to connect
func mtlsConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) (string, string, error) {
	clog.Infof("Starting mTLS Connect Request to MBG at %v for Service %v", mbgIp, svcIdDest)
	address := state.GetAddrStart() + mbgIp + "/connect"

	j, err := json.Marshal(protocol.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy, MbgID: state.GetMyId()})
	if err != nil {
		clog.Error(err)
		return "", "", err
	}
	//Send connect
	resp, err := httpAux.HttpPost(address, j, state.GetHttpClient())
	if err != nil {
		clog.Error(err)
		return "", "", err
	}
	var r protocol.ConnectReply
	err = json.Unmarshal(resp, &r)
	if err != nil {
		clog.Error(err)
		return "", "", err
	}
	if r.Connect == true {
		clog.Infof("Successfully Connected : Using Connection:Port - %s:%s", r.ConnectType, r.ConnectDest)
		return r.ConnectType, r.ConnectDest, nil
	}
	clog.Infof("Failed to Connect")

	return "", "", fmt.Errorf("failed to connect")
}

func tcpConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) (net.Conn, error) {
	clog.Printf("Starting TCP Connect Request to MBG at %v for service %v", mbgIp, svcIdDest)
	url := state.GetAddrStart() + mbgIp + "/connect"

	jsonData, err := json.Marshal(protocol.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy, MbgID: state.GetMyId()})
	if err != nil {
		clog.Error(err)
		return nil, err
	}
	c, resp := httpAux.HttpConnect(mbgIp, url, string(jsonData))
	if resp == nil {
		clog.Printf("Successfully Connected")
		return c, nil
	}

	return nil, fmt.Errorf("Connect Request Failed")
}

func hijackConn(w http.ResponseWriter) net.Conn {
	//Check if we can hijack connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "server doesn't support hijacking", http.StatusInternalServerError)
		return nil
	}
	w.WriteHeader(http.StatusOK)
	//Hijack the connection
	conn, _, _ := hj.Hijack()
	return conn
}
