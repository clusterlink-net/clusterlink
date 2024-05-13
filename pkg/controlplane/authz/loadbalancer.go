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

package authz

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"

	crds "github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

type importState struct {
	roundRobinCounter atomic.Uint32
}

type LoadBalancer struct {
	lock   sync.RWMutex
	states map[types.NamespacedName]*importState

	logger *logrus.Entry
}

type LoadBalancingResult struct {
	imp          *crds.Import
	currentIndex int
	failed       map[int]interface{}
	delayed      []int
}

func (r *LoadBalancingResult) Get() *crds.ImportSource {
	if r.currentIndex == -1 {
		return nil
	}
	return &r.imp.Spec.Sources[r.currentIndex]
}

func (r *LoadBalancingResult) IsDelayed() bool {
	return len(r.imp.Spec.Sources) == len(r.failed)
}

func (r *LoadBalancingResult) Delay() {
	r.delayed = append(r.delayed, r.currentIndex)
}

func NewLoadBalancingResult(imp *crds.Import) *LoadBalancingResult {
	return &LoadBalancingResult{
		imp:          imp,
		currentIndex: -1,
		failed:       make(map[int]interface{}),
	}
}

// NewLoadBalancer returns a new instance of a LoadBalancer object.
func NewLoadBalancer() *LoadBalancer {
	logger := logrus.WithField("component", "controlplane.authz.loadbalancer")

	return &LoadBalancer{
		states: make(map[types.NamespacedName]*importState),
		logger: logger,
	}
}

func (lb *LoadBalancer) selectRandom(result *LoadBalancingResult) {
	sources := &result.imp.Spec.Sources
	candidateCount := len(*sources)
	index := rand.Intn(candidateCount) //nolint:gosec // G404: use of weak random is fine for load balancing
	for i := 0; i < candidateCount; i++ {
		if _, ok := result.failed[index]; !ok {
			result.currentIndex = index
			return
		}

		index++
		if index == candidateCount {
			index = 0
		}
	}
}

func (lb *LoadBalancer) selectRoundRobin(result *LoadBalancingResult) {
	imp := result.imp
	sourceCount := len(imp.Spec.Sources)

	name := types.NamespacedName{
		Namespace: imp.Namespace,
		Name:      imp.Name,
	}

	lb.lock.RLock()
	state := lb.states[name]
	lb.lock.RUnlock()

	if state == nil {
		lb.lock.Lock()
		state = lb.states[name]
		if state == nil {
			state = &importState{}
			lb.states[name] = state
		}
		lb.lock.Unlock()
	}

	counter := state.roundRobinCounter.Add(1)

	if result.currentIndex != -1 {
		result.currentIndex++
		if result.currentIndex == sourceCount {
			result.currentIndex = 0
		}
		return
	}

	result.currentIndex = int(counter) % sourceCount
}

func (lb *LoadBalancer) selectStatic(result *LoadBalancingResult) {
	result.currentIndex++
}

// Select one of the import sources, based on the set load balancing scheme.
func (lb *LoadBalancer) Select(result *LoadBalancingResult) error {
	if result.currentIndex != -1 {
		result.failed[result.currentIndex] = nil
	}

	imp := result.imp
	sources := &imp.Spec.Sources
	if len(result.failed) == len(*sources) {
		if len(result.delayed) > 0 {
			result.currentIndex = result.delayed[0]
			result.delayed = result.delayed[1:]

			lb.logger.WithFields(logrus.Fields{
				"import-name":      imp.Name,
				"import-namespace": imp.Namespace,
				"result-index":     result.currentIndex,
			}).Info("Select delayed")
			return nil
		}
		return fmt.Errorf("tried out all %d sources", len(imp.Spec.Sources))
	}

	scheme := getScheme(imp)
	switch scheme {
	case crds.LBSchemeRandom:
		lb.selectRandom(result)
	case crds.LBSchemeRoundRobin:
		lb.selectRoundRobin(result)
	case crds.LBSchemeStatic:
		lb.selectStatic(result)
	}

	lb.logger.WithFields(logrus.Fields{
		"import-name":      imp.Name,
		"import-namespace": imp.Namespace,
		"scheme":           scheme,
		"attempt":          len(result.failed),
		"result-index":     result.currentIndex,
	}).Info("Select")

	return nil
}

func getScheme(imp *crds.Import) crds.LBScheme {
	if imp.Spec.LBScheme == "" {
		return crds.LBSchemeDefault
	}

	return imp.Spec.LBScheme
}
