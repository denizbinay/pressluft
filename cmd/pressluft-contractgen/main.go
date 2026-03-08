package main

import (
	"flag"
	"fmt"
	"os"

	"pressluft/internal/contract"
)

func main() {
	format := flag.String("format", "ts", "output format: ts, api-ts, or json")
	flag.Parse()

	var (
		data []byte
		err  error
	)

	switch *format {
	case "json":
		data, err = contract.JSON()
	case "api-ts":
		var rendered string
		rendered, err = contract.RenderAPITypeScriptModule()
		data = []byte(rendered)
	case "ts":
		var rendered string
		rendered, err = contract.RenderTypeScriptModule()
		data = []byte(rendered)
	default:
		err = fmt.Errorf("unsupported format %q", *format)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "pressluft-contractgen: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "pressluft-contractgen: write output: %v\n", err)
		os.Exit(1)
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		fmt.Println()
	}
}
