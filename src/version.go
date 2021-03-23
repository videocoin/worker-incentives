package app

import (
	"fmt"

	"github.com/spf13/cobra"
)

// variable is injected at compile time
var Version = "1.0.0"

func VersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version of the applcation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(Version)
			return nil
		},
	}
}
