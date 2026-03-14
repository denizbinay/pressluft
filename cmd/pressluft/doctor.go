package main

import (
	"fmt"
	"os"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"pressluft/internal/cliui"
	"pressluft/internal/devdiag"
	"pressluft/internal/envconfig"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system health",
	Long:  "Run diagnostic checks against the local Pressluft dev state and report results.",
	RunE:  runDoctor,
}

var doctorJSON bool

func init() {
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results as JSON")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	runtime, err := resolveRuntime()
	if err != nil {
		return fmt.Errorf("resolve runtime: %w", err)
	}

	report := devdiag.Inspect(runtime)

	if doctorJSON {
		data, err := report.JSON()
		if err != nil {
			return fmt.Errorf("marshal report: %w", err)
		}
		os.Stdout.Write(data)
		fmt.Println()
	} else {
		printDoctorReport(report)
	}

	if !report.Healthy() {
		if !doctorJSON {
			lipgloss.Println()
			cliui.Issues(report.Issues())
			lipgloss.Println()
			cliui.Hint("To reset local state: rm -rf .pressluft")
		}
		os.Exit(1)
	}
	return nil
}

func resolveRuntime() (envconfig.ControlPlaneRuntime, error) {
	cwd, _ := os.Getwd()
	return envconfig.ResolveControlPlaneRuntime(true, cwd)
}

func printDoctorReport(report devdiag.Report) {
	cliui.Header("doctor")

	callbackURL := strings.TrimSpace(report.Runtime.ControlPlaneURL)
	if callbackURL == "" {
		callbackURL = cliui.Dim.Render("<unset>")
	}
	durable := "no"
	if report.DurableReconnectExpected {
		durable = "yes"
	}

	cliui.KeyValue("execution_mode", string(report.Runtime.ExecutionMode))
	cliui.KeyValue("data_dir", report.Runtime.DataDir)
	cliui.KeyValue("db_path", report.Runtime.DBPath)
	cliui.KeyValue("age_key_path", report.Runtime.AgeKeyPath)
	cliui.KeyValue("ca_key_path", report.Runtime.CAKeyPath)
	cliui.KeyValue("session_key_path", report.Runtime.SessionSecretPath)
	cliui.KeyValue("callback_url", callbackURL)
	cliui.KeyValue("callback_url_mode", string(report.CallbackURLMode))
	cliui.KeyValue("durable_reconnect", durable)

	lipgloss.Println()
	for _, check := range report.Checks {
		switch check.Status {
		case devdiag.CheckStatusOK:
			lipgloss.Println(cliui.CheckOK(check.Name, check.Detail))
		case devdiag.CheckStatusWarning:
			lipgloss.Println(cliui.CheckWarn(check.Name, check.Detail))
		case devdiag.CheckStatusError:
			lipgloss.Println(cliui.CheckFail(check.Name, check.Detail))
		}
	}
}
