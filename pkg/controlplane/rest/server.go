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
	"github.com/clusterlink-net/clusterlink/pkg/controlplane"
	"github.com/clusterlink-net/clusterlink/pkg/util/rest"
)

// RegisteHandlers registers the HTTP handlers for REST requests.
func RegisterHandlers(cp *controlplane.Instance, srv *rest.Server) {
	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/peers",
		Handler:       &peerHandler{cp: cp},
		DeleteByValue: false,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/exports",
		Handler:       &exportHandler{cp: cp},
		DeleteByValue: false,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/imports",
		Handler:       &importHandler{cp: cp},
		DeleteByValue: false,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/bindings",
		Handler:       &bindingHandler{cp: cp},
		DeleteByValue: true,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/policies",
		Handler:       &accessPolicyHandler{cp: cp},
		DeleteByValue: false,
	})

	srv.AddObjectHandlers(&rest.ServerObjectSpec{
		BasePath:      "/lbpolicies",
		Handler:       &lbPolicyHandler{cp: cp},
		DeleteByValue: false,
	})
}
