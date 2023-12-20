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

package httpecho

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"

	k8serr "k8s.io/apimachinery/pkg/api/errors"

	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/services"
	"github.com/clusterlink-net/clusterlink/tests/e2e/k8s/util"
)

func GetEchoValue(cluster *util.KindCluster, server *util.Service) (string, error) {
	port, err := cluster.ExposeNodeport(server)
	if err != nil {
		if k8serr.IsNotFound(err) {
			return "", &services.ServiceNotFoundError{}
		}
		return "", err
	}

	// fresh client assures a fresh connection
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	url := fmt.Sprintf("http://%s", net.JoinHostPort(cluster.IP(), strconv.Itoa(int(port))))
	resp, err := client.Get(url)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			return "", &services.ConnectionRefusedError{}
		}

		if errors.Is(err, syscall.ECONNRESET) {
			return "", &services.ConnectionResetError{}
		}

		return "", fmt.Errorf("cannot get server response: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read server response: %w", err)
	}

	if err := resp.Body.Close(); err != nil {
		return "", fmt.Errorf("cannot close connection: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tried to access service, got HTTP %d: %s", resp.StatusCode, string(body))
	}

	return strings.TrimSpace(string(body)), nil
}
