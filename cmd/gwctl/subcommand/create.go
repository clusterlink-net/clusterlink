package subcommand

import (
	"fmt"

	"github.com/spf13/cobra"
	api "github.ibm.com/mbg-agent/pkg/controlplane/api"
)

// startCmd represents the start command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "A start command set all parameter state of gwctl (mbg control)",
	Long: `A start command set all parameter state of gwctl (mbg control)-
			The MBG that the gwctl is connected, BY default the policy engine will be same as MBG ip
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		id, _ := cmd.Flags().GetString("id")
		mbgIP, _ := cmd.Flags().GetString("mbgIP")
		caFile, _ := cmd.Flags().GetString("certca")
		certificateFile, _ := cmd.Flags().GetString("cert")
		dataplane, _ := cmd.Flags().GetString("dataplane")
		keyFile, _ := cmd.Flags().GetString("key")
		//Require gwctl
		if !cmd.Flags().Lookup("id").Changed {
			fmt.Println("The id flag must be set")
			return
		}
		api.CreateGwctl(id, mbgIP, caFile, certificateFile, keyFile, dataplane)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.Flags().String("id", "", "gwctl Id")
	createCmd.Flags().String("mbgIP", "", "IP address of the MBG (that the gwctl is connected)")
	createCmd.Flags().String("certca", "", "Path to the Root Certificate Auth File (.pem)")
	createCmd.Flags().String("cert", "", "Path to the Certificate File (.pem)")
	createCmd.Flags().String("key", "", "Path to the Key File (.pem)")
	createCmd.Flags().String("dataplane", "tcp", "tcp/mtls based data-plane proxies")

}
