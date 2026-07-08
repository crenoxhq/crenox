package config

import (
	"math"
	"testing"
)

func TestConfig_ValidatePrivate(t *testing.T) {
	// Valid configs
	c1 := &Config{EntropyThreshold: 4.5}
	if _, err := c1.validate(); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}

	// Invalid threshold < 0
	c2 := &Config{EntropyThreshold: -0.1}
	if _, err := c2.validate(); err == nil {
		t.Error("expected error for negative entropy threshold")
	}

	// Invalid threshold > 8
	c3 := &Config{EntropyThreshold: math.Log2(256) + 0.1}
	if _, err := c3.validate(); err == nil {
		t.Error("expected error for entropy threshold > 8")
	}

	// MinSecretLength < 1 adjustment
	c4 := &Config{EntropyThreshold: 4.5, MinSecretLength: 0}
	cfg4, err := c4.validate()
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if cfg4.MinSecretLength != 1 {
		t.Errorf("expected MinSecretLength to reset to 1, got %d", cfg4.MinSecretLength)
	}

	// MaxFileSizeBytes < 0 adjustment
	c5 := &Config{EntropyThreshold: 4.5, MaxFileSizeBytes: -1}
	cfg5, err := c5.validate()
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if cfg5.MaxFileSizeBytes != DefaultMaxFileSizeBytes {
		t.Errorf("expected MaxFileSizeBytes to reset to default, got %d", cfg5.MaxFileSizeBytes)
	}

	// Custom signature empty ID
	c6 := &Config{
		EntropyThreshold: 4.5,
		CustomSignatures: []CustomSignature{
			{ID: "", Prefix: "abc"},
		},
	}
	if _, err := c6.validate(); err == nil {
		t.Error("expected error for empty custom signature ID")
	}

	// Custom signature empty prefix
	c7 := &Config{
		EntropyThreshold: 4.5,
		CustomSignatures: []CustomSignature{
			{ID: "sig-1", Prefix: ""},
		},
	}
	if _, err := c7.validate(); err == nil {
		t.Error("expected error for empty custom signature prefix")
	}

	// Custom signature invalid regex
	c8 := &Config{
		EntropyThreshold: 4.5,
		CustomSignatures: []CustomSignature{
			{ID: "sig-1", Prefix: "abc", Regex: "["},
		},
	}
	if _, err := c8.validate(); err == nil {
		t.Error("expected error for invalid custom signature regex")
	}

	// Custom signature invalid severity
	c9 := &Config{
		EntropyThreshold: 4.5,
		CustomSignatures: []CustomSignature{
			{ID: "sig-1", Prefix: "abc", Severity: "CRIT"},
		},
	}
	if _, err := c9.validate(); err == nil {
		t.Error("expected error for invalid custom signature severity")
	}

	// Custom signature valid severity CRITICAL
	c10 := &Config{
		EntropyThreshold: 4.5,
		CustomSignatures: []CustomSignature{
			{ID: "sig-1", Prefix: "abc", Severity: "CRITICAL"},
		},
	}
	if _, err := c10.validate(); err != nil {
		t.Errorf("unexpected error for CRITICAL severity: %v", err)
	}
}
