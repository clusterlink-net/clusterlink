package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
)

const (
	bearerSchemaPrefix = "Bearer "
)

func (s *Server) addAuthzHandlers() {
	r := s.Router()

	r.Post(api.RemotePeerAuthorizationPath, s.PeerAuthorize)
	r.Post(api.DataplaneEgressAuthorizationPath, s.DataplaneEgressAuthorize)
	r.Post(api.DataplaneIngressAuthorizationPath, s.DataplaneIngressAuthorize)
}

// PeerAuthorize authorizes a remote peer controlplane request for accessing an exported service, yielding an access token.
func (s *Server) PeerAuthorize(w http.ResponseWriter, r *http.Request) {
	var req api.AuthorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp, err := s.cp.AuthorizeIngress(&controlplane.IngressAuthorizationRequest{Service: req.Service})
	switch {
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	case !resp.ServiceExists:
		w.WriteHeader(http.StatusNotFound)
		return
	case !resp.Allowed:
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	responseBody, err := json.Marshal(api.AuthorizationResponse{AccessToken: resp.AccessToken})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write(responseBody); err != nil {
		s.logger.Errorf("Cannot write http response: %v.", err)
	}
}

// DataplaneEgressAuthorize authorizes access to an imported service.
func (s *Server) DataplaneEgressAuthorize(w http.ResponseWriter, r *http.Request) {
	// TODO: verify that request originates from local dataplane

	ip := r.Header.Get(api.ClientIPHeader)
	if ip == "" {
		http.Error(w, fmt.Sprintf("missing '%s' header", api.ClientIPHeader), http.StatusBadRequest)
		return
	}

	imp := r.Header.Get(api.ImportHeader)
	if imp == "" {
		http.Error(w, fmt.Sprintf("missing '%s' header", api.ImportHeader), http.StatusBadRequest)
		return
	}

	resp, err := s.cp.AuthorizeEgress(&controlplane.EgressAuthorizationRequest{
		Import: imp,
		IP:     ip,
	})

	switch {
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	case !resp.ServiceExists:
		w.WriteHeader(http.StatusNotFound)
		return
	case !resp.Allowed:
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set(api.TargetClusterHeader, resp.RemotePeerCluster)
	w.Header().Set(api.AuthorizationHeader, bearerSchemaPrefix+resp.AccessToken)
}

// DataplaneIngressAuthorize authorizes a remote peer dataplane access to an exported service.
func (s *Server) DataplaneIngressAuthorize(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get(api.AuthorizationHeader)
	if authorization == "" {
		http.Error(w, fmt.Sprintf("missing '%s' header", api.AuthorizationHeader), http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(authorization, bearerSchemaPrefix) {
		http.Error(w, fmt.Sprintf("authorization header is not using the bearer scheme: %s", authorization), http.StatusBadRequest)
		return
	}
	token := strings.TrimPrefix(authorization, bearerSchemaPrefix)

	targetCluster, err := s.cp.ParseAuthorizationHeader(token)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set(api.TargetClusterHeader, targetCluster)
}
