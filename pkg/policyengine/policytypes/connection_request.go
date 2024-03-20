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

package policytypes

// Direction indicates whether a given request is for an incoming or an outgoing connection.
type Direction int

const (
	Incoming Direction = iota
	Outgoing
)

// ConnectionRequest encapsulates all the information needed to decide on a given incoming/outgoing connection.
type ConnectionRequest struct {
	SrcWorkloadAttrs WorkloadAttrs
	DstSvcName       string
	DstSvcNamespace  string

	Direction Direction
}

// ConnectionResponse encapsulates the returned decision on a given incoming incoming/outgoing connection.
type ConnectionResponse struct {
	Action       PolicyAction
	DstPeer      string
	DstNamespace string
}
