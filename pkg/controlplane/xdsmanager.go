package controlplane

import (
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

	"github.com/clusterlink-net/clusterlink/pkg/api"
	cpapi "github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
	dpapi "github.com/clusterlink-net/clusterlink/pkg/dataplane/api"
)

// xdsManager manages the core routing components of the dataplane.
// It maps the following controlplane types to xDS types:
// - Peer -> Cluster (whose name starts with a designated prefix)
// - Export -> Cluster (whose name starts with a designated prefix)
// - Import -> Listener (whose name starts with a designated prefix)
// Note that imported service bindings are handled by the egress authz server.
type xdsManager struct {
	clusters  *cache.LinearCache
	listeners *cache.LinearCache

	logger *logrus.Entry
}

// AddPeer defines a new route target for egress dataplane connections.
func (m *xdsManager) AddPeer(peer *store.Peer) error {
	m.logger.Infof("Adding peer '%s'.", peer.Name)

	clusterName := cpapi.RemotePeerClusterName(peer.Name)
	dataplaneSNI := dpapi.DataplaneSNI(peer.Name)
	c, err := makeEndpointsCluster(clusterName, peer.Gateways, dataplaneSNI)
	if err != nil {
		return err
	}

	tlsConfig := &tls.UpstreamTlsContext{
		Sni: dataplaneSNI,
		CommonTlsContext: &tls.CommonTlsContext{
			TlsCertificateSdsSecretConfigs: []*tls.SdsSecretConfig{{
				Name: cpapi.CertificateSecret,
			}},
			ValidationContextType: &tls.CommonTlsContext_ValidationContextSdsSecretConfig{
				ValidationContextSdsSecretConfig: &tls.SdsSecretConfig{
					Name: cpapi.ValidationSecret,
				},
			},
		},
	}

	pb, err := anypb.New(tlsConfig)
	if err != nil {
		return err
	}

	c.TransportSocket = &core.TransportSocket{
		Name:       wellknown.TransportSocketTLS,
		ConfigType: &core.TransportSocket_TypedConfig{TypedConfig: pb},
	}

	return m.clusters.UpdateResource(clusterName, c)
}

// DeletePeer removes the possibility for egress dataplane connections to be routed to a given peer.
func (m *xdsManager) DeletePeer(name string) error {
	m.logger.Infof("Deleting peer '%s'.", name)

	clusterName := cpapi.RemotePeerClusterName(name)
	return m.clusters.DeleteResource(clusterName)
}

// AddExport defines a new route target for ingress dataplane connections.
func (m *xdsManager) AddExport(export *store.Export) error {
	m.logger.Infof("Adding export '%s'.", export.Name)

	clusterName := cpapi.ExportClusterName(export.Name)
	c, err := makeAddressCluster(
		clusterName, export.Service.Host, export.Service.Port, "")
	if err != nil {
		return err
	}

	return m.clusters.UpdateResource(clusterName, c)
}

// DeleteExport removes the possibility for ingress dataplane connections to access a given service.
func (m *xdsManager) DeleteExport(name string) error {
	m.logger.Infof("Deleting export '%s'.", name)

	clusterName := cpapi.ExportClusterName(name)
	return m.clusters.DeleteResource(clusterName)
}

// AddImport adds a listening socket for an imported remote service.
func (m *xdsManager) AddImport(imp *store.Import) error {
	m.logger.Infof("Adding import '%s'.", imp.Name)

	listenerName := cpapi.ImportListenerName(imp.Name)
	egressRouterHostname := "egress-router:443"

	tunnelingConfig := &tcpproxy.TcpProxy_TunnelingConfig{
		Hostname: egressRouterHostname,
		UsePost:  true,
		HeadersToAdd: []*core.HeaderValueOption{
			{
				Header: &core.HeaderValue{
					Key:   cpapi.ImportHeader,
					Value: imp.Name,
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
	l := &listener.Listener{
		Name: listenerName,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address: "0.0.0.0",
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(imp.Port),
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{tcpProxyFilter},
		}},
	}

	return m.listeners.UpdateResource(listenerName, l)
}

// DeleteImport removes the listening socket of a previously imported service.
func (m *xdsManager) DeleteImport(name string) error {
	m.logger.Infof("Deleting import '%s'.", name)

	listenerName := cpapi.ImportListenerName(name)
	return m.listeners.DeleteResource(listenerName)
}

func makeAddressCluster(name, addr string, port uint16, hostname string) (*cluster.Cluster, error) {
	return makeEndpointsCluster(name, []api.Endpoint{{Host: addr, Port: port}}, hostname)
}

func makeEndpointsCluster(name string, endpoints []api.Endpoint, hostname string) (*cluster.Cluster, error) {
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

	c := &cluster.Cluster{
		Name:           name,
		ConnectTimeout: durationpb.New(time.Second),
		DnsRefreshRate: durationpb.New(time.Second),
		ClusterDiscoveryType: &cluster.Cluster_Type{
			Type: cluster.Cluster_LOGICAL_DNS,
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

	return c, nil
}

func makeTCPProxyFilter(clusterName, statPrefix string,
	tunnelingConfig *tcpproxy.TcpProxy_TunnelingConfig) (*listener.Filter, error) {
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

// newXDSManager creates an uninitialized, non-registered xDS manager.
func newXDSManager() *xdsManager {
	logger := logrus.WithField("component", "xdsmanager")

	return &xdsManager{
		clusters:  cache.NewLinearCache(resource.ClusterType, cache.WithLogger(logger)),
		listeners: cache.NewLinearCache(resource.ListenerType, cache.WithLogger(logger)),
		logger:    logger,
	}
}
