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
	"fmt"
	"math/rand"
	"sync"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// startPort is the first port number that can be allocated.
	startPort = uint16(1024)
	// endPort is the last port number that can be allocated (not including).
	endPort = uint16(49152)
	// portCount is the maximum number of ports that can be allocated.
	portCount = endPort - startPort

	// maxRandomTries is the maximum number of times we try to generate a random free port,
	// before switching to an iterative full-scan for a free port.
	// should be > 0.
	maxRandomTries = 40
)

type conflictingTargetPortError struct {
	port      uint16
	leaseName types.NamespacedName
}

func (e conflictingTargetPortError) Error() string {
	return fmt.Sprintf("port %d is already in use by service %v", e.port, e.leaseName)
}

// portManager leases ports for use by imported services.
type portManager struct {
	lock         sync.Mutex
	leasesByPort map[uint16]types.NamespacedName
	leasesByName map[types.NamespacedName]uint16

	logger *logrus.Entry
}

// getRandomFreePort returns a random free port.
// It first tries to generate a random port and checking whether it's free.
// If this fails for <maxRandomTries> times, it starts scanning the entire
// port range (starting from a random offset), looking for a free port.
func (m *portManager) getRandomFreePort() uint16 {
	// try to generate a random port number and checking if it's free
	var port uint16
	for i := 0; i < maxRandomTries; i++ {
		//#nosec G404 -- port numbers do not need secure random
		port := startPort + uint16(rand.Intn(int(endPort-startPort)))

		if _, ok := m.leasesByPort[port]; !ok {
			return port
		}
	}

	// iterate all ports to find a free one
	for i := uint16(0); i < portCount-2; i++ {
		port++
		if port == endPort {
			port = startPort
		}

		if _, ok := m.leasesByPort[port]; !ok {
			break
		}
	}

	return port
}

// Lease marks a port as taken by the given name. If port is 0, some random free port is returned.
func (m *portManager) Lease(name types.NamespacedName, port uint16) (uint16, error) {
	m.logger.Infof("Leasing %d for %v.", port, name)

	m.lock.Lock()
	defer m.lock.Unlock()

	if port == 0 {
		if len(m.leasesByPort) == int(portCount) {
			return 0, fmt.Errorf("all ports are taken")
		}

		if port = m.leasesByName[name]; port != 0 {
			m.logger.Infof("Leased existing: %d.", port)
		} else {
			port = m.getRandomFreePort()
			m.logger.Infof("Generated port: %d.", port)
		}
	} else {
		if leaseName, ok := m.leasesByPort[port]; ok && leaseName != name {
			return 0, conflictingTargetPortError{
				port:      port,
				leaseName: leaseName,
			}
		}
	}

	// mark previous port (if exists) is free
	if port, ok := m.leasesByName[name]; ok {
		delete(m.leasesByPort, port)
	}

	// mark port is leased
	m.leasesByPort[port] = name
	m.leasesByName[name] = port

	return port, nil
}

// Release returns a leased port to be re-used by others.
func (m *portManager) Release(name types.NamespacedName) {
	m.logger.Infof("Returning port for: '%v'.", name)

	m.lock.Lock()
	defer m.lock.Unlock()

	if port, ok := m.leasesByName[name]; ok {
		delete(m.leasesByName, name)
		delete(m.leasesByPort, port)
	}
}

// newPortManager returns a new empty portManager.
func newPortManager() *portManager {
	logger := logrus.WithField("component", "controlplane.control.portmanager")

	logger.WithFields(logrus.Fields{
		"start": startPort,
		"end":   endPort,
	},
	).Info("Initialized.")

	return &portManager{
		leasesByPort: make(map[uint16]types.NamespacedName),
		leasesByName: make(map[types.NamespacedName]uint16),
		logger:       logger,
	}
}
