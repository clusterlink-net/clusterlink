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

package httpecho

import (
	"context"
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

const EchoClientPodName = "echo-client"

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
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		// handle error
		return "", fmt.Errorf("cannot create request: %w", err)
	}

	resp, err := client.Do(req)
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

func RunClientInPod(cluster *util.KindCluster, server *util.Service) (string, error) {
	url := "http://" + server.Name
	body, err := cluster.RunPod(&util.Pod{
		Name:      EchoClientPodName,
		Namespace: server.Namespace,
		Image:     "curlimages/curl",
		Args:      []string{"curl", "-s", "-m", "10", "--retry", "10", "--retry-delay", "1", "--retry-all-errors", url},
	})
	return strings.TrimSpace(body), err
}
