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

package unknown

// Platform represents an unknown platform.
type Platform struct {
}

// CreateService creates a service.
func (d *Platform) CreateService(_, _ string, _, _ uint16) {
}

// UpdateService updates a service.
func (d *Platform) UpdateService(_, _ string, _, _ uint16) {
}

// DeleteService deletes a service.
func (d *Platform) DeleteService(_ string) {
}

// CreateEndpoint creates a endpoint.
func (d *Platform) CreateEndpoint(_, _ string, _ uint16) {
}

// UpdateEndpoint updates a endpoint.
func (d *Platform) UpdateEndpoint(_, _ string, _ uint16) {
}

// DeleteEndpoint deletes a endpoint.
func (d *Platform) DeleteEndpoint(_ string) {
}

// NewPlatform returns a new unknown platform.
func NewPlatform() *Platform {
	return &Platform{}
}
