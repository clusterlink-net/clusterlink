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

package authz

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/authz/connectivitypdp"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/peer"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

const (
	// the number of seconds a JWT access token is valid before it expires.
	jwtExpirySeconds = 5

	ServiceNameLabel      = "clusterlink/metadata.serviceName"
	ServiceNamespaceLabel = "clusterlink/metadata.serviceNamespace"
	GatewayNameLabel      = "clusterlink/metadata.gatewayName"
)

// egressAuthorizationRequest (from local dataplane)
// represents a request for accessing an imported service.
type egressAuthorizationRequest struct {
	// ImportName is the name of the requested imported service.
	ImportName types.NamespacedName
	// IP address of the client connecting to the service.
	IP string
}

// egressAuthorizationResponse (to local dataplane) represents a response for an egressAuthorizationRequest.
type egressAuthorizationResponse struct {
	// ServiceExists is true if the requested service exists.
	ServiceExists bool
	// Allowed is true if the request is allowed.
	Allowed bool
	// RemotePeerCluster is the cluster name of the remote peer where the connection should be routed to.
	RemotePeerCluster string
	// AccessToken is a token that allows accessing the requested service.
	AccessToken string
}

// ingressAuthorizationRequest (to remote peer controlplane) represents a request for accessing an exported service.
type ingressAuthorizationRequest struct {
	// Service is the name of the requested exported service.
	ServiceName types.NamespacedName
}

// ingressAuthorizationResponse (from remote peer controlplane) represents a response for an ingressAuthorizationRequest.
type ingressAuthorizationResponse struct {
	// ServiceExists is true if the requested service exists.
	ServiceExists bool
	// Allowed is true if the request is allowed.
	Allowed bool
	// AccessToken is a token that allows accessing the requested service.
	AccessToken string
}

type podInfo struct {
	name      string
	namespace string
	labels    map[string]string
}

// Manager manages the authorization dataplane connections.
type Manager struct {
	client    client.Client
	namespace string

	loadBalancer    *LoadBalancer
	connectivityPDP *connectivitypdp.PDP

	peerName   string
	peerTLS    *tls.ParsedCertData
	peerLock   sync.RWMutex
	peerClient map[string]*peer.Client

	podLock sync.RWMutex
	ipToPod map[string]types.NamespacedName
	podList map[types.NamespacedName]podInfo

	jwkSignKey   jwk.Key
	jwkVerifyKey jwk.Key

	logger *logrus.Entry
}

// AddPeer defines a new route target for egress dataplane connections.
func (m *Manager) AddPeer(pr *v1alpha1.Peer) {
	m.logger.Infof("Adding peer '%s'.", pr.Name)

	// initialize peer client
	cl := peer.NewClient(pr, m.peerTLS.ClientConfig(pr.Name))

	m.peerLock.Lock()
	m.peerClient[pr.Name] = cl
	m.peerLock.Unlock()
}

// DeletePeer removes the possibility for egress dataplane connections to be routed to a given peer.
func (m *Manager) DeletePeer(name string) {
	m.logger.Infof("Deleting peer '%s'.", name)

	m.peerLock.Lock()
	delete(m.peerClient, name)
	m.peerLock.Unlock()
}

// AddAccessPolicy adds an access policy to allow/deny specific connections.
func (m *Manager) AddAccessPolicy(policy *connectivitypdp.AccessPolicy) error {
	return m.connectivityPDP.AddOrUpdatePolicy(policy)
}

// DeleteAccessPolicy removes an access policy to allow/deny specific connections.
func (m *Manager) DeleteAccessPolicy(name types.NamespacedName, privileged bool) error {
	return m.connectivityPDP.DeletePolicy(name, privileged)
}

// deletePod deletes pod to ipToPod list.
func (m *Manager) deletePod(podID types.NamespacedName) {
	m.podLock.Lock()
	defer m.podLock.Unlock()

	delete(m.podList, podID)
	for key, pod := range m.ipToPod {
		if pod.Name == podID.Name && pod.Namespace == podID.Namespace {
			delete(m.ipToPod, key)
		}
	}
}

// addPod adds or updates pod to ipToPod and podList.
func (m *Manager) addPod(pod *v1.Pod) {
	m.podLock.Lock()
	defer m.podLock.Unlock()

	podID := types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}
	m.podList[podID] = podInfo{name: pod.Name, namespace: pod.Namespace, labels: pod.Labels}
	for _, ip := range pod.Status.PodIPs {
		// ignoring host-networked Pod IPs
		if ip.IP != pod.Status.HostIP {
			m.ipToPod[ip.IP] = podID
		}
	}
}

// getPodInfoByIP returns the information about the Pod with the specified IP address.
func (m *Manager) getPodInfoByIP(ip string) *podInfo {
	m.podLock.RLock()
	defer m.podLock.RUnlock()

	if p, ipExsit := m.ipToPod[ip]; ipExsit {
		if pInfo, podExist := m.podList[p]; podExist {
			return &pInfo
		}
	}
	return nil
}

// authorizeEgress authorizes a request for accessing an imported service.
func (m *Manager) authorizeEgress(ctx context.Context, req *egressAuthorizationRequest) (*egressAuthorizationResponse, error) {
	m.logger.Infof("Received egress authorization request: %v.", req)

	srcAttributes := connectivitypdp.WorkloadAttrs{GatewayNameLabel: m.peerName}
	podInfo := m.getPodInfoByIP(req.IP)
	if podInfo != nil {
		srcAttributes[ServiceNamespaceLabel] = podInfo.namespace

		if src, ok := podInfo.labels["app"]; ok { // TODO: Add support for labels other than just the "app" key.
			m.logger.Infof("Received egress authorization srcLabels[app]: %v.", podInfo.labels["app"])
			srcAttributes[ServiceNameLabel] = src
		}
	}

	var imp v1alpha1.Import
	if err := m.client.Get(ctx, req.ImportName, &imp); err != nil {
		return nil, fmt.Errorf("cannot get import %v: %w", req.ImportName, err)
	}

	lbResult := NewLoadBalancingResult(&imp)
	for {
		if err := m.loadBalancer.Select(lbResult); err != nil {
			return nil, fmt.Errorf("cannot select import source: %w", err)
		}

		importSource := lbResult.Get()
		peerName := types.NamespacedName{
			Name:      importSource.Peer,
			Namespace: m.namespace,
		}

		var pr v1alpha1.Peer
		if err := m.client.Get(ctx, peerName, &pr); err != nil {
			return nil, fmt.Errorf("cannot get peer '%s': %w", importSource.Peer, err)
		}

		if !meta.IsStatusConditionTrue(pr.Status.Conditions, v1alpha1.PeerReachable) {
			if !lbResult.IsDelayed() {
				lbResult.Delay()
				continue
			}
		}

		dstAttributes := connectivitypdp.WorkloadAttrs{
			ServiceNameLabel:      imp.Name,
			ServiceNamespaceLabel: imp.Namespace,
			GatewayNameLabel:      importSource.Peer,
		}
		decision, err := m.connectivityPDP.Decide(srcAttributes, dstAttributes, req.ImportName.Namespace)
		if err != nil {
			return nil, fmt.Errorf("error deciding on an egress connection: %w", err)
		}

		if decision.Decision != connectivitypdp.DecisionAllow {
			continue
		}

		m.peerLock.RLock()
		cl, ok := m.peerClient[importSource.Peer]
		m.peerLock.RUnlock()

		if !ok {
			return nil, fmt.Errorf("missing client for peer: %s", importSource.Peer)
		}

		DstName := importSource.ExportName
		DstNamespace := importSource.ExportNamespace
		if DstName == "" { // TODO- remove when controlplane will support only CRD mode.
			DstName = req.ImportName.Name
		}

		if DstNamespace == "" { // TODO- remove when controlplane will support only CRD mode.
			DstNamespace = req.ImportName.Namespace
		}

		peerResp, err := cl.Authorize(&cpapi.AuthorizationRequest{
			ServiceName:      DstName,
			ServiceNamespace: DstNamespace,
		})
		if err != nil {
			m.logger.Infof("Unable to get access token from peer: %v", err)
			continue
		}

		if !peerResp.ServiceExists {
			m.logger.Infof(
				"Peer %s does not have an import source for %v",
				importSource.Peer, req.ImportName)
			continue
		}

		if !peerResp.Allowed {
			m.logger.Infof(
				"Peer %s did not allow connection to import %v: %v",
				importSource.Peer, req.ImportName, err)
			continue
		}

		return &egressAuthorizationResponse{
			ServiceExists:     true,
			Allowed:           true,
			RemotePeerCluster: cpapi.RemotePeerClusterName(importSource.Peer),
			AccessToken:       peerResp.AccessToken,
		}, nil
	}
}

// parseAuthorizationHeader verifies an access token for an ingress dataplane connection.
// On success, returns the parsed target cluster name.
func (m *Manager) parseAuthorizationHeader(token string) (string, error) {
	m.logger.Debug("Parsing access token.")

	parsedToken, err := jwt.ParseString(
		token, jwt.WithVerify(cpapi.JWTSignatureAlgorithm, m.jwkVerifyKey), jwt.WithValidate(true))
	if err != nil {
		return "", err
	}

	// TODO: verify client name

	exportName, ok := parsedToken.PrivateClaims()[cpapi.ExportNameJWTClaim]
	if !ok {
		return "", fmt.Errorf("token missing '%s' claim", cpapi.ExportNameJWTClaim)
	}

	exportNamespace, ok := parsedToken.PrivateClaims()[cpapi.ExportNamespaceJWTClaim]
	if !ok {
		return "", fmt.Errorf("token missing '%s' claim", cpapi.ExportNamespaceJWTClaim)
	}

	return cpapi.ExportClusterName(exportName.(string), exportNamespace.(string)), nil
}

// authorizeIngress authorizes a request for accessing an exported service.
func (m *Manager) authorizeIngress(
	ctx context.Context,
	req *ingressAuthorizationRequest,
	pr string,
) (*ingressAuthorizationResponse, error) {
	m.logger.Infof("Received ingress authorization request: %v.", req)

	resp := &ingressAuthorizationResponse{}

	// check that a corresponding export exists
	exportName := types.NamespacedName{
		Namespace: req.ServiceName.Namespace,
		Name:      req.ServiceName.Name,
	}
	var export v1alpha1.Export
	if err := m.client.Get(ctx, exportName, &export); err != nil {
		if errors.IsNotFound(err) || !meta.IsStatusConditionTrue(export.Status.Conditions, v1alpha1.ExportValid) {
			return resp, nil
		}

		return nil, fmt.Errorf("cannot get export %v: %w", exportName, err)
	}

	resp.ServiceExists = true

	srcAttributes := connectivitypdp.WorkloadAttrs{GatewayNameLabel: pr}
	dstAttributes := connectivitypdp.WorkloadAttrs{
		ServiceNameLabel:      req.ServiceName.Name,
		ServiceNamespaceLabel: req.ServiceName.Namespace,
		GatewayNameLabel:      m.peerName,
	}
	decision, err := m.connectivityPDP.Decide(srcAttributes, dstAttributes, req.ServiceName.Namespace)
	if err != nil {
		return nil, fmt.Errorf("error deciding on an ingress connection: %w", err)
	}

	if decision.Decision != connectivitypdp.DecisionAllow {
		resp.Allowed = false
		return resp, nil
	}
	resp.Allowed = true

	// create access token
	// TODO: include client name as a claim
	token, err := jwt.NewBuilder().
		Expiration(time.Now().Add(time.Second*jwtExpirySeconds)).
		Claim(cpapi.ExportNameJWTClaim, req.ServiceName.Name).
		Claim(cpapi.ExportNamespaceJWTClaim, req.ServiceName.Namespace).
		Build()
	if err != nil {
		return nil, fmt.Errorf("unable to generate access token: %w", err)
	}

	// sign access token
	signed, err := jwt.Sign(token, cpapi.JWTSignatureAlgorithm, m.jwkSignKey)
	if err != nil {
		return nil, fmt.Errorf("unable to sign access token: %w", err)
	}
	resp.AccessToken = string(signed)

	return resp, nil
}

// NewManager returns a new authorization manager.
func NewManager(peerTLS *tls.ParsedCertData, cl client.Client, namespace string) (*Manager, error) {
	// generate RSA key-pair for JWT signing
	// TODO: instead of generating, read from k8s secret
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("unable to generate RSA keys: %w", err)
	}

	jwkSignKey, err := jwk.New(rsaKey)
	if err != nil {
		return nil, fmt.Errorf("unable to create JWK signing key: %w", err)
	}

	jwkVerifyKey, err := jwk.New(rsaKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("unable to create JWK verifing key: %w", err)
	}

	dnsNames := peerTLS.DNSNames()
	if len(dnsNames) == 0 {
		return nil, fmt.Errorf("expected peer certificate to contain at least one DNS name")
	}

	return &Manager{
		client:          cl,
		namespace:       namespace,
		connectivityPDP: connectivitypdp.NewPDP(),
		loadBalancer:    NewLoadBalancer(),
		peerName:        dnsNames[0],
		peerTLS:         peerTLS,
		peerClient:      make(map[string]*peer.Client),
		jwkSignKey:      jwkSignKey,
		jwkVerifyKey:    jwkVerifyKey,
		ipToPod:         make(map[string]types.NamespacedName),
		podList:         make(map[types.NamespacedName]podInfo),
		logger:          logrus.WithField("component", "controlplane.authz.manager"),
	}, nil
}
