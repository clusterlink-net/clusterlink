package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	"github.ibm.com/mbg-agent/pkg/controlplane/store"
)

type MbgHandler struct{}

func (m MbgHandler) Routes() chi.Router {
	r := store.GetChiRouter()

	r.Get("/", m.mbgWelcome)

	r.Route("/peer", func(r chi.Router) {
		r.Get("/", m.peerGetAll)           //GET /peer  - Get all MBG peers
		r.Get("/{mbgID}", m.peerGet)       //GET /peer/{mbgID}  - Get MBG peer Id
		r.Post("/{mbgID}", m.peerPost)     // Post /peer/{mbgID} - Add MBG peer Id to MBg peers list
		r.Delete("/{mbgID}", m.peerRemove) // Delete  /service  - Get specific local service
	})

	r.Route("/hello", func(r chi.Router) {
		r.Post("/{mbgID}", m.sendHello) // send Hello to MBG peer
		r.Post("/", m.sendHello2All)    // send Hello to MBG peer
	})

	r.Route("/hb", func(r chi.Router) {
		r.Post("/", m.handleHB) // Heartbeat messages
	})

	r.Route("/service", func(r chi.Router) {
		r.Post("/", m.addLocalServicePost)                   // Post /service  - Add local service to MBG
		r.Get("/", m.allLocalServicesGet)                    // Get  /service  - Get all local services in MBG
		r.Get("/{svcId}", m.localServiceGet)                 // Get  /service  - Get specific local service
		r.Delete("/{svcId}", m.delLocalService)              // Delete  /service  - Delete local service
		r.Delete("/{svcId}/peer", m.delLocalServiceFromPeer) // Delete  /service - Delete local service from peer

	})

	r.Route("/remoteservice", func(r chi.Router) {
		r.Post("/", m.addRemoteServicePost)      // Post /remoteservice  - Add Remote service to the MBG
		r.Get("/", m.allRemoteServicesGet)       // Get  /remoteservice  - Get all remote services in MBG
		r.Get("/{svcId}", m.remoteServiceGet)    // Get  /remoteservice  - Get specific remote service
		r.Delete("/{svcId}", m.delRemoteService) // Delete  /remoteservice  - Get specific remote service

	})

	r.Route("/expose", func(r chi.Router) {
		r.Post("/", m.exposePost) // Post /expose  - Expose mbg service
	})

	r.Route("/binding", func(r chi.Router) {
		r.Post("/", m.bindingCreate)          // Post /expose  - Bind remote service to local port
		r.Delete("/{svcId}", m.bindingDelete) // Disconnect /connect  - Disconnect mbg service
	})

	r.Route("/connect", func(r chi.Router) {
		r.Post("/", m.connectPost)      // Post /connect  - Connect mbg service
		r.Connect("/", m.handleConnect) // Connect /connect  - Connect mbg service
		r.Delete("/", m.connectDelete)  // Disconnect /connect  - Disconnect mbg service

	})

	return r
}

func (m MbgHandler) mbgWelcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to Multi-cloud Border Gateway"))
	if err != nil {
		log.Println(err)
	}
}
