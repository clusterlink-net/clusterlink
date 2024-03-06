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
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/peer"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

const (
	heartbeatInterval = 10 * time.Second
)

// peerMonitor monitors a single peer.
type peerMonitor struct {
	lock           sync.Mutex
	pr             *v1alpha1.Peer
	client         *peer.Client
	statusCallback func(*v1alpha1.Peer)

	wg     *sync.WaitGroup
	stopCh chan struct{}

	logger *logrus.Entry
}

// peerManager manages peers status.
type peerManager struct {
	client         client.Client
	peerTLS        *tls.ParsedCertData
	statusCallback func(*v1alpha1.Peer)

	lock     sync.Mutex
	monitors map[string]*peerMonitor

	stopped         bool
	monitorWG       sync.WaitGroup
	updaterWG       sync.WaitGroup
	stopCh          chan struct{}
	statusUpdatesCh chan *v1alpha1.Peer

	logger *logrus.Entry
}

func (m *peerMonitor) Peer() v1alpha1.Peer {
	m.lock.Lock()
	defer m.lock.Unlock()

	return *m.pr
}

func (m *peerMonitor) Start() {
	defer m.wg.Done()

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	backoffConfig := backoff.NewExponentialBackOff()

	reachable := false
	reachableCond := metav1.Condition{
		Type:   v1alpha1.PeerReachable,
		Status: metav1.ConditionFalse,
		Reason: "Heartbeat",
	}

	for {
		select {
		case <-m.stopCh:
			return
		default:
			break
		}

		err := backoff.Retry(m.client.GetHeartbeat, backoffConfig)
		if heartbeatOK := err == nil; heartbeatOK != reachable {
			m.logger.Infof("Heartbeat result: %v", heartbeatOK)

			if heartbeatOK {
				reachableCond.Status = metav1.ConditionTrue
				backoffConfig.MaxElapsedTime = heartbeatInterval
			} else {
				reachableCond.Status = metav1.ConditionFalse
				backoffConfig.MaxElapsedTime = 0
			}

			reachable = heartbeatOK

			m.lock.Lock()
			meta.SetStatusCondition(&m.pr.Status.Conditions, reachableCond)
			m.lock.Unlock()

			// callback for non-CRD mode, which does not watch peers/status
			m.statusCallback(m.pr)
		}

		// wait till it's time for next heartbeat round
		<-ticker.C
	}
}

func (m *peerMonitor) Stop() {
	close(m.stopCh)
}

// AddPeer defines a new route target for egress dataplane connections.
func (m *peerManager) AddPeer(pr *v1alpha1.Peer) {
	m.logger.Infof("Adding peer '%s'.", pr.Name)

	m.lock.Lock()
	defer m.lock.Unlock()

	if m.stopped {
		return
	}

	monitor, ok := m.monitors[pr.Name]
	if !ok || peerChanged(monitor.pr, pr) {
		m.monitors[pr.Name] = newPeerMonitor(pr, m)
	}
}

// DeletePeer removes the possibility for egress dataplane connections to be routed to a given peer.
func (m *peerManager) DeletePeer(name string) {
	m.logger.Infof("Deleting peer '%s'.", name)

	m.lock.Lock()
	defer m.lock.Unlock()
	delete(m.monitors, name)
}

// Name of the peer monitor runnable.
func (m *peerManager) Name() string {
	return "peerManager"
}

func (m *peerManager) SetStatusCallback(callback func(*v1alpha1.Peer)) {
	m.statusCallback = callback
}

// Start the peer manager.
func (m *peerManager) Start() error {
	m.updaterWG.Add(1)
	defer m.updaterWG.Done()

	for {
		select {
		case <-m.stopCh:
			return nil
		case pr := <-m.statusUpdatesCh:
			// retry loop
			for {
				m.lock.Lock()
				monitor, ok := m.monitors[pr.Name]
				m.lock.Unlock()

				if !ok {
					continue
				}

				currPeer := monitor.Peer()

				if m.statusCallback != nil {
					m.statusCallback(&currPeer)
				} else {
					// CRD-mode
					err := m.client.Status().Update(context.Background(), &currPeer)
					if err != nil {
						m.logger.Warnf("Cannot update peer '%s' status: %v", pr.Name, err)
						continue
					}
				}

				break
			}
		}
	}
}

// Stop the peer manager.
func (m *peerManager) Stop() error {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, monitor := range m.monitors {
		monitor.Stop()
	}
	m.monitorWG.Wait()

	close(m.stopCh)
	m.updaterWG.Wait()

	m.stopped = true
	return nil
}

// GracefulStop does a graceful stop of the peer manager.
func (m *peerManager) GracefulStop() error {
	return m.Stop()
}

func (m *peerManager) queueStatusUpdate(pr *v1alpha1.Peer) {
	m.statusUpdatesCh <- pr
}

func peerChanged(pr1, pr2 *v1alpha1.Peer) bool {
	if len(pr1.Spec.Gateways) != len(pr2.Spec.Gateways) {
		return true
	}

	for i := 0; i < len(pr1.Spec.Gateways); i++ {
		if pr1.Spec.Gateways[i].Host != pr2.Spec.Gateways[i].Host {
			return true
		}
		if pr1.Spec.Gateways[i].Port != pr2.Spec.Gateways[i].Port {
			return true
		}
	}

	return false
}

func newPeerMonitor(pr *v1alpha1.Peer, manager *peerManager) *peerMonitor {
	logger := logrus.WithFields(logrus.Fields{
		"component": "controlplane.control.peerMonitor",
		"peer":      pr.Name,
	})

	monitor := &peerMonitor{
		pr:             pr,
		client:         peer.NewClient(pr, manager.peerTLS.ClientConfig(pr.Name)),
		statusCallback: manager.queueStatusUpdate,
		wg:             &manager.monitorWG,
		stopCh:         make(chan struct{}),
		logger:         logger,
	}

	manager.monitorWG.Add(1)
	go monitor.Start()
	return monitor
}

// newPeerManager returns a new empty peerManager.
func newPeerManager(cl client.Client, peerTLS *tls.ParsedCertData) peerManager {
	logger := logrus.WithField("component", "controlplane.control.peerManager")

	return peerManager{
		client:          cl,
		peerTLS:         peerTLS,
		monitors:        make(map[string]*peerMonitor),
		stopCh:          make(chan struct{}),
		statusUpdatesCh: make(chan *v1alpha1.Peer),
		logger:          logger,
	}
}
