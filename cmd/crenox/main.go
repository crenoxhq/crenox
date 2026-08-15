// Package main implements the Crenox CLI — a `cobra`-based command dispatcher
// that exposes:
//
//	crenox run     — the core pre-commit hook (default)
//	crenox install — install the hook into a git repository
//	crenox version — print build metadata
//	crenox scan    — scan an arbitrary file or directory (ad-hoc mode)
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crenoxhq/crenox/v2/cmd/crenox/commands"
	"github.com/crenoxhq/crenox/v2/pkg/version"
)

func init() {
	// Termux self-healing mechanism for SSL certificates.
	// Go's crypto/x509 does not natively check Termux's custom certificate path.
	// If we are in Termux (e.g. the path exists or PREFIX is set),
	// and SSL_CERT_FILE is not already overridden by the user, inject it automatically.
	termuxCertPath := "/data/data/com.termux/files/usr/etc/tls/cert.pem"
	if os.Getenv("SSL_CERT_FILE") == "" {
		if _, err := os.Stat(termuxCertPath); err == nil {
			os.Setenv("SSL_CERT_FILE", termuxCertPath)
		} else if strings.Contains(os.Getenv("PREFIX"), "com.termux") {
			os.Setenv("SSL_CERT_FILE", termuxCertPath)
		}
	}
}

func main() {
	root := &cobra.Command{
		Use:   "crenox",
		Short: "Crenox — Git secret scanner and pre-commit hook",
		Long: `Crenox is a statically compiled Git pre-commit hook and credentials scanner.
It scans source code to prevent commits containing API keys, private keys, passwords,
and sensitive tokens using a three-tier pipeline:

  Tier 1 (PATTERN)   — Aho-Corasick automaton matching 125+ secret signatures in O(n) time
  Tier 2 (ENTROPY)   — Shannon entropy analysis with Look-Up Table (LUT) acceleration
  Tier 3 (CONTEXT)   — Context classifier suppressing false positives from comments and tests

Commands:
  crenox run                  Run pre-commit scan on staged files (invoked by Git hook).
  crenox scan [path...]       Scan arbitrary files, directories, or full Git commit history.
  crenox install              Install the pre-commit hook (local repository or --global).
  crenox uninstall            Remove Crenox binary, hooks, and configuration.
  crenox update               Update Crenox to the latest release version.
  crenox version              Print version and build metadata.

Flags:
  -c, --config string         Path to .crenox.yaml configuration file.
  -f, --format string         Output format: pretty | json | plain | sarif | gitlab-sast (default "pretty").
  -v, --verbose               Enable verbose diagnostic output.
  -h, --help                  Help for crenox.

Developed by: Khaled Hani | Contact: https://t.me/A245F`,
		Version:       version.Version,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.SetVersionTemplate(`Crenox version
{{printf "crenox %s" .Version}} (commit: ` + version.Commit + `, built: ` + version.Date + `)
Developed by: Khaled Hani | Contact: https://t.me/A245F
`)

	cmds := []*cobra.Command{
		commands.NewRunCmd(),
		commands.NewInstallCmd(),
		commands.NewScanCmd(),
		commands.NewUpdateCmd(),
		commands.NewUninstallCmd(),
		commands.NewVersionCmd(),
	}
	// Dashboard is only available in builds compiled with -tags dashboard.
	if dashCmd := commands.NewDashboardCmd(); dashCmd != nil {
		cmds = append(cmds, dashCmd)
	}
	root.AddCommand(cmds...)

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "crenox: error:", err)
		os.Exit(1)
	}
}
