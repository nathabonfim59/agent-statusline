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
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/nathabonfim59/agent-statusline/harness"
)

func certDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "agent-statusline", "certs")
}

func GenerateCA() (cert, key []byte, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:         "agent-statusline Local CA (self-signed)",
			Organization:       []string{"agent-statusline — local MITM proxy"},
			OrganizationalUnit: []string{"Private key stored only on this machine. https://github.com/nathabonfim59/agent-statusline"},
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

func LoadOrGenerateCA() (cert, key []byte, err error) {
	cert, key, err = LoadCA()
	if err == nil {
		return cert, key, nil
	}
	return GenerateCA()
}

func detectLinuxDistro() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	content := string(data)
	lower := strings.ToLower(content)

	switch {
	case strings.Contains(lower, "debian"), strings.Contains(lower, "ubuntu"),
		strings.Contains(lower, "linux mint"), strings.Contains(lower, "pop!_os"):
		return "debian"
	case strings.Contains(lower, "rhel"), strings.Contains(lower, "fedora"),
		strings.Contains(lower, "centos"), strings.Contains(lower, "rocky"),
		strings.Contains(lower, "alma"):
		return "rhel"
	case strings.Contains(lower, "arch"), strings.Contains(lower, "manjaro"),
		strings.Contains(lower, "endeavour"):
		return "arch"
	case strings.Contains(lower, "opensuse"), strings.Contains(lower, "suse"):
		return "suse"
	default:
		return "unknown"
	}
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

	fmt.Printf("CA certificate: %s%s%s\n\n", harness.Cyan, certPath, harness.Reset)
	fmt.Printf("To trust this certificate, run:\n\n")

	var cmds []string

	switch runtime.GOOS {
	case "linux":
		distro := detectLinuxDistro()
		switch distro {
		case "debian":
			cmds = []string{
				fmt.Sprintf("sudo cp %s /usr/local/share/ca-certificates/agent-statusline-ca.crt", certPath),
				"sudo update-ca-certificates",
			}
		case "rhel":
			cmds = []string{
				fmt.Sprintf("sudo cp %s /etc/pki/ca-trust/source/anchors/", certPath),
				"sudo update-ca-trust",
			}
		case "arch":
			cmds = []string{
				fmt.Sprintf("sudo cp %s /etc/ca-certificates/trust-source/anchors/", certPath),
				"sudo trust extract-compat",
			}
		case "suse":
			cmds = []string{
				fmt.Sprintf("sudo cp %s /etc/pki/trust/anchors/", certPath),
				"sudo update-ca-certificates",
			}
		default:
			cmds = []string{
				fmt.Sprintf("# Unknown distro — try one of:"),
				fmt.Sprintf("# Debian/Ubuntu:"),
				fmt.Sprintf("sudo cp %s /usr/local/share/ca-certificates/ && sudo update-ca-certificates", certPath),
				fmt.Sprintf("# Fedora/RHEL:"),
				fmt.Sprintf("sudo cp %s /etc/pki/ca-trust/source/anchors/ && sudo update-ca-trust", certPath),
				fmt.Sprintf("# Arch:"),
				fmt.Sprintf("sudo cp %s /etc/ca-certificates/trust-source/anchors/ && sudo trust extract-compat", certPath),
			}
		}

	case "darwin":
		cmds = []string{
			fmt.Sprintf("sudo security add-trusted-cert -d -p ssl -k /Library/Keychains/System.keychain %s", certPath),
		}

	default:
		fmt.Printf("No automatic instructions for %s.\n", runtime.GOOS)
		fmt.Printf("Install the CA certificate manually from: %s\n", certPath)
		return nil
	}

	for _, c := range cmds {
		fmt.Printf("%s%s%s\n", harness.Green, c, harness.Reset)
	}

	return nil
}
