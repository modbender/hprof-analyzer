package main

import (
	"fmt"
	"os"

	"github.com/modbender/hprof-analyzer/internal/selfupdate"
	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade hprof-analyzer to the latest release",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Checking for updates...")
			newVersion, err := selfupdate.Upgrade(version)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if newVersion == "" {
				fmt.Println("Already up to date.")
				return
			}
			fmt.Printf("Successfully upgraded to %s\n", newVersion)
		},
	}
}
