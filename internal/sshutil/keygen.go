// Package sshutil provides provider-agnostic SSH key utilities.
package sshutil

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// GenerateKeyPair creates a new Ed25519 SSH key pair.
// Returns the public key in OpenSSH authorized_keys format and the private
// key in PEM format. The comment is appended to the public key and used as
// the PEM header comment.
func GenerateKeyPair(comment string) (publicKey, privateKey string, err error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("create ssh public key: %w", err)
	}
	authorizedKey := ssh.MarshalAuthorizedKey(sshPubKey)
	publicKeyStr := strings.TrimSpace(string(authorizedKey))
	if comment != "" {
		publicKeyStr = publicKeyStr + " " + comment
	}

	pemBlock, err := ssh.MarshalPrivateKey(privKey, comment)
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(pemBlock)

	return publicKeyStr, string(privateKeyPEM), nil
}
