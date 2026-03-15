package main

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"pressluft/internal/shared/security"

	_ "modernc.org/sqlite"
)

type sshAccessTarget struct {
	ID   string
	Name string
	IPv4 string
	IPv6 string
	Key  string
}

var serverSSHCmd = &cobra.Command{
	Use:   "server-ssh TARGET",
	Short: "SSH access for a managed server",
	Long: `Print or execute SSH access for a managed server.

By default, prints connection details and a ready-to-copy SSH command.
Use --exec to open an interactive SSH session directly.
Use --print-key to output the decrypted private key to stdout.`,
	Args: cobra.ExactArgs(1),
	RunE: runServerSSH,
}

var (
	sshExec     bool
	sshPrintKey bool
)

func init() {
	serverSSHCmd.Flags().BoolVar(&sshExec, "exec", false, "Open an interactive SSH session")
	serverSSHCmd.Flags().BoolVar(&sshPrintKey, "print-key", false, "Print the decrypted SSH private key to stdout")
	serverSSHCmd.MarkFlagsMutuallyExclusive("exec", "print-key")
}

func runServerSSH(cmd *cobra.Command, args []string) error {
	target := args[0]

	runtime, err := resolveRuntime()
	if err != nil {
		return fmt.Errorf("resolve runtime: %w", err)
	}

	db, err := openExistingDB(runtime.DBPath)
	if err != nil {
		return err
	}
	defer db.Close()

	access, err := lookupSSHAccessTarget(db, target)
	if err != nil {
		return err
	}
	decryptedKey, err := security.Decrypt(access.Key)
	if err != nil {
		return fmt.Errorf("decrypt server ssh key: %w", err)
	}

	if sshPrintKey {
		_, err := os.Stdout.Write(decryptedKey)
		return err
	}

	host := strings.TrimSpace(access.IPv4)
	if host == "" {
		host = strings.TrimSpace(access.IPv6)
	}
	if host == "" {
		return fmt.Errorf("server %q has no recorded IP address", access.ID)
	}

	keyFile, err := os.CreateTemp("", "pressluft-server-ssh-*.key")
	if err != nil {
		return fmt.Errorf("create temp key file: %w", err)
	}
	defer keyFile.Close()
	if err := keyFile.Chmod(0o600); err != nil {
		return fmt.Errorf("chmod temp key file: %w", err)
	}
	if _, err := keyFile.Write(decryptedKey); err != nil {
		return fmt.Errorf("write temp key file: %w", err)
	}

	sshArgs := []string{"-i", keyFile.Name(), "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null", "root@" + host}
	if sshExec {
		sshCmd := exec.Command("ssh", sshArgs...)
		sshCmd.Stdin = os.Stdin
		sshCmd.Stdout = os.Stdout
		sshCmd.Stderr = os.Stderr
		if err := sshCmd.Run(); err != nil {
			return fmt.Errorf("run ssh: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\nSSH key left at %s\n", keyFile.Name())
		return nil
	}

	fmt.Printf("server: %s (%s)\n", access.Name, access.ID)
	fmt.Printf("host: %s\n", host)
	fmt.Printf("key_file: %s\n", keyFile.Name())
	fmt.Printf("ssh -i %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@%s\n", keyFile.Name(), host)
	return nil
}

func lookupSSHAccessTarget(db *sql.DB, target string) (*sshAccessTarget, error) {
	rows, err := db.Query(`
		SELECT s.id, s.name, s.ipv4, s.ipv6, k.private_key_encrypted
		FROM servers s
		JOIN server_keys k ON k.server_id = s.id
		WHERE s.id = ? OR s.name = ?
		ORDER BY s.created_at DESC
	`, target, target)
	if err != nil {
		return nil, fmt.Errorf("query server ssh access: %w", err)
	}
	defer rows.Close()

	var matches []sshAccessTarget
	for rows.Next() {
		var item sshAccessTarget
		var ipv4 sql.NullString
		var ipv6 sql.NullString
		if err := rows.Scan(&item.ID, &item.Name, &ipv4, &ipv6, &item.Key); err != nil {
			return nil, fmt.Errorf("scan server ssh access: %w", err)
		}
		if ipv4.Valid {
			item.IPv4 = ipv4.String
		}
		if ipv6.Valid {
			item.IPv6 = ipv6.String
		}
		matches = append(matches, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate server ssh access: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no server with stored SSH key matched %q", target)
	}
	if len(matches) > 1 {
		ids := make([]string, 0, len(matches))
		for _, match := range matches {
			ids = append(ids, fmt.Sprintf("%s (%s)", match.Name, match.ID))
		}
		return nil, fmt.Errorf("multiple servers matched %q: %s", target, strings.Join(ids, ", "))
	}
	return &matches[0], nil
}

func openExistingDB(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("stat db: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}
