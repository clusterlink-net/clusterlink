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

package tcp

import (
	"net"

	"github.com/sirupsen/logrus"
)

// Listener is a wrapper of a TCP listener.
type Listener struct {
	name     string
	address  string
	listener net.Listener

	logger *logrus.Entry
}

// Listen starts the listener.
func (l *Listener) Listen(address string) error {
	l.logger.Infof("Creating listener on %s.", address)

	lis, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	l.address = address
	l.listener = lis
	return nil
}

// GetAddress returns the listening address.
func (l *Listener) GetAddress() string {
	return l.address
}

// GetListener returns the wrapped listener.
func (l *Listener) GetListener() net.Listener {
	return l.listener
}

// Name returns the name of listener.
func (l *Listener) Name() string {
	return l.name
}

// Close the listener.
func (l *Listener) Close() error {
	l.logger.Infof("Closing listener.")

	if l.listener != nil {
		return l.listener.Close()
	}
	return nil
}

// NewListener returns a new listener.
func NewListener(name string) Listener {
	return Listener{
		name: name,
		logger: logrus.WithFields(logrus.Fields{
			"component": "listener",
			"name":      name,
		}),
	}
}
