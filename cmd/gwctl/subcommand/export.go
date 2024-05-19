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

// Copyright (c) 2022 The ClusterLink Authors.
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

// Copyright (C) The ClusterLink Authors.
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
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/clusterlink-net/clusterlink/cmd/gwctl/config"
	cmdutil "github.com/clusterlink-net/clusterlink/cmd/util"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
)

// exportCreateOptions is the command line options for 'create export' or 'update export'.
type exportCreateOptions struct {
	myID     string
	name     string
	host     string
	port     uint16
	external string
}

// ExportCreateCmd - Create an exported service.
func ExportCreateCmd() *cobra.Command {
	o := exportCreateOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Create an exported service",
		Long:  `Create an exported service that can be accessed by other peers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(false)
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "port"})
	return cmd
}

// ExportUpdateCmd - Update an exported service.
func ExportUpdateCmd() *cobra.Command {
	o := exportCreateOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Update an exported service",
		Long:  `Update an exported service that can be accessed by other peers`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run(true)
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name", "port"})
	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportCreateOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name")
	fs.StringVar(&o.host, "host", "", "Exported service endpoint hostname (IP/DNS), if unspecified, uses the service name")
	fs.Uint16Var(&o.port, "port", 0, "Exported service port")
	fs.StringVar(&o.external, "external", "",
		"External endpoint <host>:<port, which the exported service will be connected")
}

// run performs the execution of the 'create export' or 'update export' subcommand.
func (o *exportCreateOptions) run(isUpdate bool) error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}
	exportOperation := g.Exports.Create
	if isUpdate {
		exportOperation = g.Exports.Update
	}

	err = exportOperation(&v1alpha1.Export{
		ObjectMeta: metav1.ObjectMeta{
			Name: o.name,
		},
		Spec: v1alpha1.ExportSpec{
			Host: o.host,
			Port: o.port,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// exportDeleteOptions is the command line options for 'delete export'.
type exportDeleteOptions struct {
	myID string
	name string
}

// ExportDeleteCmd - delete an exported service command.
func ExportDeleteCmd() *cobra.Command {
	o := exportDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Delete an exported service",
		Long:  `Delete an exported service`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())
	cmdutil.MarkFlagsRequired(cmd, []string{"name"})

	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportDeleteOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name")
}

// run performs the execution of the 'delete export' subcommand.
func (o *exportDeleteOptions) run() error {
	g, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	err = g.Exports.Delete(o.name)
	if err != nil {
		return err
	}

	return nil
}

// exportGetOptions is the command line options for 'get export'.
type exportGetOptions struct {
	myID string
	name string
}

// ExportGetCmd - get an exported service command.
func ExportGetCmd() *cobra.Command {
	o := exportGetOptions{}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Get an exported service list",
		Long:  `Get an exported service list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.run()
		},
	}

	o.addFlags(cmd.Flags())

	return cmd
}

// addFlags registers flags for the CLI.
func (o *exportGetOptions) addFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.myID, "myid", "", "gwctl ID")
	fs.StringVar(&o.name, "name", "", "Exported service name. If empty gets all exported services.")
}

// run performs the execution of the 'get export' subcommand.
func (o *exportGetOptions) run() error {
	exportClient, err := config.GetClientFromID(o.myID)
	if err != nil {
		return err
	}

	if o.name == "" {
		sArr, err := exportClient.Exports.List()
		if err != nil {
			return err
		}

		exports, ok := sArr.(*[]v1alpha1.Export)
		if !ok {
			return fmt.Errorf("cannot decode exports list")
		}

		fmt.Printf("Exported services:\n")
		for i := range *exports {
			export := &(*exports)[i]
			fmt.Printf(
				"%d. Service Name: %s. Host: %s. Port: %d\n",
				i+1, export.Name, export.Spec.Host, export.Spec.Port)
		}
	} else {
		s, err := exportClient.Exports.Get(o.name)
		if err != nil {
			return err
		}
		fmt.Printf("Exported service :%+v\n", s.(*v1alpha1.Export))
	}

	return nil
}
