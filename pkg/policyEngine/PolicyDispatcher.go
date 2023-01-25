/**********************************************************/
/* Package Policy contain all Policies and data structure
/* related to Policy that can run in mbg
/**********************************************************/
package policyEngine

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	event "github.ibm.com/mbg-agent/pkg/eventManager"
)

var plog = logrus.WithField("component", "PolicyEngine")

type MbgState struct {
	mbgPeers *[]string
}

type PolicyHandler struct {
	SubscriptionMap map[string][]string
	accessControl   *AccessControl
	loadBalancer    *LoadBalancer
	mbgState        MbgState
}

func (pH PolicyHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", pH.policyWelcome)

	r.Route("/"+event.NewConnectionRequest, func(r chi.Router) {
		r.Post("/", pH.newConnectionRequest) // New connection Request
	})

	r.Route("/"+event.AddPeerRequest, func(r chi.Router) {
		r.Post("/", pH.addPeerRequest) // New connection Request
	})
	r.Route("/"+event.NewRemoteService, func(r chi.Router) {
		r.Post("/", pH.newRemoteService) // New connection Request
	})
	r.Route("/"+event.ExposeRequest, func(r chi.Router) {
		r.Post("/", pH.exposeRequest) // New connection Request
	})

	r.Route("/acl", func(r chi.Router) {
		r.Get("/", pH.accessControl.GetRuleReq)
		r.Post("/add", pH.accessControl.AddRuleReq) // Add ACL Rule
		r.Post("/delete", pH.accessControl.DelRuleReq)
	})

	r.Route("/lb/", func(r chi.Router) {
		r.Post("/setPolicy", pH.loadBalancer.SetPolicyReq) // Add LB Policy
	})
	return r
}

func (pH PolicyHandler) addPeer(peerMbg string) {
	*pH.mbgState.mbgPeers = append(*pH.mbgState.mbgPeers, peerMbg)
	plog.Infof("Added Peer %+v", pH.mbgState.mbgPeers)
}

func (pH PolicyHandler) newConnectionRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.ConnectionRequestAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("New connection request : %+v -> %+v", requestAttr, pH.SubscriptionMap[event.NewConnectionRequest])

	var action int
	var targetMbg string
	var bitrate int
	for _, agent := range pH.SubscriptionMap[event.NewConnectionRequest] {
		plog.Infof("Applying Policy %s", agent)
		switch agent {
		case "AccessControl":
			if requestAttr.Direction == event.Outgoing {
				action, bitrate = pH.accessControl.Lookup(requestAttr.SrcService, requestAttr.DstService, targetMbg)
			} else {
				action, bitrate = pH.accessControl.Lookup(requestAttr.SrcService, requestAttr.DstService, requestAttr.OtherMbg)
			}
		case "LoadBalancer":
			plog.Infof("Looking up loadbalancer drection %d", requestAttr.Direction)
			if requestAttr.Direction == event.Outgoing {
				targetMbg = pH.loadBalancer.Lookup(requestAttr.DstService)
			}
		default:
			plog.Errorf("Unrecognized Policy Agent")
		}
	}
	respJson, err := json.Marshal(event.ConnectionRequestResp{Action: action, TargetMbg: targetMbg, BitRate: bitrate})
	if err != nil {
		panic(err)
	}
	plog.Infof("Response : %+v", event.ConnectionRequestResp{Action: action, TargetMbg: targetMbg, BitRate: bitrate})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(respJson)
	if err != nil {
		plog.Errorf("Unable to write response %v", err)
	}
}

func (pH PolicyHandler) addPeerRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.AddPeerAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Add Peer reqest : %+v -> %+v", requestAttr, pH.SubscriptionMap[event.AddPeerRequest])
	//TODO : Convert this into standard interfaces. This requires formalizing Policy I/O
	var action int

	for _, agent := range pH.SubscriptionMap[event.AddPeerRequest] {
		switch agent {
		case "AccessControl":
			_, action, _ = pH.accessControl.RulesLookup(event.Wildcard, event.Wildcard, requestAttr.PeerMbg)
		default:
			plog.Errorf("Unrecognized Policy Agent")
		}
	}
	respJson, err := json.Marshal(event.AddPeerResp{Action: action})
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(respJson)
	if err != nil {
		plog.Errorf("Unable to write response %v", err)
	}
	// Update States
	if action != event.Deny {
		pH.addPeer(requestAttr.PeerMbg)
	}

}

func (pH PolicyHandler) newRemoteService(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.NewRemoteServiceAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("New Remote Service request : %+v -> %+v", requestAttr, pH.SubscriptionMap[event.NewRemoteService])
	//TODO : Convert this into standard interfaces. This requires formalizing Policy I/O
	var action int

	for _, agent := range pH.SubscriptionMap[event.NewRemoteService] {
		switch agent {
		case "AccessControl":
			action, _ = pH.accessControl.Lookup(event.Wildcard, requestAttr.Service, requestAttr.Mbg)
		default:
			plog.Errorf("Unrecognized Policy Agent")
		}
	}
	respJson, err := json.Marshal(event.NewRemoteServiceResp{Action: action})
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(respJson)
	if err != nil {
		plog.Errorf("Unable to write response %v", err)
	}
	// Update States
	if action != event.Deny {
		pH.loadBalancer.AddToServiceMap(requestAttr.Service, requestAttr.Mbg)
	}
}

func (pH PolicyHandler) exposeRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.ExposeRequestAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("New Expose request : %+v -> %+v", requestAttr, pH.SubscriptionMap[event.ExposeRequest])
	//TODO : Convert this into standard interfaces. This requires formalizing Policy I/O
	action := event.AllowAll
	var mbgPeers []string

	for _, agent := range pH.SubscriptionMap[event.ExposeRequest] {
		switch agent {
		case "AccessControl":
			plog.Infof("Checking accesses for %+v", pH.mbgState.mbgPeers)
			action, mbgPeers = pH.accessControl.LookupTarget(requestAttr.Service, pH.mbgState.mbgPeers)
		default:
			plog.Errorf("Unrecognized Policy Agent")
		}
	}
	respJson, err := json.Marshal(event.ExposeRequestResp{Action: action, TargetMbgs: mbgPeers})
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_, err = w.Write(respJson)
	if err != nil {
		plog.Errorf("Unable to write response %v", err)
	}
}

func (pH PolicyHandler) policyWelcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to Policy Engine"))
	if err != nil {
		log.Println(err)
	}
}
func (pH PolicyHandler) init(router *chi.Mux, ip string) {
	pH.SubscriptionMap = make(map[string][]string)
	pH.mbgState.mbgPeers = &([]string{})
	policyList1 := []string{"LoadBalancer", "AccessControl"}
	policyList2 := []string{"AccessControl"}

	pH.accessControl = &AccessControl{}
	pH.loadBalancer = &LoadBalancer{}
	pH.accessControl.Init()
	pH.loadBalancer.Init()

	pH.SubscriptionMap[event.NewConnectionRequest] = policyList1
	pH.SubscriptionMap[event.AddPeerRequest] = policyList2
	pH.SubscriptionMap[event.NewRemoteService] = policyList2
	pH.SubscriptionMap[event.ExposeRequest] = policyList2

	plog.Infof("Subscription Map - %+v", pH.SubscriptionMap)

	routes := pH.Routes()

	router.Mount("/policy", routes)
	plog.Infof("Policy Routes : %+v", routes)

	//Use router to start the server
	plog.Infof("Starting HTTP server, listening to: %v", ip)
	err := http.ListenAndServe(ip, router)
	if err != nil {
		log.Println(err)
	}
}

func StartPolicyDispatcher(router *chi.Mux, ip string) {
	plog.Infof("Policy Engine started")

	var myPolicyHandler PolicyHandler

	myPolicyHandler.init(router, ip)

}
