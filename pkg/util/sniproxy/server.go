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

package sniproxy

import (
	"net"
	"time"

	"github.com/inetaf/tcpproxy"
	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/util/tcp"
)

// Server for proxying connections by checking client SNI.
type Server struct {
	tcp.Listener

	routes map[string]string
	server *tcpproxy.Proxy

	logger *logrus.Entry
}

// Start the server.
func (s *Server) Start() error {
	listenAddress := s.GetAddress()
	for sni, targetAddress := range s.routes {
		target := tcpproxy.To(targetAddress)
		target.DialTimeout = 100 * time.Millisecond
		s.server.AddSNIRoute(listenAddress, sni, target)
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
		"component": "sni-proxy",
	})

	sniproxy := &Server{
		Listener: tcp.NewListener("sni-proxy"),
		routes:   routes,
		server:   &tcpproxy.Proxy{},
		logger:   logger,
	}

	sniproxy.server.ListenFunc = func(_, _ string) (net.Listener, error) {
		return sniproxy.GetListener(), nil
	}

	return sniproxy
}
