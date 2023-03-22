package cmd

import (
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/pkg/api"
)

// startCmd represents the start command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "A start command set all parameter state of mbgctl (mbg control)",
	Long: `A start command set all parameter state of mbgctl (mbg control)-
			1) The MBG that the mbgctl is connected
			2) The IP of the mbgctl
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		mbgIP, _ := cmd.Flags().GetString("mbgIP")
		caFile, _ := cmd.Flags().GetString("rootCa")
		certificateFile, _ := cmd.Flags().GetString("certificate")
		dataplane, _ := cmd.Flags().GetString("dataplane")
		keyFile, _ := cmd.Flags().GetString("key")

		api.CreateMbgctl(id, mbgIP, caFile, certificateFile, keyFile, dataplane)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().String("id", "", "mbgctl Id")
	createCmd.Flags().String("mbgIP", "", "IP address of the MBG (that the mbgctl is connected)")
	createCmd.Flags().String("rootCa", "", "Path to the Root Certificate Auth File (.pem)")
	createCmd.Flags().String("certificate", "", "Path to the Certificate File (.pem)")
	createCmd.Flags().String("key", "", "Path to the Key File (.pem)")
	createCmd.Flags().String("dataplane", "tcp", "tcp/mtls based data-plane proxies")

}
