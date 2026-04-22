package sshkeys

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DiscoveredKey describes a key discovered outside the manager.
type DiscoveredKey struct {
	Path        string
	BaseName    string
	Algorithm   string
	Comment     string
	Fingerprint string
}

// DuplicateGroup groups keys by the same identifier (name/fingerprint).
type DuplicateGroup struct {
	Identifier string
	Keys       []DiscoveredKey
}

// DiscoveryReport helps understand duplicate files and ownership.
type DiscoveryReport struct {
	Keys                   []DiscoveredKey
	DuplicateByBaseName    []DuplicateGroup
	DuplicateByFingerprint []DuplicateGroup
}

func Discover(paths []string) (DiscoveryReport, error) {
	if len(paths) == 0 {
		return DiscoveryReport{}, errors.New("at least one path is required")
	}

	all := make([]DiscoveredKey, 0, 32)
	for _, root := range paths {
		abs, err := filepath.Abs(root)
		if err != nil {
			return DiscoveryReport{}, fmt.Errorf("resolve path %q: %w", root, err)
		}

		if _, err := os.Stat(abs); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return DiscoveryReport{}, fmt.Errorf("stat path %q: %w", abs, err)
		}

		err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}

			if strings.HasSuffix(path, ".pub") {
				return nil
			}
			if strings.HasSuffix(path, ".crt") || strings.HasSuffix(path, ".cer") {
				return nil
			}

			isKey, err := isPrivateKeyFile(path)
			if err != nil || !isKey {
				return nil
			}

			pubPath := path + ".pub"
			algorithm, comment := readPublicKeyParts(pubPath)
			fingerprint := ""
			if _, err := os.Stat(pubPath); err == nil {
				out, err := runOutput("ssh-keygen", "-lf", pubPath)
				if err == nil {
					fingerprint = strings.TrimSpace(out)
				}
			}

			all = append(all, DiscoveredKey{
				Path:        path,
				BaseName:    filepath.Base(path),
				Algorithm:   algorithm,
				Comment:     comment,
				Fingerprint: fingerprint,
			})
			return nil
		})
		if err != nil {
			return DiscoveryReport{}, err
		}
	}

	sort.Slice(all, func(i, j int) bool {
		return all[i].Path < all[j].Path
	})

	byName := make(map[string][]DiscoveredKey)
	byFingerprint := make(map[string][]DiscoveredKey)
	for _, key := range all {
		byName[key.BaseName] = append(byName[key.BaseName], key)
		if key.Fingerprint != "" {
			byFingerprint[key.Fingerprint] = append(byFingerprint[key.Fingerprint], key)
		}
	}

	return DiscoveryReport{
		Keys:                   all,
		DuplicateByBaseName:    buildDuplicateGroups(byName),
		DuplicateByFingerprint: buildDuplicateGroups(byFingerprint),
	}, nil
}

func buildDuplicateGroups(groups map[string][]DiscoveredKey) []DuplicateGroup {
	out := make([]DuplicateGroup, 0)
	for key, values := range groups {
		if len(values) <= 1 {
			continue
		}
		sort.Slice(values, func(i, j int) bool {
			return values[i].Path < values[j].Path
		})
		out = append(out, DuplicateGroup{
			Identifier: key,
			Keys:       values,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Identifier < out[j].Identifier
	})
	return out
}

func isPrivateKeyFile(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if len(content) > 16*1024 {
		content = content[:16*1024]
	}
	return bytes.Contains(content, []byte("PRIVATE KEY")), nil
}
