package commands

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestScanCmd_SimultaneousFindingAndUnreadableFile_SingleJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. File with finding
	findingFile := filepath.Join(tmpDir, "leak.txt")
	if err := os.WriteFile(findingFile, []byte("export AWS_ACCESS_KEY_ID=\"AKIAIOSFODNN7EXAMPLE\"\n"), 0644); err != nil {
		t.Fatalf("write leak.txt: %v", err)
	}

	// 2. Unreadable file
	unreadableFile := filepath.Join(tmpDir, "unreadable.key")
	if err := os.WriteFile(unreadableFile, []byte("super secret binary\n"), 0644); err != nil {
		t.Fatalf("write unreadable.key: %v", err)
	}
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Skip("skipping test: chmod 0000 not supported")
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
		t.Fatalf("runAdHocScan returned unexpected error: %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	reportBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read report file: %v", err)
	}

	// Verify report is valid single JSON object (no dual JSON collision)
	var report map[string]interface{}
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatalf("dual JSON collision detected! Output is not valid single JSON: %v, raw:\n%s", err, string(reportBytes))
	}

	// Status must be scan_error due to fail-closed policy
	if report["status"] != "scan_error" {
		t.Errorf("expected status 'scan_error', got %v", report["status"])
	}

	// Both findings and failed files must be populated in the same JSON object
	findings, ok := report["findings"].([]interface{})
	if !ok || len(findings) == 0 {
		t.Errorf("expected findings to be present, got %v", report["findings"])
	}

	failedFiles, ok := report["failed_files"].([]interface{})
	if !ok || len(failedFiles) == 0 {
		t.Errorf("expected failed_files to be present, got %v", report["failed_files"])
	}
}

func TestScanCmd_SARIF_ScanError(t *testing.T) {
	tmpDir := t.TempDir()

	unreadableFile := filepath.Join(tmpDir, "locked.env")
	if err := os.WriteFile(unreadableFile, []byte("SECRET=123\n"), 0644); err != nil {
		t.Fatalf("write locked.env: %v", err)
	}
	if err := os.Chmod(unreadableFile, 0000); err != nil {
		t.Skip("skipping test: chmod 0000 not supported")
	}
	defer os.Chmod(unreadableFile, 0644)

	oldExit := exitFunc
	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = oldExit }()

	outFile := filepath.Join(tmpDir, "report.sarif")
	err := runAdHocScan([]string{tmpDir}, "", "sarif", true, false, false, outFile, false)
	if err != nil {
		t.Fatalf("runAdHocScan returned unexpected error: %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	reportBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read report file: %v", err)
	}

	var sarif map[string]interface{}
	if err := json.Unmarshal(reportBytes, &sarif); err != nil {
		t.Fatalf("failed to parse SARIF JSON on scan_error: %v, raw:\n%s", err, string(reportBytes))
	}

	runs := sarif["runs"].([]interface{})
	if len(runs) == 0 {
		t.Fatalf("expected runs in SARIF")
	}
	run := runs[0].(map[string]interface{})
	invocations, ok := run["invocations"].([]interface{})
	if !ok || len(invocations) == 0 {
		t.Fatalf("expected invocations in SARIF scan_error run")
	}
	inv := invocations[0].(map[string]interface{})
	if inv["executionSuccessful"] != false {
		t.Errorf("expected executionSuccessful false in SARIF on scan_error")
	}
}

func TestScanCmd_History_GitLogFailure_EmitsScanError(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize git repo and commit a file
	runGit := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = tmpDir
		c.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		_ = c.Run()
	}
	runGit("init", "-b", "main")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	runGit("config", "core.hooksPath", "")

	_ = os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("content\n"), 0644)
	runGit("add", "file.txt")
	runGit("commit", "-m", "init")

	// Corrupt ref so git log fails with exit code 128 (bad object)
	_ = os.WriteFile(filepath.Join(tmpDir, ".git", "refs", "heads", "main"), []byte("0123456789abcdef0123456789abcdef01234567\n"), 0644)

	oldExit := exitFunc
	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = oldExit }()

	outFile := filepath.Join(tmpDir, "report.json")
	err := runAdHocScan([]string{tmpDir}, "", "json", false, false, true, outFile, false)
	if err != nil {
		t.Fatalf("unexpected Go error from scan: %v", err)
	}

	if exitCode != 1 {
		t.Errorf("expected exit code 1 on git history failure, got %d", exitCode)
	}

	reportBytes, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read report file: %v", err)
	}

	var report map[string]interface{}
	if err := json.Unmarshal(reportBytes, &report); err != nil {
		t.Fatalf("failed to parse JSON from git log failure: %v, raw:\n%s", err, string(reportBytes))
	}

	if report["status"] != "scan_error" {
		t.Errorf("expected status 'scan_error' on git log failure, got %v", report["status"])
	}
}

func TestUpdate_MissingChecksum_AbortsAndCleansUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/binary" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-binary-content"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "crenox-bin")

	err := verifyAndInstallBinary(server.Client(), targetPath, server.URL+"/binary", "", false)
	if err == nil {
		t.Fatalf("expected error when sha256 checksum asset is missing, got nil")
	}
	if !strings.Contains(err.Error(), "checksum asset not found") {
		t.Errorf("expected error message about checksum asset not found, got: %v", err)
	}

	// Verify temp file was cleaned up
	tmpPath := targetPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("expected temp file %s to be deleted after abort, but it exists", tmpPath)
	}
}

func TestUpdate_MalformedChecksum_AbortsAndCleansUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/binary" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-binary-content"))
			return
		}
		if r.URL.Path == "/sha256" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-a-valid-64-hex-hash"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "crenox-bin")

	err := verifyAndInstallBinary(server.Client(), targetPath, server.URL+"/binary", server.URL+"/sha256", false)
	if err == nil {
		t.Fatalf("expected error on malformed checksum, got nil")
	}
	if !strings.Contains(err.Error(), "invalid SHA-256 checksum format") {
		t.Errorf("expected invalid SHA-256 format error, got: %v", err)
	}

	tmpPath := targetPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("expected temp file %s to be deleted after abort, but it exists", tmpPath)
	}
}

func TestUpdate_MismatchedChecksum_AbortsAndCleansUp(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/binary" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("fake-binary-content"))
			return
		}
		if r.URL.Path == "/sha256" {
			w.WriteHeader(http.StatusOK)
			// Valid 64-hex format, but completely wrong hash
			_, _ = w.Write([]byte("0000000000000000000000000000000000000000000000000000000000000000  binary"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "crenox-bin")

	err := verifyAndInstallBinary(server.Client(), targetPath, server.URL+"/binary", server.URL+"/sha256", false)
	if err == nil {
		t.Fatalf("expected error on mismatched checksum, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("expected checksum mismatch error, got: %v", err)
	}

	tmpPath := targetPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("expected temp file %s to be deleted after abort, but it exists", tmpPath)
	}
}

func TestUpdate_SkipVerify_BypassesMissingOrMismatchedChecksum(t *testing.T) {
	binaryContent := []byte("executable-binary-payload")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/binary" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(binaryContent)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "crenox-bin")

	// Even with empty sha256URL, skipVerify=true must succeed
	err := verifyAndInstallBinary(server.Client(), targetPath, server.URL+"/binary", "", true)
	if err != nil {
		t.Fatalf("expected skip-verify to succeed, got: %v", err)
	}

	installed, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read installed binary: %v", err)
	}
	if !bytes.Equal(installed, binaryContent) {
		t.Errorf("installed binary content mismatch")
	}
}

func TestUpdate_ValidChecksum_InstallsSuccessfully(t *testing.T) {
	binaryContent := []byte("verified-binary-payload-v2")
	hashBytes := sha256.Sum256(binaryContent)
	expectedHex := hex.EncodeToString(hashBytes[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/binary" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(binaryContent)
			return
		}
		if r.URL.Path == "/sha256" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(fmt.Sprintf("%s  crenox-linux-amd64\n", expectedHex)))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	targetDir := t.TempDir()
	targetPath := filepath.Join(targetDir, "crenox-bin")

	err := verifyAndInstallBinary(server.Client(), targetPath, server.URL+"/binary", server.URL+"/sha256", false)
	if err != nil {
		t.Fatalf("expected valid checksum update to succeed, got: %v", err)
	}

	installed, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read installed binary: %v", err)
	}
	if !bytes.Equal(installed, binaryContent) {
		t.Errorf("installed binary content mismatch")
	}
}

func TestRunCmd_ZeroTolerance_LowSeverityBlocks(t *testing.T) {
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

	runGit("init", "-b", "main")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	runGit("config", "core.hooksPath", "")

	// Create file with LOW severity finding (Stripe test key)
	f := filepath.Join(tmpDir, "stripe.py")
	if err := os.WriteFile(f, []byte("stripe_key = \"sk_test_51MzZ1234567890abcdefghijklmnopqrstuvwxyz\"\n"), 0644); err != nil {
		t.Fatalf("write stripe.py: %v", err)
	}
	runGit("add", "stripe.py")

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir tmpDir: %v", err)
	}
	defer os.Chdir(origWd)

	oldExit := exitFunc
	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = oldExit }()

	// Intercept stderr to keep test output clean and verify JSON output
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	err = runScan("", "json", false, false)
	w.Close()
	os.Stderr = oldStderr

	var outBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, r)

	if err != nil {
		t.Fatalf("runScan returned unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for LOW severity finding under zero-tolerance, got %d", exitCode)
	}

	var rep map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &rep); err != nil {
		t.Fatalf("failed to parse JSON from runScan: %v, raw:\n%s", err, outBuf.String())
	}

	if rep["status"] != "blocked" {
		t.Errorf("expected status 'blocked', got %v", rep["status"])
	}
}

func TestRunCmd_FailClosedOnDiffOrShowFailure(t *testing.T) {
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

	runGit("init", "-b", "main")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	runGit("config", "core.hooksPath", "")

	// Create and stage a clean file
	cleanFile := filepath.Join(tmpDir, "staged.txt")
	if err := os.WriteFile(cleanFile, []byte("initial clean text\n"), 0644); err != nil {
		t.Fatalf("write staged.txt: %v", err)
	}
	runGit("add", "staged.txt")

	// Corrupt git objects by removing the blob object, causing git show :staged.txt to fail
	objDirs, _ := filepath.Glob(filepath.Join(tmpDir, ".git", "objects", "??"))
	for _, dir := range objDirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*"))
		for _, file := range files {
			_ = os.Remove(file)
		}
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir tmpDir: %v", err)
	}
	defer os.Chdir(origWd)

	oldExit := exitFunc
	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = oldExit }()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stderr = w

	err = runScan("", "json", false, false)
	w.Close()
	os.Stderr = oldStderr

	var outBuf bytes.Buffer
	_, _ = io.Copy(&outBuf, r)

	if err != nil {
		t.Fatalf("runScan returned unexpected error: %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for git blob read failure, got %d", exitCode)
	}

	var rep map[string]interface{}
	if err := json.Unmarshal(outBuf.Bytes(), &rep); err != nil {
		t.Fatalf("failed to parse JSON from runScan: %v, raw:\n%s", err, outBuf.String())
	}

	if rep["status"] != "scan_error" {
		t.Errorf("expected status 'scan_error', got %v", rep["status"])
	}

	failedFiles, ok := rep["failed_files"].([]interface{})
	if !ok || len(failedFiles) == 0 {
		t.Errorf("expected failed_files to be present in scan_error, got: %v", rep["failed_files"])
	}
}



