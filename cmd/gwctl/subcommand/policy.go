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
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
)

// PolicyOptions is the command line options for 'create policy' or 'update policy'.
type policyOptions struct {
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
	o := policyOptions{}
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Create a policy in the gateway",
		Long:  `Create a load-balancing policy or an access policy in the gateway.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(false)
		},
	}
	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"type"})

	return cmd
}

// PolicyUpdateCmd - update a policy - TODO update this command after integration.
func PolicyUpdateCmd() *cobra.Command {
	o := policyOptions{}
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Update a policy in the gateway",
		Long:  `Update a load-balancing policy or an access policy in the gateway.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(true)
		},
	}
	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"type"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *policyOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.pType, "type", "", "Policy agent command (For now: lb, access)")
	fs.StringVar(&o.serviceSrc, "serviceSrc", "*", "Name of Source Service (* for wildcard)")
	fs.StringVar(&o.serviceDst, "serviceDst", "*", "Name of Dest Service (* for wildcard)")
	fs.StringVar(&o.gwDest, "gwDest", "*", "Name of gateway the dest service belongs to (* for wildcard)")
	fs.StringVar(&o.policy, "policy", "random", "lb policy: random, ecmp, static")
	fs.StringVar(&o.policyFile, "policyFile", "", "File to load access policy from")
}

// run performs the execution of the 'create policy' or 'update policy' subcommand.
func (o *policyOptions) run(isUpdate bool) error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}
	switch o.pType {
	case policyengine.LbType:
		policy, err := lbPolicyFromParams(o.serviceSrc, o.serviceDst, o.policy, o.gwDest)
		if err != nil {
			return err
		}

		lbOperation := g.LBPolicies.Create
		if isUpdate {
			lbOperation = g.LBPolicies.Update
		}

		err = lbOperation(policy)
		if err != nil {
			return err
		}

		return nil

	case policyengine.AccessType:
		policy, err := policyFromFile(o.policyFile)
		if err != nil {
			return err
		}

		acOperation := g.AccessPolicies.Create
		if isUpdate {
			acOperation = g.AccessPolicies.Update
		}

		err = acOperation(policy)
		if err != nil {
			return err
		}

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

func lbPolicyFromParams(serviceSrc, serviceDst, scheme, defaultPeer string) (api.Policy, error) {
	lbPolicy := policyengine.LBPolicy{
		ServiceSrc:  serviceSrc,
		ServiceDst:  serviceDst,
		Scheme:      policyengine.LBScheme(scheme),
		DefaultPeer: defaultPeer,
	}
	blob, err := json.Marshal(lbPolicy)
	if err != nil {
		return api.Policy{}, fmt.Errorf("error marshaling a load-balancing policy: %w", err)
	}
	policyName := fmt.Sprintf("%s%%%s", serviceSrc, serviceDst)
	return api.Policy{Name: policyName, Spec: api.PolicySpec{Blob: blob}}, nil
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
		policy, err := lbPolicyFromParams(o.serviceSrc, o.serviceDst, o.policy, o.gwDest)
		if err != nil {
			return err
		}
		err = g.LBPolicies.Delete(policy)
		if err != nil {
			return err
		}

		return nil

	case policyengine.AccessType:
		policy, err := policyFromFile(o.policyFile)
		if err != nil {
			return err
		}
		err = g.AccessPolicies.Delete(policy)
		if err != nil {
			return err
		}

		return nil
	default:
		return fmt.Errorf("unknown policy type")
	}
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

	accessPolicies, err := g.AccessPolicies.List()
	if err != nil {
		return err
	}

	fmt.Printf("Access policies\n")
	for d, policy := range *accessPolicies.(*[]api.Policy) {
		fmt.Printf("Access policy %d: %s\n", d, policy.Name)
	}

	lbPolicies, err := g.LBPolicies.List()
	if err != nil {
		return err
	}

	fmt.Printf("\nLoad balancing policies\n")
	for d, policy := range *lbPolicies.(*[]api.Policy) {
		fmt.Printf("Load-balancing policy %d: %s\n", d, policy.Name)
	}

	return nil
}
