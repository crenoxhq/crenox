package tests

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/crenoxhq/crenox/v2/internal/reporter"
	"github.com/crenoxhq/crenox/v2/internal/scanner"
)

// Simple structures to unmarshal and validate SARIF output
type testSarifReport struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []struct {
		Tool struct {
			Driver struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Rules   []struct {
					ID               string `json:"id"`
					ShortDescription struct {
						Text string `json:"text"`
					} `json:"shortDescription"`
				} `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
		Results []struct {
			RuleID  string `json:"ruleId"`
			Message struct {
				Text string `json:"text"`
			} `json:"message"`
			Locations []struct {
				PhysicalLocation struct {
					ArtifactLocation struct {
						URI string `json:"uri"`
					} `json:"artifactLocation"`
					Region struct {
						StartLine int `json:"startLine"`
					} `json:"region"`
				} `json:"physicalLocation"`
			} `json:"locations"`
		} `json:"results"`
	} `json:"runs"`
}

func TestSARIFReporter(t *testing.T) {
	// 1. Verify ParseFormat
	if reporter.ParseFormat("sarif") != reporter.FormatSARIF {
		t.Fatalf("expected ParseFormat('sarif') to be FormatSARIF")
	}

	// 2. Mock findings
	mockFindings := []scanner.Finding{
		{
			FilePath:      "main.go",
			Line:          15,
			Severity:      "HIGH",
			DetectionTier: scanner.TierTrie,
			SignatureID:   "slack-webhook",
			Description:   "Slack Webhook URL",
			Token:         "https://hooks.slack.com/services/T_MOCK_VAL/B_MOCK_VAL/MOCK_SECRET_XYZ",
		},
	}

	buf := new(bytes.Buffer)
	rep := reporter.New(buf, reporter.FormatSARIF)

	// Print summary
	rep.PrintSummary(mockFindings, 120*time.Millisecond, 5)

	// Validate JSON structure
	var report testSarifReport
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("failed to unmarshal SARIF report: %v", err)
	}

	if report.Version != "2.1.0" {
		t.Errorf("expected version 2.1.0, got %s", report.Version)
	}
	if len(report.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(report.Runs))
	}
	run := report.Runs[0]
	if run.Tool.Driver.Name != "Crenox" {
		t.Errorf("expected driver name Crenox, got %s", run.Tool.Driver.Name)
	}
	if len(run.Tool.Driver.Rules) != 1 || run.Tool.Driver.Rules[0].ID != "slack-webhook" {
		t.Errorf("expected rule ID slack-webhook, got %+v", run.Tool.Driver.Rules)
	}
	if len(run.Results) != 1 || run.Results[0].RuleID != "slack-webhook" {
		t.Errorf("expected 1 result with rule ID slack-webhook")
	}
	res := run.Results[0]
	if len(res.Locations) != 1 || res.Locations[0].PhysicalLocation.ArtifactLocation.URI != "main.go" {
		t.Errorf("expected location main.go, got %+v", res.Locations)
	}
	if res.Locations[0].PhysicalLocation.Region.StartLine != 15 {
		t.Errorf("expected line 15, got %d", res.Locations[0].PhysicalLocation.Region.StartLine)
	}
}

func TestJSONReporter(t *testing.T) {
	if reporter.ParseFormat("json") != reporter.FormatJSON {
		t.Fatalf("expected ParseFormat('json') to be FormatJSON")
	}

	mockFindings := []scanner.Finding{
		{
			FilePath:      "config.json",
			Line:          10,
			Severity:      "CRITICAL",
			DetectionTier: scanner.TierTrie,
			SignatureID:   "aws-access-key",
			Description:   "AWS Access Key",
			Token:         "AKIAIOSFODNN7EXAMPLE",
		},
	}

	buf := new(bytes.Buffer)
	rep := reporter.New(buf, reporter.FormatJSON)
	rep.PrintSummary(mockFindings, 50*time.Millisecond, 1)

	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to unmarshal JSON report: %v", err)
	}

	findingsRaw, ok := output["findings"].([]interface{})
	if !ok || len(findingsRaw) != 1 {
		t.Fatalf("expected 1 finding in JSON, got %v", output["findings"])
	}
	finding := findingsRaw[0].(map[string]interface{})
	if finding["file_path"] != "config.json" {
		t.Errorf("expected file config.json, got %v", finding["file_path"])
	}
}

func TestPlainReporter(t *testing.T) {
	if reporter.ParseFormat("plain") != reporter.FormatPlain {
		t.Fatalf("expected ParseFormat('plain') to be FormatPlain")
	}

	mockFindings := []scanner.Finding{
		{
			FilePath:      "README.md",
			Line:          5,
			Severity:      "LOW",
			DetectionTier: scanner.TierTrie,
			SignatureID:   "test-key",
			Description:   "Test Key",
			Token:         "TEST1234",
		},
	}

	buf := new(bytes.Buffer)
	rep := reporter.New(buf, reporter.FormatPlain)
	rep.PrintFindings(mockFindings)

	outStr := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("README.md:5")) {
		t.Errorf("expected plain output to contain 'README.md:5', got %s", outStr)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Test Key")) {
		t.Errorf("expected plain output to contain 'Test Key'")
	}
}

func TestParseFormat_Default(t *testing.T) {
	if reporter.ParseFormat("unknown") != reporter.FormatPretty {
		t.Fatalf("expected unknown format to default to FormatPretty")
	}
	if reporter.ParseFormat("") != reporter.FormatPretty {
		t.Fatalf("expected empty format to default to FormatPretty")
	}
}

func TestGitLabSASTReporter(t *testing.T) {
	if reporter.ParseFormat("gitlab-sast") != reporter.FormatGitLabSAST {
		t.Fatalf("expected ParseFormat('gitlab-sast') to be FormatGitLabSAST")
	}
	if reporter.ParseFormat("sast") != reporter.FormatGitLabSAST {
		t.Fatalf("expected ParseFormat('sast') to be FormatGitLabSAST")
	}

	mockFindings := []scanner.Finding{
		{
			FilePath:      "server/main.go",
			Line:          42,
			Severity:      "CRITICAL",
			DetectionTier: scanner.TierTrie,
			SignatureID:   "github-pat",
			Description:   "GitHub Personal Access Token",
			Token:         "ghp_0123456789abcdefghijklmnopqrstuvwxyz",
		},
	}

	buf := new(bytes.Buffer)
	rep := reporter.New(buf, reporter.FormatGitLabSAST)
	rep.PrintSummary(mockFindings, 80*time.Millisecond, 3)

	var report map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("failed to unmarshal GitLab SAST JSON report: %v", err)
	}

	if report["version"] != "15.0.0" {
		t.Errorf("expected version 15.0.0, got %v", report["version"])
	}

	vulns, ok := report["vulnerabilities"].([]interface{})
	if !ok || len(vulns) != 1 {
		t.Fatalf("expected 1 vulnerability in GitLab SAST, got %v", report["vulnerabilities"])
	}

	vuln := vulns[0].(map[string]interface{})
	if vuln["category"] != "secret_detection" {
		t.Errorf("expected category secret_detection, got %v", vuln["category"])
	}
	if vuln["severity"] != "Critical" {
		t.Errorf("expected severity Critical, got %v", vuln["severity"])
	}

	loc := vuln["location"].(map[string]interface{})
	if loc["file"] != "server/main.go" || loc["start_line"] != float64(42) {
		t.Errorf("expected location server/main.go:42, got %+v", loc)
	}
}
