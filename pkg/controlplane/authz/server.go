// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authz

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	utilhttp "github.com/clusterlink-net/clusterlink/pkg/util/http"
)

const (
	bearerSchemaPrefix = "Bearer "
)

type server struct {
	manager *Manager
	logger  *logrus.Entry
}

// RegisterHandlers registers the HTTP handlers for dataplane authz requests.
func RegisterHandlers(manager *Manager, srv *utilhttp.Server) {
	router := srv.Router()
	server := &server{
		manager: manager,
		logger:  logrus.WithField("component", "controlplane.authz.server"),
	}

	router.Post(api.DataplaneEgressAuthorizationPath, server.DataplaneEgressAuthorize)
	router.Post(api.DataplaneIngressAuthorizationPath, server.DataplaneIngressAuthorize)

	router.Get(api.HeartbeatPath, server.Heartbeat)
	router.Post(api.RemotePeerAuthorizationPath, server.PeerAuthorize)
}

// DataplaneEgressAuthorize authorizes access to an imported service.
func (s *server) DataplaneEgressAuthorize(w http.ResponseWriter, r *http.Request) {
	// TODO: verify that request originates from local dataplane

	ip := r.Header.Get(api.ClientIPHeader)
	if ip == "" {
		http.Error(w, fmt.Sprintf("missing '%s' header", api.ClientIPHeader), http.StatusBadRequest)
		return
	}

	importName := r.Header.Get(api.ImportNameHeader)
	if importName == "" {
		http.Error(w, fmt.Sprintf("missing '%s' header", api.ImportNameHeader), http.StatusBadRequest)
		return
	}

	importNamespace := r.Header.Get(api.ImportNamespaceHeader)
	if importNamespace == "" {
		http.Error(w, fmt.Sprintf("missing '%s' header", api.ImportNamespaceHeader), http.StatusBadRequest)
		return
	}

	resp, err := s.manager.authorizeEgress(r.Context(), &egressAuthorizationRequest{
		ImportName: types.NamespacedName{
			Namespace: importNamespace,
			Name:      importName,
		},
		IP: ip,
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
func (s *server) DataplaneIngressAuthorize(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get(api.AuthorizationHeader)
	if authorization == "" {
		http.Error(w, fmt.Sprintf("missing '%s' header", api.AuthorizationHeader), http.StatusBadRequest)
		return
	}

	if !strings.HasPrefix(authorization, bearerSchemaPrefix) {
		http.Error(w, fmt.Sprintf("authorization header is not using the bearer scheme: %s", authorization),
			http.StatusBadRequest)
		return
	}
	token := strings.TrimPrefix(authorization, bearerSchemaPrefix)

	targetCluster, err := s.manager.parseAuthorizationHeader(token)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.Header().Set(api.TargetClusterHeader, targetCluster)
}

// Heartbeat returns a response for heartbeat checks from remote peers.
func (s *server) Heartbeat(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// PeerAuthorize authorizes a remote peer controlplane request for accessing an exported service,
// yielding an access token.
func (s *server) PeerAuthorize(w http.ResponseWriter, r *http.Request) {
	var req api.AuthorizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 || len(r.TLS.PeerCertificates[0].DNSNames) != 2 ||
		r.TLS.PeerCertificates[0].DNSNames[0] == "" {
		http.Error(w, "certificate does not contain a valid DNS name for the peer gateway", http.StatusBadRequest)
		return
	}

	peerName := r.TLS.PeerCertificates[0].DNSNames[0]
	resp, err := s.manager.authorizeIngress(
		r.Context(),
		&ingressAuthorizationRequest{
			ServiceName: types.NamespacedName{
				Namespace: req.ServiceNamespace,
				Name:      req.ServiceName,
			},
		},
		peerName)
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
