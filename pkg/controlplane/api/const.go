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

package api

const (
	// ListenPort is the port used by the dataplane to access the controlplane.
	ListenPort = 4444
	// ReadinessListenPort is the port used to probe for controlplane readiness.
	ReadinessListenPort = 4445
	// Name is the controlplane name.
	Name = "cl-controlplane"
)
