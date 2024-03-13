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
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/peer"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

const (
	// the number of seconds a JWT access token is valid before it expires.
	jwtExpirySeconds = 5
)

// egressAuthorizationRequest (from local dataplane)
// represents a request for accessing an imported service.
type egressAuthorizationRequest struct {
	// ImportName is the name of the requested imported service.
	ImportName string
	// ImportNamespace is the namespace of the requested imported service.
	ImportNamespace string
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
	ServiceName string
	// ServiceNamespace is the namespace of the requested exported service.
	ServiceNamespace string
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
	policyDecider policyengine.PolicyDecider

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
	client := peer.NewClient(pr, m.peerTLS.ClientConfig(pr.Name))

	m.peerLock.Lock()
	m.peerClient[pr.Name] = client
	m.peerLock.Unlock()

	if meta.IsStatusConditionTrue(pr.Status.Conditions, v1alpha1.PeerReachable) {
		m.policyDecider.AddPeer(pr.Name)
	} else {
		m.policyDecider.DeletePeer(pr.Name)
	}
}

// DeletePeer removes the possibility for egress dataplane connections to be routed to a given peer.
func (m *Manager) DeletePeer(name string) {
	m.logger.Infof("Deleting peer '%s'.", name)

	m.peerLock.Lock()
	delete(m.peerClient, name)
	m.peerLock.Unlock()

	m.policyDecider.DeletePeer(name)
}

// AddImport adds a listening socket for an imported remote service.
func (m *Manager) AddImport(imp *v1alpha1.Import) {
	m.logger.Infof("Adding import '%s/%s'.", imp.Namespace, imp.Name)

	m.policyDecider.AddImport(imp)
}

// DeleteImport removes the listening socket of a previously imported service.
func (m *Manager) DeleteImport(name types.NamespacedName) error {
	m.logger.Infof("Deleting import '%v'.", name)
	m.policyDecider.DeleteImport(name)
	return nil
}

// AddExport defines a new route target for ingress dataplane connections.
func (m *Manager) AddExport(export *v1alpha1.Export) {
	m.logger.Infof("Adding export '%s/%s'.", export.Namespace, export.Name)

	// TODO: m.policyDecider.AddExport()
}

// DeleteExport removes the possibility for ingress dataplane connections to access a given service.
func (m *Manager) DeleteExport(name types.NamespacedName) {
	m.logger.Infof("Deleting export '%v'.", name)

	// TODO: pass on namespace
	m.policyDecider.DeleteExport(name.Name)
}

// AddAccessPolicy adds an access policy to allow/deny specific connections.
// TODO: switch from api.Policy to v1alpha1.Policy.
func (m *Manager) AddAccessPolicy(policy *api.Policy) error {
	return m.policyDecider.AddAccessPolicy(policy)
}

// DeleteAccessPolicy removes an access policy to allow/deny specific connections.
// TODO: switch from api.Policy to v1alpha1.Policy.
func (m *Manager) DeleteAccessPolicy(policy *api.Policy) error {
	return m.policyDecider.DeleteAccessPolicy(policy)
}

// AddLBPolicy adds a load-balancing policy to set a load-balancing scheme for specific connections.
// TODO: merge this with AddImport.
func (m *Manager) AddLBPolicy(policy *api.Policy) error {
	return m.policyDecider.AddLBPolicy(policy)
}

// DeleteLBPolicy removes a load-balancing policy.
// TODO: merge this with DeleteImport.
func (m *Manager) DeleteLBPolicy(policy *api.Policy) error {
	return m.policyDecider.DeleteLBPolicy(policy)
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

func (m *Manager) deleteAccessPolicy(_ types.NamespacedName) {
	// TODO: call policy decider
}

func (m *Manager) addAccessPolicy(accessPolicy *v1alpha1.AccessPolicy) error {
	convert := func(list v1alpha1.WorkloadSetOrSelectorList) policytypes.WorkloadSetOrSelectorList {
		out := make(policytypes.WorkloadSetOrSelectorList, len(list))
		for i, elem := range list {
			out[i] = policytypes.WorkloadSetOrSelector{
				WorkloadSets:     elem.WorkloadSets,
				WorkloadSelector: elem.WorkloadSelector,
			}
		}

		return out
	}

	policyData, err := json.Marshal(&policytypes.ConnectivityPolicy{
		Name:       accessPolicy.Name,
		Privileged: accessPolicy.Spec.Privileged,
		Action:     policytypes.PolicyAction(accessPolicy.Spec.Action),
		From:       convert(accessPolicy.Spec.From),
		To:         convert(accessPolicy.Spec.To),
	})
	if err != nil {
		return err
	}

	return m.policyDecider.AddAccessPolicy(&api.Policy{
		Name: accessPolicy.Name,
		Spec: api.PolicySpec{Blob: policyData},
	})
}

// getLabelsFromIP returns the labels associated with Pod with the specified IP address.
func (m *Manager) getLabelsFromIP(ip string) map[string]string {
	m.podLock.RLock()
	defer m.podLock.RUnlock()

	if p, ipExsit := m.ipToPod[ip]; ipExsit {
		if pInfo, podExist := m.podList[p]; podExist {
			return pInfo.labels
		}
	}
	return nil
}

// authorizeEgress authorizes a request for accessing an imported service.
func (m *Manager) authorizeEgress(req *egressAuthorizationRequest) (*egressAuthorizationResponse, error) {
	m.logger.Infof("Received egress authorization request: %v.", req)

	connReq := policytypes.ConnectionRequest{
		DstSvcName:      req.ImportName,
		DstSvcNamespace: req.ImportNamespace,
		Direction:       policytypes.Outgoing,
	}
	srcLabels := m.getLabelsFromIP(req.IP)
	if src, ok := srcLabels["app"]; ok { // TODO: Add support for labels other than just the "app" key.
		m.logger.Infof("Received egress authorization srcLabels[app]: %v.", srcLabels["app"])
		connReq.SrcWorkloadAttrs = policytypes.WorkloadAttrs{policyengine.ServiceNameLabel: src}
	}

	authResp, err := m.policyDecider.AuthorizeAndRouteConnection(&connReq)
	if err != nil {
		return nil, err
	}

	if authResp.Action != policytypes.ActionAllow {
		return &egressAuthorizationResponse{Allowed: false}, nil
	}

	target := authResp.DstPeer

	m.peerLock.RLock()
	client, ok := m.peerClient[target]
	m.peerLock.RUnlock()

	if !ok {
		return nil, fmt.Errorf("missing client for peer: %s", target)
	}

	serverResp, err := client.Authorize(&cpapi.AuthorizationRequest{
		ServiceName:      req.ImportName,
		ServiceNamespace: req.ImportNamespace,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to get access token from peer: %w", err)
	}

	resp := &egressAuthorizationResponse{
		ServiceExists: serverResp.ServiceExists,
		Allowed:       serverResp.Allowed,
	}

	if serverResp.Allowed {
		resp.RemotePeerCluster = cpapi.RemotePeerClusterName(target)
		resp.AccessToken = serverResp.AccessToken
	}

	return resp, nil
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
func (m *Manager) authorizeIngress(req *ingressAuthorizationRequest, pr string) (*ingressAuthorizationResponse, error) {
	m.logger.Infof("Received ingress authorization request: %v.", req)

	resp := &ingressAuthorizationResponse{}

	// TODO: set this from autoResp below
	resp.ServiceExists = true

	connReq := policytypes.ConnectionRequest{
		DstSvcName:       req.ServiceName,
		DstSvcNamespace:  req.ServiceNamespace,
		Direction:        policytypes.Incoming,
		SrcWorkloadAttrs: policytypes.WorkloadAttrs{policyengine.GatewayNameLabel: pr},
	}
	authResp, err := m.policyDecider.AuthorizeAndRouteConnection(&connReq)
	if err != nil {
		return nil, err
	}
	if authResp.Action != policytypes.ActionAllow {
		resp.Allowed = false
		return resp, nil
	}
	resp.Allowed = true

	// create access token
	// TODO: include client name as a claim
	token, err := jwt.NewBuilder().
		Expiration(time.Now().Add(time.Second*jwtExpirySeconds)).
		Claim(cpapi.ExportNameJWTClaim, req.ServiceName).
		Claim(cpapi.ExportNamespaceJWTClaim, req.ServiceNamespace).
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
func NewManager(peerTLS *tls.ParsedCertData) (*Manager, error) {
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

	return &Manager{
		policyDecider: policyengine.NewPolicyHandler(),
		peerTLS:       peerTLS,
		peerClient:    make(map[string]*peer.Client),
		jwkSignKey:    jwkSignKey,
		jwkVerifyKey:  jwkVerifyKey,
		ipToPod:       make(map[string]types.NamespacedName),
		podList:       make(map[types.NamespacedName]podInfo),
		logger:        logrus.WithField("component", "controlplane.authz.manager"),
	}, nil
}
