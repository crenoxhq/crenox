package reporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/crenoxhq/crenox/v2/internal/scanner"
)

func TestReporter_PrintClean(t *testing.T) {
	buf := new(bytes.Buffer)
	rep := New(buf, FormatJSON)
	rep.PrintClean(10*time.Millisecond, 5)

	var res map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if res["status"] != "clean" {
		t.Errorf("expected status 'clean', got %v", res["status"])
	}
	if res["scanned_files"] != float64(5) {
		t.Errorf("expected scanned_files 5, got %v", res["scanned_files"])
	}
}

func TestReporter_PrintIncomplete_JSON(t *testing.T) {
	buf := new(bytes.Buffer)
	rep := New(buf, FormatJSON)

	failed := []FailedFile{
		{Path: "secret.env", Err: errors.New("permission denied")},
	}
	rep.PrintIncomplete(failed, nil, 15*time.Millisecond, 3)

	var res map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	if res["status"] != "scan_error" {
		t.Errorf("expected status 'scan_error', got %v", res["status"])
	}

	ffList, ok := res["failed_files"].([]interface{})
	if !ok || len(ffList) != 1 {
		t.Fatalf("expected 1 failed_file, got %v", res["failed_files"])
	}

	first := ffList[0].(map[string]interface{})
	if first["path"] != "secret.env" {
		t.Errorf("expected path 'secret.env', got %v", first["path"])
	}
	if !strings.Contains(first["error"].(string), "permission denied") {
		t.Errorf("expected error to contain 'permission denied', got %v", first["error"])
	}
}

func TestReporter_PrintIncomplete_JSON_WithFindings(t *testing.T) {
	buf := new(bytes.Buffer)
	rep := New(buf, FormatJSON)

	failed := []FailedFile{
		{Path: "unreadable.key", Err: errors.New("permission denied")},
	}
	findings := []scanner.Finding{
		{
			FilePath:      "app/config.py",
			Line:          14,
			LineContent:   "AWS_SECRET = 'AKIAIOSFODNN7EXAMPLE'",
			Token:         "AKIAIOSFODNN7EXAMPLE",
			DetectionTier: scanner.TierTrie,
			SignatureID:   "aws-access-token",
			Description:   "AWS Access Key ID",
			Severity:      "CRITICAL",
		},
	}

	rep.PrintIncomplete(failed, findings, 25*time.Millisecond, 2)

	var res map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse single JSON object: %v", err)
	}

	if res["status"] != "scan_error" {
		t.Errorf("expected status 'scan_error', got %v", res["status"])
	}

	ffList := res["failed_files"].([]interface{})
	if len(ffList) != 1 {
		t.Fatalf("expected 1 failed file, got %d", len(ffList))
	}

	fList := res["findings"].([]interface{})
	if len(fList) != 1 {
		t.Fatalf("expected 1 finding preserved in scan_error, got %d", len(fList))
	}
}

func TestReporter_PrintIncomplete_SARIF(t *testing.T) {
	buf := new(bytes.Buffer)
	rep := New(buf, FormatSARIF)

	failed := []FailedFile{
		{Path: "locked/key.pem", Err: errors.New("access denied")},
	}
	findings := []scanner.Finding{
		{
			FilePath:      "src/main.go",
			Line:          42,
			SignatureID:   "github-pat",
			Description:   "GitHub Personal Access Token",
			Severity:      "HIGH",
			DetectionTier: scanner.TierTrie,
		},
	}

	rep.PrintIncomplete(failed, findings, 20*time.Millisecond, 2)

	var report sarifReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("failed to parse SARIF JSON: %v", err)
	}

	if report.Version != "2.1.0" {
		t.Errorf("expected SARIF version 2.1.0, got %s", report.Version)
	}
	if len(report.Runs) == 0 {
		t.Fatalf("expected at least one SARIF run")
	}

	run := report.Runs[0]
	if len(run.Invocations) == 0 {
		t.Fatalf("expected invocations recorded in SARIF run")
	}
	if run.Invocations[0].ExecutionSuccessful {
		t.Errorf("expected executionSuccessful to be false on scan error")
	}
	if len(run.Invocations[0].ToolExecutionNotifications) == 0 {
		t.Errorf("expected toolExecutionNotifications on scan error")
	}
	if len(run.Results) != 1 {
		t.Errorf("expected 1 result in SARIF results, got %d", len(run.Results))
	}
}

func TestReporter_PrintIncomplete_GitLab(t *testing.T) {
	buf := new(bytes.Buffer)
	rep := New(buf, FormatGitLabSAST)

	failed := []FailedFile{
		{Path: "unreadable.conf", Err: errors.New("io error")},
	}
	findings := []scanner.Finding{
		{
			FilePath:      "settings.yaml",
			Line:          8,
			SignatureID:   "generic-secret",
			Description:   "Secret Key",
			Severity:      "CRITICAL",
			DetectionTier: scanner.TierTrie,
		},
	}

	rep.PrintIncomplete(failed, findings, 20*time.Millisecond, 1)

	var report gitlabReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("failed to parse GitLab SAST JSON: %v", err)
	}

	if report.Scan.Status != "failed" {
		t.Errorf("expected scan status 'failed', got %s", report.Scan.Status)
	}
	if len(report.Vulnerabilities) != 1 {
		t.Errorf("expected 1 vulnerability, got %d", len(report.Vulnerabilities))
	}
}

func TestReporter_PrintIncomplete_Plain(t *testing.T) {
	buf := new(bytes.Buffer)
	rep := New(buf, FormatPlain)

	failed := []FailedFile{
		{Path: "locked/key.pem", Err: errors.New("access denied")},
	}
	rep.PrintIncomplete(failed, nil, 20*time.Millisecond, 2)

	out := buf.String()
	if !strings.Contains(out, "scan incomplete — 1 file(s) failed") {
		t.Errorf("expected plain output to mention scan incomplete, got: %s", out)
	}
	if !strings.Contains(out, "locked/key.pem") {
		t.Errorf("expected plain output to mention locked/key.pem, got: %s", out)
	}
	if !strings.Contains(out, "commit blocked because the scan was incomplete") {
		t.Errorf("expected plain output to mention commit blocked, got: %s", out)
	}
}

func TestReporter_PrintIncomplete_Pretty(t *testing.T) {
	buf := new(bytes.Buffer)
	rep := New(buf, FormatPretty)

	failed := []FailedFile{
		{Path: "protected/db.yml", Err: errors.New("device busy")},
	}
	rep.PrintIncomplete(failed, nil, 5*time.Millisecond, 1)

	out := buf.String()
	if !strings.Contains(out, "SCAN INCOMPLETE") {
		t.Errorf("expected pretty output to mention SCAN INCOMPLETE, got: %s", out)
	}
	if !strings.Contains(out, "protected/db.yml") {
		t.Errorf("expected pretty output to mention protected/db.yml, got: %s", out)
	}
	if !strings.Contains(out, "Commit blocked because the scan was incomplete") {
		t.Errorf("expected pretty output to mention Commit blocked, got: %s", out)
	}
}
