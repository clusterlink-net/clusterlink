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

package api

const (
	// RemotePeerAuthorizationPath is the path remote peers use to send an authorization request.
	RemotePeerAuthorizationPath = "/authz"
	// DataplaneEgressAuthorizationPath is the path the dataplane uses to authorize an egress connection.
	DataplaneEgressAuthorizationPath = "/authz/egress/"
	// DataplaneIngressAuthorizationPath is the path the dataplane uses to authorize an ingress connection.
	DataplaneIngressAuthorizationPath = "/authz/ingress/"

	// ImportHeader holds the name of the imported service.
	ImportHeader = "x-import"
	// ClientIPHeader holds the IP address of the source client.
	ClientIPHeader = "x-forwarded-for"

	// AuthorizationHeader holds a signed token allowing ingress connections to access the dataplane.
	AuthorizationHeader = "authorization"

	// TargetClusterHeader holds the name of the target cluster.
	TargetClusterHeader = "host"

	// ExportNameJWTClaim holds the name of the requested exported service.
	ExportNameJWTClaim = "export_name"
)

// AuthorizationRequest represents an authorization request for accessing an exported service.
type AuthorizationRequest struct {
	// Service is the name of the requested exported service.
	Service string
}

// AuthorizationResponse represents a response for a successful AuthorizationRequest.
type AuthorizationResponse struct {
	// AccessToken holds an access token which can be used to access the requested exported service.
	AccessToken string
}
