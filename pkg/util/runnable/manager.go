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

package runnable

import (
	"errors"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// Runnable represents a runnable instance.
type Instance interface {
	Name() string
	Start() error
	Stop() error
	GracefulStop() error
}

// Server represents a runnable server.
type Server interface {
	Instance
	Listen(address string) error
	Close() error
}

// Manager manages a set of runnables.
type Manager struct {
	lock sync.Mutex

	runnables     []Instance
	serverAddress map[Server]string
	errors        map[Instance]error

	logger *logrus.Entry
}

// AddServer adds a new server.
func (c *Manager) AddServer(listenAddress string, server Server) {
	c.Add(server)
	c.serverAddress[server] = listenAddress
}

// Add a new runnable.
func (c *Manager) Add(runnable Instance) {
	c.runnables = append(c.runnables, runnable)
}

// Run starts all runnables.
func (c *Manager) Run() error {
	defer func() {
		// close all servers
		for server := range c.serverAddress {
			if err := server.Close(); err != nil {
				c.logger.Warnf("Error closing server '%s': %v.", server.Name(), err)
			}
		}
	}()

	// start server listeners
	for server, listenAddress := range c.serverAddress {
		if err := server.Listen(listenAddress); err != nil {
			return fmt.Errorf("unable to create listener for server '%s' on %s: %w",
				server.Name(), listenAddress, err)
		}
	}

	lock := &sync.Mutex{}
	stop := sync.NewCond(lock)

	// goroutine for stopping all runnables if one fails
	go func(stop *sync.Cond) {
		stop.L.Lock()
		stop.Wait()
		stop.L.Unlock()

		c.lock.Lock()
		pending := len(c.errors) < len(c.runnables)
		c.lock.Unlock()

		if pending {
			if err := c.Stop(); err != nil {
				c.logger.Warnf("Error stopping: %v.", err)
			} else {
				c.logger.Infof("Asked all runnables to stop.")
			}
		}
	}(stop)

	// initialize wait group
	wg := &sync.WaitGroup{}
	wg.Add(len(c.runnables))

	// start runnables in goroutines
	for _, runnable := range c.runnables {
		go func(runnable Instance) {
			defer wg.Done()

			c.logger.Infof("Starting runnable '%s'.", runnable.Name())
			err := runnable.Start()
			c.logger.Infof("Runnable '%s' stopped: %v.", runnable.Name(), err)

			c.lock.Lock()
			c.errors[runnable] = err
			c.lock.Unlock()

			if err != nil {
				// signal to stop other runnables
				stop.Signal()
			}
		}(runnable)
	}

	// wait for all runnables to stop
	wg.Wait()

	// terminate error-waiting goroutine
	stop.Signal()

	// collect and return errors
	var errs []error
	for runnable, err := range c.errors {
		if err != nil {
			errs = append(errs, fmt.Errorf(
				"error running '%s': %w", runnable.Name(), err))
		}
	}
	return errors.Join(errs...)
}

// Stop all runnables.
func (c *Manager) Stop() error {
	c.logger.Info("Stopping.")

	var errs []error
	for _, runnable := range c.runnables {
		if err := runnable.Stop(); err != nil {
			errs = append(errs, fmt.Errorf(
				"unable to stop '%s': %w", runnable.Name(), err))
		}
	}

	return errors.Join(errs...)
}

// GracefulStop gracefully stops all runnables.
func (c *Manager) GracefulStop() error {
	c.logger.Info("Gracefully stopping.")

	var errs []error
	for _, runnable := range c.runnables {
		if err := runnable.GracefulStop(); err != nil {
			errs = append(errs, fmt.Errorf(
				"unable to gracefully stop '%s': %w", runnable.Name(), err))
		}
	}

	return errors.Join(errs...)
}

// NewManager returns a new empty runnable manager.
func NewManager() *Manager {
	return &Manager{
		serverAddress: make(map[Server]string),
		errors:        make(map[Instance]error),
		logger:        logrus.WithField("component", "util.controller"),
	}
}
