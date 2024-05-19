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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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

package control

import (
	"context"

	"github.com/clusterlink-net/clusterlink/pkg/util/controller"
	v1 "k8s.io/api/core/v1"
	discv1 "k8s.io/api/discovery/v1"

	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// CreateControllers creates the various k8s controllers used to update the control manager.
func CreateControllers(mgr *Manager, controllerManager ctrl.Manager, crdMode bool) error {
	if crdMode {
		err := controller.AddToManager(controllerManager, &controller.Spec{
			Name:   "control.peer",
			Object: &v1alpha1.Peer{},
			AddHandler: func(ctx context.Context, object any) error {
				mgr.AddPeer(object.(*v1alpha1.Peer))
				return nil
			},
			DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
				mgr.DeletePeer(name.Name)
				return nil
			},
		})
		if err != nil {
			return err
		}
		err = controller.AddToManager(controllerManager, &controller.Spec{
			Name:   "control.service",
			Object: &v1.Service{},
			AddHandler: func(ctx context.Context, object any) error {
				return mgr.addService(ctx, object.(*v1.Service))
			},
			DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
				return mgr.deleteService(ctx, name)
			},
		})
		if err != nil {
			return err
		}

		err = controller.AddToManager(controllerManager, &controller.Spec{
			Name:   "control.export",
			Object: &v1alpha1.Export{},
			AddHandler: func(ctx context.Context, object any) error {
				return mgr.AddExport(ctx, object.(*v1alpha1.Export))
			},
			DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
				return nil
			},
		})
		if err != nil {
			return err
		}

		err = controller.AddToManager(controllerManager, &controller.Spec{
			Name:   "control.import",
			Object: &v1alpha1.Import{},
			AddHandler: func(ctx context.Context, object any) error {
				return mgr.AddImport(ctx, object.(*v1alpha1.Import))
			},
			DeleteHandler: mgr.DeleteImport,
		})
		if err != nil {
			return err
		}
	}

	return controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "control.endpointslice",
		Object: &discv1.EndpointSlice{},
		AddHandler: func(ctx context.Context, object any) error {
			return mgr.addEndpointSlice(ctx, object.(*discv1.EndpointSlice))
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			return mgr.deleteEndpointSlice(ctx, name)
		},
	})
}
