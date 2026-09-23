package reporter

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
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
	rep.PrintIncomplete(failed, 15*time.Millisecond, 3)

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

func TestReporter_PrintIncomplete_Plain(t *testing.T) {
	buf := new(bytes.Buffer)
	rep := New(buf, FormatPlain)

	failed := []FailedFile{
		{Path: "locked/key.pem", Err: errors.New("access denied")},
	}
	rep.PrintIncomplete(failed, 20*time.Millisecond, 2)

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
	rep.PrintIncomplete(failed, 5*time.Millisecond, 1)

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
