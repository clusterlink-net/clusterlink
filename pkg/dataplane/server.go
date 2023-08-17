package dataplane

import (
	"io"
	"net/http"

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

	d.Router.Route("/imports", func(r chi.Router) {
		// Dataplane functions
		r.Route("/serviceEndpoint", func(r chi.Router) {
			r.Post("/", d.AddImportServiceEndpointHandler)          // Post /imports/serviceEndpoint            - Add endpoint for import service
			r.Delete("/{svcId}", d.DelImportServiceEndpointHandler) // Delete  /imports/serviceEndpoint/{svcId} - Delete sendpoint for import service
		})
		// Controlplane Forwarding
		r.Post("/", d.controlPlaneForwardingHandler)       // Post /remoteservice            - Add Remote service
		r.Get("/", d.controlPlaneForwardingHandler)        // Get  /remoteservice            - Get all remote services
		r.Get("/{id}", d.controlPlaneForwardingHandler)    // Get  /remoteservice/{svcId}    - Get specific remote service
		r.Delete("/{id}", d.controlPlaneForwardingHandler) // Delete  /remoteservice/{svcId} - Delete specific remote service
	})

	d.Router.Route("/exports", func(r chi.Router) {
		// Dataplane functions
		r.Route("/serviceEndpoint", func(r chi.Router) {
			r.Post("/", d.MTLSexportServiceEndpointHandler)   // Post /exports/serviceEndpoint    - Create Connection to a service (mTLS) for export service
			r.Connect("/", d.TCPexportServiceEndpointHandler) // Connect /exports/serviceEndpoint - Create Connection to a service (TCP) for export service
		})
		// Controlplane Forwarding
		r.Post("/", d.controlPlaneForwardingHandler)       // Post /service    - Add local service
		r.Get("/", d.controlPlaneForwardingHandler)        // Get  /service    - Get all local services
		r.Get("/{id}", d.controlPlaneForwardingHandler)    // Get  /service    - Get specific local service
		r.Delete("/{id}", d.controlPlaneForwardingHandler) // Delete  /service - Delete local service
	})
	// Controlplane Forwarding
	d.Router.Route("/peers", func(r chi.Router) {
		r.Get("/", d.controlPlaneForwardingHandler)        // GET    /peers      - Get all peers
		r.Post("/", d.controlPlaneForwardingHandler)       // Post   /peers      - Add peer Id to peers list
		r.Get("/{id}", d.controlPlaneForwardingHandler)    // GET    /peers/{id} - Get peer Id
		r.Delete("/{id}", d.controlPlaneForwardingHandler) // Delete /peers/{id} - Delete peer
	})

	d.Router.Route("/hb", func(r chi.Router) {
		r.Post("/", d.controlPlaneForwardingHandler) // Heartbeat messages
	})

	d.Router.Route("/bindings", func(r chi.Router) {
		r.Post("/", d.controlPlaneForwardingHandler)       // Post   /bindings   - Bind remote service to local port
		r.Get("/{id}", d.controlPlaneForwardingHandler)    // Get    /bindings   - Bind remote service to local port
		r.Delete("/{id}", d.controlPlaneForwardingHandler) // Delete /bindings - Remove Binding of remote service to local port
	})

	d.Router.Route("/newRemoteConnection", func(r chi.Router) {
		r.Post("/", d.controlPlaneForwardingHandler) // Post /newRemoteConnection - New remote connection parameters check
	})

	d.Router.Route("/newLocalConnection", func(r chi.Router) {
		r.Post("/", d.controlPlaneForwardingHandler) // Post /newLocalConnection  - New connection parameters check
	})

	d.Router.Route("/policy/acl", func(r chi.Router) {
		r.Get("/", d.controlPlaneForwardingHandler)
		r.Post("/add", d.controlPlaneForwardingHandler)    // Add ACL Rule
		r.Post("/delete", d.controlPlaneForwardingHandler) // Delete ACL Rule
	})

	d.Router.Route("/policy/lb", func(r chi.Router) {
		r.Get("/", d.controlPlaneForwardingHandler)
		r.Post("/add", d.controlPlaneForwardingHandler)    // Add LB Policy
		r.Post("/delete", d.controlPlaneForwardingHandler) // Delete LB Policy
	})

	d.Router.Route("/metrics", func(r chi.Router) {
		r.Get("/ConnectionStatus", d.controlPlaneForwardingHandler)
	})
}

// Welcome message to data-plane
func (d *Dataplane) welcome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Welcome to dataplane agent"))
	clog.Info("Welcome to dataplane agent")
	if err != nil {
		clog.Println(err)
	}
}

// Forwarding request to control-plane
func (d *Dataplane) controlPlaneForwardingHandler(w http.ResponseWriter, r *http.Request) {
	forwardingURL := d.Store.GetControlPlaneAddr() + r.URL.Path
	// Create a new request to the forwarding URL
	forwardingReq, err := http.NewRequest(r.Method, forwardingURL, r.Body)
	if err != nil {
		clog.Error("Forwarding error in NewRequest", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	client := d.Store.GetLocalHttpClient()
	// Send the forwardingURL request
	resp, err := client.Do(forwardingReq)
	forwardingReq.Close = true
	if err != nil {
		clog.Error("Forwarding error in sending operation", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Copy the response headers from the forwarding response
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set the response status code
	w.WriteHeader(resp.StatusCode)

	// Copy the response body from the forwarding response
	if _, err = io.Copy(w, resp.Body); err != nil && err != io.EOF {
		clog.Warnf("failed to copy response body in forwarding: %+v", err)
	}
}
