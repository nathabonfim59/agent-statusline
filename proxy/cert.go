package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

func certDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "claude-statusline", "certs")
}

func GenerateCA() (cert, key []byte, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "claude-statusline CA",
			Organization: []string{"claude-statusline"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLenZero:        true,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	cert = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	key = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})

	dir := certDir()
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "ca-cert.pem"), cert, 0o600)
	os.WriteFile(filepath.Join(dir, "ca-key.pem"), key, 0o600)

	return cert, key, nil
}

func LoadCA() (cert, key []byte, err error) {
	dir := certDir()
	cert, err = os.ReadFile(filepath.Join(dir, "ca-cert.pem"))
	if err != nil {
		return nil, nil, err
	}
	key, err = os.ReadFile(filepath.Join(dir, "ca-key.pem"))
	if err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func InstallCA() error {
	dir := certDir()
	certPath := filepath.Join(dir, "ca-cert.pem")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		fmt.Println("CA cert not found, generating...")
		if _, _, err := GenerateCA(); err != nil {
			return fmt.Errorf("failed to generate CA: %w", err)
		}
	}

	switch runtime.GOOS {
	case "linux":
		dst := "/usr/local/share/ca-certificates/claude-statusline-ca.crt"
		cmd := exec.Command("sudo", "cp", certPath, dst)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Manual install: sudo cp %s %s && sudo update-ca-certificates\n", certPath, dst)
			return err
		}
		cmd = exec.Command("sudo", "update-ca-certificates")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Manual: sudo update-ca-certificates\n")
			return err
		}
		fmt.Println("CA installed to system trust store (Linux).")
		return nil

	case "darwin":
		cmd := exec.Command("security", "add-trusted-cert", "-d", "-p", "ssl", "-k",
			"/Library/Keychains/System.keychain", certPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Manual install: sudo security add-trusted-cert -d -p ssl %s\n", certPath)
			return err
		}
		fmt.Println("CA installed to system trust store (macOS).")
		return nil

	default:
		fmt.Printf("Automatic CA install not supported on %s.\n", runtime.GOOS)
		fmt.Printf("Install manually: %s\n", certPath)
		return nil
	}
}

func LoadOrGenerateCA() (cert, key []byte, err error) {
	cert, key, err = LoadCA()
	if err == nil {
		return cert, key, nil
	}
	return GenerateCA()
}