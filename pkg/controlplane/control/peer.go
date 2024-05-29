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

package control

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/peer"
	"github.com/clusterlink-net/clusterlink/pkg/util/tls"
)

const (
	// time interval between health check requests when peer has recently responded.
	healthyInterval = 1 * time.Second
	// time interval between health check requests when peer has not recently responded.
	unhealthyInterval = 10 * time.Second
	// number of consecutive successful healthchecks for a peer to be declared reachable.
	healthyThreshold = 3
	// number of consecutive unsuccessful healthchecks for a peer to be declared unreachable.
	unhealthyThreshold = 5
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
	client client.Client

	peerTLSLock sync.RWMutex
	peerTLS     *tls.ParsedCertData

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

func (m *peerMonitor) SetPeer(pr *v1alpha1.Peer) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.pr = pr
}

func (m *peerMonitor) SetClientCertificates(peerTLS *tls.ParsedCertData) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.client = peer.NewClient(m.pr, peerTLS.ClientConfig(m.pr.Name))
}

func (m *peerMonitor) getClient() *peer.Client {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.client
}

func (m *peerMonitor) Start() {
	defer m.wg.Done()

	ticker := time.NewTicker(healthyInterval)
	defer ticker.Stop()

	healthy := meta.IsStatusConditionTrue(m.pr.Status.Conditions, v1alpha1.PeerReachable)
	strikeCount := 0
	threshold := 1 // require a single request on startup
	reachableCond := metav1.Condition{
		Type:   v1alpha1.PeerReachable,
		Reason: "Heartbeat",
	}

	for {
		select {
		case <-m.stopCh:
			return
		default:
			break
		}

		heartbeatOK := m.getClient().GetHeartbeat() == nil
		if healthy == heartbeatOK {
			if !healthy {
				ticker.Reset(unhealthyInterval)
			}
			strikeCount = 0
		} else {
			if heartbeatOK {
				// switch to healthy interval (even though not yet declared healthy)
				ticker.Reset(healthyInterval)
			}
			strikeCount++
		}

		if strikeCount < threshold {
			<-ticker.C
			continue
		}

		m.logger.Infof("Peer reachable status changed to: %v", heartbeatOK)

		if heartbeatOK {
			reachableCond.Status = metav1.ConditionTrue
			threshold = unhealthyThreshold
		} else {
			reachableCond.Status = metav1.ConditionFalse
			threshold = healthyThreshold
			ticker.Reset(unhealthyInterval)
		}

		strikeCount = 0
		healthy = heartbeatOK

		m.lock.Lock()
		meta.SetStatusCondition(&m.pr.Status.Conditions, reachableCond)
		m.lock.Unlock()

		m.statusCallback(m.pr)

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
		if monitor != nil {
			monitor.Stop()
		}
		m.monitors[pr.Name] = newPeerMonitor(pr, m)
	} else {
		monitor.SetPeer(pr)
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

				err := m.client.Status().Update(context.Background(), &currPeer)
				if err != nil {
					m.logger.Warnf("Cannot update peer '%s' status: %v", pr.Name, err)
					continue
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

	if len(pr1.Status.Conditions) != len(pr2.Status.Conditions) {
		return true
	}

	for i := 0; i < len(pr1.Status.Conditions); i++ {
		if pr1.Status.Conditions[i] != pr2.Status.Conditions[i] {
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

func (m *peerManager) SetPeerCertificates(peerTLS *tls.ParsedCertData) {
	m.peerTLSLock.Lock()
	defer m.peerTLSLock.Unlock()

	m.peerTLS = peerTLS

	m.lock.Lock()
	defer m.lock.Unlock()

	for _, mon := range m.monitors {
		mon.SetClientCertificates(peerTLS)
	}
}

// newPeerManager returns a new empty peerManager.
func newPeerManager(cl client.Client) peerManager {
	logger := logrus.WithField("component", "controlplane.control.peerManager")

	return peerManager{
		client:          cl,
		monitors:        make(map[string]*peerMonitor),
		stopCh:          make(chan struct{}),
		statusUpdatesCh: make(chan *v1alpha1.Peer),
		logger:          logger,
	}
}
