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

package xds

import (
	"fmt"
	"time"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	tcpproxy "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	getaddrinfo "github.com/envoyproxy/go-control-plane/envoy/extensions/network/dns_resolver/getaddrinfo/v3"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	utiltls "github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

// Manager manages the core routing components of the dataplane.
// It maps the following controlplane types to xDS types:
// - Peer -> Cluster (whose name starts with a designated prefix)
// - Export -> Cluster (whose name starts with a designated prefix)
// - Import -> Listener (whose name starts with a designated prefix)
// Note that imported service bindings are handled by the egress authz server.
type Manager struct {
	clusters  *cache.LinearCache
	listeners *cache.LinearCache
	secrets   *cache.LinearCache

	logger *logrus.Entry
}

// AddPeer defines a new route target for egress dataplane connections.
func (m *Manager) AddPeer(peer *v1alpha1.Peer) error {
	m.logger.Infof("Adding peer '%s'.", peer.Name)

	clusterName := cpapi.RemotePeerClusterName(peer.Name)
	epc, err := makeEndpointsCluster(clusterName, peer.Spec.Gateways, peer.Name+":443")
	if err != nil {
		return err
	}

	sdsConfig := &core.ConfigSource{
		ConfigSourceSpecifier: &core.ConfigSource_Ads{
			Ads: &core.AggregatedConfigSource{},
		},
		InitialFetchTimeout: durationpb.New(time.Second),
		ResourceApiVersion:  core.ApiVersion_V3,
	}

	tlsConfig := &tls.UpstreamTlsContext{
		Sni: peer.Name,
		CommonTlsContext: &tls.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*tls.SdsSecretConfig{{
				Name:      cpapi.CertificateSecret,
				SdsConfig: sdsConfig,
			}},
			ValidationContextType: &tls.CommonTlsContext_ValidationContextSdsSecretConfig{
				ValidationContextSdsSecretConfig: &tls.SdsSecretConfig{
					Name:      cpapi.ValidationSecret,
					SdsConfig: sdsConfig,
				},
			},
		},
	}

	pb, err := anypb.New(tlsConfig)
	if err != nil {
		return err
	}

	epc.TransportSocket = &core.TransportSocket{
		Name:       wellknown.TransportSocketTLS,
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: pb},
	}

	return m.clusters.UpdateResource(clusterName, epc)
}

// DeletePeer removes the possibility for egress dataplane connections to be routed to a given peer.
func (m *Manager) DeletePeer(name string) error {
	m.logger.Infof("Deleting peer '%s'.", name)

	clusterName := cpapi.RemotePeerClusterName(name)
	return m.clusters.DeleteResource(clusterName)
}

// AddExport defines a new route target for ingress dataplane connections.
func (m *Manager) AddExport(export *v1alpha1.Export) error {
	m.logger.Infof("Adding export '%s/%s'.", export.Namespace, export.Name)

	host := export.Spec.Host
	if host == "" {
		host = fmt.Sprintf("%s.%s.svc.cluster.local", export.Name, export.Namespace)
	}

	clusterName := cpapi.ExportClusterName(export.Name, export.Namespace)
	cc, err := makeAddressCluster(
		clusterName,
		host,
		export.Spec.Port, "")
	if err != nil {
		return err
	}

	return m.clusters.UpdateResource(clusterName, cc)
}

// DeleteExport removes the possibility for ingress dataplane connections to access a given service.
func (m *Manager) DeleteExport(name types.NamespacedName) error {
	m.logger.Infof("Deleting export '%v'.", name)

	clusterName := cpapi.ExportClusterName(name.Name, name.Namespace)
	return m.clusters.DeleteResource(clusterName)
}

// AddImport adds a listening socket for an imported remote service.
func (m *Manager) AddImport(imp *v1alpha1.Import) error {
	m.logger.Infof("Adding import '%s/%s'.", imp.Namespace, imp.Name)

	if !meta.IsStatusConditionTrue(imp.Status.Conditions, v1alpha1.ImportTargetPortValid) {
		// target port not yet allocated, skip
		m.logger.Infof("Skipping import with no valid target port '%s/%s'.", imp.Namespace, imp.Name)
		return nil
	}

	listenerName := cpapi.ImportListenerName(imp.Name, imp.Namespace)
	egressRouterHostname := "egress-router:443"

	tunnelingConfig := &tcpproxy.TcpProxy_TunnelingConfig{
		Hostname: egressRouterHostname,
		HeadersToAdd: []*core.HeaderValueOption{
			{
				Header: &core.HeaderValue{
					Key:   cpapi.ImportNameHeader,
					Value: imp.Name,
				},
				KeepEmptyValue: true,
			},
			{
				Header: &core.HeaderValue{
					Key:   cpapi.ImportNamespaceHeader,
					Value: imp.Namespace,
				},
				KeepEmptyValue: true,
			},
			{
				Header: &core.HeaderValue{
					Key:   cpapi.ClientIPHeader,
					Value: "%DOWNSTREAM_REMOTE_ADDRESS_WITHOUT_PORT%",
				},
				KeepEmptyValue: true,
			},
		},
	}

	tcpProxyFilter, err := makeTCPProxyFilter(
		cpapi.EgressRouterCluster, imp.Name, tunnelingConfig)
	if err != nil {
		return err
	}

	// TODO: listen on a more specific address (i.e. not 0.0.0.0)
	ln := &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(imp.Spec.TargetPort),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{tcpProxyFilter},
		}},
	}

	return m.listeners.UpdateResource(listenerName, ln)
}

// DeleteImport removes the listening socket of a previously imported service.
func (m *Manager) DeleteImport(name types.NamespacedName) error {
	m.logger.Infof("Deleting import '%v'.", name)

	listenerName := cpapi.ImportListenerName(name.Name, name.Namespace)
	return m.listeners.DeleteResource(listenerName)
}

// SetPeerCertificates sets the TLS certificates used for peer-to-peer communication.
func (m *Manager) SetPeerCertificates(rawCertData *utiltls.RawCertData) error {
	m.logger.Info("Setting peer certificates.")

	certificateSecret := &tls.Secret{
		Name: cpapi.CertificateSecret,
		Type: &tls.Secret_TlsCertificate{
			TlsCertificate: &tls.TlsCertificate{
				CertificateChain: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: rawCertData.Certificate(),
					},
				},
				PrivateKey: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: rawCertData.Key(),
					},
				},
			},
		},
	}

	if err := m.secrets.UpdateResource(certificateSecret.Name, certificateSecret); err != nil {
		return fmt.Errorf("error setting certificate secret: %w", err)
	}

	validationSecret := &tls.Secret{
		Name: cpapi.ValidationSecret,
		Type: &tls.Secret_ValidationContext{
			ValidationContext: &tls.CertificateValidationContext{
				TrustedCa: &core.DataSource{
					Specifier: &core.DataSource_InlineBytes{
						InlineBytes: rawCertData.CA(),
					},
				},
			},
		},
	}

	if err := m.secrets.UpdateResource(validationSecret.Name, validationSecret); err != nil {
		return fmt.Errorf("error setting validation secret: %w", err)
	}

	return nil
}

func makeAddressCluster(name, addr string, port uint16, hostname string) (*cluster.Cluster, error) {
	return makeEndpointsCluster(name, []v1alpha1.Endpoint{{Host: addr, Port: port}}, hostname)
}

func makeEndpointsCluster(name string, endpoints []v1alpha1.Endpoint, hostname string) (*cluster.Cluster, error) {
	lbEndpoints := make([]*endpoint.LbEndpoint, len(endpoints))

	for i, ep := range endpoints {
		lbEndpoints[i] = &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Address: ep.Host,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: uint32(ep.Port),
								},
							},
						},
					},
					Hostname: hostname,
				},
			},
		}
	}

	pb, err := anypb.New(&getaddrinfo.GetAddrInfoDnsResolverConfig{})
	if err != nil {
		return nil, err
	}

	cc := &cluster.Cluster{
		Name:           name,
		ConnectTimeout: durationpb.New(time.Second),
		DnsRefreshRate: durationpb.New(time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{
			Type: cluster.Cluster_STRICT_DNS,
		},
		TypedDnsResolverConfig: &core.TypedExtensionConfig{
			Name:        "envoy.network.dns_resolver.getaddrinfo",
			TypedConfig: pb,
		},
		LoadAssignment: &endpoint.ClusterLoadAssignment{
			ClusterName: name,
			Endpoints: []*endpoint.LocalityLbEndpoints{{
				LbEndpoints: lbEndpoints,
			}},
		},
	}

	return cc, nil
}

func makeTCPProxyFilter(clusterName, statPrefix string,
	tunnelingConfig *tcpproxy.TcpProxy_TunnelingConfig,
) (*listener.Filter, error) {
	tcpProxyConfig := &tcpproxy.TcpProxy{
		StatPrefix: "tcp-proxy-" + statPrefix,
		ClusterSpecifier: &tcpproxy.TcpProxy_Cluster{
			Cluster: clusterName,
		},
		TunnelingConfig: tunnelingConfig,
	}

	pb, err := anypb.New(tcpProxyConfig)
	if err != nil {
		return nil, err
	}

	return &listener.Filter{
		Name: wellknown.TCPProxy,
		ConfigType: &listener.Filter_TypedConfig{
			TypedConfig: pb,
		},
	}, nil
}

// NewManager creates an uninitialized, non-registered xDS manager.
func NewManager() *Manager {
	logger := logrus.WithField("component", "controlplane.xds.manager")

	return &Manager{
		clusters:  cache.NewLinearCache(resource.ClusterType, cache.WithLogger(logger)),
		listeners: cache.NewLinearCache(resource.ListenerType, cache.WithLogger(logger)),
		secrets:   cache.NewLinearCache(resource.SecretType, cache.WithLogger(logger)),
		logger:    logger,
	}
}
