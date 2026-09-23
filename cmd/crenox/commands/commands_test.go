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

	runGit("init", "-b", "master")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	runGit("config", "core.hooksPath", "")

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

func TestScanCmd_MultipleFilesWithSameSecret_AllReported(t *testing.T) {
	tmpDir := t.TempDir()
	f1 := filepath.Join(tmpDir, "file1.txt")
	f2 := filepath.Join(tmpDir, "file2.txt")
	secretContent := []byte("aws_key = \"AKIAIOSFODNN7EXAMPLE\"\n")
	if err := os.WriteFile(f1, secretContent, 0644); err != nil {
		t.Fatalf("write f1: %v", err)
	}
	if err := os.WriteFile(f2, secretContent, 0644); err != nil {
		t.Fatalf("write f2: %v", err)
	}

	oldExit := exitFunc
	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = oldExit }()

	outFile := filepath.Join(tmpDir, "report.json")
	err := runAdHocScan([]string{f1, f2}, "", "json", false, false, false, outFile, false)
	if err != nil {
		t.Fatalf("runAdHocScan error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	reportBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}
	reportStr := string(reportBytes)

	if !strings.Contains(reportStr, "file1.txt") {
		t.Errorf("expected report to contain file1.txt, got: %s", reportStr)
	}
	if !strings.Contains(reportStr, "file2.txt") {
		t.Errorf("expected report to contain file2.txt, got: %s", reportStr)
	}
}

func TestCommands_RejectUnexpectedArguments(t *testing.T) {
	// 1. Run command rejects unexpected arguments
	runCmd := NewRunCmd()
	runCmd.SetArgs([]string{"extra-arg"})
	if err := runCmd.Execute(); err == nil {
		t.Errorf("expected NewRunCmd to reject unexpected arguments, got nil")
	}

	// 2. Install command rejects unexpected arguments
	installCmd := NewInstallCmd()
	installCmd.SetArgs([]string{"unexpected-arg"})
	if err := installCmd.Execute(); err == nil {
		t.Errorf("expected NewInstallCmd to reject unexpected arguments, got nil")
	}

	// 3. Update command rejects unexpected arguments
	updateCmd := NewUpdateCmd()
	updateCmd.SetArgs([]string{"unexpected-arg"})
	if err := updateCmd.Execute(); err == nil {
		t.Errorf("expected NewUpdateCmd to reject unexpected arguments, got nil")
	}
}

func TestUpdateCmd_Hex64Regex(t *testing.T) {
	validHashes := []string{
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		"E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
		"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	for _, h := range validHashes {
		if !hex64Regex.MatchString(h) {
			t.Errorf("expected valid hash %q to match hex64Regex", h)
		}
	}

	invalidHashes := []string{
		"",                                                                    // empty
		"e3b0c442",                                                            // too short
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855aa", // too long (66 chars)
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b85g", // non-hex char 'g'
		"not-a-hash",
	}
	for _, h := range invalidHashes {
		if hex64Regex.MatchString(h) {
			t.Errorf("expected invalid hash %q to be rejected by hex64Regex", h)
		}
	}
}

func TestScanCmd_FailClosedOnUnreadableFile(t *testing.T) {
	tmpDir := t.TempDir()
	cleanFile := filepath.Join(tmpDir, "clean.txt")
	if err := os.WriteFile(cleanFile, []byte("clean regular text\n"), 0644); err != nil {
		t.Fatalf("write clean.txt: %v", err)
	}

	unreadableFile := filepath.Join(tmpDir, "locked.txt")
	if err := os.WriteFile(unreadableFile, []byte("secret text\n"), 0644); err != nil {
		t.Fatalf("write locked.txt: %v", err)
	}
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Skip("skipping test: chmod 0000 not supported in this filesystem")
	}
	defer os.Chmod(unreadableFile, 0644)

	oldExit := exitFunc
	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = oldExit }()

	outFile := filepath.Join(tmpDir, "report.json")
	err := runAdHocScan([]string{tmpDir}, "", "json", true, false, false, outFile, false)
	if err != nil {
		t.Fatalf("runAdHocScan returned unexpected Go error: %v", err)
	}

	// Must fail-closed with exitCode 1
	if exitCode != 1 {
		t.Errorf("expected exit code 1 on unreadable file, got %d", exitCode)
	}

	reportBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read report file: %v", err)
	}
	reportStr := string(reportBytes)

	// Must report status "scan_error" and record the failed file
	if !strings.Contains(reportStr, `"scan_error"`) {
		t.Errorf("expected report status 'scan_error', got: %s", reportStr)
	}
	if !strings.Contains(reportStr, "locked.txt") {
		t.Errorf("expected report to contain unreadable file locked.txt, got: %s", reportStr)
	}
}

