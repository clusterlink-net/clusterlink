package dataplane

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
	"github.ibm.com/mbg-agent/pkg/api"
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/dataplane/store"
	"github.ibm.com/mbg-agent/pkg/utils/httputils"
)

const TCP_TYPE = "tcp"
const MTLS_TYPE = "mtls"

var clog = logrus.WithField("component", "DataPlane")

type Dataplane struct {
	Store  *store.Store
	Router *chi.Mux
}

// Set the data-plane store according the bootstrap
func NewDataplane(s *store.Store, controlplane string) *Dataplane {
	return &Dataplane{Store: store.NewStore(s, controlplane)}

}

// Connect HTTP handler for post request (use for MTLS data plane)
func (d *Dataplane) MTLSexportServiceEndpointHandler(w http.ResponseWriter, r *http.Request) {
	// Parse struct from request
	var c apiObject.ConnectRequest
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Connect data plane logic
	mbgIP := strings.Split(r.RemoteAddr, ":")[0]
	clog.Infof("Received connect to service %s from MBG: %s", c.Id, mbgIP)
	connect, connectType, connectDest := d.startListenerToExportServiceEndpoint(c, mbgIP, nil)

	clog.Infof("Got {%+v, %+v, %+v} from connect \n", connect, connectType, connectDest)
	// Set Connect response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(apiObject.ConnectReply{Connect: connect, ConnectType: connectType, ConnectDest: connectDest}); err != nil {
		clog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}

}

// Connect HTTP handler for connect request (use for TCP data plane)
func (d *Dataplane) TCPexportServiceEndpointHandler(w http.ResponseWriter, r *http.Request) {
	//Parse struct from request
	var c apiObject.ConnectRequest
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	clog.Infof("Received connect to service: %v", c.Id)
	mbgIP := strings.Split(r.RemoteAddr, ":")[0]
	clog.Infof("Received connect to service %s from MBG: %s", c.Id, mbgIP)
	allow, _, _ := d.startListenerToExportServiceEndpoint(c, mbgIP, w)

	// Write response for error
	if !allow {
		w.WriteHeader(http.StatusForbidden)
	}
}

// ConnectExportServiceEndpoint waiting for connection from host and do two things:
// 1. Create tcp connection to destination (Not Secure)- TODO support also secure connection
// 2. Register new handle function and hijack the connection
func (d *Dataplane) startListenerToExportServiceEndpoint(c apiObject.ConnectRequest, targetMbgIP string, w http.ResponseWriter) (bool, string, string) {
	clog.Infof("Received Incoming Connect request from service: %v to service: %v", c.Id, c.IdDest)
	connectionID := createConnId(c.Id, c.IdDest)
	endpoint := connectionID + ksuid.New().String()

	dataplane := d.Store.GetDataplane()
	rep, err := d.SendToControlPlaneNewExportConnRequest(c.Id, c.MbgID, c.IdDest)
	if err != nil {
		clog.Error("Unable to raise connection request event ")
		return false, "", ""
	}
	if rep.Action == eventManager.Deny.String() {
		clog.Infof("Denying incoming connect request (%s,%s) due to policy", c.Id, c.IdDest)
		return false, "", ""
	}
	clog.Infof("Received control plane response for service %s ,connection information : %v ", c.Id, rep)
	switch dataplane {
	case TCP_TYPE:
		clog.Infof("Sending Connect reply to Connection(%v) to use Dest:%v", rep.ConnId, "use connect hijack")
		conn := hijackConn(w)
		if conn == nil {
			clog.Error("Hijack Failure")
			return false, "", ""
		}
		go d.startTCPListenerService("httpconnect", rep.DestSvcEndpoint, c.Policy, rep.ConnId, conn, nil, eventManager.Incoming)
		return true, dataplane, endpoint
	case MTLS_TYPE:
		clog.Infof("Starting a Receiver service for %s Using serviceEndpoint : %s/%s",
			rep.DestSvcEndpoint, rep.SrcGwEndpoint, endpoint)

		go d.StartMTLSListenerToExportServiceEndpoint(rep.DestSvcEndpoint, rep.SrcGwEndpoint, endpoint, rep.ConnId)
		return true, dataplane, endpoint
	default:
		return false, "", ""
	}
}

// Send request to control-plane to check connection permission and parameters
func (d *Dataplane) SendToControlPlaneNewExportConnRequest(srcId, srcGwId, destId string) (apiObject.NewExportConnParmaReply, error) {
	var rep apiObject.NewExportConnParmaReply
	address := d.Store.GetControlPlaneAddr() + "/exports/newConnection"

	j, err := json.Marshal(apiObject.NewExportConnParmaReq{SrcId: srcId, SrcGwId: srcGwId, DestId: destId})
	if err != nil {
		clog.Error(err)
		return rep, err
	}
	resp, err := httputils.HttpPost(address, j, d.Store.GetLocalHttpClient())
	if err := json.Unmarshal(resp, &rep); err != nil {
		clog.Error("Unable to Unmarshal json NewConnParmaReply ", err)
	}

	return rep, err
}

// Receiver service is run at the gw which receives connection from a remote peer
func (d *Dataplane) StartMTLSListenerToExportServiceEndpoint(exportServicePort, targetMbgIPPort, importEndPoint, connId string) error {
	conn, err := net.Dial("tcp", exportServicePort) //Todo - support destination with secure connection
	if err != nil {
		clog.Errorf("Dial to export service failed: %v", err)
		return err
	}
	clog.Infof("Received new Connection at %s, %s", conn.LocalAddr().String(), importEndPoint)
	MTLSForward := MTLSForwarder{ChiRouter: d.Router}
	incomingBytes, outgoingBytes, startTstamp, endTstamp, _ := MTLSForward.StartMTLSForwarderServer(targetMbgIPPort, importEndPoint, "", "", "", conn)
	d.SendToControlPlaneConnStatus(connId, incomingBytes, outgoingBytes, startTstamp, endTstamp, eventManager.Incoming, eventManager.Complete)

	return nil
}

func (d *Dataplane) startMTLSListenerService(mbgIP, connectDest, rootCA, certificate, key, serverName string, ac net.Conn, connId string) {
	MTLSForward := MTLSForwarder{ChiRouter: d.Router}

	incomingBytes, outgoingBytes, startTstamp, endTstamp, _ := MTLSForward.StartMTLSForwarderClient(mbgIP, connectDest, rootCA, certificate, key, serverName, ac)

	d.SendToControlPlaneConnStatus(connId, incomingBytes, outgoingBytes, startTstamp, endTstamp, eventManager.Outgoing, eventManager.Complete)
}

// Run server for Data connection - we have one server and client that we can add some network functions e.g: TCP-split
// By default we just forward the data
func (d *Dataplane) startTCPListenerService(svcListenPort, svcIp, policy, connId string, serverConn, clientConn net.Conn, direction eventManager.Direction) {

	srcIp := svcListenPort
	destIp := svcIp

	// No Policy to be applied
	var forward TCPForwarder
	forward.Init(srcIp, destIp, connId)
	if serverConn != nil {
		forward.SetServerConnection(serverConn)
	}
	if clientConn != nil {
		forward.SetClientConnection(clientConn)
	}

	incomingBytes, outgoingBytes, startTstamp, endTstamp, _ := forward.RunTcpForwarder(direction)
	d.SendToControlPlaneConnStatus(connId, incomingBytes, outgoingBytes, startTstamp, endTstamp, direction, eventManager.Complete)
}

// Add import service - HTTP handler
func (d *Dataplane) AddImportServiceEndpointHandler(w http.ResponseWriter, r *http.Request) {

	// Parse add service struct from request
	var e api.Import
	err := json.NewDecoder(r.Body).Decode(&e)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	// AddService control plane logic
	d.addImportServiceEndpoint(e)

	// Response
	w.WriteHeader(http.StatusOK)
	rep := apiObject.ImportReply{Id: e.Name, Port: d.Store.GetSvcPort(e.Name)}
	if err := json.NewEncoder(w).Encode(rep); err != nil {
		clog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

// Add import service - control logic
func (d *Dataplane) addImportServiceEndpoint(e api.Import) {
	err := d.createImportServiceEndpoint(e.Name, false)
	if err != nil {
		return
	}
}

// Create import service endpoint
func (d *Dataplane) createImportServiceEndpoint(svcId string, force bool) error {
	connPort, err := d.Store.GetFreePorts(svcId)
	if err != nil {
		if err.Error() != store.ConnExist {
			return err
		}
		if !force {
			return nil
		}
	}
	certca, certFile, keyFile := d.Store.GetCerts()
	clog.Infof("Starting an service endpoint for import service %s at port %s with certs(%s,%s,%s)", svcId, connPort, certca, certFile, keyFile)
	go d.CreateListenerToImportServiceEndpoint(svcId, connPort, certca, certFile, keyFile)
	return nil
}

// Start a listener to Import Service (which connect to export service)
// It receives connections from remote peer and performs Connect API
// and sets up an mTLS forwarding to the remote peer upon accepted (policy checks, etc)
func (d *Dataplane) CreateListenerToImportServiceEndpoint(serviceId, servicePort, certca, certificate, key string) {
	clog.Infof("Starting an service endpoint for Export service %s at port %s ", serviceId, servicePort)
	acceptor, err := net.Listen("tcp", servicePort) //TODO- need to support secure endpoint
	if err != nil {
		clog.Infof("Error Listen: to port  %v", err)
	}
	clog.Infof("Accept a connection service endpoint for import service %s at port %s ", serviceId, servicePort)

	go d.StartListenerToImportServiceEndpoint(serviceId, acceptor, servicePort, certca, certificate, key)
	d.Store.WaitServiceStopCh(serviceId, servicePort)
	acceptor.Close()
}

// Start listener to import service endpoint
func (d *Dataplane) StartListenerToImportServiceEndpoint(destId string, acceptor net.Listener, servicePort, certca, certificate, key string) error {
	dataplane := d.Store.GetDataplane()
	// loop until signalled to stop
	for {
		ac, err := acceptor.Accept()
		clog = logrus.WithFields(logrus.Fields{
			"component": d.Store.GetMyId() + "-Dataplane",
		})
		if err != nil {
			clog.Infof("Accept() returned error: %v", err)
			return err
		}
		srcIp := ac.RemoteAddr().String()
		destIp := ac.LocalAddr().String()

		//Send Request to control plane if connection is valid and destination
		r, err := d.SendToControlPlaneNewImportConnRequest(srcIp, destIp, destId)
		clog.Printf("Got policy response for new connection to service %s with response %s", destId, r)

		if err != nil {
			clog.Errorf("SendToControlPlaneNewConnRequest returned error: %v", err)
			ac.Close()
			continue
		}
		if r.Action == eventManager.Deny.String() {
			clog.Infof("Denying Outgoing connection due to policy")
			ac.Close()
			continue
		}

		switch dataplane {
		case TCP_TYPE:
			connDest, err := d.TCPConnectReq(r.SrcId, destId, "forward", r.Target)

			if err != nil {
				clog.Infof("Unable to connect(tcp): %v ", err.Error())
				ac.Close()
				d.SendToControlPlaneConnStatus(r.ConnId, 0, 0, time.Now(), time.Now(), eventManager.Outgoing, eventManager.PeerDenied)
				continue
			}
			connectDest := "Use open connect socket" //not needed ehr we use connect - destSvc.Service.Ip + ":" + connectDest
			clog.Infof("Using %s for  %s/%s to connect to Service-%v", dataplane, r.Target, connectDest, destId)
			go d.startTCPListenerService(servicePort, connectDest, "forward", r.ConnId, ac, connDest, eventManager.Outgoing)

		case MTLS_TYPE:
			//Send connection request to other MBG
			connectType, connectDest, err := d.mTLSConnectReq(r.SrcId, destId, "forward", r.Target)

			if err != nil {
				clog.Infof("Unable to connect(MTLS): %v ", err.Error())
				ac.Close()
				d.SendToControlPlaneConnStatus(r.ConnId, 0, 0, time.Now(), time.Now(), eventManager.Outgoing, eventManager.PeerDenied)
				continue
			}
			clog.Infof("Using %s for  %s/%s to connect to Service-%v", connectType, r.Target, connectDest, destId)
			serverName := d.Store.GetMyId()
			go d.startMTLSListenerService(r.Target, connectDest, certca, certificate, key, serverName, ac, r.ConnId)
		default:
			clog.Errorf("%v -Not supported", dataplane)

		}
	}
}

// Send control request to connect
func (d *Dataplane) mTLSConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) (string, string, error) {
	clog.Infof("Starting mTLS Connect Request to MBG at %v for Service %v", mbgIp, svcIdDest)
	address := d.Store.GetProtocolPrefix() + mbgIp + "/exports/serviceEndpoint"

	j, err := json.Marshal(apiObject.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy, MbgID: d.Store.GetMyId()})
	if err != nil {
		clog.Error(err)
		return "", "", err
	}
	// Send connect
	resp, err := httputils.HttpPost(address, j, d.Store.GetRemoteHttpClient())
	if err != nil {
		clog.Error(err)
		return "", "", err
	}
	var r apiObject.ConnectReply
	err = json.Unmarshal(resp, &r)
	if err != nil {
		clog.Error(err)
		return "", "", err
	}
	if r.Connect {
		clog.Infof("Successfully Connected : Using Connection:Port - %s:%s", r.ConnectType, r.ConnectDest)
		return r.ConnectType, r.ConnectDest, nil
	}
	clog.Infof("Failed to Connect")

	return "", "", fmt.Errorf("failed to connect")
}

// TCP connection request to other peer
func (d *Dataplane) TCPConnectReq(svcId, svcIdDest, svcPolicy, mbgIp string) (net.Conn, error) {
	clog.Printf("Starting TCP Connect Request to peer at %v for service %v", mbgIp, svcIdDest)
	url := d.Store.GetProtocolPrefix() + mbgIp + "/exports/serviceEndpoint"

	jsonData, err := json.Marshal(apiObject.ConnectRequest{Id: svcId, IdDest: svcIdDest, Policy: svcPolicy, MbgID: d.Store.GetMyId()})
	if err != nil {
		clog.Error(err)
		return nil, err
	}
	c, resp := httputils.HttpConnect(mbgIp, url, string(jsonData))
	if resp == nil {
		clog.Printf("Successfully Connected")
		return c, nil
	}

	return nil, fmt.Errorf("Connect Request Failed")
}

// Hijack the HTTP connection and use the TCP session
func hijackConn(w http.ResponseWriter) net.Conn {
	// Check if we can hijack connection
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

// Creating connection id for the store
func createConnId(srcId, destId string) string {
	connectionID := srcId + ":" + destId
	connectionID = strings.Replace(connectionID, "*", "wildcard", 2)
	return connectionID
}

// Send request to control-plane to check connection permission and parameters
func (d *Dataplane) SendToControlPlaneNewImportConnRequest(srcIp, destIp, destId string) (apiObject.NewImportConnParmaReply, error) {
	var rep apiObject.NewImportConnParmaReply
	address := d.Store.GetControlPlaneAddr() + "/imports/newConnection"

	j, err := json.Marshal(apiObject.NewImportConnParmaReq{SrcIp: srcIp, DestIp: destIp, DestId: destId})
	if err != nil {
		clog.Error(err)
		return rep, err
	}
	resp, err := httputils.HttpPost(address, j, d.Store.GetLocalHttpClient())
	if err := json.Unmarshal(resp, &rep); err != nil {
		clog.Error("Unable to Unmarshal json NewConnParmaReply ", err)
	}

	return rep, err
}

// Delete import service - HTTP handler
func (d *Dataplane) DelImportServiceEndpointHandler(w http.ResponseWriter, r *http.Request) {
	// Parse del service struct from request
	svcId := chi.URLParam(r, "svcId")
	// Parse add service struct from request
	var s api.Import
	err := json.NewDecoder(r.Body).Decode(&s)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// AddService control plane logic
	d.delImportServiceEndpoint(svcId)

	// Response
	w.WriteHeader(http.StatusOK)
	_, err = w.Write([]byte("Service deleted successfully"))
	if err != nil {
		clog.Println(err)
	}
}

// Delete import service - control logic
func (d *Dataplane) delImportServiceEndpoint(svcId string) {
	//Todo
}

// Send request to control-plane to check connection permission and parameters
func (d *Dataplane) SendToControlPlaneConnStatus(connId string, incomingBytes, outgoingBytes int, startTstamp, endTstamp time.Time, direction eventManager.Direction, state eventManager.ConnectionState) error {
	address := d.Store.GetControlPlaneAddr() + "/connectionStatus"

	connStatus := apiObject.ConnectionStatus{ConnectionId: connId,
		IncomingBytes: incomingBytes,
		OutgoingBytes: outgoingBytes,
		StartTstamp:   startTstamp,
		LastTstamp:    endTstamp,
		Direction:     direction,
		State:         state}
	j, err := json.Marshal(connStatus)
	if err != nil {
		clog.Error(err)
		return err
	}
	_, err = httputils.HttpPost(address, j, d.Store.GetLocalHttpClient())

	return err
}
