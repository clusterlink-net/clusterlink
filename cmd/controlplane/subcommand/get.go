package subcommand

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

func emptyRun(*cobra.Command, []string) {}

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get",
	Long:  `Get`,
	Run:   emptyRun,
}

var logGetCmd = &cobra.Command{
	Use:   "log",
	Short: "Get mbg log file",
	Long:  `Get mbg log file`,
	Run: func(cmd *cobra.Command, args []string) {
		RunCmd("cat /root/.gw/gw.log")
	},
}

var stateGetCmd = &cobra.Command{
	Use:   "state",
	Short: "Get mbg state",
	Long:  `Get mbg state`,
	Run: func(cmd *cobra.Command, args []string) {
		RunCmd("cat /root/.mbg/mbgApp")
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	// Get Log
	getCmd.AddCommand(logGetCmd)
	// Get mbg state
	getCmd.AddCommand(stateGetCmd)
}

func RunCmd(c string) { //Execute command and print in the end the result
	argSplit := strings.Split(c, " ")
	//fmt.Println(argSplit[0], argSplit[1:])
	cmd := exec.Command(argSplit[0], argSplit[1:]...)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(err.Error())
		return
	}

	// Print the output
	fmt.Println(string(stdout))
}
