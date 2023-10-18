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

package sniproxy

import (
	"net"

	"github.com/sirupsen/logrus"
	"inet.af/tcpproxy"

	"github.com/clusterlink-net/clusterlink/pkg/util/tcp"
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
