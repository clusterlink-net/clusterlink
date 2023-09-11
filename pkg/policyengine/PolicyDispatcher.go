/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyengine

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	event "github.ibm.com/mbg-agent/pkg/controlplane/eventmanager"
	"github.ibm.com/mbg-agent/pkg/policyengine/connectivitypdp"
	"github.ibm.com/mbg-agent/pkg/policyengine/policytypes"
)

const (
	LbType     = "lb"     // Type for load-balancing policies
	AccessType = "access" // Type for access policies

	PolicyRoute = "/policy"        // Parent route for all kinds of policies
	LbRoute     = "/" + LbType     // Route for managing LoadBalancer policies
	AccessRoute = "/" + AccessType // Route for managing Access policies (Connectivity policies)

	GetRoute = "/"       // Route for getting a list of active policies
	AddRoute = "/add"    // Route for adding policies
	DelRoute = "/delete" // Route for deleting policies

	ServiceNameLabel = "clusterlink/metadata.serviceName"
	MbgNameLabel     = "clusterlink/metadata.gatewayName"
)

var plog = logrus.WithField("component", "PolicyEngine")
var MyPolicyHandler PolicyHandler

type MbgState struct {
	mbgPeers *[]string
}

type PolicyHandler struct {
	loadBalancer    *LoadBalancer
	connectivityPDP *connectivitypdp.PDP
	mbgState        MbgState
}

func (pH *PolicyHandler) Routes(r *chi.Mux) chi.Router {

	r.Get("/policywelcome/", pH.policyWelcome)

	r.Route("/"+event.NewConnectionRequest, func(r chi.Router) {
		r.Post("/", pH.newConnectionRequest) // New connection request
	})

	r.Route("/"+event.AddPeerRequest, func(r chi.Router) {
		r.Post("/", pH.addPeerRequest) // New peer request
	})

	r.Route("/"+event.RemovePeerRequest, func(r chi.Router) {
		r.Post("/", pH.removePeerRequest) // Remove peer request
	})

	r.Route("/"+event.NewRemoteService, func(r chi.Router) {
		r.Post("/", pH.newRemoteService) // New remote service request
	})
	r.Route("/"+event.RemoveRemoteService, func(r chi.Router) {
		r.Post("/", pH.removeRemoteServiceRequest) // Remove remote service request
	})
	r.Route("/"+event.ExposeRequest, func(r chi.Router) {
		r.Post("/", pH.exposeRequest) // New expose request
	})

	r.Route(AccessRoute, func(r chi.Router) {
		r.Get(GetRoute, pH.getConnPoliciesReq)
		r.Post(AddRoute, pH.addConnPolicyReq) // Add Access Policy
		r.Post(DelRoute, pH.delConnPolicyReq) // Delete Access policies
	})

	r.Route(LbRoute, func(r chi.Router) {
		r.Get(GetRoute, pH.loadBalancer.GetPolicyReq)
		r.Post(AddRoute, pH.loadBalancer.SetPolicyReq)    // Add LB Policy
		r.Post(DelRoute, pH.loadBalancer.DeletePolicyReq) // Delete LB Policy

	})
	return r
}

func exists(slice []string, entry string) (int, bool) {
	for i, e := range slice {
		if e == entry {
			return i, true
		}
	}
	return -1, false
}

func (pH *PolicyHandler) addPeer(peerMbg string) {
	_, exist := exists(*pH.mbgState.mbgPeers, peerMbg)
	if exist {
		return
	}
	*pH.mbgState.mbgPeers = append(*pH.mbgState.mbgPeers, peerMbg)
	plog.Infof("Added Peer %+v", pH.mbgState.mbgPeers)
}

func (pH *PolicyHandler) removePeer(peerMbg string) {
	index, exist := exists(*pH.mbgState.mbgPeers, peerMbg)
	if !exist {
		return
	}
	*pH.mbgState.mbgPeers = append((*pH.mbgState.mbgPeers)[:index], (*pH.mbgState.mbgPeers)[index+1:]...)
	plog.Infof("Removed Peer(%s, %d) %+v", peerMbg, index, *pH.mbgState.mbgPeers)
}

func getServiceAttrs(serviceName, peer string) policytypes.WorkloadAttrs {
	ret := policytypes.WorkloadAttrs{ServiceNameLabel: serviceName}
	if len(peer) > 0 {
		ret[MbgNameLabel] = peer
	}
	return ret
}

func getServiceAttrsForMultiplePeers(serviceName string, peers []string) []policytypes.WorkloadAttrs {
	res := []policytypes.WorkloadAttrs{}
	for _, peer := range peers {
		res = append(res, getServiceAttrs(serviceName, peer))
	}
	return res
}

func (pH *PolicyHandler) decideIncomingConnection(requestAttr *event.ConnectionRequestAttr) event.Action {
	src := getServiceAttrs(requestAttr.SrcService, requestAttr.OtherMbg)
	dest := getServiceAttrs(requestAttr.DstService, "")
	decisions, err := pH.connectivityPDP.Decide(src, []policytypes.WorkloadAttrs{dest})
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return event.Deny
	}
	if decisions[0].Decision == policytypes.PolicyDecisionAllow {
		return event.Allow
	}
	return event.Deny
}

func (pH *PolicyHandler) decideOutgoingConnection(requestAttr *event.ConnectionRequestAttr) (event.Action, string) {
	// Get a list of MBGs for the service
	mbgList, err := pH.loadBalancer.GetTargetMbgs(requestAttr.DstService)
	if err != nil {
		plog.Errorf("error getting target peers: %v", err)
		return event.Deny, ""
	}

	src := getServiceAttrs(requestAttr.SrcService, "")
	dsts := getServiceAttrsForMultiplePeers(requestAttr.DstService, mbgList)
	decisions, err := pH.connectivityPDP.Decide(src, dsts)
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return event.Deny, ""
	}

	allowedMbgs := []string{}
	for _, decision := range decisions {
		if decision.Decision == policytypes.PolicyDecisionAllow {
			allowedMbgs = append(allowedMbgs, decision.Destination[MbgNameLabel])
		}
	}

	// Perform load-balancing using the filtered mbgList
	targetMbg, err := pH.loadBalancer.LookupWith(requestAttr.SrcService, requestAttr.DstService, allowedMbgs)
	if err != nil {
		return event.Deny, ""
	}
	return event.Allow, targetMbg
}

func (pH *PolicyHandler) newConnectionRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.ConnectionRequestAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("New connection request : %+v", requestAttr)

	var action event.Action
	var targetMbg string
	var bitrate int
	if requestAttr.Direction == event.Incoming {
		action = pH.decideIncomingConnection(&requestAttr)
	} else if requestAttr.Direction == event.Outgoing {
		action, targetMbg = pH.decideOutgoingConnection(&requestAttr)
	}

	plog.Infof("Response : %+v", event.ConnectionRequestResp{Action: action, TargetMbg: targetMbg, BitRate: bitrate})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event.ConnectionRequestResp{Action: action, TargetMbg: targetMbg, BitRate: bitrate}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func (pH *PolicyHandler) addPeerRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.AddPeerAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plog.Infof("Add Peer request : %+v", requestAttr)
	// Currently, request is always allowed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event.AddPeerResp{Action: event.Allow}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}

	pH.addPeer(requestAttr.PeerMbg)
}

func (pH *PolicyHandler) removePeerRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.AddPeerAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Remove Peer request : %+v ", requestAttr)
	pH.removePeer(requestAttr.PeerMbg)
	pH.loadBalancer.RemoveMbgFromServiceMap(requestAttr.PeerMbg)
	w.WriteHeader(http.StatusOK)
}

func (pH *PolicyHandler) newRemoteService(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.NewRemoteServiceAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plog.Infof("New Remote Service request : %+v", requestAttr)
	// Currently, request is always allowed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event.NewRemoteServiceResp{Action: event.Allow}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
	}

	// Update States
	pH.loadBalancer.AddToServiceMap(requestAttr.Service, requestAttr.Mbg)
}

func (pH *PolicyHandler) removeRemoteServiceRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.RemoveRemoteServiceAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Remove remote service request : %+v ", requestAttr)
	pH.loadBalancer.RemoveDestService(requestAttr.Service, requestAttr.Mbg)
	w.WriteHeader(http.StatusOK)
}

func (pH *PolicyHandler) exposeRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.ExposeRequestAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	plog.Infof("New Expose request : %+v", requestAttr)
	// Currently, request is always allowed
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(event.ExposeRequestResp{Action: event.AllowAll, TargetMbgs: *pH.mbgState.mbgPeers}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func (pH *PolicyHandler) getConnPoliciesReq(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	policies := pH.connectivityPDP.GetPolicies()

	if err := json.NewEncoder(w).Encode(policies); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func (pH *PolicyHandler) addConnPolicyReq(w http.ResponseWriter, r *http.Request) {
	var policy policytypes.ConnectivityPolicy
	err := json.NewDecoder(r.Body).Decode(&policy)
	if err != nil {
		plog.Errorf("failed decoding connectivity policy: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = pH.connectivityPDP.AddOrUpdatePolicy(policy)
	if err != nil { // policy is syntactically ok, but semantically invalid - 422 is the status to return
		plog.Errorf("failed adding connectivity policy: %v", err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	plog.Infof("Added policy : %+v", policy)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

func (pH *PolicyHandler) delConnPolicyReq(w http.ResponseWriter, r *http.Request) {
	var policy policytypes.ConnectivityPolicy
	err := json.NewDecoder(r.Body).Decode(&policy)
	if err != nil {
		plog.Errorf("failed decoding connectivity policy: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = pH.connectivityPDP.DeletePolicy(policy.Name, policy.Privileged)
	if err != nil {
		plog.Errorf("failed deleting connectivity policy: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	plog.Infof("Deleted policy : %+v", policy)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (pH *PolicyHandler) policyWelcome(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("Welcome to Policy Engine"))
	if err != nil {
		log.Println(err)
	}
}

func (pH *PolicyHandler) init(router *chi.Mux, defaultRule event.Action) {
	pH.mbgState.mbgPeers = &([]string{})
	pH.loadBalancer = &LoadBalancer{}
	pH.loadBalancer.Init()
	pH.connectivityPDP = connectivitypdp.NewPDP()

	routes := pH.Routes(router)
	router.Mount(PolicyRoute, routes)
}

func StartPolicyDispatcher(router *chi.Mux, defaultRule event.Action) {
	plog.Infof("Policy Engine started")
	MyPolicyHandler.init(router, defaultRule)
}
