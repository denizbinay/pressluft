package main

import (
	"os"

	"github.com/spf13/cobra"

	"pressluft/internal/cliui"
)

var rootCmd = &cobra.Command{
	Use:           "pressluft",
	Short:         "CLI for the Pressluft hosting panel",
	Long:          "pressluft is the development and operations CLI for the Pressluft hosting panel.",
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(testCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(serverSSHCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		cliui.Errf("%v", err)
		os.Exit(1)
	}
}
