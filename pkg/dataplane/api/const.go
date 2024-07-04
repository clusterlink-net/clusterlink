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
	// ListenPort is the dataplane external listening port.
	ListenPort = 4443
	// Name is the dataplane name.
	Name = "cl-dataplane"
	// Name of the go-dataplane image.
	GoDataplaneName = "cl-go-dataplane"
	// ReadinessListenPort is the port used to probe for dataplane readiness.
	ReadinessListenPort = 4445
)
