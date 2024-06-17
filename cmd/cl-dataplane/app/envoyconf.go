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

package app

const (
	envoyConfigurationTemplate = `
node:
  id: {{.dataplaneID}}
  cluster: cl-dataplane
admin:
  address:
    socket_address:
      address: 127.0.0.1
      port_value: 1500
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
        cluster_name: {{.controlplaneCluster}}
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
  clusters:
  - name: {{.controlplaneCluster}}
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
      cluster_name: {{.controlplaneCluster}}
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
        max_session_keys: 0 # TODO: remove once controlplane no longer uses inet.af/tcpproxy
        common_tls_context:
          tls_certificates:
            - certificate_chain:
                filename: {{.certificateFile}}
              private_key:
                filename: {{.keyFile}}
          validation_context:
            trusted_ca:
              filename: {{.caFile}}
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
                  connect_matcher: {}
                route:
                  cluster_header: {{.targetClusterHeader}}
                  auto_host_rewrite: true
          upgrade_configs:
          - upgrade_type: CONNECT
          http_filters:
          - name: envoy.filters.http.ext_authz
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
              grpc_service:
                envoy_grpc:
                  cluster_name: {{.controlplaneCluster}}
                  retry_policy:
                    retry_back_off:
                      base_interval: 0.5s
                      max_interval: 1s
              clear_route_cache: true
              transport_api_version: V3
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
  - name: {{.ingressRouterListener}}
    address:
      socket_address:
        address: 0.0.0.0
        port_value: {{.dataplaneListenPort}}
    filter_chains:
    - transport_socket:
          name: envoy.transport_sockets.tls
          typed_config:
            "@type": type.googleapis.com/envoy.extensions.transport_sockets.tls.v3.DownstreamTlsContext
            require_client_certificate: true
            common_tls_context:
              tls_certificate_sds_secret_configs:
              - name: {{.certificateSecret}}
                sds_config:
                  resource_api_version: V3
                  initial_fetch_timeout: 1s
                  ads: {}
              validation_context_sds_secret_config:
                name: {{.validationSecret}}
                sds_config:
                  resource_api_version: V3
                  initial_fetch_timeout: 1s
                  ads: {}
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
                  connect_matcher: {}
                route:
                  cluster_header: {{.targetClusterHeader}}
                  upgrade_configs:
                  - upgrade_type: CONNECT
                    connect_config: {}
              - match:
                  prefix: /
                direct_response:
                  status: 200
          upgrade_configs:
          - upgrade_type: CONNECT
          http_filters:
          - name: composite
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.common.matching.v3.ExtensionWithMatcher
              extension_config:
                name: composite
                typed_config:
                  "@type": type.googleapis.com/envoy.extensions.filters.http.composite.v3.Composite
              matcher:
                on_no_match:
                  action:
                    name: action-no-match
                    typed_config:
                      "@type": type.googleapis.com/envoy.extensions.filters.http.composite.v3.ExecuteFilterAction
                      typed_config:
                        name: envoy.filters.http.ext_authz
                        typed_config:
                          "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
                          grpc_service:
                            envoy_grpc:
                              cluster_name: {{.controlplaneCluster}}
                              retry_policy:
                                retry_back_off:
                                  base_interval: 0.5s
                                  max_interval: 1s
                          clear_route_cache: true
                          include_peer_certificate: true
                          with_request_body:
                            max_request_bytes: 65536
                          transport_api_version: V3
                          allowed_headers:
                            patterns:
                            - exact: {{.authorizationHeader}}
                matcher_list:
                  matchers:
                  - predicate:
                      single_predicate:
                        input:
                          name: method-matcher
                          typed_config:
                            "@type": type.googleapis.com/envoy.type.matcher.v3.HttpRequestHeaderMatchInput
                            header_name: :method
                        value_match:
                          exact: CONNECT
                          ignore_case: true
                    on_match:
                      action:
                        name: connect-action
                        typed_config:
                          "@type": type.googleapis.com/envoy.extensions.filters.http.composite.v3.ExecuteFilterAction
                          typed_config:
                            name: envoy.filters.http.ext_authz
                            typed_config:
                              "@type": type.googleapis.com/envoy.extensions.filters.http.ext_authz.v3.ExtAuthz
                              grpc_service:
                                envoy_grpc:
                                  cluster_name: {{.controlplaneCluster}}
                                  retry_policy:
                                    retry_back_off:
                                      base_interval: 0.5s
                                      max_interval: 1s
                              clear_route_cache: true
                              include_peer_certificate: true
                              transport_api_version: V3
                              allowed_headers:
                                patterns:
                                - exact: {{.authorizationHeader}}
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
`
)
