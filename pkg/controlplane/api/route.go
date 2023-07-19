package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"

	cp "github.ibm.com/mbg-agent/pkg/controlplane"
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/health"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/k8s/kubernetes"
)

type MbgHandler struct{}

func (m MbgHandler) Routes() chi.Router {
	r := store.GetChiRouter()

	r.Get("/", m.controlplaneWelcome)

	r.Route("/peer", func(r chi.Router) {
		r.Get("/", cp.GetAllPeersHandler)       // GET    /peer      - Get all peers
		r.Post("/", cp.AddPeerHandler)          // Post   /peer      - Add peer Id to peers list
		r.Get("/{id}", cp.GetPeerHandler)       // GET    /peer/{id} - Get peer Id
		r.Delete("/{id}", cp.RemovePeerHandler) // Delete /peer/{id} - Delete peer
	})

	r.Route("/hello", func(r chi.Router) {
		r.Post("/", cp.SendHelloToAllHandler)    // Post /hello Hello to all peers
		r.Post("/{peerID}", cp.SendHelloHandler) // Post /hello/{peerID} send Hello to a peer
	})

	r.Route("/hb", func(r chi.Router) {
		r.Post("/", health.HandleHB) // Heartbeat messages
	})

	r.Route("/service", func(r chi.Router) {
		r.Post("/", cp.AddLocalServiceHandler)                    // Post /service    - Add local service
		r.Get("/", cp.GetAllLocalServicesHandler)                 // Get  /service    - Get all local services
		r.Get("/{id}", cp.GetLocalServiceHandler)                 // Get  /service    - Get specific local service
		r.Delete("/{id}", cp.DelLocalServiceHandler)              // Delete  /service - Delete local service
		r.Delete("/{id}/peer", cp.DelLocalServiceFromPeerHandler) // Delete  /service - Delete local service from peer

	})

	r.Route("/remoteservice", func(r chi.Router) {
		r.Post("/", cp.AddRemoteServiceHandler)          // Post /remoteservice            - Add Remote service
		r.Get("/", cp.GetAllRemoteServicesHandler)       // Get  /remoteservice            - Get all remote services
		r.Get("/{svcId}", cp.GetRemoteServiceHandler)    // Get  /remoteservice/{svcId}    - Get specific remote service
		r.Delete("/{svcId}", cp.DelRemoteServiceHandler) // Delete  /remoteservice/{svcId} - Delete specific remote service

	})

	r.Route("/expose", func(r chi.Router) {
		r.Post("/", cp.ExposeHandler) // Post /expose  - Expose  service
	})

	r.Route("/binding", func(r chi.Router) {
		r.Post("/", cp.CreateBindingHandler)          // Post /binding   - Bind remote service to local port
		r.Delete("/{svcId}", cp.DeleteBindingHandler) // Delete /binding - Remove Binding of remote service to local port
	})

	r.Route("/imports", func(r chi.Router) {
		r.Post("/newConnection", setupNewImportConnHandler) // Post /newImportConnection - New connection parameters check
	})
	r.Route("/exports", func(r chi.Router) {
		r.Post("/newConnection", setupNewExportConnHandler) // Post /newExportConnection  - New connection parameters check
	})
	r.Route("/connectionStatus", func(r chi.Router) {
		r.Post("/", connStatusHandler) // Post /connectionStatus  - Metrics regarding connections
	})
	return r
}

func (m MbgHandler) controlplaneWelcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to control plane Gateway"))
	if err != nil {
		log.Println(err)
	}
}

// New connection request to import service- HTTP handler
func setupNewImportConnHandler(w http.ResponseWriter, r *http.Request) {
	// Parse expose struct from request
	var c apiObject.NewImportConnParmaReq
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	log.Infof("Got new connection check for dataplane: %+v", c)
	connReply := setupNewImportConn(c.SrcIp, c.DestIp, c.DestId)
	// Response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(connReply); err != nil {
		log.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}

}

// New connection request to import service-control plane logic that check the policy and connection parameters
func setupNewImportConn(srcIp, destIp, destSvcId string) apiObject.NewImportConnParmaReply {
	// Need to look up the label to find local service
	// If label isnt found, Check for IP.
	// If we cant find the service, we get the "service id" as a wildcard
	// which is sent to the policy engine to decide.

	// Ideally do a control plane connect API, Policy checks, and then create a mTLS forwarder
	// ImportEndPoint has to be in the connect Request/Response
	appLabel, err := kubernetes.Data.GetLabel(strings.Split(srcIp, ":")[0], kubernetes.AppLabel)
	if err != nil {
		log.Errorf("Unable to get App Info :%+v", err)
	}
	log.Infof("Receiving Outgoing connection %s(%s)->%s ", srcIp, destIp, appLabel)
	srcSvc, err := store.LookupLocalService(appLabel, srcIp)
	if err != nil {
		log.Infof("Unable to lookup local service :%v", err)
	}

	policyResp, err := store.GetEventManager().RaiseNewConnectionRequestEvent(eventManager.ConnectionRequestAttr{SrcService: srcSvc.Id, DstService: destSvcId, Direction: eventManager.Outgoing, OtherMbg: eventManager.Wildcard})
	if err != nil {
		log.Errorf("Unable to raise connection request event")
		return apiObject.NewImportConnParmaReply{Action: eventManager.Deny.String()}
	}
	connectionId := srcSvc.Id + ":" + destSvcId + ":" + ksuid.New().String()
	connectionStatus := eventManager.ConnectionStatusAttr{ConnectionId: connectionId,
		SrcService:      srcSvc.Id,
		DstService:      destSvcId,
		DestinationPeer: policyResp.TargetMbg,
		StartTstamp:     time.Now(),
		Direction:       eventManager.Outgoing,
		State:           eventManager.Ongoing}

	if policyResp.Action == eventManager.Deny {
		connectionStatus.State = eventManager.Denied
	}
	store.GetEventManager().RaiseConnectionStatusEvent(connectionStatus)

	log.Infof("Accepting Outgoing Connect request from service: %v to service: %v", srcSvc.Id, destSvcId)

	var target string
	if policyResp.TargetMbg == "" {
		// Policy Agent hasnt suggested anything any target MBG, hence we fall back to our defaults
		target = store.GetMbgTarget(destSvcId)
	} else {
		target = store.GetMbgTarget(policyResp.TargetMbg)
	}
	return apiObject.NewImportConnParmaReply{Action: policyResp.Action.String(), Target: target, SrcId: srcSvc.Id, ConnId: connectionId}
}

// New connection request to export service- HTTP handler
func setupNewExportConnHandler(w http.ResponseWriter, r *http.Request) {
	// Parse struct from request
	var c apiObject.NewExportConnParmaReq
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	log.Infof("Got new connection check for dataplane: %+v", c)
	connReply := setupNewExportConn(c.SrcId, c.SrcGwId, c.DestId)
	// Response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(connReply); err != nil {
		log.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}

}

// New connection request  to export service-control plane logic that check the policy and connection parameters
func setupNewExportConn(srcSvcId, srcGwId, destSvcId string) apiObject.NewExportConnParmaReply {
	localSvc := store.GetLocalService(destSvcId)
	policyResp, err := store.GetEventManager().RaiseNewConnectionRequestEvent(eventManager.ConnectionRequestAttr{SrcService: srcSvcId, DstService: destSvcId, Direction: eventManager.Incoming, OtherMbg: srcGwId})

	if err != nil {
		log.Error("Unable to raise connection request event ", store.GetMyId())
		return apiObject.NewExportConnParmaReply{Action: eventManager.Deny.String()}
	}

	connectionId := srcSvcId + ":" + destSvcId + ":" + ksuid.New().String()
	connectionStatus := eventManager.ConnectionStatusAttr{ConnectionId: connectionId,
		SrcService:      srcSvcId,
		DstService:      destSvcId,
		DestinationPeer: srcGwId,
		StartTstamp:     time.Now(),
		Direction:       eventManager.Incoming,
		State:           eventManager.Ongoing}

	if policyResp.Action == eventManager.Deny {
		connectionStatus.State = eventManager.Denied
	}
	store.GetEventManager().RaiseConnectionStatusEvent(connectionStatus)

	srcGw := store.GetMbgTarget(srcGwId)
	return apiObject.NewExportConnParmaReply{Action: policyResp.Action.String(), SrcGwEndpoint: srcGw, DestSvcEndpoint: localSvc.GetIpAndPort(), ConnId: connectionId}
}

// Connection Status handler to receive metrics regarding connection from the dataplane
func connStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Parse expose struct from request
	var c apiObject.ConnectionStatus
	err := json.NewDecoder(r.Body).Decode(&c)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return

	}
	connectionStatus := eventManager.ConnectionStatusAttr{ConnectionId: c.ConnectionId,
		IncomingBytes: c.IncomingBytes,
		OutgoingBytes: c.OutgoingBytes,
		StartTstamp:   c.StartTstamp,
		LastTstamp:    c.LastTstamp,
		Direction:     c.Direction,
		State:         c.State}

	store.GetEventManager().RaiseConnectionStatusEvent(connectionStatus)

	w.WriteHeader(http.StatusOK)

}
