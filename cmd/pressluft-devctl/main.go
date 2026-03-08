package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"pressluft/internal/devdiag"
	"pressluft/internal/envconfig"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	cwd, _ := os.Getwd()
	runtime, err := envconfig.ResolveControlPlaneRuntime(true, cwd)
	if err != nil {
		log.Fatalf("resolve control-plane runtime: %v", err)
	}

	switch args[0] {
	case "status":
		printReport(devdiag.Inspect(runtime))
	case "preflight":
		workflow, err := parseWorkflow(args[1:])
		if err != nil {
			log.Fatalf("parse workflow: %v", err)
		}
		report := devdiag.Inspect(runtime)
		printReport(report)
		if !report.HealthyFor(workflow) {
			fmt.Println()
			fmt.Printf("workflow=%s preflight failed\n", workflow)
			for _, issue := range report.WorkflowIssues(workflow) {
				fmt.Printf("- %s\n", issue)
			}
			fmt.Println("Suggested next steps: make dev-status ; make dev-reset CONFIRM=1")
			os.Exit(1)
		}
	case "reset":
		force := len(args) > 1 && args[1] == "--force"
		if !force {
			log.Fatal("reset requires --force")
		}
		if err := reset(runtime); err != nil {
			log.Fatalf("reset local state: %v", err)
		}
		fmt.Printf("Removed Pressluft local state bundle:\n- %s\n- %s\n- %s\n- %s\n", runtime.DBPath, runtime.AgeKeyPath, runtime.CAKeyPath, runtime.SessionSecretPath)
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("pressluft-devctl")
	fmt.Println("  status                   Inspect local dev state and callback durability")
	fmt.Println("  preflight --workflow=X   Validate local state for dev or lab workflow")
	fmt.Println("  reset --force            Remove the local Pressluft state bundle")
}

func parseWorkflow(args []string) (devdiag.Workflow, error) {
	workflow := devdiag.WorkflowDev
	for _, arg := range args {
		if strings.HasPrefix(arg, "--workflow=") {
			value := strings.TrimPrefix(arg, "--workflow=")
			switch value {
			case string(devdiag.WorkflowDev):
				workflow = devdiag.WorkflowDev
			case string(devdiag.WorkflowLab):
				workflow = devdiag.WorkflowLab
			default:
				return "", fmt.Errorf("unsupported workflow %q", value)
			}
		}
	}
	return workflow, nil
}

func printReport(report devdiag.Report) {
	fmt.Println("Pressluft dev state")
	fmt.Printf("execution_mode: %s\n", report.Runtime.ExecutionMode)
	fmt.Printf("data_dir: %s\n", report.Runtime.DataDir)
	fmt.Printf("db_path: %s\n", report.Runtime.DBPath)
	fmt.Printf("age_key_path: %s\n", report.Runtime.AgeKeyPath)
	fmt.Printf("ca_key_path: %s\n", report.Runtime.CAKeyPath)
	fmt.Printf("session_key_path: %s\n", report.Runtime.SessionSecretPath)
	if strings.TrimSpace(report.Runtime.ControlPlaneURL) == "" {
		fmt.Println("callback_url: <unset>")
	} else {
		fmt.Printf("callback_url: %s\n", report.Runtime.ControlPlaneURL)
	}
	fmt.Printf("callback_url_mode: %s\n", report.CallbackURLMode)
	if report.DurableReconnectExpected {
		fmt.Println("durable_reconnect: yes")
	} else {
		fmt.Println("durable_reconnect: no")
	}
	fmt.Println("checks:")
	for _, check := range report.Checks {
		fmt.Printf("- [%s] %s: %s\n", check.Status, check.Name, check.Detail)
	}
}

func reset(runtime envconfig.ControlPlaneRuntime) error {
	paths := []string{
		runtime.DBPath,
		runtime.AgeKeyPath,
		runtime.CAKeyPath,
		runtime.SessionSecretPath,
	}
	for _, path := range paths {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		dir := filepath.Dir(path)
		_ = os.Remove(dir)
	}
	return nil
}
