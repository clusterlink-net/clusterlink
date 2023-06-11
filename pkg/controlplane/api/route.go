package api

import (
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"

	cp "github.ibm.com/mbg-agent/pkg/controlplane"
	"github.ibm.com/mbg-agent/pkg/controlplane/health"
	"github.ibm.com/mbg-agent/pkg/controlplane/store"
	dp "github.ibm.com/mbg-agent/pkg/dataplane"
)

type MbgHandler struct{}

func (m MbgHandler) Routes() chi.Router {
	r := store.GetChiRouter()

	r.Get("/", m.mbgWelcome)

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

	r.Route("/connect", func(r chi.Router) {
		r.Post("/", dp.MTLSConnectHandler)   // Post /connect    - Create Connection to a service (mTLS)
		r.Connect("/", dp.TCPConnectHandler) // Connect /connect - Create Connection to a service (TCP)
	})

	return r
}

func (m MbgHandler) mbgWelcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to Multi-cloud Border Gateway"))
	if err != nil {
		log.Println(err)
	}
}
