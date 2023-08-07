package api

import "fmt"

const (
	// ListenPort is the port used by the dataplane to access the controlplane.
	ListenPort = 444

	// gRPCServerNamePrefix is the prefix such that <gRPCServerNamePrefix>.<peer name> is the gRPC server name.
	gRPCServerNamePrefix = "grpc"
)

// GRPCServerName returns the gRPC server name of a specific peer.
func GRPCServerName(peer string) string {
	return fmt.Sprintf("%s.%s", gRPCServerNamePrefix, peer)
}
