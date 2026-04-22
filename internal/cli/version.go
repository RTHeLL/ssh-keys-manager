package cli

import (
	"fmt"

	"github.com/RTHeLL/ssh-keys-manager/internal/buildinfo"
	"github.com/spf13/cobra"
)

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print build version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "version=%s commit=%s date=%s\n", buildinfo.Version, buildinfo.Commit, buildinfo.Date)
		},
	}
}
