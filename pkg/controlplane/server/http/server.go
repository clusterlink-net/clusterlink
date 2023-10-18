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
	s.addHeartbeatHandler()

	return s
}
