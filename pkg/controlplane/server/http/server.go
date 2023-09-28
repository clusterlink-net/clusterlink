package http

import (
	"crypto/tls"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane"
	"github.com/clusterlink-net/clusterlink/pkg/util/rest"
)

// Server implementing the management API, allowing to manage the set of peers, imports, exports and bindings.
// Furthermore, this server implements the various authorization APIs.
type Server struct {
	rest.Server
	cp *controlplane.Instance

	logger *logrus.Entry
}

// NewServer returns a new controlplane HTTP server.
func NewServer(cp *controlplane.Instance, tlsConfig *tls.Config) *Server {
	s := &Server{
		Server: rest.NewServer("controlplane-http", tlsConfig),
		cp:     cp,
		logger: logrus.WithField("component", "controlplane.server.http"),
	}

	s.addAPIHandlers()
	s.addAuthzHandlers()

	return s
}
