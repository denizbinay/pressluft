package pki

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"filippo.io/age/armor"
)

const (
	caLifetimeYears = 10
)

type CA struct {
	cert        *x509.Certificate
	key         *ecdsa.PrivateKey
	ageIdentity age.Identity
	db          *sql.DB
}

type NodeCertificate struct {
	ID           int64
	ServerID     int64
	Fingerprint  string
	SerialNumber string
	IssuedAt     time.Time
	ExpiresAt    time.Time
	RevokedAt    *time.Time
}

func LoadOrCreateCA(db *sql.DB, ageKeyPath, caKeyPath string) (*CA, error) {
	var cert *x509.Certificate
	var key *ecdsa.PrivateKey

	row := db.QueryRow("SELECT certificate FROM ca_certificates ORDER BY id DESC LIMIT 1")
	var certPEM []byte
	err := row.Scan(&certPEM)
	if err == nil {
		block, _ := pem.Decode(certPEM)
		if block == nil {
			return nil, fmt.Errorf("failed to decode CA certificate")
		}
		cert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse CA certificate: %w", err)
		}

		key, err = loadCAKey(caKeyPath, ageKeyPath)
		if err != nil {
			return nil, err
		}
	} else if err == sql.ErrNoRows {
		key, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("generate CA key: %w", err)
		}

		serialNumber, err := generateSerialNumber()
		if err != nil {
			return nil, err
		}

		template := &x509.Certificate{
			SerialNumber: serialNumber,
			Subject: pkix.Name{
				Organization: []string{"Pressluft"},
				CommonName:   "Pressluft CA",
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(caLifetimeYears, 0, 0),
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			BasicConstraintsValid: true,
			IsCA:                  true,
			MaxPathLen:            0,
		}

		certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
		if err != nil {
			return nil, fmt.Errorf("create CA certificate: %w", err)
		}

		cert, err = x509.ParseCertificate(certDER)
		if err != nil {
			return nil, fmt.Errorf("parse CA certificate: %w", err)
		}

		if err := saveCAKey(caKeyPath, ageKeyPath, key); err != nil {
			return nil, err
		}

		certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
		_, err = db.Exec("INSERT INTO ca_certificates (fingerprint, certificate) VALUES (?, ?)",
			calculateFingerprint(cert), certPEM)
		if err != nil {
			return nil, fmt.Errorf("save CA certificate: %w", err)
		}
	} else {
		return nil, fmt.Errorf("lookup CA certificate: %w", err)
	}

	ageId, err := loadAgeIdentity(ageKeyPath)
	if err != nil {
		return nil, err
	}

	return &CA{cert: cert, key: key, ageIdentity: ageId, db: db}, nil
}

func (ca *CA) CertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(ca.cert)
	return pool
}

func (ca *CA) Certificate() *x509.Certificate {
	return ca.cert
}

func (ca *CA) SignCSR(csr *x509.CertificateRequest, validityDays int) (*x509.Certificate, error) {
	serialNumber, err := generateSerialNumber()
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(0, validityDays, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, csr.PublicKey, ca.key)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	return x509.ParseCertificate(certDER)
}

func GenerateServerCert(ca *CA, hostname string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate server key: %w", err)
	}

	serialNumber, err := generateSerialNumber()
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: hostname,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:    []string{hostname, "localhost"},
		IPAddresses: []net.IP{},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create server certificate: %w", err)
	}

	parsedCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse server certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
		Leaf:        parsedCert,
	}, nil
}

func ParseCertificateFromPEM(pemData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParseCertificate(block.Bytes)
}

func ParseCSRFromPEM(pemData []byte) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found")
	}
	return x509.ParseCertificateRequest(block.Bytes)
}

func calculateFingerprint(cert *x509.Certificate) string {
	hash := sha256.Sum256(cert.Raw)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func generateSerialNumber() (*big.Int, error) {
	serial := make([]byte, 20)
	_, err := rand.Read(serial)
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}
	return new(big.Int).SetBytes(serial), nil
}

func loadCAKey(caKeyPath, ageKeyPath string) (*ecdsa.PrivateKey, error) {
	encryptedKey, err := os.ReadFile(caKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read CA key: %w", err)
	}

	ageId, err := loadAgeIdentity(ageKeyPath)
	if err != nil {
		return nil, err
	}

	decrypted, err := age.Decrypt(strings.NewReader(string(encryptedKey)), ageId)
	if err != nil {
		return nil, fmt.Errorf("decrypt CA key: %w", err)
	}

	keyData, err := io.ReadAll(decrypted)
	if err != nil {
		return nil, fmt.Errorf("read decrypted key: %w", err)
	}

	key, err := x509.ParseECPrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}

	return key, nil
}

func saveCAKey(caKeyPath, ageKeyPath string, key *ecdsa.PrivateKey) error {
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal CA key: %w", err)
	}

	recipients, err := loadAgeRecipients(ageKeyPath)
	if err != nil {
		return err
	}

	var encrypted bytes.Buffer
	armorWriter := armor.NewWriter(&encrypted)
	writer, err := age.Encrypt(armorWriter, recipients...)
	if err != nil {
		return fmt.Errorf("encrypt CA key: %w", err)
	}

	if _, err := writer.Write(keyBytes); err != nil {
		return fmt.Errorf("write encrypted CA key: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close encrypted CA key: %w", err)
	}

	if err := armorWriter.Close(); err != nil {
		return fmt.Errorf("close armor: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(caKeyPath), 0700); err != nil {
		return fmt.Errorf("create CA key directory: %w", err)
	}

	if err := os.WriteFile(caKeyPath, encrypted.Bytes(), 0600); err != nil {
		return fmt.Errorf("write CA key: %w", err)
	}

	return nil
}

func loadAgeIdentity(path string) (age.Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read age key: %w", err)
	}

	identities, err := age.ParseIdentities(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse age identities: %w", err)
	}

	if len(identities) == 0 {
		return nil, fmt.Errorf("no age identities found")
	}

	return identities[0], nil
}

func loadAgeRecipients(path string) ([]age.Recipient, error) {
	identity, err := loadAgeIdentity(path)
	if err != nil {
		return nil, err
	}

	recipient, ok := identity.(*age.X25519Identity)
	if !ok {
		return nil, fmt.Errorf("unsupported age identity type")
	}

	return []age.Recipient{recipient.Recipient()}, nil
}
