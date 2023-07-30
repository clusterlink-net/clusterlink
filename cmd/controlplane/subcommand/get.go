package subcommand

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// LogGetCmd prints out the controlplane log
var LogGetCmd = &cobra.Command{
	Use:   "log",
	Short: "Get mbg log file",
	Long:  `Get mbg log file`,
	Run: func(cmd *cobra.Command, args []string) {
		RunCmd("cat /root/.gw/gw.log")
	},
}

// StateGetCmd prints out the controlplane state
var StateGetCmd = &cobra.Command{
	Use:   "state",
	Short: "Get mbg state",
	Long:  `Get mbg state`,
	Run: func(cmd *cobra.Command, args []string) {
		RunCmd("cat /root/.gw/gwApp")
	},
}

// RunCmd executes os cmd and print the output
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
