package sniproxy

import (
	"net"

	"github.com/sirupsen/logrus"
	"inet.af/tcpproxy"

	"github.com/clusterlink-org/clusterlink/pkg/util/tcp"
)

// Server for proxying connections by checking client SNI.
type Server struct {
	tcp.Listener

	routes map[string]string
	server *tcpproxy.Proxy

	logger *logrus.Entry
}

// Serve starts the server.
func (s *Server) Serve() error {
	listenAddress := s.GetAddress()
	for sni, targetAddress := range s.routes {
		s.server.AddSNIRoute(listenAddress, sni, tcpproxy.To(targetAddress))
	}

	return s.server.Run()
}

// Stop the server.
func (s *Server) Stop() error {
	return s.server.Close()
}

// GracefulStop does a graceful stop of the server.
func (s *Server) GracefulStop() error {
	return s.server.Close()
}

// NewServer returns a new server.
// routes map (server name) -> (target host:port).
func NewServer(routes map[string]string) *Server {
	logger := logrus.WithFields(logrus.Fields{
		"component": "sni-proxy"})

	s := &Server{
		Listener: tcp.NewListener("sni-proxy"),
		routes:   routes,
		server:   &tcpproxy.Proxy{},
		logger:   logger,
	}

	s.server.ListenFunc = func(_, _ string) (net.Listener, error) {
		return s.GetListener(), nil
	}

	return s
}
