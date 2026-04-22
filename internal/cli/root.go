package cli

import (
	"errors"
	"fmt"
	"os"
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
	return &cobra.Command{
		Use:   "list",
		Short: "List managed keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager, err := sshkeys.NewManager()
			if err != nil {
				return err
			}
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
		},
	}
}

func newGenerateCommand() *cobra.Command {
	var (
		keyType      string
		comment      string
		bits         int
		force        bool
		passphrase   string
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
