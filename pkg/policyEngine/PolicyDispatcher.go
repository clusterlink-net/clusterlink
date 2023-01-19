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

type PolicyHandler struct{}

func (pH PolicyHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", pH.policyWelcome)

	r.Route("/NewConnectionRequest", func(r chi.Router) {
		r.Post("/", pH.newConnectionRequest) // New connection Request
	})

	r.Route("/AddPeerRequest", func(r chi.Router) {
		r.Post("/", pH.addPeerRequest) // New connection Request
	})
	return r
}

func (pH PolicyHandler) newConnectionRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.ConnectionRequestAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

func (pH PolicyHandler) addPeerRequest(w http.ResponseWriter, r *http.Request) {
	var requestAttr event.AddPeerAttr
	err := json.NewDecoder(r.Body).Decode(&requestAttr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	plog.Infof("Add Peer reqest : %+v", requestAttr)
	respJson, err := json.Marshal(event.AddPeerResp{Action: event.Allow})
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

func StartPolicyDispatcher(router *chi.Mux, ip string) {
	plog.Infof("Policy Engine [%v] started")

	router.Mount("/policy", PolicyHandler{}.Routes())

	//Use router to start the server
	plog.Infof("Starting HTTP server, listening to: %v", ip)
	err := http.ListenAndServe(ip, router)
	if err != nil {
		log.Println(err)
	}

}
