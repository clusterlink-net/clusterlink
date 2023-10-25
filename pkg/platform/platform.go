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

package platform

// Platform abstracts all operations which are handled by the specific platform (e.g. Kubernetes).
type Platform interface {
	CreateService(name, targetApp string, port, targetPort uint16)
	UpdateService(name, targetApp string, port, targetPort uint16)
	DeleteService(name string)
	CreateEndpoint(name, targetIP string, targetPort uint16)
	UpdateEndpoint(name, targetIP string, targetPort uint16)
	DeleteEndpoint(name string)
	GetLabelsFromIP(ip string) map[string]string
}
