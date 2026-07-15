package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

func NewUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Completely remove Crenox from the system",
		Long: `Completely remove Crenox, its pre-commit hooks, and global configurations from your system.
This command performs the following cleanup steps:
  1. Unsets the global git config 'core.hooksPath' if it was configured for Crenox.
  2. Deletes the 'crenox' executable binary from your system path (unless managed by a package manager).
  3. Removes the global configuration and hook folder located at '~/.config/crenox'.
  4. Deletes the local pre-commit hook file '.git/hooks/pre-commit' in the current working directory.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Uninstalling Crenox...")

			// a) Unset global hooks path
			exec.Command("git", "config", "--global", "--unset", "core.hooksPath").Run()

			// Check if binary is managed by a package manager
			var pmName, pmCmd string
			exePath, err := os.Executable()
			if err == nil {
				// Ensure absolute path
				absPath, err := filepath.EvalSymlinks(exePath)
				if err == nil {
					exePath = absPath
				}
				pmName, pmCmd = isManagedByPackageManager(exePath)
			}

			// b) Remove the binary itself
			removedBin := false
			if pmName != "" {
				fmt.Printf("⚠️  Crenox is managed by the %s package manager.\n", pmName)
				fmt.Printf("   To uninstall the binary cleanly, please run:\n")
				fmt.Printf("       \033[1;36m%s\033[0m\n\n", pmCmd)
				removedBin = true // skipped because managed
			} else if exePath != "" {
				if err := os.Remove(exePath); err == nil {
					removedBin = true
					fmt.Printf("Removed binary: %s\n", exePath)
				}
			}

			if !removedBin {
				fmt.Printf("Could not automatically remove the crenox binary. You may need to remove it manually from your PATH.\n")
			}

			// c) Remove global hooks directory
			homeDir, err := os.UserHomeDir()
			if err == nil {
				os.RemoveAll(filepath.Join(homeDir, ".config", "crenox"))
			}

			// d) Remove local pre-commit hook
			os.Remove(".git/hooks/pre-commit")

			fmt.Println("✔ Crenox hooks and configurations have been successfully removed.")
			return nil
		},
	}
}

func isManagedByPackageManager(exePath string) (string, string) {
	// 1. Check dpkg/apt (Debian, Ubuntu, Termux)
	if _, err := exec.LookPath("dpkg"); err == nil {
		cmd := exec.Command("dpkg", "-S", exePath)
		if err := cmd.Run(); err == nil {
			if _, err := exec.LookPath("pkg"); err == nil {
				return "pkg", "pkg uninstall crenox"
			}
			return "dpkg/apt", "sudo apt remove crenox"
		}
	}

	// 2. Check Homebrew (macOS / Linux)
	if _, err := exec.LookPath("brew"); err == nil {
		cmd := exec.Command("brew", "list", "crenox")
		if err := cmd.Run(); err == nil {
			return "Homebrew", "brew uninstall crenox"
		}
	}

	// 3. Check Pacman (Arch Linux)
	if _, err := exec.LookPath("pacman"); err == nil {
		cmd := exec.Command("pacman", "-Qo", exePath)
		if err := cmd.Run(); err == nil {
			return "pacman", "sudo pacman -R crenox"
		}
	}

	// 4. Check RPM (Fedora, RHEL, CentOS)
	if _, err := exec.LookPath("rpm"); err == nil {
		cmd := exec.Command("rpm", "-qf", exePath)
		if err := cmd.Run(); err == nil {
			return "rpm", "sudo dnf remove crenox"
		}
	}

	return "", ""
}
