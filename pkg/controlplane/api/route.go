package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
	cp "github.ibm.com/mbg-agent/pkg/controlplane"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
)

type MbgHandler struct{}

func (m MbgHandler) Routes() chi.Router {
	r := store.GetChiRouter()

	r.Get("/", m.mbgWelcome)

	r.Route("/peer", func(r chi.Router) {
		r.Get("/", cp.PeerGetAllHandler)           //GET /peer  - Get all MBG peers
		r.Get("/{mbgID}", cp.PeerGetHandler)       //GET /peer/{mbgID}  - Get MBG peer Id
		r.Post("/{mbgID}", cp.PeerPostHandler)     // Post /peer/{mbgID} - Add MBG peer Id to MBg peers list
		r.Delete("/{mbgID}", cp.PeerRemoveHandler) // Delete  /service  - Get specific local service
	})

	r.Route("/hello", func(r chi.Router) {
		r.Post("/{mbgID}", cp.SendHelloHandler) // send Hello to MBG peer
		r.Post("/", cp.SendHello2AllHandler)    // send Hello to MBG peer
	})

	r.Route("/hb", func(r chi.Router) {
		r.Post("/", cp.HandleHB) // Heartbeat messages
	})

	r.Route("/service", func(r chi.Router) {
		r.Post("/", cp.AddLocalServicePostHandler)                   // Post /service  - Add local service to MBG
		r.Get("/", cp.AllLocalServicesGetHandler)                    // Get  /service  - Get all local services in MBG
		r.Get("/{svcId}", cp.LocalServiceGetHandler)                 // Get  /service  - Get specific local service
		r.Delete("/{svcId}", cp.DelLocalServiceHandler)              // Delete  /service  - Delete local service
		r.Delete("/{svcId}/peer", cp.DelLocalServiceFromPeerHandler) // Delete  /service - Delete local service from peer

	})

	r.Route("/remoteservice", func(r chi.Router) {
		r.Post("/", cp.AddRemoteServicePostHandler)      // Post /remoteservice  - Add Remote service to the MBG
		r.Get("/", cp.AllRemoteServicesGetHandler)       // Get  /remoteservice  - Get all remote services in MBG
		r.Get("/{svcId}", cp.RemoteServiceGetHandler)    // Get  /remoteservice  - Get specific remote service
		r.Delete("/{svcId}", cp.DelRemoteServiceHandler) // Delete  /remoteservice  - Get specific remote service

	})

	r.Route("/expose", func(r chi.Router) {
		r.Post("/", cp.ExposePostHandler) // Post /expose  - Expose mbg service
	})

	r.Route("/binding", func(r chi.Router) {
		r.Post("/", cp.BindingCreateHandler)          // Post /expose  - Bind remote service to local port
		r.Delete("/{svcId}", cp.BindingDeleteHandler) // Disconnect /connect  - Disconnect mbg service
	})

	r.Route("/connect", func(r chi.Router) {
		r.Post("/", cp.ConnectPostHandler)      // Post /connect  - Connect mbg service
		r.Connect("/", cp.HandleConnectHandler) // Connect /connect  - Connect mbg service
		r.Delete("/", cp.ConnectDeleteHandler)  // Disconnect /connect  - Disconnect mbg service

	})

	return r
}

func (m MbgHandler) mbgWelcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to Multi-cloud Border Gateway"))
	if err != nil {
		log.Println(err)
	}
}
