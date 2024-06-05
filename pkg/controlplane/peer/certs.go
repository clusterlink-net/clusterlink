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

package peer

import (
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"

	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

// CertsConsumer represents a consumer of peer TLS certificates.
type CertsConsumer interface {
	SetPeerCertificates(parsedCertData *tls.ParsedCertData, rawCertData *tls.RawCertData) error
}

// parseError represents an error in parsing the peer certificates.
type parseError struct {
	err error
}

func (e parseError) Error() string {
	return e.err.Error()
}

// CertsWatcher watches certificate updates.
type CertsWatcher struct {
	caPath   string
	certPath string
	keyPath  string

	stopCh    chan struct{}
	consumers []CertsConsumer

	logger *logrus.Entry
}

// Name of the watcher.
func (w *CertsWatcher) Name() string {
	return "certs-watcher"
}

// AddConsumer adds a new peer certificates consumer.
// This function is not thread-safe.
func (w *CertsWatcher) AddConsumer(consumer CertsConsumer) {
	w.consumers = append(w.consumers, consumer)
}

// ReadCertsAndUpdateConsumers reads the peer certificates and updates the consumers.
func (w *CertsWatcher) ReadCertsAndUpdateConsumers() error {
	w.logger.Infof("Updating certificates.")

	parsedCertData, rawCertData, err := tls.ParseFiles(w.caPath, w.certPath, w.keyPath)
	if err != nil {
		return &parseError{err: err}
	}

	for _, consumer := range w.consumers {
		if err := consumer.SetPeerCertificates(parsedCertData, rawCertData); err != nil {
			return fmt.Errorf("error setting peer certificates on %v: %w", consumer, err)
		}
	}

	return nil
}

// Start the certs watcher.
func (w *CertsWatcher) Start() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("cannot initialize file watcher: %w", err)
	}

	defer func() {
		if err := watcher.Close(); err != nil {
			w.logger.Warnf("Cannot close watcher: %v", err)
		}
	}()

	watchedFiles := []string{w.caPath, w.certPath, w.keyPath}
	watchedDirs := make(map[string]interface{})
	for _, file := range watchedFiles {
		dir := path.Dir(file)
		if _, ok := watchedDirs[dir]; !ok {
			w.logger.Infof("Watching: %s.", dir)
			if err := watcher.Add(dir); err != nil {
				return fmt.Errorf("cannot watch directory '%s': %w", dir, err)
			}
			watchedDirs[dir] = nil
		}
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	certsModified := false
	for {
		select {
		case <-w.stopCh:
			return nil
		case event := <-watcher.Events:
			w.logger.Debugf("Event: %v", event)
			certsModified = true
		case err := <-watcher.Errors:
			w.logger.Errorf("Error: %v", err)
			return err
		case <-ticker.C:
			if !certsModified {
				continue
			}

			w.logger.Infof("Certificates modified.")
			certsModified = false

			if err = w.ReadCertsAndUpdateConsumers(); err == nil {
				continue
			}

			w.logger.Infof("Error: %v", err)

			if !errors.Is(err, &parseError{}) {
				return err
			}

			w.logger.Errorf("Error parsing peer TLS certificates: %v.", err)
		}
	}
}

// Stop the watcher.
func (w *CertsWatcher) Stop() error {
	close(w.stopCh)
	return nil
}

// GracefulStop does a graceful stop of the watcher.
func (w *CertsWatcher) GracefulStop() error {
	return w.Stop()
}

// NewWatcher returns a new certificate files watcher.
func NewWatcher(caPath, certPath, keyPath string) *CertsWatcher {
	return &CertsWatcher{
		caPath:   caPath,
		certPath: certPath,
		keyPath:  keyPath,
		stopCh:   make(chan struct{}),
		logger:   logrus.WithField("component", "controlplane.peer.certs-watcher"),
	}
}
