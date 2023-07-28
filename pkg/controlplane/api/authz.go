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
