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

package subcommand

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/cmd/gwctl/config"
	cmdutil "github.com/clusterlink-net/clusterlink/cmd/util"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

// PolicyOptions is the command line options for 'create policy' or 'update policy'.
type policyOptions struct {
	myID       string
	policyFile string
}

// PolicyCreateCmd - create a new policy - TODO update this command after integration.
func PolicyCreateCmd() *cobra.Command {
	o := policyOptions{}
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Create an access policy in the gateway.",
		Long:  `Create an access policy in the gateway.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(false)
		},
	}
	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"policyFile"})

	return cmd
}

// PolicyUpdateCmd - update a policy - TODO update this command after integration.
func PolicyUpdateCmd() *cobra.Command {
	o := policyOptions{}
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Update an access policy in the gateway.",
		Long:  `Update an access policy in the gateway.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(true)
		},
	}
	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"policyFile"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *policyOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.policyFile, "policyFile", "", "File to load access policy from")
}

// run performs the execution of the 'create policy' or 'update policy' subcommand.
func (o *policyOptions) run(isUpdate bool) error {
	policyClient, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	policy, err := policyFromFile(o.policyFile)
	if err != nil {
		return err
	}

	acOperation := policyClient.AccessPolicies.Create
	if isUpdate {
		acOperation = policyClient.AccessPolicies.Update
	}

	err = acOperation(policy)
	if err != nil {
		return err
	}

	return nil
}

func policyFromFile(filename string) (*v1alpha1.AccessPolicy, error) {
	fileBuf, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading policy file: %w", err)
	}
	var policy v1alpha1.AccessPolicy
	err = json.Unmarshal(fileBuf, &policy)
	if err != nil {
		return nil, fmt.Errorf("error parsing Json in policy file: %w", err)
	}

	return &policy, nil
}

// PolicyDeleteOptions is the command line options for 'delete policy'.
type policyDeleteOptions struct {
	myID       string
	policyFile string
}

// PolicyDeleteCmd - delete a policy command - TODO change after the policy integration.
func PolicyDeleteCmd() *cobra.Command {
	o := policyDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Delete service policy from gateway.",
		Long:  `Delete service policy from gateway.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"policyFile"})

	return cmd
}

// addFlags registers flags for the CLI.I.
func (o *policyDeleteOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.policyFile, "policyFile", "", "File to load access policy from")
}

// run performs the execution of the 'delete policy' subcommand.
func (o *policyDeleteOptions) run() error {
	policyClient, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	policy, err := policyFromFile(o.policyFile)
	if err != nil {
		return err
	}
	err = policyClient.AccessPolicies.Delete(types.NamespacedName{
		Name: policy.Name,
	})
	if err != nil {
		return err
	}

	return nil
}

// PolicyGetOptions is the command line options for 'get policy'.
type policyGetOptions struct {
	myID string
}

// PolicyGetCmd - get a policy command.
func PolicyGetCmd() *cobra.Command {
	o := policyGetOptions{}
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Get policy list from the GW",
		Long:  `Get policy list from the GW`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *policyGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
}

// run performs the execution of the 'delete policy' subcommand.
func (o *policyGetOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	accessPolicies, err := g.AccessPolicies.List()
	if err != nil {
		return err
	}

	fmt.Printf("Access policies\n")
	policies := *accessPolicies.(*[]v1alpha1.AccessPolicy)
	for i := 0; i < len(policies); i++ {
		fmt.Printf("Access policy %d: %s\n", i, policies[i].Name)
	}

	return nil
}
