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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Restart coredns deployment.
func restartCoreDNS(ctx context.Context, mClient client.Client, logger *logrus.Entry) error {
	logger.Infof("restarting coredns deployment")
	patch := []byte(
		fmt.Sprintf(
			`{"spec": {"template": {"metadata": {"annotations":{"kubectl.kubernetes.io/restartedAt": %q}}}}}`,
			time.Now().String(),
		),
	)

	return mClient.Patch(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kube-system",
			Name:      "coredns",
		},
	}, client.RawPatch(types.StrategicMergePatchType, patch))
}

// Add coredns rewrite for a given external dns service.
func addCoreDNSRewrite(ctx context.Context, mClient client.Client, logger *logrus.Entry, name *types.NamespacedName,
	alias string,
) error {
	corednsName := types.NamespacedName{
		Name:      "coredns",
		Namespace: "kube-system",
	}
	var cm v1.ConfigMap

	if err := mClient.Get(ctx, corednsName, &cm); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Warnf("coredns configmap not found.")
			return nil
		}
		return err
	}
	if data, ok := cm.Data["Corefile"]; ok {
		// remove trailing end-of-line
		data := strings.TrimSuffix(data, "\n")
		// break into lines
		lines := strings.Split(data, "\n")
		serviceFqdn := fmt.Sprintf("%s.%s.svc.cluster.local", name.Name, name.Namespace)

		coreFileUpdated := false
		rewriteLine := ""
		for i, line := range lines {
			if strings.Contains(line, serviceFqdn) {
				// matched line already exists
				break
			}
			// ready marker is reached - matched line not found, append it here
			if strings.Contains(line, "    ready") {
				if strings.HasPrefix(alias, "*.") { // wildcard DNS
					alias = strings.TrimPrefix(alias, "*")
					alias = strings.ReplaceAll(alias, ".", "\\.")
					alias = "(.*)" + alias

					rewriteLine = fmt.Sprintf("    rewrite name regex %s %s answer auto", alias, serviceFqdn)
				} else {
					rewriteLine = fmt.Sprintf("    rewrite name %s %s", alias, serviceFqdn)
				}
				// add matched line
				lines = append(lines[:i+1], lines[i:]...)
				lines[i] = rewriteLine
				coreFileUpdated = true
				break
			}
		}

		if coreFileUpdated {
			// update configmap and restart the pods
			var newLines string
			for _, line := range lines {
				// return back EOL
				newLines += (line + "\n")
			}
			cm.Data["Corefile"] = newLines
			if err := mClient.Update(ctx, &cm); err != nil {
				return err
			}

			if err := restartCoreDNS(ctx, mClient, logger); err != nil {
				return err
			}
		}
	} else {
		return errors.New("coredns configmap['Corefile'] not found")
	}

	return nil
}

// Remove coredns rewrite for a given external dns service.
func removeCoreDNSRewrite(ctx context.Context, mClient client.Client, logger *logrus.Entry, name *types.NamespacedName) error {
	corednsName := types.NamespacedName{
		Name:      "coredns",
		Namespace: "kube-system",
	}
	var cm v1.ConfigMap

	if err := mClient.Get(ctx, corednsName, &cm); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Warnf("coredns configmap not found.")
			return nil
		}
		return err
	}
	if data, ok := cm.Data["Corefile"]; ok {
		// remove trailing end-of-line
		dataEol := strings.TrimSuffix(data, "\n")
		// break into lines
		lines := strings.Split(dataEol, "\n")
		serviceFqdn := fmt.Sprintf("%s.%s.svc.cluster.local", name.Name, name.Namespace)

		coreFileUpdated := false
		for i, line := range lines {
			if strings.Contains(line, serviceFqdn) {
				// remove matched line
				lines = append(lines[:i], lines[i+1:]...)
				coreFileUpdated = true
				break
			}
		}

		if coreFileUpdated {
			// update configmap and restart the pods
			var newLines string
			for _, line := range lines {
				// return back EOL
				newLines += (line + "\n")
			}
			cm.Data["Corefile"] = newLines
			if err := mClient.Update(ctx, &cm); err != nil {
				return err
			}

			if err := restartCoreDNS(ctx, mClient, logger); err != nil {
				return err
			}
		}
	} else {
		return errors.New("coredns configmap['Corefile'] not found")
	}

	return nil
}
