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

package controlplane

// Runtime Types.
const (
	k8s = "k8s"
)

// MyRunTimeEnv defines the runtime environment where the controlplane is deployed.
var MyRunTimeEnv runtimeEnv

type runtimeEnv struct {
	rtenvType string
}

// IsRuntimeEnvK8s returns if the runtime environment of the controlplane is Kubernetes based.
func (r *runtimeEnv) IsRuntimeEnvK8s() bool {
	return (r.rtenvType == k8s)
}

// SetRuntimeEnv sets the runtime environment of the controlplane.
func (r *runtimeEnv) SetRuntimeEnv(rtenv string) {
	r.rtenvType = rtenv
}
