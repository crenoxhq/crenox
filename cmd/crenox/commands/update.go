package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var hex64Regex = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

func NewUpdateCmd() *cobra.Command {
	var (
		allowBeta  bool
		skipVerify bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update Crenox to the latest stable version over-the-air",
		Long: `Check GitHub Releases for the latest version of Crenox and automatically update the active binary.
This command performs:
  1. Automatic detection of your operating system and CPU architecture.
  2. Querying the GitHub Releases API for the latest release tag.
  3. Downloading the binary and verifying its cryptographic SHA-256 checksum.
  4. Atomically replacing the active 'crenox' executable without service disruption.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Updating Crenox to the latest version...")

			// 1. Detect environment
			goos := runtime.GOOS
			goarch := runtime.GOARCH

			// On Android/Termux, the binary may be compiled with GOOS=linux
			// but the device is actually Android. Detect this by checking
			// for Termux-specific paths or the ANDROID_ROOT env variable.
			if goos == "linux" {
				if _, err := os.Stat("/data/data/com.termux"); err == nil {
					goos = "android"
				} else if os.Getenv("ANDROID_ROOT") != "" || os.Getenv("TERMUX_VERSION") != "" {
					goos = "android"
				}
			}

			exePath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("could not determine executable path: %w", err)
			}
			if absPath, err := filepath.EvalSymlinks(exePath); err == nil {
				exePath = absPath
			}

			// 2. Query GitHub Releases
			fmt.Println("Checking GitHub for the latest release...")

			// Try standard HTTP client first, falling back to Google DNS resolver if system resolution fails
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Get("https://api.github.com/repos/crenoxhq/crenox/releases")
			if err != nil {
				fallbackClient := &http.Client{
					Timeout: 15 * time.Second,
					Transport: &http.Transport{
						DialContext: (&net.Dialer{
							Resolver: &net.Resolver{
								PreferGo: true,
								Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
									d := net.Dialer{}
									return d.DialContext(ctx, "udp", "8.8.8.8:53")
								},
							},
						}).DialContext,
					},
				}
				resp, err = fallbackClient.Get("https://api.github.com/repos/crenoxhq/crenox/releases")
				if err != nil {
					return fmt.Errorf("failed to reach github: %w", err)
				}
				client = fallbackClient
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to get latest release: http status %d", resp.StatusCode)
			}

			type ReleaseInfo struct {
				TagName    string `json:"tag_name"`
				Prerelease bool   `json:"prerelease"`
				Assets     []struct {
					Name               string `json:"name"`
					BrowserDownloadURL string `json:"browser_download_url"`
				} `json:"assets"`
			}
			var releases []ReleaseInfo

			if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
				return fmt.Errorf("failed to decode release JSON: %w", err)
			}

			var release *ReleaseInfo
			for i, r := range releases {
				isBeta := r.Prerelease || strings.Contains(r.TagName, "-beta") || strings.Contains(r.TagName, "-rc")
				if allowBeta || !isBeta {
					release = &releases[i]
					break
				}
			}

			if release == nil {
				return fmt.Errorf("no matching release found")
			}

			var downloadURL string
			var downloadName string
			for _, asset := range release.Assets {
				lowerName := strings.ToLower(asset.Name)
				hasOS := strings.Contains(lowerName, goos) || (goos == "darwin" && strings.Contains(lowerName, "macos"))
				hasArch := strings.Contains(lowerName, goarch) || (goarch == "amd64" && strings.Contains(lowerName, "x86_64")) || (goarch == "386" && strings.Contains(lowerName, "i386"))

				if hasOS && hasArch {
					// We prefer raw binaries, avoid compressed archives for direct replacement.
					if !strings.HasSuffix(lowerName, ".tar.gz") && !strings.HasSuffix(lowerName, ".zip") && !strings.HasSuffix(lowerName, ".sha256") {
						downloadURL = asset.BrowserDownloadURL
						downloadName = asset.Name
						break
					}
				}
			}

			var sha256URL string
			if downloadURL != "" {
				for _, asset := range release.Assets {
					if asset.Name == downloadName+".sha256" {
						sha256URL = asset.BrowserDownloadURL
						break
					}
				}
			}

			// Fallback to go install if no raw binary is available.
			if downloadURL == "" {
				fmt.Printf("No matching pre-compiled binary found for %s/%s. Falling back to 'go install'...\n", goos, goarch)
				c := exec.Command("go", "install", "github.com/crenoxhq/crenox/v2/cmd/crenox@latest")
				c.Stdout = os.Stdout
				c.Stderr = os.Stderr
				if err := c.Run(); err != nil {
					return fmt.Errorf("update failed: %w", err)
				}
				fmt.Println("✔ Crenox successfully updated to the latest version!")
				return nil
			}

			fmt.Printf("Found binary for %s/%s. Downloading %s...\n", goos, goarch, release.TagName)

			// 3. Safe Binary Replacement
			tmpPath := exePath + ".tmp"

			// Download to temporary file
			out, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return fmt.Errorf("failed to create temporary file: %w", err)
			}

			dlResp, err := client.Get(downloadURL)
			if err != nil {
				out.Close()
				os.Remove(tmpPath)
				return fmt.Errorf("failed to download binary: %w", err)
			}
			defer dlResp.Body.Close()

			if dlResp.StatusCode != http.StatusOK {
				out.Close()
				os.Remove(tmpPath)
				return fmt.Errorf("download failed with status: %d", dlResp.StatusCode)
			}

			if _, err := io.Copy(out, dlResp.Body); err != nil {
				out.Close()
				os.Remove(tmpPath)
				return fmt.Errorf("failed to write binary to disk: %w", err)
			}
			out.Close()

			if skipVerify {
				fmt.Println("WARNING: Bypassing SHA-256 cryptographic verification via --skip-verify.")
			} else {
				if sha256URL == "" {
					os.Remove(tmpPath)
					return fmt.Errorf("SHA-256 checksum asset not found for release %s (use --skip-verify to override)", release.TagName)
				}

				fmt.Println("Verifying SHA-256 checksum...")
				shaResp, err := client.Get(sha256URL)
				if err != nil || shaResp.StatusCode != http.StatusOK {
					os.Remove(tmpPath)
					errMsg := "could not fetch SHA-256 checksum file"
					if err != nil {
						errMsg = fmt.Sprintf("failed to fetch SHA-256 checksum file: %v", err)
					} else {
						errMsg = fmt.Sprintf("checksum download failed with HTTP status: %d", shaResp.StatusCode)
						shaResp.Body.Close()
					}
					return fmt.Errorf("%s (use --skip-verify to override)", errMsg)
				}
				defer shaResp.Body.Close()

				shaBytes, err := io.ReadAll(shaResp.Body)
				if err != nil {
					os.Remove(tmpPath)
					return fmt.Errorf("failed to read SHA-256 checksum file: %w", err)
				}

				fields := strings.Fields(string(shaBytes))
				if len(fields) == 0 {
					os.Remove(tmpPath)
					return fmt.Errorf("SHA-256 checksum file format is empty or invalid (use --skip-verify to override)")
				}

				expectedHash := strings.ToLower(fields[0])
				if !hex64Regex.MatchString(expectedHash) {
					os.Remove(tmpPath)
					return fmt.Errorf("invalid SHA-256 checksum format: expected 64 hex characters, got %q (use --skip-verify to override)", expectedHash)
				}

				f, err := os.Open(tmpPath)
				if err != nil {
					os.Remove(tmpPath)
					return fmt.Errorf("failed to open downloaded binary for checksum verification: %w", err)
				}
				h := sha256.New()
				if _, err := io.Copy(h, f); err != nil {
					f.Close()
					os.Remove(tmpPath)
					return fmt.Errorf("failed to compute checksum: %w", err)
				}
				f.Close()

				actualHash := strings.ToLower(hex.EncodeToString(h.Sum(nil)))
				if actualHash != expectedHash {
					os.Remove(tmpPath)
					return fmt.Errorf("cryptographic checksum mismatch: expected %s, got %s", expectedHash, actualHash)
				}
				fmt.Println("✔ Cryptographic SHA-256 integrity verified")
			}

			// Overwrite running executable atomically
			if err := os.Rename(tmpPath, exePath); err != nil {
				os.Remove(tmpPath)
				return fmt.Errorf("failed to safely replace binary (text file busy?): %w", err)
			}

			fmt.Printf("✔ Crenox successfully updated to %s!\n", release.TagName)
			return nil
		},
	}
	cmd.Flags().BoolVar(&allowBeta, "beta", false, "Allow updating to pre-release (beta) versions")
	cmd.Flags().BoolVar(&skipVerify, "skip-verify", false, "Bypass SHA-256 cryptographic checksum verification")
	return cmd
}
