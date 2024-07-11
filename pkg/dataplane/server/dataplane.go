// Copyright (c) The ClusterLink Authors.
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

package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tlsv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	utiltls "github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

// Dataplane implements the server and api client which sends authorization to the control plane.
// Assumption: The caller implements lock mechanism which operating with clusters and listeners.
type Dataplane struct {
	ID             string
	router         *chi.Mux
	authzClient    authv3.AuthorizationClient
	parsedCertData *utiltls.ParsedCertData
	clusters       map[string]*cluster.Cluster
	listeners      map[string]*listener.Listener
	listenerEnd    map[string]chan bool

	tlsConfigLock sync.RWMutex
	tlsConfig     *tls.Config

	logger *logrus.Entry
}

// GetClusterTarget returns the cluster address:port from the cluster map.
func (d *Dataplane) GetClusterTarget(name string) (string, error) {
	if _, ok := d.clusters[name]; !ok {
		return "", fmt.Errorf("unable to find %s in cluster map", name)
	}
	ep := d.clusters[name].LoadAssignment.GetEndpoints()[0].LbEndpoints[0].GetEndpoint().Address.GetSocketAddress()
	return ep.GetAddress() + ":" + strconv.Itoa(int(ep.GetPortValue())), nil
}

// GetClusterHost returns the cluster hostname after trimming ":".
func (d *Dataplane) GetClusterHost(name string) (string, error) {
	hostname, err := d.GetClusterHostname(name)
	if err != nil {
		return "", err
	}

	host, _, err := net.SplitHostPort(hostname)
	if err != nil {
		return "", fmt.Errorf("cluster hostname '%s' cannot be parsed: %w", hostname, err)
	}
	return host, nil
}

// GetClusterHostname returns the cluster hostname.
func (d *Dataplane) GetClusterHostname(name string) (string, error) {
	if _, ok := d.clusters[name]; !ok {
		return "", fmt.Errorf("unable to find %s in cluster map", name)
	}
	return d.clusters[name].LoadAssignment.GetEndpoints()[0].LbEndpoints[0].GetEndpoint().Hostname, nil
}

// AddCluster adds/updates a cluster to the map.
func (d *Dataplane) AddCluster(c *cluster.Cluster) {
	d.clusters[c.Name] = c
}

// RemoveCluster adds a cluster to the map.
func (d *Dataplane) RemoveCluster(name string) {
	delete(d.clusters, name)
}

// GetClusters returns the clusters map.
func (d *Dataplane) GetClusters() map[string]*cluster.Cluster {
	return d.clusters
}

// AddListener adds a listener to the map.
func (d *Dataplane) AddListener(ln *listener.Listener) {
	listenerName := strings.TrimPrefix(ln.Name, api.ImportListenerPrefix)
	if le, ok := d.listeners[listenerName]; ok {
		// Check if there is an update to the listener address/port
		if ln.Address.GetSocketAddress().GetAddress() == le.Address.GetSocketAddress().GetAddress() &&
			ln.Address.GetSocketAddress().GetPortValue() == le.Address.GetSocketAddress().GetPortValue() {
			return
		}
		d.listenerEnd[listenerName] <- true
	}
	d.listeners[listenerName] = ln
	go func() {
		d.CreateListener(listenerName,
			ln.Address.GetSocketAddress().GetAddress(),
			ln.Address.GetSocketAddress().GetPortValue())
	}()
}

// RemoveListener removes a listener.
func (d *Dataplane) RemoveListener(name string) {
	delete(d.listeners, name)
	d.listenerEnd[name] <- true
}

// GetListeners returns the listeners map.
func (d *Dataplane) GetListeners() map[string]*listener.Listener {
	return d.listeners
}

// AddSecret adds a secret (dataplane cert or CA).
func (d *Dataplane) AddSecret(secret *tlsv3.Secret) error {
	switch secret.Name {
	case api.CertificateSecret:
		return d.addCertificateSecret(secret)
	case api.ValidationSecret:
		return d.addValidationSecret(secret)
	}

	return fmt.Errorf("unknown secret: %s", secret.Name)
}

func (d *Dataplane) addCertificateSecret(secret *tlsv3.Secret) error {
	tlsSecret := secret.GetTlsCertificate()
	if tlsSecret == nil {
		return fmt.Errorf("not a TLS certificate secret")
	}

	certChain := tlsSecret.CertificateChain
	if certChain == nil {
		return fmt.Errorf("no certificate chain")
	}

	certBytes := certChain.GetInlineBytes()
	if certBytes == nil {
		return fmt.Errorf("no certificate chain bytes embedded")
	}

	privateKey := tlsSecret.PrivateKey
	if privateKey == nil {
		return fmt.Errorf("no private key")
	}

	keyBytes := privateKey.GetInlineBytes()
	if keyBytes == nil {
		return fmt.Errorf("no private key bytes embedded")
	}

	certificate, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		return fmt.Errorf("error parsing certificate: %w", err)
	}

	d.tlsConfigLock.Lock()
	defer d.tlsConfigLock.Unlock()
	newTLSConfig := d.tlsConfig.Clone()
	newTLSConfig.Certificates = []tls.Certificate{certificate}
	d.tlsConfig = newTLSConfig

	return nil
}

func (d *Dataplane) addValidationSecret(secret *tlsv3.Secret) error {
	validationContext := secret.GetValidationContext()
	if validationContext == nil {
		return fmt.Errorf("not a validation context secret")
	}

	trustedCa := validationContext.TrustedCa
	if trustedCa == nil {
		return fmt.Errorf("no trusted CA")
	}

	caBytes := trustedCa.GetInlineBytes()
	if caBytes == nil {
		return fmt.Errorf("no CA bytes embedded")
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caBytes) {
		return fmt.Errorf("error parsing CA")
	}

	d.tlsConfigLock.Lock()
	defer d.tlsConfigLock.Unlock()
	newTLSConfig := d.tlsConfig.Clone()
	newTLSConfig.ClientCAs = caCertPool
	newTLSConfig.RootCAs = caCertPool
	d.tlsConfig = newTLSConfig

	return nil
}

// NewDataplane returns a new dataplane HTTP server.
func NewDataplane(
	dataplaneID string,
	controlplaneClient grpc.ClientConnInterface,
	parsedCertData *utiltls.ParsedCertData,
) *Dataplane {
	logger := logrus.WithField("component", "dataplane.server.http")

	router := chi.NewRouter()
	router.Use(middleware.Recoverer)
	if logrus.GetLevel() >= logrus.DebugLevel {
		router.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
			Logger:  logger,
			NoColor: true,
		}))
	}

	dp := &Dataplane{
		ID:             dataplaneID,
		router:         router,
		authzClient:    authv3.NewAuthorizationClient(controlplaneClient),
		parsedCertData: parsedCertData,
		clusters:       make(map[string]*cluster.Cluster),
		listeners:      make(map[string]*listener.Listener),
		listenerEnd:    make(map[string]chan bool),
		tlsConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ClientAuth:         tls.RequireAndVerifyClientCert,
			ClientSessionCache: tls.NewLRUClientSessionCache(64),
		},
		logger: logger,
	}

	dp.addAuthzHandlers()
	return dp
}
