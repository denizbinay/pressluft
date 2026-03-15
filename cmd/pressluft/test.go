package main

import (
	"github.com/spf13/cobra"

	"pressluft/internal/cli/test"
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run Go tests",
	Long:  "Run Go tests for the project.",
	RunE:  runTest,
}

func runTest(cmd *cobra.Command, args []string) error {
	return test.Run()
}
