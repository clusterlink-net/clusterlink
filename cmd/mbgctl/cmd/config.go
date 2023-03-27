package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.ibm.com/mbg-agent/pkg/api"
)

// startCmd represents the start command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "config",
	Long:  `config`,
	Run:   emptyRun,
}

var getContextCmd = &cobra.Command{
	Use:   "current-context",
	Short: "Get mbgctl current context.",
	Long:  `Get mbgctl current context.`,
	Run: func(cmd *cobra.Command, args []string) {
		m := api.Mbgctl{}
		s, err := m.ConfigCurrentContext()
		if err != nil {
			fmt.Printf("Failed to get current state :%v\n", err)
			return
		}
		sJSON, _ := json.MarshalIndent(s, "", " ")
		fmt.Println("mbgctl current state\n", string(sJSON))
	},
}
var useContextCmd = &cobra.Command{
	Use:   "use-context",
	Short: "use mbgctl context.",
	Long:  `use mbgctl context.`,
	Run: func(cmd *cobra.Command, args []string) {
		mId, _ := cmd.Flags().GetString("myid")
		m := api.Mbgctl{mId}
		err := m.ConfigUseContext()
		if err != nil {
			fmt.Printf("Failed to use context %v: %v\n", mId, err)
		}
		fmt.Println("mbgctl use context ", mId)
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	//current-context
	configCmd.AddCommand(getContextCmd)
	//use-context
	configCmd.AddCommand(useContextCmd)
	useContextCmd.Flags().String("myid", "", "MBGCtl Id")

}
