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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

import "github.com/lestrrat-go/jwx/jwa"

const (
	// RemotePeerAuthorizationPath is the path remote peers use to send an authorization request.
	RemotePeerAuthorizationPath = "/authz"
	// DataplaneEgressAuthorizationPath is the path the dataplane uses to authorize an egress connection.
	DataplaneEgressAuthorizationPath = "/authz/egress/"
	// DataplaneIngressAuthorizationPath is the path the dataplane uses to authorize an ingress connection.
	DataplaneIngressAuthorizationPath = "/authz/ingress/"

	// ImportNameHeader holds the name of the imported service.
	ImportNameHeader = "x-import-name"
	// ImportNamespaceHeader holds the namespace of the imported service.
	ImportNamespaceHeader = "x-import-namespace"
	// ClientIPHeader holds the IP address of the source client.
	ClientIPHeader = "x-client-ip"

	// AuthorizationHeader holds a signed token allowing ingress connections to access the dataplane.
	AuthorizationHeader = "authorization"

	// TargetClusterHeader holds the name of the target cluster.
	TargetClusterHeader = "host"

	// JWTSignatureAlgorithm defines the signing algorithm for JWT tokens.
	JWTSignatureAlgorithm = jwa.RS256
	// ExportNameJWTClaim holds the name of the requested exported service.
	ExportNameJWTClaim = "export_name"
	// ExportNamespaceJWTClaim holds the namespace of the requested exported service.
	ExportNamespaceJWTClaim = "export_namespace"
)

// AuthorizationRequest represents an authorization request for accessing an exported service.
type AuthorizationRequest struct {
	// ServiceName is the name of the requested exported service.
	ServiceName string
	// ServiceNamespace is the namespace of the requested exported service.
	ServiceNamespace string
}

// AuthorizationResponse represents a response for a successful AuthorizationRequest.
type AuthorizationResponse struct {
	// AccessToken holds an access token which can be used to access the requested exported service.
	AccessToken string
}
