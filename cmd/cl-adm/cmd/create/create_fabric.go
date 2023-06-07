package create

import (
	"github.com/spf13/cobra"

	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/config"
	"github.com/clusterlink-net/clusterlink/cmd/cl-adm/util"
)

// NewCmdCreateFabric returns a cobra.Command to run the 'create fabric' subcommand.
func NewCmdCreateFabric() *cobra.Command {
	return &cobra.Command{
		Use:   "fabric",
		Short: "Create a fabric",
		Long:  `Create a fabric`,

		RunE: func(cmd *cobra.Command, args []string) error {
			return util.CreateCertificate(&util.CertificateConfig{
				Name:              "root",
				IsCA:              true,
				CertOutPath:       config.CertificateFileName,
				PrivateKeyOutPath: config.PrivateKeyFileName,
			})
		},
	}
}
