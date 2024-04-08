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

package connectivitypdp

import (
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
)

// AccessPolicy is an opaque, PDP-internal, generalized representation of AccessPolicy and PrivilegedAccessPolicy CRDs.
type AccessPolicy struct {
	name       types.NamespacedName
	privileged bool
	spec       v1alpha1.AccessPolicySpec
}

// PolicyFromCRD converts the AccessPolicy CRD into the PDP's AccessPolicy.
func PolicyFromCRD(vap *v1alpha1.AccessPolicy) *AccessPolicy {
	return &AccessPolicy{
		name:       types.NamespacedName{Namespace: vap.Namespace, Name: vap.Name},
		privileged: false,
		spec:       vap.Spec,
	}
}

// PolicyFromPrivilegedCRD converts the PrivilegedAccessPolicy CRD into the PDP's AccessPolicy.
func PolicyFromPrivilegedCRD(vap *v1alpha1.PrivilegedAccessPolicy) *AccessPolicy {
	return &AccessPolicy{
		name:       types.NamespacedName{Name: vap.Name},
		privileged: true,
		spec:       vap.Spec,
	}
}
