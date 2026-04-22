package sshkeys

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const (
	privateKeyPerm = 0o600
	publicKeyPerm  = 0o644
	sshDirPerm     = 0o700
)

// KeyType is a supported SSH key algorithm.
type KeyType string

const (
	KeyTypeED25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
)

// GenerateOptions controls key generation.
type GenerateOptions struct {
	Name       string
	Type       KeyType
	Comment    string
	Bits       int
	Passphrase string
	Force      bool
}

// KeyInfo represents a managed key pair.
type KeyInfo struct {
	Name           string
	PrivateKeyPath string
	PublicKeyPath  string
}

// Manager manages key files in ~/.ssh/sshkm.
type Manager struct {
	baseDir string
}

func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve user home: %w", err)
	}

	base := filepath.Join(home, ".ssh", "sshkm")
	if err := os.MkdirAll(base, sshDirPerm); err != nil {
		return nil, fmt.Errorf("create manager directory: %w", err)
	}
	if err := os.Chmod(base, sshDirPerm); err != nil {
		return nil, fmt.Errorf("set manager directory permission: %w", err)
	}

	return &Manager{baseDir: base}, nil
}

func (m *Manager) BaseDir() string {
	return m.baseDir
}

func (m *Manager) Generate(opts GenerateOptions) (KeyInfo, error) {
	if err := validateKeyName(opts.Name); err != nil {
		return KeyInfo{}, err
	}
	if opts.Type == "" {
		opts.Type = KeyTypeED25519
	}
	if opts.Type != KeyTypeED25519 && opts.Type != KeyTypeRSA {
		return KeyInfo{}, fmt.Errorf("unsupported key type %q", opts.Type)
	}
	if opts.Type == KeyTypeRSA && opts.Bits == 0 {
		opts.Bits = 4096
	}

	keyPath := m.privatePath(opts.Name)
	pubPath := m.publicPath(opts.Name)
	if !opts.Force {
		if _, err := os.Stat(keyPath); err == nil {
			return KeyInfo{}, fmt.Errorf("private key already exists: %s", keyPath)
		}
		if _, err := os.Stat(pubPath); err == nil {
			return KeyInfo{}, fmt.Errorf("public key already exists: %s", pubPath)
		}
	}

	args := []string{"-t", string(opts.Type), "-f", keyPath, "-N", opts.Passphrase}
	if opts.Comment != "" {
		args = append(args, "-C", opts.Comment)
	}
	if opts.Type == KeyTypeRSA {
		args = append(args, "-b", fmt.Sprintf("%d", opts.Bits))
	}

	if err := run("ssh-keygen", args...); err != nil {
		return KeyInfo{}, err
	}

	if err := os.Chmod(keyPath, privateKeyPerm); err != nil {
		return KeyInfo{}, fmt.Errorf("set private key permission: %w", err)
	}
	if err := os.Chmod(pubPath, publicKeyPerm); err != nil {
		return KeyInfo{}, fmt.Errorf("set public key permission: %w", err)
	}

	return KeyInfo{
		Name:           opts.Name,
		PrivateKeyPath: keyPath,
		PublicKeyPath:  pubPath,
	}, nil
}

func (m *Manager) List() ([]KeyInfo, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read manager directory: %w", err)
	}

	keys := make([]KeyInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".pub") {
			continue
		}
		privatePath := filepath.Join(m.baseDir, name)
		publicPath := privatePath + ".pub"
		if _, err := os.Stat(publicPath); err != nil {
			continue
		}
		keys = append(keys, KeyInfo{
			Name:           name,
			PrivateKeyPath: privatePath,
			PublicKeyPath:  publicPath,
		})
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Name < keys[j].Name
	})
	return keys, nil
}

func (m *Manager) Import(name, sourcePath string, overwrite bool) (KeyInfo, error) {
	if err := validateKeyName(name); err != nil {
		return KeyInfo{}, err
	}
	if sourcePath == "" {
		return KeyInfo{}, errors.New("source path is required")
	}
	if strings.HasSuffix(sourcePath, ".pub") {
		return KeyInfo{}, errors.New("source path must be a private key, not .pub")
	}

	keyBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return KeyInfo{}, fmt.Errorf("read source key: %w", err)
	}
	if !bytes.Contains(keyBytes, []byte("PRIVATE KEY")) {
		return KeyInfo{}, errors.New("source file does not look like a private key")
	}

	sourcePub := sourcePath + ".pub"
	pubBytes, err := os.ReadFile(sourcePub)
	if err != nil {
		return KeyInfo{}, fmt.Errorf("read source public key (%s): %w", sourcePub, err)
	}

	dstKey := m.privatePath(name)
	dstPub := m.publicPath(name)
	if !overwrite {
		if _, err := os.Stat(dstKey); err == nil {
			return KeyInfo{}, fmt.Errorf("private key already exists: %s", dstKey)
		}
	}

	if err := os.WriteFile(dstKey, keyBytes, privateKeyPerm); err != nil {
		return KeyInfo{}, fmt.Errorf("write private key: %w", err)
	}
	if err := os.WriteFile(dstPub, pubBytes, publicKeyPerm); err != nil {
		return KeyInfo{}, fmt.Errorf("write public key: %w", err)
	}

	return KeyInfo{
		Name:           name,
		PrivateKeyPath: dstKey,
		PublicKeyPath:  dstPub,
	}, nil
}

func (m *Manager) Delete(name string) error {
	if err := validateKeyName(name); err != nil {
		return err
	}
	privatePath := m.privatePath(name)
	publicPath := m.publicPath(name)

	privateRemoved := false
	if err := os.Remove(privatePath); err == nil {
		privateRemoved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove private key: %w", err)
	}

	publicRemoved := false
	if err := os.Remove(publicPath); err == nil {
		publicRemoved = true
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove public key: %w", err)
	}

	if !privateRemoved && !publicRemoved {
		return fmt.Errorf("key %q was not found", name)
	}
	return nil
}

func (m *Manager) PublicKey(name string) (string, error) {
	if err := validateKeyName(name); err != nil {
		return "", err
	}
	content, err := os.ReadFile(m.publicPath(name))
	if err != nil {
		return "", fmt.Errorf("read public key: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

func (m *Manager) Fingerprint(name string) (string, error) {
	if err := validateKeyName(name); err != nil {
		return "", err
	}
	out, err := runOutput("ssh-keygen", "-lf", m.publicPath(name))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (m *Manager) AddToAgent(name string) error {
	if err := validateKeyName(name); err != nil {
		return err
	}
	return run("ssh-add", m.privatePath(name))
}

func (m *Manager) RemoveFromAgent(name string) error {
	if err := validateKeyName(name); err != nil {
		return err
	}
	return run("ssh-add", "-d", m.privatePath(name))
}

func (m *Manager) privatePath(name string) string {
	return filepath.Join(m.baseDir, name)
}

func (m *Manager) publicPath(name string) string {
	return m.privatePath(name) + ".pub"
}

func validateKeyName(name string) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		switch r {
		case '-', '_', '.':
			continue
		default:
			return fmt.Errorf("key name %q contains forbidden character %q", name, r)
		}
	}
	return nil
}

func run(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return fmt.Errorf("%s failed: %w", command, err)
		}
		return fmt.Errorf("%s failed: %s", command, msg)
	}
	return nil
}

func runOutput(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", fmt.Errorf("%s failed: %w", command, err)
		}
		return "", fmt.Errorf("%s failed: %s", command, msg)
	}
	return stdout.String(), nil
}

func CopyStream(dst io.Writer, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	_, err = dst.Write(data)
	return err
}
