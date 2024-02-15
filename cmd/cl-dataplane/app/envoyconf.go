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

package app

const (
	envoyConfigurationTemplate = `
node:
  id: {{.dataplaneID}}
  cluster: {{.peerName}}
admin:
  address:
    socket_address:
      address: 127.0.0.1
      port_value: 1000
bootstrap_extensions:
- name: envoy.bootstrap.internal_listener
  typed_config:
    "@type": type.googleapis.com/envoy.extensions.bootstrap.internal_listener.v3.InternalListener
layered_runtime:
  layers:
  - name: static_layer
    static_layer:
      overload:
        global_downstream_max_connections: 50000
dynamic_resources:
  ads_config:
    api_type: DELTA_GRPC
    transport_api_version: V3
    grpc_services:
    - envoy_grpc:
        cluster_name: {{.controlplaneGRPCCluster}}
        retry_policy:
          retry_back_off:
            base_interval: 0.5s
            max_interval: 1s
  lds_config:
    resource_api_version: V3
    initial_fetch_timeout: 1s
    ads: {}
  cds_config:
    resource_api_version: V3
    initial_fetch_timeout: 1s
    ads: {}
static_resources:
  secrets:
  - name: {{.certificateSecret}}
    tls_certificate:
      certificate_chain:
        filename: {{.certificateFile}}
      private_key:
        filename: {{.keyFile}}
  - name: {{.validationSecret}}
    validation_context:
      trusted_ca:
        filename: {{.caFile}}
  clusters:
  - name: {{.controlplaneGRPCCluster}}
    type: LOGICAL_DNS
    dns_refresh_rate: 1s
    connect_timeout: 1s
    typed_dns_resolver_config:
      name: envoy.network.dns_resolver.getaddrinfo
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.network.dns_resolver.getaddrinfo.v3.GetAddrInfoDnsResolverConfig
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        "@type": type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
        explicit_http_config:
          http2_protocol_options: {}
    load_assignment:
      cluster_name: {{.controlplaneGRPCCluster}}
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: {{.controlplaneHost}}
                port_value: {{.controlplanePort}}
    transport_socket:
      name: envoy.transport_sockets.tls
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
        sni: {{.controlplaneGRPCSNI}}
        common_tls_context:
          tls_certificate_sds_secret_configs:
          - name: {{.certificateSecret}}
          validation_context_sds_secret_config:
            name: {{.validationSecret}}
  - name: {{.controlplaneInternalHTTPCluster}}
    type: LOGICAL_DNS
    dns_refresh_rate: 1s
    connect_timeout: 1s
    typed_dns_resolver_config:
      name: envoy.network.dns_resolver.getaddrinfo
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.network.dns_resolver.getaddrinfo.v3.GetAddrInfoDnsResolverConfig
    load_assignment:
      cluster_name: {{.controlplaneInternalHTTPCluster}}
      endpoints:
      - lb_endpoints:
        - endpoint:
            hostname: {{.peerName}}
            address:
              socket_address:
                address: {{.controlplaneHost}}
                port_value: {{.controlplanePort}}
    transport_socket:
      name: envoy.transport_sockets.tls
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.UpstreamTlsContext
        sni: {{.peerName}}
        common_tls_context:
          tls_certificate_sds_secret_configs:
          - name: {{.certificateSecret}}
          validation_context_sds_secret_config:
            name: {{.validationSecret}}
  - name: {{.controlplaneExternalHTTPCluster}}
    type: LOGICAL_DNS
    dns_refresh_rate: 1s
    connect_timeout: 1s
    typed_dns_resolver_config:
      name: envoy.network.dns_resolver.getaddrinfo
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.network.dns_resolver.getaddrinfo.v3.GetAddrInfoDnsResolverConfig
    load_assignment:
      cluster_name: {{.controlplaneInternalHTTPCluster}}
      endpoints:
      - lb_endpoints:
        - endpoint:
            hostname: {{.peerName}}
            address:
              socket_address:
                address: {{.controlplaneHost}}
                port_value: {{.controlplanePort}}
  - name: {{.egressRouterCluster}}
    connect_timeout: 1s
    typed_extension_protocol_options:
      envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
        "@type": type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
        explicit_http_config:
          http2_protocol_options:
            max_outbound_frames: 50000
    load_assignment:
      cluster_name: {{.egressRouterCluster}}
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              envoy_internal_address:
                server_listener_name: {{.egressRouterListener}}
  listeners:
  - name: {{.egressRouterListener}}
    internal_listener: {}
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: hcm-egress
          route_config:
            virtual_hosts:
            - name: egress
              domains: ["*"]
              routes:
              - match:
                  path: /
                route:
                  cluster_header: {{.targetClusterHeader}}
                  auto_host_rewrite: true
                  prefix_rewrite: /
          upgrade_configs:
          - upgrade_type: CONNECT
          http_filters:
          - name: envoy.filters.http.ext_authz
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
              http_service:
                server_uri:
                  uri: {{.peerName}}
                  cluster: {{.controlplaneInternalHTTPCluster}}
                  timeout: 0.250s
                path_prefix: {{.dataplaneEgressAuthorizationPrefix}}
                authorization_response:
                  allowed_upstream_headers:
                    patterns:
                    - exact: {{.targetClusterHeader}}
                    - exact: {{.authorizationHeader}}
              clear_route_cache: true
              transport_api_version: V3
              allowed_headers:
                patterns:
                - exact: {{.importNameHeader}}
                - exact: {{.importNamespaceHeader}}
                - exact: {{.clientIPHeader}}
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
  - name: {{.ingressRouterListener}}
    address:
      socket_address:
        address: 0.0.0.0
        port_value: {{.dataplaneListenPort}}
    listener_filters:
    - name: envoy.filters.listener.tls_inspector
      typed_config:
        "@type": type.googleapis.com/envoy.extensions.filters.listener.tls_inspector.v3.TlsInspector
    filter_chains:
    - filter_chain_match:
        server_names: ["{{.peerName}}"]
      filters:
      - name: envoy.filters.network.tcp_proxy
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
          stat_prefix: tcp-proxy-controlplane
          cluster: {{.controlplaneExternalHTTPCluster}}
    - filter_chain_match:
        server_names: ["{{.dataplaneSNI}}"]
      transport_socket:
          name: envoy.transport_sockets.tls
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
            require_client_certificate: true
            common_tls_context:
              tls_certificate_sds_secret_configs:
              - name: {{.certificateSecret}}
              validation_context_sds_secret_config:
                name: {{.validationSecret}}
      filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: hcm-ingress
          route_config:
            virtual_hosts:
            - name: ingress
              domains: ["*"]
              routes:
              - match:
                  path: /
                route:
                  cluster_header: {{.targetClusterHeader}}
                  upgrade_configs:
                  - upgrade_type: CONNECT
                    connect_config:
                      allow_post: true
          upgrade_configs:
          - upgrade_type: CONNECT
          http_filters:
          - name: envoy.filters.http.ext_authz
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
              http_service:
                server_uri:
                  uri: {{.peerName}}
                  cluster: {{.controlplaneInternalHTTPCluster}}
                  timeout: 0.250s
                path_prefix: {{.dataplaneIngressAuthorizationPrefix}}
                authorization_response:
                  allowed_upstream_headers:
                    patterns:
                    - exact: {{.targetClusterHeader}}
              clear_route_cache: true
              transport_api_version: V3
              allowed_headers:
                patterns:
                - exact: {{.authorizationHeader}}
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
`
)
