package sshkeys

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const metadataPerm = 0o600

// KeyMetadata stores human-readable ownership and purpose.
type KeyMetadata struct {
	Purpose   string    `json:"purpose,omitempty"`
	Project   string    `json:"project,omitempty"`
	Owner     string    `json:"owner,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// KeyDetails is an extended view over a managed key.
type KeyDetails struct {
	KeyInfo
	Algorithm   string
	Comment     string
	Fingerprint string
	Metadata    KeyMetadata
}

func (m *Manager) SetMetadata(name string, metadata KeyMetadata) error {
	if err := validateKeyName(name); err != nil {
		return err
	}
	if _, err := os.Stat(m.privatePath(name)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("key %q does not exist", name)
		}
		return fmt.Errorf("check key existence: %w", err)
	}

	metadata.Tags = normalizeTags(metadata.Tags)
	metadata.UpdatedAt = time.Now().UTC()

	store, err := m.loadMetadataStore()
	if err != nil {
		return err
	}
	store[name] = metadata
	return m.saveMetadataStore(store)
}

func (m *Manager) KeyDetails(name string) (KeyDetails, error) {
	if err := validateKeyName(name); err != nil {
		return KeyDetails{}, err
	}

	privatePath := m.privatePath(name)
	publicPath := m.publicPath(name)
	if _, err := os.Stat(privatePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return KeyDetails{}, fmt.Errorf("key %q not found", name)
		}
		return KeyDetails{}, fmt.Errorf("check private key: %w", err)
	}
	if _, err := os.Stat(publicPath); err != nil {
		return KeyDetails{}, fmt.Errorf("public key for %q not found", name)
	}

	algorithm, comment := readPublicKeyParts(publicPath)
	fingerprint, _ := m.Fingerprint(name)
	metadata, _ := m.GetMetadata(name)

	return KeyDetails{
		KeyInfo: KeyInfo{
			Name:           name,
			PrivateKeyPath: privatePath,
			PublicKeyPath:  publicPath,
		},
		Algorithm:   algorithm,
		Comment:     comment,
		Fingerprint: fingerprint,
		Metadata:    metadata,
	}, nil
}

func (m *Manager) ListDetails() ([]KeyDetails, error) {
	keys, err := m.List()
	if err != nil {
		return nil, err
	}

	store, err := m.loadMetadataStore()
	if err != nil {
		return nil, err
	}

	result := make([]KeyDetails, 0, len(keys))
	for _, key := range keys {
		algorithm, comment := readPublicKeyParts(key.PublicKeyPath)
		fingerprint, _ := runOutput("ssh-keygen", "-lf", key.PublicKeyPath)

		meta := store[key.Name]
		result = append(result, KeyDetails{
			KeyInfo:     key,
			Algorithm:   algorithm,
			Comment:     comment,
			Fingerprint: strings.TrimSpace(fingerprint),
			Metadata:    meta,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func (m *Manager) GetMetadata(name string) (KeyMetadata, error) {
	if err := validateKeyName(name); err != nil {
		return KeyMetadata{}, err
	}
	store, err := m.loadMetadataStore()
	if err != nil {
		return KeyMetadata{}, err
	}
	return store[name], nil
}

func (m *Manager) metadataPath() string {
	return filepath.Join(m.baseDir, ".metadata.json")
}

func (m *Manager) loadMetadataStore() (map[string]KeyMetadata, error) {
	path := m.metadataPath()
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]KeyMetadata{}, nil
		}
		return nil, fmt.Errorf("read metadata store: %w", err)
	}

	if len(content) == 0 {
		return map[string]KeyMetadata{}, nil
	}

	var store map[string]KeyMetadata
	if err := json.Unmarshal(content, &store); err != nil {
		return nil, fmt.Errorf("parse metadata store: %w", err)
	}
	if store == nil {
		store = map[string]KeyMetadata{}
	}
	return store, nil
}

func (m *Manager) saveMetadataStore(store map[string]KeyMetadata) error {
	path := m.metadataPath()
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize metadata store: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, metadataPerm); err != nil {
		return fmt.Errorf("write metadata store: %w", err)
	}
	return nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	uniq := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		t := strings.TrimSpace(strings.ToLower(tag))
		if t == "" {
			continue
		}
		uniq[t] = struct{}{}
	}
	out := make([]string, 0, len(uniq))
	for tag := range uniq {
		out = append(out, tag)
	}
	sort.Strings(out)
	return out
}

func readPublicKeyParts(path string) (algorithm string, comment string) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	fields := strings.Fields(strings.TrimSpace(string(content)))
	if len(fields) < 2 {
		return "", ""
	}
	algorithm = fields[0]
	if len(fields) > 2 {
		comment = strings.Join(fields[2:], " ")
	}
	return algorithm, comment
}
