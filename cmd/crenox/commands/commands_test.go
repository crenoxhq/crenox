package commands

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewVersionCmd(t *testing.T) {
	cmd := NewVersionCmd()

	if cmd.Use != "version" {
		t.Errorf("expected Use 'version', got %q", cmd.Use)
	}

	if cmd.Short == "" || cmd.Long == "" {
		t.Errorf("expected Short and Long descriptions to be populated")
	}

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	// Execute the command without arguments
	cmd.SetArgs([]string{})
	err := cmd.Execute()

	if err != nil {
		t.Fatalf("unexpected error running version command: %v", err)
	}

	// We can't capture the fmt.Printf easily without os.Stdout redirection,
	// but we can ensure it executes cleanly without panic.
}

func TestNewScanCmd(t *testing.T) {
	cmd := NewScanCmd()

	if cmd.Use != "scan [path...]" {
		t.Errorf("expected Use 'scan [path...]', got %q", cmd.Use)
	}

	// Verify flags are registered
	flags := []string{"config", "format", "recursive", "verbose", "history", "output", "fail-fast"}
	for _, f := range flags {
		if flag := cmd.Flag(f); flag == nil {
			t.Errorf("expected flag %q to be defined", f)
		}
	}
}

func TestNewRunCmd(t *testing.T) {
	cmd := NewRunCmd()

	if cmd.Use != "run" {
		t.Errorf("expected Use 'run', got %q", cmd.Use)
	}

	// Verify flags are registered
	flags := []string{"config", "format", "verbose", "fail-fast"}
	for _, f := range flags {
		if flag := cmd.Flag(f); flag == nil {
			t.Errorf("expected flag %q to be defined", f)
		}
	}
}

func TestNewUpdateCmd(t *testing.T) {
	cmd := NewUpdateCmd()

	if !strings.HasPrefix(cmd.Use, "update") {
		t.Errorf("expected Use to start with 'update', got %q", cmd.Use)
	}
}

func TestNewInstallCmd(t *testing.T) {
	cmd := NewInstallCmd()

	if !strings.HasPrefix(cmd.Use, "install") {
		t.Errorf("expected Use to start with 'install', got %q", cmd.Use)
	}
}

func TestNewUninstallCmd(t *testing.T) {
	cmd := NewUninstallCmd()

	if !strings.HasPrefix(cmd.Use, "uninstall") {
		t.Errorf("expected Use to start with 'uninstall', got %q", cmd.Use)
	}
}

func TestStripANSI(t *testing.T) {
	input := []byte("\x1b[32m+apiKey := \"sk_live_1234567890abcdef\"\x1b[m")
	want := []byte("+apiKey := \"sk_live_1234567890abcdef\"")
	got := stripANSI(input)
	if !bytes.Equal(got, want) {
		t.Errorf("stripANSI() = %q; want %q", string(got), string(want))
	}
}

func TestScanCmd_NonExistentPath(t *testing.T) {
	err := runAdHocScan([]string{"/tmp/path-that-does-not-exist-123456"}, "", "pretty", false, false, false, "", false)
	if err == nil {
		t.Fatalf("expected error for non-existent path, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist or is inaccessible") {
		t.Errorf("expected error message mentioning 'does not exist or is inaccessible', got: %v", err)
	}
}

func TestScanCmd_InvalidFormat(t *testing.T) {
	err := runAdHocScan([]string{"."}, "", "nope", false, false, false, "", false)
	if err == nil {
		t.Fatalf("expected error for invalid format 'nope', got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format: nope") {
		t.Errorf("expected error message mentioning 'unsupported output format: nope', got: %v", err)
	}
}

func TestRunCmd_InvalidFormat(t *testing.T) {
	err := runScan("", "nope", false, false)
	if err == nil {
		t.Fatalf("expected error for invalid format 'nope', got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format: nope") {
		t.Errorf("expected error message mentioning 'unsupported output format: nope', got: %v", err)
	}
}

func TestScanCmd_History_NonGitRepo(t *testing.T) {
	tmpDir := t.TempDir()
	err := runAdHocScan([]string{tmpDir}, "", "pretty", false, false, true, "", false)
	if err == nil {
		t.Fatalf("expected error running --history on non-git directory, got nil")
	}
	if !strings.Contains(err.Error(), "is not a git repository") {
		t.Errorf("expected error to mention 'is not a git repository', got: %v", err)
	}
}

func TestScanCmd_History_GitRepo(t *testing.T) {
	tmpDir := t.TempDir()

	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = tmpDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v, out: %s", args, err, string(out))
		}
	}

	runGit("init")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")

	// Commit 1: Secret in regular file
	f1 := filepath.Join(tmpDir, "file1.txt")
	if err := os.WriteFile(f1, []byte("AKIAIOSFODNN7EXAMPLE\n"), 0644); err != nil {
		t.Fatalf("write file1: %v", err)
	}
	runGit("add", "file1.txt")
	runGit("commit", "-m", "initial commit with secret")

	// Commit 2: Secret in commit message
	cleanFile := filepath.Join(tmpDir, "clean.txt")
	if err := os.WriteFile(cleanFile, []byte("clean content\n"), 0644); err != nil {
		t.Fatalf("write clean.txt: %v", err)
	}
	runGit("add", "clean.txt")
	runGit("commit", "-m", "fix: updated token to AKIAIOSFODNN7EXAMPLB")

	// Commit 3 & 4 on separate branches, then merge commit with resolution secret
	runGit("checkout", "-b", "feature")
	f2 := filepath.Join(tmpDir, "feature.txt")
	if err := os.WriteFile(f2, []byte("feature branch text\n"), 0644); err != nil {
		t.Fatalf("write feature.txt: %v", err)
	}
	runGit("add", "feature.txt")
	runGit("commit", "-m", "feature commit")

	runGit("checkout", "master")
	f3 := filepath.Join(tmpDir, "feature.txt")
	if err := os.WriteFile(f3, []byte("master branch text\n"), 0644); err != nil {
		t.Fatalf("write master feature.txt: %v", err)
	}
	runGit("add", "feature.txt")
	runGit("commit", "-m", "master commit")

	// Merge feature into master (conflict)
	cmd := exec.Command("git", "merge", "--no-ff", "feature", "-m", "merge conflict")
	cmd.Dir = tmpDir
	_ = cmd.Run()

	// Resolve conflict with a secret
	if err := os.WriteFile(f3, []byte("resolved: AKIAIOSFODNN7EXAMPLC\n"), 0644); err != nil {
		t.Fatalf("write resolved feature.txt: %v", err)
	}
	runGit("add", "feature.txt")
	runGit("commit", "-m", "Merge branch feature with secret resolution")

	// Intercept exitFunc so the test runner doesn't exit
	oldExit := exitFunc
	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = oldExit }()

	// Output report to a temp file
	outFile := filepath.Join(tmpDir, "report.json")
	err := runAdHocScan([]string{tmpDir}, "", "json", true, false, true, outFile, false)
	if err != nil {
		t.Fatalf("runAdHocScan error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1 due to detected secrets, got %d", exitCode)
	}

	reportBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}
	reportStr := string(reportBytes)

	// Verify all 3 secrets are reported in JSON
	if !strings.Contains(reportStr, "AKIAIO**********MPLE") {
		t.Errorf("expected commit 1 secret in history scan report: %s", reportStr)
	}
	if !strings.Contains(reportStr, "AKIAIO**********MPLB") {
		t.Errorf("expected commit message secret in history scan report: %s", reportStr)
	}
	if !strings.Contains(reportStr, "AKIAIO**********MPLC") {
		t.Errorf("expected merge commit secret in history scan report: %s", reportStr)
	}
}
