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
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/stretchr/testify/suite"
	v1 "k8s.io/api/core/v1"

	"github.com/clusterlink-net/clusterlink/pkg/bootstrap/platform"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

const (
	clusterCount     = 3
	exportedLogsPath = "/tmp/clusterlink-k8s-tests"
)

var images = [...]string{"cl-controlplane", "cl-dataplane", "cl-go-dataplane"}

var iperfService = util.Service{
	Name:      "iperf3-server",
	Namespace: v1.NamespaceDefault,
	Port:      5201,
}

var httpEchoService = util.Service{
	Name:      "http-echo",
	Namespace: v1.NamespaceDefault,
	Port:      8080,
}

func httpEchoPodAndService(echoValue string) *util.PodAndService {
	return &util.PodAndService{
		Service: httpEchoService,
		Image:   "hashicorp/http-echo",
		Args:    []string{"-listen=:8080", "-text=" + echoValue},
	}
}

// TestSuite is a suite for e2e testing on k8s clusters.
type TestSuite struct {
	suite.Suite
	fabric   *util.Fabric
	clusters []*util.KindCluster
}

// SetupSuite creates the k8s kind clusters and the clusterlink certificates.
func (s *TestSuite) SetupSuite() {
	fabric, err := util.NewFabric()
	if err != nil {
		s.T().Fatal(err)
	}

	// create clusters and fabric
	s.fabric = fabric
	s.clusters = make([]*util.KindCluster, clusterCount)
	for i := 0; i < clusterCount; i++ {
		s.clusters[i] = util.NewKindCluster(fmt.Sprintf("peer%d", i))
		fabric.CreatePeer(s.clusters[i])

		s.clusters[i].Start()
		for _, image := range images {
			s.clusters[i].LoadImage(image)
		}
	}

	// prepare logs directory
	if err := os.RemoveAll(exportedLogsPath); err != nil {
		s.T().Fatal(fmt.Errorf("cannot cleanup logs directory: %w", err))
	}
	if err := os.MkdirAll(exportedLogsPath, 0755); err != nil {
		s.T().Fatal(fmt.Errorf("cannot create logs directory: %w", err))
	}

	// wait for clusters
	for i := 0; i < clusterCount; i++ {
		if err := s.clusters[i].Wait(); err != nil {
			s.T().Fatal(err)
		}

		// create http-echo service which echoes the cluster name
		err := s.clusters[i].CreatePodAndService(httpEchoPodAndService(s.clusters[i].Name()))
		if err != nil {
			s.T().Fatal(fmt.Errorf("cannot create http-echo service: %w", err))
		}

		// create iperf3 server service
		err = s.clusters[i].CreatePodAndService(&util.PodAndService{
			Service: iperfService,
			Image:   "networkstatic/iperf3",
			Args:    []string{"-s", "-p", "5201"},
		})
		if err != nil {
			s.T().Fatal(fmt.Errorf("cannot create iperf3-server service: %w", err))
		}
	}

	// wait for fabric
	if err := fabric.Wait(); err != nil {
		s.T().Fatal(err)
	}
}

// TearDownSuite deletes the k8s kind clusters.
func (s *TestSuite) TearDownSuite() {
	for _, cluster := range s.clusters {
		if err := cluster.Destroy(); err != nil {
			s.T().Fatal(err)
		}
	}
}

// convert e.g. TestBlaBla to test-bla-bla
func convertCaseCamelToKebab(s string) string {
	s = regexp.MustCompile("(.)([A-Z][a-z]+)").ReplaceAllString(s, "${1}-${2}")
	s = regexp.MustCompile("([a-z0-9])([A-Z])").ReplaceAllString(s, "${1}-${2}")
	return strings.ToLower(s)
}

// BeforeTest creates the test namespace before each test, and removes the previous test namespace.
func (s *TestSuite) BeforeTest(_, testName string) {
	testName = convertCaseCamelToKebab(testName)
	if err := s.fabric.SwitchToNewNamespace(testName, false); err != nil {
		s.T().Fatal(err)
	}
}

func (s *TestSuite) exportLogs() {
	var runner util.AsyncRunner
	for _, cluster := range s.clusters {
		runner.Run(func(cluster *util.KindCluster) func() error {
			return func() error {
				return cluster.ExportLogs(exportedLogsPath)
			}
		}(cluster))
	}

	if err := runner.Wait(); err != nil {
		s.T().Fatal(err)
	}
}

// AfterTest exports logs after each test.
func (s *TestSuite) AfterTest(_, _ string) {
	s.exportLogs()
}

// RunSubTest creates the test namespace before each subtest, runs the subtest, and finally export logs.
func (s *TestSuite) RunSubTest(subTestName string, subtest func()) bool {
	subTestName = convertCaseCamelToKebab(subTestName)
	if err := s.fabric.SwitchToNewNamespace(subTestName, true); err != nil {
		s.T().Fatal(err)
	}

	ret := s.Run(subTestName, subtest)
	s.exportLogs()
	return ret
}

// RunOnAllDataplaneTypes runs the given test function on all dataplane types (envoy / go).
func (s *TestSuite) RunOnAllDataplaneTypes(test func(cfg *util.PeerConfig)) {
	// DataplaneTypeConfigs holds a single simple configuration per each dataplane type.
	dataplaneTypes := []string{platform.DataplaneTypeEnvoy, platform.DataplaneTypeGo}

	for _, dataplaneType := range dataplaneTypes {
		s.RunSubTest(dataplaneType, func() {
			test(&util.PeerConfig{
				DataplaneType: dataplaneType,
				Dataplanes:    1,
			})
		})
	}
}
