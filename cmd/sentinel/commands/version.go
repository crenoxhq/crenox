package commands

import (
	"fmt"

	"github.com/sentinel-cli/sentinel/pkg/version"
	"github.com/spf13/cobra"
)

// NewVersionCmd builds the `sentinel version` sub-command.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print Sentinel version and build metadata",
		Long: `Print build and version metadata for the Sentinel executable, including the version tag, commit hash, build date, and developer contact.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Sentinel version\n")
			fmt.Printf("sentinel %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
			fmt.Printf("Developed by: Khaled Hani | Contact: https://t.me/A245F\n")
		},
	}
}
