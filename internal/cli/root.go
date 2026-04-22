package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/RTHeLL/ssh-keys-manager/internal/sshkeys"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "sshkm",
		Short: "Professional SSH keys manager for terminal workflows",
		Long:  "sshkm manages SSH keys in an isolated ~/.ssh/sshkm directory and integrates with ssh-agent.",
	}

	root.AddCommand(
		newInitCommand(),
		newListCommand(),
		newInfoCommand(),
		newAnnotateCommand(),
		newDiscoverCommand(),
		newGenerateCommand(),
		newImportCommand(),
		newPublicCommand(),
		newFingerprintCommand(),
		newAgentCommand(),
		newDeleteCommand(),
		newVersionCommand(),
	)
	return root
}

func newInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize manager directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Initialized manager directory: %s\n", manager.BaseDir())
			return nil
		},
	}
}

func newListCommand() *cobra.Command {
	var details bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List managed keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}

			if !details {
				keys, err := manager.List()
				if err != nil {
					return err
				}
				if len(keys) == 0 {
					fmt.Fprintln(cmd.OutOrStdout(), "No managed keys found.")
					return nil
				}
				for _, key := range keys {
					fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", key.Name)
				}
				return nil
			}

			keys, err := manager.ListDetails()
			if err != nil {
				return err
			}
			if len(keys) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No managed keys found.")
				return nil
			}
			for _, key := range keys {
				fmt.Fprintf(cmd.OutOrStdout(), "- %s\n", key.Name)
				if key.Algorithm != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  algorithm: %s\n", key.Algorithm)
				}
				if key.Comment != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  comment: %s\n", key.Comment)
				}
				if key.Metadata.Purpose != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  purpose: %s\n", key.Metadata.Purpose)
				}
				if key.Metadata.Project != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "  project: %s\n", key.Metadata.Project)
				}
				if len(key.Metadata.Tags) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "  tags: %s\n", strings.Join(key.Metadata.Tags, ", "))
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  fingerprint: %s\n", shortFingerprint(key.Fingerprint))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&details, "details", false, "Show extended key details and metadata")
	return cmd
}

func newInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show full details for one key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}
			details, err := manager.KeyDetails(args[0])
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "name: %s\n", details.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "private: %s\n", details.PrivateKeyPath)
			fmt.Fprintf(cmd.OutOrStdout(), "public: %s\n", details.PublicKeyPath)
			if details.Algorithm != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "algorithm: %s\n", details.Algorithm)
			}
			if details.Comment != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "comment: %s\n", details.Comment)
			}
			if details.Fingerprint != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "fingerprint: %s\n", details.Fingerprint)
			}
			if details.Metadata.Purpose != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "purpose: %s\n", details.Metadata.Purpose)
			}
			if details.Metadata.Project != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "project: %s\n", details.Metadata.Project)
			}
			if details.Metadata.Owner != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "owner: %s\n", details.Metadata.Owner)
			}
			if len(details.Metadata.Tags) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "tags: %s\n", strings.Join(details.Metadata.Tags, ", "))
			}
			if details.Metadata.Notes != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "notes: %s\n", details.Metadata.Notes)
			}
			return nil
		},
	}
}

func newAnnotateCommand() *cobra.Command {
	var (
		purpose string
		project string
		owner   string
		tags    string
		notes   string
	)

	cmd := &cobra.Command{
		Use:   "annotate <name>",
		Short: "Attach purpose/project/owner metadata to key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}

			if purpose == "" && project == "" && owner == "" && tags == "" && notes == "" {
				return errors.New("at least one metadata flag must be provided")
			}

			err = manager.SetMetadata(args[0], sshkeys.KeyMetadata{
				Purpose: purpose,
				Project: project,
				Owner:   owner,
				Tags:    parseTags(tags),
				Notes:   notes,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated metadata for key: %s\n", args[0])
			return nil
		},
	}

	cmd.Flags().StringVar(&purpose, "purpose", "", "Business purpose of this key")
	cmd.Flags().StringVar(&project, "project", "", "Project or service name")
	cmd.Flags().StringVar(&owner, "owner", "", "Owner or responsible person/team")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma separated tags")
	cmd.Flags().StringVar(&notes, "notes", "", "Free-form notes")
	return cmd
}

func newDiscoverCommand() *cobra.Command {
	var paths []string
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover keys and explain duplicate filenames",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(paths) == 0 {
				home, err := os.UserHomeDir()
				if err != nil {
					return err
				}
				paths = []string{
					filepath.Join(home, ".ssh"),
				}
			}

			report, err := sshkeys.Discover(paths)
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Discovered private keys: %d\n", len(report.Keys))

			if len(report.DuplicateByBaseName) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Duplicate filenames: not found")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Duplicate filenames:")
				for _, group := range report.DuplicateByBaseName {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s (%d files)\n", group.Identifier, len(group.Keys))
					for _, key := range group.Keys {
						fmt.Fprintf(cmd.OutOrStdout(), "    • %s\n", key.Path)
					}
				}
			}

			if len(report.DuplicateByFingerprint) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Duplicate key material: not found")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Duplicate key material (same fingerprint):")
				for _, group := range report.DuplicateByFingerprint {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", shortFingerprint(group.Identifier))
					for _, key := range group.Keys {
						fmt.Fprintf(cmd.OutOrStdout(), "    • %s (%s)\n", key.Path, key.BaseName)
					}
				}
			}

			orphanNames := detectUnclearNames(report.Keys)
			if len(orphanNames) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Keys with low-context names (recommend annotate/import with clear name):")
				for _, name := range orphanNames {
					fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", name)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringSliceVarP(&paths, "path", "p", nil, "Path(s) to scan recursively for private keys")
	return cmd
}

func newGenerateCommand() *cobra.Command {
	var (
		keyType       string
		comment       string
		bits          int
		force         bool
		passphrase    string
		passphraseEnv string
	)

	cmd := &cobra.Command{
		Use:   "generate <name>",
		Short: "Generate a new SSH key pair",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}

			phrase, err := resolvePassphrase(passphrase, passphraseEnv)
			if err != nil {
				return err
			}

			info, err := manager.Generate(sshkeys.GenerateOptions{
				Name:       args[0],
				Type:       sshkeys.KeyType(keyType),
				Comment:    comment,
				Bits:       bits,
				Passphrase: phrase,
				Force:      force,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Generated key: %s\n", info.PrivateKeyPath)
			fmt.Fprintf(cmd.OutOrStdout(), "Public key: %s\n", info.PublicKeyPath)
			return nil
		},
	}

	cmd.Flags().StringVarP(&keyType, "type", "t", string(sshkeys.KeyTypeED25519), "Key type: ed25519 or rsa")
	cmd.Flags().StringVarP(&comment, "comment", "c", "", "Comment embedded into public key")
	cmd.Flags().IntVar(&bits, "bits", 4096, "RSA bits (used only with --type rsa)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing key files")
	cmd.Flags().StringVar(&passphrase, "passphrase", "", "Passphrase value (unsafe for shell history)")
	cmd.Flags().StringVar(&passphraseEnv, "passphrase-env", "", "Environment variable name containing passphrase")
	return cmd
}

func newImportCommand() *cobra.Command {
	var (
		sourcePath string
		overwrite  bool
	)
	cmd := &cobra.Command{
		Use:   "import <name>",
		Short: "Import existing private/public key pair",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}
			info, err := manager.Import(args[0], sourcePath, overwrite)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported key: %s\n", info.PrivateKeyPath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&sourcePath, "from", "f", "", "Path to source private key")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "Overwrite destination key if exists")
	_ = cmd.MarkFlagRequired("from")
	return cmd
}

func newPublicCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "public <name>",
		Short: "Print public key line",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}
			pub, err := manager.PublicKey(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), pub)
			return nil
		},
	}
}

func newFingerprintCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "fingerprint <name>",
		Short: "Print key fingerprint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}
			fp, err := manager.Fingerprint(args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), fp)
			return nil
		},
	}
}

func newAgentCommand() *cobra.Command {
	agent := &cobra.Command{
		Use:   "agent",
		Short: "Interact with ssh-agent",
	}

	agent.AddCommand(
		&cobra.Command{
			Use:   "add <name>",
			Short: "Add managed key to ssh-agent",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				manager, err := sshkeys.NewManager()
				if err != nil {
					return err
				}
				if err := manager.AddToAgent(args[0]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Added to agent: %s\n", args[0])
				return nil
			},
		},
		&cobra.Command{
			Use:   "remove <name>",
			Short: "Remove managed key from ssh-agent",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				manager, err := sshkeys.NewManager()
				if err != nil {
					return err
				}
				if err := manager.RemoveFromAgent(args[0]); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Removed from agent: %s\n", args[0])
				return nil
			},
		},
	)

	return agent
}

func newDeleteCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete managed key files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				ok, err := askConfirmation(cmd, "Delete key "+args[0]+"? [y/N]: ")
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}
			if err := manager.Delete(args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted key: %s\n", args[0])
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func resolvePassphrase(direct, envName string) (string, error) {
	if direct != "" && envName != "" {
		return "", errors.New("use either --passphrase or --passphrase-env, not both")
	}
	if direct != "" {
		return direct, nil
	}
	if envName != "" {
		val := os.Getenv(envName)
		if val == "" {
			return "", fmt.Errorf("environment variable %q is empty or missing", envName)
		}
		return val, nil
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return "", nil
	}
	return "", nil
}

func askConfirmation(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprint(cmd.OutOrStdout(), prompt)
	var answer string
	_, err := fmt.Fscanln(cmd.InOrStdin(), &answer)
	if err != nil {
		if errors.Is(err, os.ErrClosed) {
			return false, err
		}
		// Empty input means default "no".
		if strings.Contains(err.Error(), "unexpected newline") {
			return false, nil
		}
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func parseTags(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func shortFingerprint(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return "n/a"
	}
	if len(full) <= 64 {
		return full
	}
	return full[:64] + "..."
}

func detectUnclearNames(keys []sshkeys.DiscoveredKey) []string {
	ambiguous := make(map[string]struct{})
	for _, key := range keys {
		base := strings.ToLower(strings.TrimSpace(key.BaseName))
		switch base {
		case "id_rsa", "id_ed25519", "id_ecdsa", "id_dsa", "key", "ssh_key", "private_key":
			ambiguous[key.BaseName] = struct{}{}
		}
	}
	out := make([]string, 0, len(ambiguous))
	for value := range ambiguous {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
