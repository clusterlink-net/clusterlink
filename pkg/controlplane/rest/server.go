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

package rest

import (
	"github.com/clusterlink-net/clusterlink/pkg/util/rest"
)

// RegisteHandlers registers the HTTP handlers for REST requests.
func RegisterHandlers(manager *Manager, srv *rest.Server) {
	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/peers",
		Handler:       &peerHandler{manager: manager},
		DeleteByValue: false,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/exports",
		Handler:       &exportHandler{manager: manager},
		DeleteByValue: false,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/imports",
		Handler:       &importHandler{manager: manager},
		DeleteByValue: false,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/bindings",
		Handler:       &bindingHandler{manager: manager},
		DeleteByValue: true,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/policies",
		Handler:       &accessPolicyHandler{manager: manager},
		DeleteByValue: false,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/lbpolicies",
		Handler:       &lbPolicyHandler{manager: manager},
		DeleteByValue: false,
	})
}
