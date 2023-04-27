package cmd

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/cmd/mbg/state"
	api "github.ibm.com/mbg-agent/pkg/api"
)

// / startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A start command set all parameter state of the Multi-cloud Border Gateway",
	Long: `A start command set all parameter state of the MBg-
			The  id, IP cport(Cntrol port for grpc) and localDataPortRange,externalDataPortRange
			TBD now is done manually need to call some external `,
	Run: func(cmd *cobra.Command, args []string) {
		ip, _ := cmd.Flags().GetString("ip")
		id, _ := cmd.Flags().GetString("id")
		cportLocal, _ := cmd.Flags().GetString("cportLocal")
		cport, _ := cmd.Flags().GetString("cport")
		localDataPortRange, _ := cmd.Flags().GetString("localDataPortRange")
		externalDataPortRange, _ := cmd.Flags().GetString("externalDataPortRange")
		caFile, _ := cmd.Flags().GetString("rootCa")
		certificateFile, _ := cmd.Flags().GetString("certificate")
		keyFile, _ := cmd.Flags().GetString("key")
		dataplane, _ := cmd.Flags().GetString("dataplane")
		startPolicyEngine, _ := cmd.Flags().GetBool("startPolicyEngine")
		policyEngineTarget, _ := cmd.Flags().GetString("policyEngineIp")
		restore, _ := cmd.Flags().GetBool("restore")
		logFile, _ := cmd.Flags().GetBool("logFile")
		logLevel, _ := cmd.Flags().GetString("logLevel")
		if ip == "" || id == "" || cport == "" {
			fmt.Println("Error: please insert all flag arguments for Mbg start command")
			os.Exit(1)
		}
		var m api.Mbg
		var err error
		if restore {
			if startPolicyEngine && policyEngineTarget == "" {
				fmt.Println("Error: Please specify policyEngineTarget")
				os.Exit(1)
			}
			m, _ = api.RestoreMbg(id, policyEngineTarget, logLevel, logFile, startPolicyEngine)
			log.Infof("Restoring MBG")
			state.PrintState()
			m.StartMbg()
		}

		m, err = api.CreateMbg(id, ip, cportLocal, cport, localDataPortRange, externalDataPortRange, dataplane,
			caFile, certificateFile, keyFile, logLevel, logFile, restore)
		if err != nil {
			fmt.Println("Error: Unable to create MBG: ", err)
			os.Exit(1)
		}

		if startPolicyEngine {
			m.AddPolicyEngine("localhost:"+cportLocal, true)
		}

		state.PrintState()

		m.StartMbg()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("id", "", "Multi-cloud Border Gateway id")
	startCmd.Flags().String("ip", "", "Multi-cloud Border Gateway ip")
	startCmd.Flags().String("cportLocal", "8443", "Multi-cloud Border Gateway control local port inside the MBG")
	startCmd.Flags().String("cport", "8443", "Multi-cloud Border Gateway control external port for the MBG neighbors ")
	startCmd.Flags().String("localDataPortRange", "5000", "Set the port range for data connection in the MBG")
	startCmd.Flags().String("externalDataPortRange", "30000", "Set the port range for exposing data connection (each expose port connect to localDataPort")
	startCmd.Flags().String("rootCa", "", "Path to the Root Certificate Auth File (.pem)")
	startCmd.Flags().String("certificate", "", "Path to the Certificate File (.pem)")
	startCmd.Flags().String("key", "", "Path to the Key File (.pem)")
	startCmd.Flags().String("dataplane", "mtls", "tcp/mtls based data-plane proxies")
	startCmd.Flags().Bool("startPolicyEngine", true, "Start policy engine in port")
	startCmd.Flags().String("policyEngineIp", "", "Set the policy engine ip")
	startCmd.Flags().Bool("restore", false, "Restore existing stored MBG states")
	startCmd.Flags().Bool("logFile", true, "Save the outputs to file")
	startCmd.Flags().String("logLevel", "info", "Log level: debug, info, warning, error")

}
