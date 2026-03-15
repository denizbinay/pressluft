package main

import (
	"github.com/spf13/cobra"

	"pressluft/internal/cli/validate"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Run the full project validation pipeline",
	Long:  "Run gofmt, go vet, go tests, profile tests, Ansible syntax check, and frontend generation.",
	RunE:  runValidate,
}

func runValidate(cmd *cobra.Command, args []string) error {
	return validate.Run()
}
