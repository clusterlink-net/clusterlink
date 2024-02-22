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

package k8s

import (
	"context"

	logrusr "github.com/bombsimon/logrusr/v4"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// Platform represents a k8s platform.
type Platform struct {
	podReconciler *PodReconciler
	client        client.Client
	logger        *logrus.Entry
}

// GetLabelsFromIP return all the labels for specific ip.
func (p *Platform) GetLabelsFromIP(ip string) map[string]string {
	return p.podReconciler.GetLabelsFromIP(ip)
}

// NewPlatform returns a new Kubernetes platform.
func NewPlatform() (*Platform, error) {
	logger := logrus.WithField("component", "platform.k8s")
	ctrl.SetLogger(logrusr.New(logrus.WithField("component", "k8s.controller-runtime")))

	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	manager, err := ctrl.NewManager(cfg, ctrl.Options{Metrics: metricsserver.Options{BindAddress: "0"}})
	if err != nil {
		return nil, err
	}
	podReconciler, err := NewPodReconciler(manager)
	if err != nil {
		return nil, err
	}

	err = ctrl.NewControllerManagedBy(manager).
		For(&corev1.Pod{}).
		Complete(podReconciler)
	if err != nil {
		return nil, err
	}

	// Start manger and all the controllers.
	go func() {
		if err := manager.Start(context.Background()); err != nil {
			logger.Error(err, "problem running manager")
		}
	}()

	return &Platform{
		client:        manager.GetClient(),
		podReconciler: podReconciler,
		logger:        logger,
	}, nil
}
