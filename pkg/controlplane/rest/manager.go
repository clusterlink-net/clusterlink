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

package rest

import (
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/authz"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/control"
	cpstore "github.com/clusterlink-net/clusterlink/pkg/controlplane/store"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/xds"
	"github.com/clusterlink-net/clusterlink/pkg/store"
)

// Manager of a controlplane, where all API servers delegate their requested actions to.
type Manager struct {
	namespace string

	peers      *cpstore.Peers
	exports    *cpstore.Exports
	imports    *cpstore.Imports
	acPolicies *cpstore.AccessPolicies

	xdsManager     *xds.Manager
	authzManager   *authz.Manager
	controlManager *control.Manager

	initialized bool

	logger *logrus.Entry
}

// init initializes the controlplane manager.
func (m *Manager) init() error {
	// add peers
	for _, p := range m.GetAllPeers() {
		if err := m.CreatePeer(p); err != nil {
			return err
		}
	}

	// add exports
	for _, export := range m.GetAllExports() {
		if err := m.CreateExport(export); err != nil {
			return err
		}
	}

	// add exports
	for _, imp := range m.GetAllImports() {
		if err := m.CreateImport(imp); err != nil {
			return err
		}
	}

	// add access policies
	for _, policy := range m.GetAllAccessPolicies() {
		if err := m.CreateAccessPolicy(policy); err != nil {
			return err
		}
	}

	m.initialized = true

	return nil
}

// NewManager returns a new controlplane CRUD manager.
func NewManager(
	namespace string,
	storeManager store.Manager,
	xdsManager *xds.Manager,
	authzManager *authz.Manager,
	controlManager *control.Manager,
) (*Manager, error) {
	logger := logrus.WithField("component", "controlplane.rest.manager")

	peers, err := cpstore.NewPeers(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load peers from store: %w", err)
	}
	logger.Infof("Loaded %d peers.", peers.Len())

	exports, err := cpstore.NewExports(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load exports from store: %w", err)
	}
	logger.Infof("Loaded %d exports.", exports.Len())

	imports, err := cpstore.NewImports(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load imports from store: %w", err)
	}
	logger.Infof("Loaded %d imports.", imports.Len())

	acPolicies, err := cpstore.NewAccessPolicies(storeManager)
	if err != nil {
		return nil, fmt.Errorf("cannot load access policies from store: %w", err)
	}
	logger.Infof("Loaded %d access policies.", acPolicies.Len())

	m := &Manager{
		namespace:      namespace,
		peers:          peers,
		exports:        exports,
		imports:        imports,
		acPolicies:     acPolicies,
		xdsManager:     xdsManager,
		authzManager:   authzManager,
		controlManager: controlManager,
		initialized:    false,
		logger:         logger,
	}

	// initialize instance
	if err := m.init(); err != nil {
		return nil, err
	}

	return m, nil
}
