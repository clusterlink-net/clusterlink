package grpc

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/clusterlink-net/clusterlink/pkg/util/tcp"
)

// Server is a wrapper of a gRPC server.
type Server struct {
	tcp.Listener

	server *grpc.Server
}

// GetGRPCServer returns the underlying gRPC server instance.
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.server
}

// Serve starts the server.
func (s *Server) Serve() error {
	return s.server.Serve(s.GetListener())
}

// Stop the server.
func (s *Server) Stop() error {
	s.server.Stop()
	return nil
}

// GracefulStop does a graceful stop of the server.
func (s *Server) GracefulStop() error {
	s.server.GracefulStop()
	return nil
}

// NewServer returns a new server.
func NewServer(name string, tlsConfig *tls.Config) Server {
	return Server{
		Listener: tcp.NewListener(name),
		server:   grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig))),
	}
}
