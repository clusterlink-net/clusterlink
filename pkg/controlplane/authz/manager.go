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
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/control"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/peer"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

const (
	// the number of seconds a JWT access token is valid before it expires.
	jwtExpirySeconds = 5

	ClientNamespaceLabel  = "clusterlink/metadata.clientNamespace"
	ClientSALabel         = "clusterlink/metadata.clientServiceAccount"
	ClientLabelsPrefix    = "client/metadata.labels."
	ServiceNameLabel      = "clusterlink/metadata.serviceName"
	ServiceNamespaceLabel = "clusterlink/metadata.serviceNamespace"
	ServiceLabelsPrefix   = "service/metadata."
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
	// Attributes of the source workload, to be used by the PDP on the remote peer
	SrcAttributes connectivitypdp.WorkloadAttrs
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
	name           string
	namespace      string
	serviceAccount string
	labels         map[string]string
}

// Manager manages the authorization dataplane connections.
type Manager struct {
	client    client.Client
	namespace string

	loadBalancer    *LoadBalancer
	connectivityPDP *connectivitypdp.PDP

	selfPeerLock sync.RWMutex
	peerTLS      *tls.ParsedCertData
	peerName     string

	peerClientLock sync.RWMutex
	peerClient     map[string]*peer.Client

	podLock sync.RWMutex
	ipToPod map[string]types.NamespacedName
	podList map[types.NamespacedName]podInfo

	jwksLock     sync.RWMutex
	jwkSignKey   jwk.Key
	jwkVerifyKey jwk.Key

	logger *logrus.Entry
}

// AddPeer defines a new route target for egress dataplane connections.
func (m *Manager) AddPeer(pr *v1alpha1.Peer) {
	m.logger.Infof("Adding peer '%s'.", pr.Name)

	// initialize peer client
	m.selfPeerLock.RLock()
	cl := peer.NewClient(pr, m.peerTLS.ClientConfig(pr.Name))
	m.selfPeerLock.RUnlock()

	m.peerClientLock.Lock()
	m.peerClient[pr.Name] = cl
	m.peerClientLock.Unlock()
}

// DeletePeer removes the possibility for egress dataplane connections to be routed to a given peer.
func (m *Manager) DeletePeer(name string) {
	m.logger.Infof("Deleting peer '%s'.", name)

	m.peerClientLock.Lock()
	delete(m.peerClient, name)
	m.peerClientLock.Unlock()
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
	m.podList[podID] = podInfo{
		name:           pod.Name,
		namespace:      pod.Namespace,
		labels:         pod.Labels,
		serviceAccount: pod.Spec.ServiceAccountName,
	}
	for _, ip := range pod.Status.PodIPs {
		// ignoring host-networked Pod IPs
		if ip.IP != pod.Status.HostIP {
			m.ipToPod[ip.IP] = podID
		}
	}
}

// addSecret adds a new secret.
func (m *Manager) addSecret(secret *v1.Secret) error {
	if secret.Namespace != m.namespace || secret.Name != control.JWKSecretName {
		return nil
	}

	privateKey, err := control.ParseJWKSSecret(secret)
	if err != nil {
		return fmt.Errorf("cannot parse JWKS secret: %w", err)
	}

	jwkSignKey, err := jwk.New(privateKey)
	if err != nil {
		return fmt.Errorf("unable to create JWK signing key: %w", err)
	}

	jwkVerifyKey, err := jwk.New(privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("unable to create JWK verifing key: %w", err)
	}

	m.jwksLock.Lock()
	defer m.jwksLock.Unlock()
	m.jwkSignKey = jwkSignKey
	m.jwkVerifyKey = jwkVerifyKey

	return nil
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

func (m *Manager) getClientAttributes(req *egressAuthorizationRequest) connectivitypdp.WorkloadAttrs {
	podInfo := m.getPodInfoByIP(req.IP)
	if podInfo == nil {
		m.logger.Infof("Pod has no info: IP=%v.", req.IP)
		return nil
	}

	clientAttrs := connectivitypdp.WorkloadAttrs{
		GatewayNameLabel:      m.getPeerName(),
		ServiceNamespaceLabel: podInfo.namespace, // deprecated
		ClientNamespaceLabel:  podInfo.namespace,
		ClientSALabel:         podInfo.serviceAccount,
	}

	if src, ok := podInfo.labels["app"]; ok {
		clientAttrs[ServiceNameLabel] = src // deprecated
	}

	for k, v := range podInfo.labels {
		clientAttrs[ClientLabelsPrefix+k] = v
	}

	m.logger.Infof("Client attributes: %v.", clientAttrs)

	return clientAttrs
}

// authorizeEgress authorizes a request for accessing an imported service.
func (m *Manager) authorizeEgress(ctx context.Context, req *egressAuthorizationRequest) (*egressAuthorizationResponse, error) {
	m.logger.Infof("Received egress authorization request: %v.", req)

	srcAttributes := m.getClientAttributes(req)
	if len(srcAttributes) == 0 && m.connectivityPDP.DependsOnClientAttrs() {
		return nil, fmt.Errorf("failed to extract client attributes, however, access policies depend on such attributes")
	}

	var imp v1alpha1.Import
	if err := m.client.Get(ctx, req.ImportName, &imp); err != nil {
		return nil, fmt.Errorf("cannot get import %v: %w", req.ImportName, err)
	}

	dstAttributes := connectivitypdp.WorkloadAttrs{}
	for k, v := range imp.Labels { // add import labels to destination attributes
		dstAttributes[ServiceLabelsPrefix+k] = v
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

		dstAttributes[ServiceNameLabel] = importSource.ExportName
		dstAttributes[ServiceNamespaceLabel] = importSource.ExportNamespace
		dstAttributes[GatewayNameLabel] = importSource.Peer

		decision, err := m.connectivityPDP.Decide(srcAttributes, dstAttributes, req.ImportName.Namespace)
		if err != nil {
			return nil, fmt.Errorf("error deciding on an egress connection: %w", err)
		}

		if decision.Decision != connectivitypdp.DecisionAllow {
			continue
		}

		m.peerClientLock.RLock()
		cl, ok := m.peerClient[importSource.Peer]
		m.peerClientLock.RUnlock()

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

		accessToken, err := cl.Authorize(&cpapi.AuthorizationRequest{
			ServiceName:      DstName,
			ServiceNamespace: DstNamespace,
			SrcAttributes:    srcAttributes,
		})
		if err != nil {
			m.logger.Infof("Unable to get access token from peer: %v", err)
			continue
		}

		return &egressAuthorizationResponse{
			Allowed:           true,
			RemotePeerCluster: cpapi.RemotePeerClusterName(importSource.Peer),
			AccessToken:       accessToken,
		}, nil
	}
}

// parseAuthorizationHeader verifies an access token for an ingress dataplane connection.
// On success, returns the parsed target cluster name.
func (m *Manager) parseAuthorizationHeader(token string) (string, error) {
	m.logger.Debug("Parsing access token.")

	m.jwksLock.RLock()
	jwkVerifyKey := m.jwkVerifyKey
	m.jwksLock.RUnlock()

	if jwkVerifyKey == nil {
		return "", fmt.Errorf("jwk verify key undefined")
	}

	parsedToken, err := jwt.ParseString(
		token, jwt.WithVerify(cpapi.JWTSignatureAlgorithm, jwkVerifyKey), jwt.WithValidate(true))
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

	dstAttributes := connectivitypdp.WorkloadAttrs{
		ServiceNameLabel:      export.Name,
		ServiceNamespaceLabel: export.Namespace,
		GatewayNameLabel:      m.getPeerName(),
	}
	for k, v := range export.Labels { // add export labels to destination attributes
		dstAttributes[ServiceLabelsPrefix+k] = v
	}

	decision, err := m.connectivityPDP.Decide(req.SrcAttributes, dstAttributes, req.ServiceName.Namespace)
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

	m.jwksLock.RLock()
	jwkSignKey := m.jwkSignKey
	m.jwksLock.RUnlock()

	if jwkSignKey == nil {
		return nil, fmt.Errorf("jwk sign key undefined")
	}

	// sign access token
	signed, err := jwt.Sign(token, cpapi.JWTSignatureAlgorithm, jwkSignKey)
	if err != nil {
		return nil, fmt.Errorf("unable to sign access token: %w", err)
	}
	resp.AccessToken = string(signed)

	return resp, nil
}

func (m *Manager) getPeerName() string {
	m.selfPeerLock.RLock()
	defer m.selfPeerLock.RUnlock()
	return m.peerName
}

func (m *Manager) SetPeerCertificates(peerTLS *tls.ParsedCertData, _ *tls.RawCertData) error {
	m.logger.Info("Setting peer certificates.")

	dnsNames := peerTLS.DNSNames()
	if len(dnsNames) == 0 {
		return fmt.Errorf("expected peer certificate to contain at least one DNS name")
	}

	m.selfPeerLock.Lock()
	defer m.selfPeerLock.Unlock()

	m.peerName = dnsNames[0]
	m.peerTLS = peerTLS

	m.peerClientLock.Lock()
	defer m.peerClientLock.Unlock()

	// re-initialize peer clients
	for pr, cl := range m.peerClient {
		m.peerClient[pr] = peer.NewClient(cl.Peer(), m.peerTLS.ClientConfig(pr))
	}

	return nil
}

func (m *Manager) IsReady() bool {
	m.jwksLock.RLock()
	defer m.jwksLock.RUnlock()
	return m.jwkSignKey != nil
}

// NewManager returns a new authorization manager.
func NewManager(cl client.Client, namespace string) *Manager {
	return &Manager{
		client:          cl,
		namespace:       namespace,
		connectivityPDP: connectivitypdp.NewPDP(),
		loadBalancer:    NewLoadBalancer(),
		peerClient:      make(map[string]*peer.Client),
		ipToPod:         make(map[string]types.NamespacedName),
		podList:         make(map[types.NamespacedName]podInfo),
		logger:          logrus.WithField("component", "controlplane.authz.manager"),
	}
}
