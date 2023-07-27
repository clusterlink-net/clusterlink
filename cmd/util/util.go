package util

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// MarkFlagsRequired mark the flags that required for the command line.
func MarkFlagsRequired(cmd *cobra.Command, flags []string) {
	for _, f := range flags {
		if err := cmd.MarkFlagRequired(f); err != nil {
			fmt.Printf("Error marking required flag '%s': %v\n", f, err)
			os.Exit(1)
		}
	}
}
