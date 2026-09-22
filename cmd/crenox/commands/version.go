package commands

import (
	"fmt"

	"github.com/crenoxhq/crenox/v2/pkg/version"
	"github.com/spf13/cobra"
)

// NewVersionCmd builds the `crenox version` sub-command.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Crenox version and build metadata",
		Long:  `Print build and version metadata for the Crenox executable, including the version tag, commit hash, build date, and developer contact.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Crenox version\n")
			fmt.Printf("crenox %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
			fmt.Printf("Maintained by: CrenoxHQ | https://github.com/crenoxhq/crenox\n")
		},
	}
}
