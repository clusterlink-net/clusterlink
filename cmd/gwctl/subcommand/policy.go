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

	"github.com/clusterlink-net/clusterlink/cmd/gwctl/config"
	cmdutil "github.com/clusterlink-net/clusterlink/cmd/util"
	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/client"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
)

// PolicyCreateOptions is the command line options for 'create policy'
type policyCreateOptions struct {
	myID       string
	pType      string
	serviceSrc string
	serviceDst string
	gwDest     string
	policy     string
	policyFile string
}

// PolicyCreateCmd - create a new policy - TODO update this command after integration.
func PolicyCreateCmd() *cobra.Command {
	o := policyCreateOptions{}
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Create/replace a policy in the gateway",
		Long:  `Create/replace a load-balancing policy or an access policy in the gateway.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}
	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"type"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *policyCreateOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.pType, "type", "", "Policy agent command (For now: lb, access)")
	fs.StringVar(&o.serviceSrc, "serviceSrc", "*", "Name of Source Service (* for wildcard)")
	fs.StringVar(&o.serviceDst, "serviceDst", "*", "Name of Dest Service (* for wildcard)")
	fs.StringVar(&o.gwDest, "gwDest", "*", "Name of gateway the dest service belongs to (* for wildcard)")
	fs.StringVar(&o.policy, "policy", "random", "lb policy: random, ecmp, static")
	fs.StringVar(&o.policyFile, "policyFile", "", "File to load access policy from")
}

// run performs the execution of the 'create policy' subcommand
func (o *policyCreateOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}
	switch o.pType {
	case policyengine.LbType:
		return g.SendLBPolicy(o.serviceSrc, o.serviceDst, policyengine.LBScheme(o.policy), o.gwDest, client.Add)
	case policyengine.AccessType:
		policy, err := policyFromFile(o.policyFile)
		if err != nil {
			return err
		}
		err = g.Policies.Create(policy)
		if err != nil {
			return err
		}

		fmt.Printf("Access Policy %s created successfully\n", policy.Name)
		return nil

	default:
		return fmt.Errorf("unknown policy type")
	}
}

func policyFromFile(filename string) (api.Policy, error) {
	fileBuf, err := os.ReadFile(filename)
	if err != nil {
		return api.Policy{}, fmt.Errorf("error reading policy file: %w", err)
	}
	var policy api.Policy
	err = json.Unmarshal(fileBuf, &policy)
	if err != nil {
		return api.Policy{}, fmt.Errorf("error parsing Json in policy file: %w", err)
	}
	policy.Spec.Blob = fileBuf
	return policy, nil
}

// PolicyDeleteOptions is the command line options for 'delete policy'
type policyDeleteOptions struct {
	myID       string
	pType      string
	serviceSrc string
	serviceDst string
	gwDest     string
	policy     string
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
	cmdutil.MarkFlagsRequired(cmd, []string{"type"})

	return cmd
}

// addFlags registers flags for the CLI.I.
func (o *policyDeleteOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.pType, "type", "", "Policy agent command (For now: lb, access)")
	fs.StringVar(&o.serviceSrc, "serviceSrc", "*", "Name of Source Service (* for wildcard)")
	fs.StringVar(&o.serviceDst, "serviceDst", "*", "Name of Dest Service (* for wildcard)")
	fs.StringVar(&o.gwDest, "gwDest", "*", "Name of gateway the dest service belongs to (* for wildcard)")
	fs.StringVar(&o.policy, "policy", "random", "lb policy: random, ecmp, static")
	fs.StringVar(&o.policyFile, "policyFile", "", "File to load access policy from")
}

// run performs the execution of the 'delete policy' subcommand
func (o *policyDeleteOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}
	switch o.pType {
	case policyengine.LbType:
		err = g.SendLBPolicy(o.serviceSrc, o.serviceDst, policyengine.LBScheme(o.policy), o.gwDest, client.Del)
	case policyengine.AccessType:
		policy, err := policyFromFile(o.policyFile)
		if err != nil {
			return err
		}
		err = g.Policies.Delete(policy)
		if err != nil {
			return err
		}

		fmt.Printf("Access policy %s was deleted successfully\n", policy.Name)
		return nil
	default:
		return fmt.Errorf("unknown policy type")
	}
	return err
}

// PolicyGetOptions is the command line options for 'get policy'
type policyGetOptions struct {
	myID string
}

// PolicyGetCmd - get a policy command
func PolicyGetCmd() *cobra.Command {
	o := policyGetOptions{}
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Get policy list from the GW",
		Long:  `Get policy list from the GW (Access and Load-Balancing)`,
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

// run performs the execution of the 'delete policy' subcommand
func (o *policyGetOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	// TODO: Get Load-balancing policies

	accessPolicies, err := g.Policies.List()
	if err != nil {
		return err
	}

	fmt.Printf("Access policies\n")
	for d, policy := range *accessPolicies.(*[]api.Policy) {
		fmt.Printf("Access policy %d: %s\n", d, policy.Name)
	}
	return nil
}
