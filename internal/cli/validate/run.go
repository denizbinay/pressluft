package validate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"pressluft/internal/cli/cliui"
	"pressluft/internal/cli/cliutil"
)

func Run() error {
	rootDir, err := cliutil.FindRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	if err := runGofmtCheck(rootDir); err != nil {
		return err
	}

	if err := runGoVet(rootDir); err != nil {
		return err
	}

	if err := runGoTest(rootDir); err != nil {
		return err
	}

	if err := runProfileSchemaTest(rootDir); err != nil {
		return err
	}

	if err := runProfileRegistryTest(rootDir); err != nil {
		return err
	}

	if err := runAnsibleSyntaxCheck(rootDir); err != nil {
		return err
	}

	if err := runFrontendGenerate(rootDir); err != nil {
		return err
	}

	return nil
}

func runGofmtCheck(rootDir string) error {
	cliui.Step("Checking Go formatting")
	cmd := exec.Command("gofmt", "-l", "cmd", "internal")
	cmd.Dir = rootDir
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("gofmt check: %w", err)
	}
	if len(out) > 0 {
		cliui.StepDone("Checking Go formatting")
		return fmt.Errorf("files need formatting:\n%s", out)
	}
	cliui.StepDone("Checking Go formatting")
	return nil
}

func runGoVet(rootDir string) error {
	cliui.Step("Running go vet")
	cmd := exec.Command(cliutil.GoCmd(), "vet", "./...")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go vet failed: %w", err)
	}
	cliui.StepDone("Running go vet")
	return nil
}

func runGoTest(rootDir string) error {
	cliui.Step("Running go tests")
	cmd := exec.Command(cliutil.GoCmd(), "test", "./...")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test failed: %w", err)
	}
	cliui.StepDone("Running go tests")
	return nil
}

func runProfileSchemaTest(rootDir string) error {
	cliui.Step("Running profile schema validation")
	cmd := exec.Command(cliutil.GoCmd(), "test", "./internal/controlplane/server/profiles", "-run", "TestProfileArtifactsSatisfySchema", "-count=1")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("profile schema test failed: %w", err)
	}
	cliui.StepDone("Running profile schema validation")
	return nil
}

func runProfileRegistryTest(rootDir string) error {
	cliui.Step("Running profile registry consistency test")
	cmd := exec.Command(cliutil.GoCmd(), "test", "./internal/controlplane/server/profiles", "-run", "TestRegistryMatchesProfileArtifacts", "-count=1")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("profile registry test failed: %w", err)
	}
	cliui.StepDone("Running profile registry consistency test")
	return nil
}

func runAnsibleSyntaxCheck(rootDir string) error {
	cliui.Step("Checking Ansible playbook syntax")

	_, err := exec.LookPath("ansible-playbook")
	if err != nil {
		cliui.WarnBox([]string{
			"ansible-playbook not found, skipping Ansible syntax check.",
			"Install Ansible to enable this validation step.",
		})
		cliui.StepDone("Checking Ansible playbook syntax")
		return nil
	}

	playbooksDir := filepath.Join(rootDir, "ops", "ansible", "playbooks")
	entries, err := os.ReadDir(playbooksDir)
	if err != nil {
		return fmt.Errorf("read playbooks directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yml" {
			continue
		}
		playbookPath := filepath.Join(playbooksDir, entry.Name())
		cmd := exec.Command("ansible-playbook", "--syntax-check", playbookPath)
		cmd.Dir = rootDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ansible syntax check failed for %s: %w", entry.Name(), err)
		}
	}

	cliui.StepDone("Checking Ansible playbook syntax")
	return nil
}

func runFrontendGenerate(rootDir string) error {
	cliui.Step("Running frontend static generation")
	webDir := filepath.Join(rootDir, "web")
	cmd := exec.Command(cliutil.NpmCmd(), "--prefix", webDir, "run", "generate")
	cmd.Dir = rootDir
	cmd.Env = append(os.Environ(), "NODE_OPTIONS=--max-old-space-size=8192")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("frontend generation failed: %w", err)
	}

	indexPath := filepath.Join(webDir, ".output", "public", "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return fmt.Errorf("frontend build did not produce %s", indexPath)
	}

	cliui.StepDone("Running frontend static generation")
	return nil
}
