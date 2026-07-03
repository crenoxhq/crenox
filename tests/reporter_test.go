package tests

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/sentinel-cli/sentinel/internal/reporter"
	"github.com/sentinel-cli/sentinel/internal/scanner"
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
	if run.Tool.Driver.Name != "Sentinel" {
		t.Errorf("expected driver name Sentinel, got %s", run.Tool.Driver.Name)
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
