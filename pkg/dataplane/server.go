package dataplane

import (
	"net/http"
	"net/http/httputil"

	"github.com/go-chi/chi"

	"github.ibm.com/mbg-agent/pkg/utils/netutils"
)

func (d *Dataplane) StartServer(port string) {
	clog.Infoln("Start Dataplane")
	// Set router
	d.Router = chi.NewRouter()
	d.routes()
	// start mTLS
	clog.Infoln("Dataplane server listen to port:", port)
	ca, cert, key := d.Store.GetCerts()
	if d.Store.Dataplane == "mtls" {
		netutils.StartMTLSServer(":"+port, ca, cert, key, d.Router)
	} else {
		netutils.StartHTTPServer(":"+port, d.Router)
	}
}
func (d *Dataplane) routes() {
	d.Router.Get("/", d.welcome)
	// Dataplane functions
	d.Router.Route("/imports", func(r chi.Router) {
		r.Route("/serviceEndpoint", func(r chi.Router) {
			r.Post("/", d.AddImportServiceEndpointHandler)          // Post /imports/serviceEndpoint            - Add endpoint for import service
			r.Delete("/{svcId}", d.DelImportServiceEndpointHandler) // Delete  /imports/serviceEndpoint/{svcId} - Delete sendpoint for import service
		})

	})

	d.Router.Route("/exports", func(r chi.Router) {
		r.Route("/serviceEndpoint", func(r chi.Router) {
			r.Post("/", d.MTLSexportServiceEndpointHandler)   // Post /exports/serviceEndpoint    - Create Connection to a service (mTLS) for export service
			r.Connect("/", d.TCPexportServiceEndpointHandler) // Connect /exports/serviceEndpoint - Create Connection to a service (TCP) for export service
		})
	})
	// Routing to controlplane functions
	d.Router.Route("/peer", func(r chi.Router) {
		r.Get("/", d.controlPlaneRedirectHandler)        // GET    /peer      - Get all peers
		r.Post("/", d.controlPlaneRedirectHandler)       // Post   /peer      - Add peer Id to peers list
		r.Get("/{id}", d.controlPlaneRedirectHandler)    // GET    /peer/{id} - Get peer Id
		r.Delete("/{id}", d.controlPlaneRedirectHandler) // Delete /peer/{id} - Delete peer
	})

	d.Router.Route("/hello", func(r chi.Router) {
		r.Post("/", d.controlPlaneRedirectHandler)         // Post /hello Hello to all peers
		r.Post("/{peerID}", d.controlPlaneRedirectHandler) // Post /hello/{peerID} send Hello to a peer
	})

	d.Router.Route("/hb", func(r chi.Router) {
		r.Post("/", d.controlPlaneRedirectHandler) // Heartbeat messages
	})

	d.Router.Route("/service", func(r chi.Router) {
		r.Post("/", d.controlPlaneRedirectHandler)            // Post /service    - Add local service
		r.Get("/", d.controlPlaneRedirectHandler)             // Get  /service    - Get all local services
		r.Get("/{id}", d.controlPlaneRedirectHandler)         // Get  /service    - Get specific local service
		r.Delete("/{id}", d.controlPlaneRedirectHandler)      // Delete  /service - Delete local service
		r.Delete("/{id}/peer", d.controlPlaneRedirectHandler) // Delete  /service - Delete local service from peer

	})

	d.Router.Route("/remoteservice", func(r chi.Router) {
		r.Post("/", d.controlPlaneRedirectHandler)          // Post /remoteservice            - Add Remote service
		r.Get("/", d.controlPlaneRedirectHandler)           // Get  /remoteservice            - Get all remote services
		r.Get("/{svcId}", d.controlPlaneRedirectHandler)    // Get  /remoteservice/{svcId}    - Get specific remote service
		r.Delete("/{svcId}", d.controlPlaneRedirectHandler) // Delete  /remoteservice/{svcId} - Delete specific remote service

	})

	d.Router.Route("/expose", func(r chi.Router) {
		r.Post("/", d.controlPlaneRedirectHandler) // Post /expose  - Expose  service
	})

	d.Router.Route("/binding", func(r chi.Router) {
		r.Post("/", d.controlPlaneRedirectHandler)          // Post /binding   - Bind remote service to local port
		r.Delete("/{svcId}", d.controlPlaneRedirectHandler) // Delete /binding - Remove Binding of remote service to local port
	})

	d.Router.Route("/newRemoteConnection", func(r chi.Router) {
		r.Post("/", d.controlPlaneRedirectHandler) // Post /newRemoteConnection - New remote connection parameters check
	})
	d.Router.Route("/newLocalConnection", func(r chi.Router) {
		r.Post("/", d.controlPlaneRedirectHandler) // Post /newLocalConnection  - New connection parameters check
	})

}

// Welcome message to data-plane
func (d Dataplane) welcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to dataplane agent"))
	clog.Info("Welcome to dataplane agent")
	if err != nil {
		clog.Println(err)
	}
}

// Forwarding request to control-plane
func (d *Dataplane) controlPlaneRedirectHandler(w http.ResponseWriter, r *http.Request) {

	redirectURL := d.Store.GetControlPlaneAddr() + r.URL.Path

	// Create a new request to the redirect URL
	redirectReq, err := http.NewRequest(r.Method, redirectURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := d.Store.GetLocalHttpClient()
	// Send the redirect request
	resp, err := client.Do(redirectReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the response headers from the redirect response
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set the response status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body from the redirect response
	_, err = httputil.DumpResponse(resp, true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
