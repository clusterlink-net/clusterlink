/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyengine

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	event "github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/connectivitypdp"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
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

type PolicyDecider interface {
	AddLBPolicy(lbPolicy *LBPolicy) error
	DeleteLBPolicy(lbPolicy *LBPolicy) error

	AddAccessPolicy(policy *api.Policy) error
	DeleteAccessPolicy(policy *api.Policy) error

	AuthorizeAndRouteConnection(connReq *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error)

	AddPeer(peer *api.Peer)
	DeletePeer(name string)

	AddBinding(imp *api.Binding) (event.Action, error)
	DeleteBinding(imp *api.Binding)

	AddExport(exp *api.Export) (event.ExposeRequestResp, error)
	DeleteExport(name string)
}

type MbgState struct {
	mbgPeers []string
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
	_, exist := exists(pH.mbgState.mbgPeers, peerMbg)
	if exist {
		return
	}
	pH.mbgState.mbgPeers = append(pH.mbgState.mbgPeers, peerMbg)
	plog.Infof("Added Peer %+v", pH.mbgState.mbgPeers)
}

func (pH *PolicyHandler) removePeer(peerMbg string) {
	index, exist := exists(pH.mbgState.mbgPeers, peerMbg)
	if !exist {
		return
	}
	pH.mbgState.mbgPeers = append(pH.mbgState.mbgPeers[:index], pH.mbgState.mbgPeers[index+1:]...)
	plog.Infof("Removed Peer(%s, %d) %+v", peerMbg, index, pH.mbgState.mbgPeers)
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

func (pH *PolicyHandler) decideIncomingConnection(requestAttr *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error) {
	src := getServiceAttrs(requestAttr.SrcService, requestAttr.OtherMbg)
	dest := getServiceAttrs(requestAttr.DstService, "")
	decisions, err := pH.connectivityPDP.Decide(src, []policytypes.WorkloadAttrs{dest})
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return event.ConnectionRequestResp{Action: event.Deny}, err
	}
	if decisions[0].Decision == policytypes.PolicyDecisionAllow {
		return event.ConnectionRequestResp{Action: event.Allow}, nil
	}
	return event.ConnectionRequestResp{Action: event.Deny}, nil
}

func (pH *PolicyHandler) decideOutgoingConnection(requestAttr *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error) {
	// Get a list of MBGs for the service
	mbgList, err := pH.loadBalancer.GetTargetMbgs(requestAttr.DstService)
	if err != nil || len(mbgList) == 0 {
		plog.Errorf("error getting target peers for service %s: %v", requestAttr.DstService, err)
		return event.ConnectionRequestResp{Action: event.Deny}, nil // this can be caused by a user typo - so only log this error
	}

	src := getServiceAttrs(requestAttr.SrcService, "")
	dsts := getServiceAttrsForMultiplePeers(requestAttr.DstService, mbgList)
	decisions, err := pH.connectivityPDP.Decide(src, dsts)
	if err != nil {
		plog.Errorf("error deciding on a connection: %v", err)
		return event.ConnectionRequestResp{Action: event.Deny}, err
	}

	allowedMbgs := []string{}
	for _, decision := range decisions {
		if decision.Decision == policytypes.PolicyDecisionAllow {
			allowedMbgs = append(allowedMbgs, decision.Destination[MbgNameLabel])
		}
	}

	if len(allowedMbgs) == 0 {
		plog.Infof("access policies deny connections to service %s in all peers", requestAttr.DstService)
		return event.ConnectionRequestResp{Action: event.Deny}, nil
	}

	// Perform load-balancing using the filtered mbgList
	targetMbg, err := pH.loadBalancer.LookupWith(requestAttr.SrcService, requestAttr.DstService, allowedMbgs)
	if err != nil {
		return event.ConnectionRequestResp{Action: event.Deny}, err
	}
	return event.ConnectionRequestResp{Action: event.Allow, TargetMbg: targetMbg}, nil
}

func (pH *PolicyHandler) newConnectionRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.ConnectionRequestAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := pH.AuthorizeAndRouteConnection(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func (pH *PolicyHandler) AuthorizeAndRouteConnection(connReq *event.ConnectionRequestAttr) (event.ConnectionRequestResp, error) {
	plog.Infof("New connection request : %+v", connReq)

	var resp event.ConnectionRequestResp
	var err error
	if connReq.Direction == event.Incoming {
		resp, err = pH.decideIncomingConnection(connReq)
	} else if connReq.Direction == event.Outgoing {
		resp, err = pH.decideOutgoingConnection(connReq)
	}

	plog.Infof("Response : %+v", resp)
	return resp, err
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

func (pH *PolicyHandler) AddPeer(peer *api.Peer) {
	pH.addPeer(peer.Name)
}

func (pH *PolicyHandler) DeletePeer(name string) {
	pH.removePeer(name)
	pH.loadBalancer.RemoveMbgFromServiceMap(name)
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

func (pH *PolicyHandler) AddBinding(binding *api.Binding) (event.Action, error) {
	pH.loadBalancer.AddToServiceMap(binding.Spec.Import, binding.Spec.Peer)
	return event.Allow, nil
}

func (pH *PolicyHandler) DeleteBinding(binding *api.Binding) {
	pH.loadBalancer.RemoveDestService(binding.Spec.Import, binding.Spec.Peer)
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
	if err := json.NewEncoder(w).Encode(event.ExposeRequestResp{Action: event.AllowAll, TargetMbgs: pH.mbgState.mbgPeers}); err != nil {
		plog.Errorf("Error happened in JSON encode. Err: %s", err)
		return
	}
}

func (pH *PolicyHandler) AddExport(_ *api.Export) (event.ExposeRequestResp, error) {
	return event.ExposeRequestResp{Action: event.AllowAll, TargetMbgs: pH.mbgState.mbgPeers}, nil
}

func (pH *PolicyHandler) DeleteExport(_ string) {
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
	policy, err := connPolicyFromBlob(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = pH.connectivityPDP.AddOrUpdatePolicy(*policy)
	if err != nil { // policy is syntactically ok, but semantically invalid - 422 is the status to return
		plog.Errorf("failed adding connectivity policy: %v", err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	plog.Infof("Added policy : %+v", *policy)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
}

func (pH *PolicyHandler) delConnPolicyReq(w http.ResponseWriter, r *http.Request) {
	policy, err := connPolicyFromBlob(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = pH.connectivityPDP.DeletePolicy(policy.Name, policy.Privileged)
	if err != nil {
		plog.Errorf("failed deleting connectivity policy: %v", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	plog.Infof("Deleted policy : %+v", *policy)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func connPolicyFromBlob(blob io.Reader) (*policytypes.ConnectivityPolicy, error) {
	connPolicy := &policytypes.ConnectivityPolicy{}
	err := json.NewDecoder(blob).Decode(connPolicy)
	if err != nil {
		plog.Errorf("failed decoding connectivity policy: %v", err)
		return nil, err
	}
	return connPolicy, nil
}

func (pH *PolicyHandler) AddLBPolicy(lbPolicy *LBPolicy) error {
	pH.loadBalancer.SetPolicy(lbPolicy.ServiceSrc, lbPolicy.ServiceDst, lbPolicy.Scheme, lbPolicy.DefaultMbg)
	return nil
}

func (pH *PolicyHandler) DeleteLBPolicy(lbPolicy *LBPolicy) error {
	pH.loadBalancer.deletePolicy(lbPolicy.ServiceSrc, lbPolicy.ServiceDst, lbPolicy.Scheme, lbPolicy.DefaultMbg)
	return nil
}

func (pH *PolicyHandler) AddAccessPolicy(policy *api.Policy) error {
	connPolicy, err := connPolicyFromBlob(bytes.NewReader(policy.Spec.Blob))
	if err != nil {
		return err
	}
	return pH.connectivityPDP.AddOrUpdatePolicy(*connPolicy)
}

func (pH *PolicyHandler) DeleteAccessPolicy(policy *api.Policy) error {
	connPolicy, err := connPolicyFromBlob(bytes.NewReader(policy.Spec.Blob))
	if err != nil {
		return err
	}
	return pH.connectivityPDP.DeletePolicy(connPolicy.Name, connPolicy.Privileged)
}

func (pH *PolicyHandler) policyWelcome(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("Welcome to Policy Engine"))
	if err != nil {
		log.Println(err)
	}
}

func (pH *PolicyHandler) init(router *chi.Mux) {
	pH.loadBalancer = NewLoadBalancer()
	pH.connectivityPDP = connectivitypdp.NewPDP()

	routes := pH.Routes(router)
	router.Mount(PolicyRoute, routes)
}

func StartPolicyDispatcher(router *chi.Mux) {
	plog.Infof("Policy Engine started")
	MyPolicyHandler.init(router)
}

func NewPolicyHandler() PolicyDecider {
	return &PolicyHandler{
		loadBalancer:    NewLoadBalancer(),
		connectivityPDP: connectivitypdp.NewPDP(),
	}
}
