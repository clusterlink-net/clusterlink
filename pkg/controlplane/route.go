package controlplane

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/segmentio/ksuid"
	log "github.com/sirupsen/logrus"
	apiObject "github.ibm.com/mbg-agent/pkg/controlplane/api/object"
	"github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/controlplane/health"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	"github.ibm.com/mbg-agent/pkg/k8s/kubernetes"
)

// MbgHandler struct
type MbgHandler struct{}

// Routes create all the sub routes for the CP server
func (m MbgHandler) Routes() chi.Router {
	r := store.GetChiRouter()

	r.Get("/", m.controlplaneWelcome)

	r.Route("/peers", func(r chi.Router) {
		r.Get("/", GetAllPeersHandler)       // GET    /peer      - Get all peers
		r.Post("/", AddPeerHandler)          // Post   /peer      - Add peer Id to peers list
		r.Get("/{id}", GetPeerHandler)       // GET    /peer/{id} - Get peer Id
		r.Delete("/{id}", RemovePeerHandler) // Delete /peer/{id} - Delete peer
	})

	r.Route("/hb", func(r chi.Router) {
		r.Post("/", health.HandleHB) // Heartbeat messages
	})

	r.Route("/exports", func(r chi.Router) {
		r.Post("/", AddExportServiceHandler)                // Post /exports    - Add export service
		r.Get("/", GetAllExportServicesHandler)             // Get  /exports    - Get all export services
		r.Get("/{id}", GetExportServiceHandler)             // Get  /exports    - Get specific export service
		r.Delete("/{id}", DelExportServiceHandler)          // Delete  /exports - Delete export service
		r.Post("/newConnection", setupNewExportConnHandler) // Post /newExportConnection  - New connection parameters check
	})

	r.Route("/imports", func(r chi.Router) {
		r.Post("/", AddImportServiceHandler)                // Post /imports            - Add Remote service
		r.Get("/", GetAllImportServicesHandler)             // Get  /imports            - Get all remote services
		r.Get("/{id}", GetImportServiceHandler)             // Get  /imports/{svcId}    - Get specific remote service
		r.Delete("/{id}", DelImportServiceHandler)          // Delete  /imports/{svcId} - Delete specific remote service
		r.Post("/newConnection", setupNewImportConnHandler) // Post /newImportConnection - New connection parameters check
	})

	r.Route("/bindings", func(r chi.Router) {
		r.Post("/", CreateBindingHandler) // Post /binding   - Bind remote service to local port
		r.Get("/{id}", GetBindingHandler) // get /binding   - Bind remote service to local port
		r.Delete("/", DelBindingHandler)  // Delete /binding - Remove Binding of remote service to local port
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
func setupNewImportConn(srcIP, destIP, destSvcID string) apiObject.NewImportConnParmaReply {
	// Need to look up the label to find local service
	// If label isnt found, Check for IP.
	// If we cant find the service, we get the "service id" as a wildcard
	// which is sent to the policy engine to decide.

	// Ideally do a control plane connect API, Policy checks, and then create a mTLS forwarder
	// ImportEndPoint has to be in the connect Request/Response
	appLabel, err := kubernetes.Data.GetLabel(strings.Split(srcIP, ":")[0], kubernetes.AppLabel)
	if err != nil {
		log.Errorf("Unable to get App Info :%+v", err)
	}
	log.Infof("Receiving Outgoing connection %s(%s)->%s ", srcIP, destIP, appLabel)
	srcSvc, err := store.LookupLocalService(appLabel, srcIP)
	if err != nil {
		log.Infof("Unable to lookup local service :%v", err)
	}

	policyResp, err := store.GetEventManager().RaiseNewConnectionRequestEvent(eventManager.ConnectionRequestAttr{SrcService: srcSvc.Id, DstService: destSvcID, Direction: eventManager.Outgoing, OtherMbg: eventManager.Wildcard})
	if err != nil {
		log.Errorf("Unable to raise connection request event")
		return apiObject.NewImportConnParmaReply{Action: eventManager.Deny.String()}
	}
	connectionId := srcSvc.Id + ":" + destSvcID + ":" + ksuid.New().String()
	connectionStatus := eventManager.ConnectionStatusAttr{ConnectionId: connectionId,
		SrcService:      srcSvc.Id,
		DstService:      destSvcID,
		DestinationPeer: policyResp.TargetMbg,
		StartTstamp:     time.Now(),
		Direction:       eventManager.Outgoing,
		State:           eventManager.Ongoing}

	if policyResp.Action == eventManager.Deny {
		connectionStatus.State = eventManager.Denied
	}
	store.GetEventManager().RaiseConnectionStatusEvent(connectionStatus)

	log.Infof("Accepting Outgoing Connect request from service: %v to service: %v", srcSvc.Id, destSvcID)

	var target string
	if policyResp.TargetMbg == "" {
		// Policy Agent hasnt suggested anything any target MBG, hence we fall back to our defaults
		target = store.GetMbgTarget(destSvcID)
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
func setupNewExportConn(srcSvcID, srcGwID, destSvcID string) apiObject.NewExportConnParmaReply {
	localSvc := store.GetLocalService(destSvcID)
	policyResp, err := store.GetEventManager().RaiseNewConnectionRequestEvent(eventManager.ConnectionRequestAttr{SrcService: srcSvcID, DstService: destSvcID, Direction: eventManager.Incoming, OtherMbg: srcGwID})

	if err != nil {
		log.Error("Unable to raise connection request event ", store.GetMyId())
		return apiObject.NewExportConnParmaReply{Action: eventManager.Deny.String()}
	}

	connectionId := srcSvcID + ":" + destSvcID + ":" + ksuid.New().String()
	connectionStatus := eventManager.ConnectionStatusAttr{ConnectionId: connectionId,
		SrcService:      srcSvcID,
		DstService:      destSvcID,
		DestinationPeer: srcGwID,
		StartTstamp:     time.Now(),
		Direction:       eventManager.Incoming,
		State:           eventManager.Ongoing}

	if policyResp.Action == eventManager.Deny {
		connectionStatus.State = eventManager.Denied
	}
	store.GetEventManager().RaiseConnectionStatusEvent(connectionStatus)

	srcGw := store.GetMbgTarget(srcGwID)
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
