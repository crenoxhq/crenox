// Package tests contains unit and boundary tests for the Tier 3
// context-aware false-positive suppression engine.
package tests

import (
	"testing"

	crenoxcontext "github.com/crenoxhq/crenox/v2/internal/context"
)

// ──────────────────────────────────────────────────────────────────────────────
// True positives — Classify should return Real
// ──────────────────────────────────────────────────────────────────────────────

func TestClassify_RealSecret_ProductionFile(t *testing.T) {
	d := crenoxcontext.Classify("internal/auth/client.go", `token := "ghp_REALAPITOKEN1234567890abcdef"`, "ghp_REALAPITOKEN1234567890abcdef", "")
	if d != crenoxcontext.Real {
		t.Errorf("expected Real, got %s", d)
	}
}

func TestClassify_RealSecret_ConfigFile(t *testing.T) {
	d := crenoxcontext.Classify("config/production.yaml", `stripe_secret: sk_live_fake_key_for_testing_purposes`, "sk_live_fake_key_for_testing_purposes", "")
	if d != crenoxcontext.Real {
		t.Errorf("expected Real for production config, got %s", d)
	}
}

func TestClassify_RealSecret_EnvFile(t *testing.T) {
	d := crenoxcontext.Classify(".env", `OPENAI_TOKEN=sk-proj-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef`, "sk-proj-ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef", "")
	if d != crenoxcontext.Real {
		t.Errorf("expected Real for .env file, got %s", d)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// True negatives — Classify should suppress the finding
// ──────────────────────────────────────────────────────────────────────────────

func TestClassify_SafeComment_HashStyle(t *testing.T) {
	d := crenoxcontext.Classify("main.go", `  # token = "ghp_OLDTOKEN123456"`, "ghp_OLDTOKEN123456", "")
	if d != crenoxcontext.SafeComment {
		t.Errorf("expected SafeComment for hash-comment line, got %s", d)
	}
}

func TestClassify_SafeComment_SlashStyle(t *testing.T) {
	d := crenoxcontext.Classify("main.go", `  // apiKey = "sk_live_example"`, "sk_live_example", "")
	if d != crenoxcontext.SafeComment {
		t.Errorf("expected SafeComment for slash-comment line, got %s", d)
	}
}

func TestClassify_SafeTestFile_GoTest(t *testing.T) {
	d := crenoxcontext.Classify("auth/auth_test.go", `token := "ghp_TESTTOKEN123456789012345678"`, "ghp_TESTTOKEN123456789012345678", "")
	if d != crenoxcontext.SafeTestFile {
		t.Errorf("expected SafeTestFile for _test.go, got %s", d)
	}
}

func TestClassify_SafeTestFile_TestDirectory(t *testing.T) {
	d := crenoxcontext.Classify("tests/fixtures/creds.go", `key := "sk_live_TESTTOKEN1234567890"`, "sk_live_TESTTOKEN1234567890", "")
	if d != crenoxcontext.SafeTestFile {
		t.Errorf("expected SafeTestFile for tests/ directory, got %s", d)
	}
}

func TestClassify_SafeTestFile_SpecFile(t *testing.T) {
	d := crenoxcontext.Classify("spec/auth_spec.rb", `let(:token) { "xoxb-12345-67890-testvalue" }`, "xoxb-12345-67890-testvalue", "")
	if d != crenoxcontext.SafeTestFile {
		t.Errorf("expected SafeTestFile for _spec.rb, got %s", d)
	}
}

func TestClassify_SafeVariableName_Dummy(t *testing.T) {
	d := crenoxcontext.Classify("cmd/setup.go", `dummy_key = "ghp_DUMMYTOKEN123456789012"`, "ghp_DUMMYTOKEN123456789012", "")
	if d != crenoxcontext.SafeVariableName {
		t.Errorf("expected SafeVariableName for 'dummy_key', got %s", d)
	}
}

func TestClassify_SafeVariableName_Placeholder(t *testing.T) {
	// This file is in docs/ which triggers SafeTestFile — that is still valid
	// suppression. Accept any non-Real decision.
	d := crenoxcontext.Classify("docs/setup.go", `example_key = "xoxb-XXXXXXXXXX-YYYYYYY"`, "xoxb-XXXXXXXXXX-YYYYYYY", "")
	if d == crenoxcontext.Real {
		t.Errorf("expected suppression for docs/ file with example varname, got Real")
	}
}

func TestClassify_SafeVariableName_FakeToken(t *testing.T) {
	d := crenoxcontext.Classify("cmd/main.go", `fake_token := "sk_test_1234567890abcdefghij"`, "sk_test_1234567890abcdefghij", "")
	if d != crenoxcontext.SafeVariableName {
		t.Errorf("expected SafeVariableName for 'fake_token', got %s", d)
	}
}

func TestClassify_SafeVariableName_Mock(t *testing.T) {
	d := crenoxcontext.Classify("internal/client.go", `mock_api_key := "AKIAIOSFODNN7EXAMPLE"`, "AKIAIOSFODNN7EXAMPLE", "")
	if d != crenoxcontext.SafeVariableName {
		t.Errorf("expected SafeVariableName for 'mock_api_key', got %s", d)
	}
}

func TestClassify_SafePlaceholder_EnvVar(t *testing.T) {
	d := crenoxcontext.Classify("deploy.sh", `TOKEN=$MY_SECRET_TOKEN`, "$MY_SECRET_TOKEN", "")
	if d != crenoxcontext.SafePlaceholder {
		t.Errorf("expected SafePlaceholder for env var reference, got %s", d)
	}
}

func TestClassify_SafePlaceholder_BraceEnvVar(t *testing.T) {
	d := crenoxcontext.Classify("config.yaml", `token: "${SECRET_TOKEN}"`, "${SECRET_TOKEN}", "")
	if d != crenoxcontext.SafePlaceholder {
		t.Errorf("expected SafePlaceholder for brace env var, got %s", d)
	}
}

func TestClassify_SafeUUID(t *testing.T) {
	d := crenoxcontext.Classify("internal/service.go", `id := "550e8400-e29b-41d4-a716-446655440000"`, "550e8400-e29b-41d4-a716-446655440000", "")
	if d != crenoxcontext.SafeUUID {
		t.Errorf("expected SafeUUID for UUID pattern, got %s", d)
	}
}

func TestClassify_SafeReadme(t *testing.T) {
	d := crenoxcontext.Classify("README.md", `export GITHUB_TOKEN=ghp_YOURTOKENHERE`, "ghp_YOURTOKENHERE", "")
	// README.md should match the safe file pattern for .md files.
	if d == crenoxcontext.Real {
		t.Errorf("expected suppression for README.md, got Real")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// IsTestFilePath
// ──────────────────────────────────────────────────────────────────────────────

func TestIsTestFilePath_GoTest(t *testing.T) {
	if !crenoxcontext.IsTestFilePath("pkg/auth/auth_test.go") {
		t.Error("expected IsTestFilePath=true for *_test.go")
	}
}

func TestIsTestFilePath_TestsDir(t *testing.T) {
	if !crenoxcontext.IsTestFilePath("tests/unit/runner.go") {
		t.Error("expected IsTestFilePath=true for tests/ directory")
	}
}

func TestIsTestFilePath_ProductionFile(t *testing.T) {
	if crenoxcontext.IsTestFilePath("internal/auth/client.go") {
		t.Error("expected IsTestFilePath=false for production file")
	}
}

func TestIsTestFilePath_MarkdownDoc(t *testing.T) {
	if !crenoxcontext.IsTestFilePath("docs/setup.md") {
		t.Error("expected IsTestFilePath=true for .md docs file")
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Decision.String()
// ──────────────────────────────────────────────────────────────────────────────

func TestDecisionString(t *testing.T) {
	cases := []struct {
		d    crenoxcontext.Decision
		want string
	}{
		{crenoxcontext.Real, "real"},
		{crenoxcontext.SafeComment, "safe:comment"},
		{crenoxcontext.SafeTestFile, "safe:test-file"},
		{crenoxcontext.SafeVariableName, "safe:variable-name"},
		{crenoxcontext.SafePlaceholder, "safe:placeholder"},
		{crenoxcontext.SafeUUID, "safe:uuid"},
		{crenoxcontext.SafeVersionString, "safe:version"},
	}
	for _, tc := range cases {
		if got := tc.d.String(); got != tc.want {
			t.Errorf("Decision(%d).String() = %q; want %q", tc.d, got, tc.want)
		}
	}
}

func TestClassify_SafeComment_Nosec(t *testing.T) {
	d := crenoxcontext.Classify("main.go", `const authTokenHeader = "X-Auth-Token" //#nosec G101 -- This is a false positive`, "X-Auth-Token", "high-entropy")
	if d != crenoxcontext.SafeComment {
		t.Errorf("expected SafeComment for #nosec, got %s", d)
	}
}

func TestClassify_SafePlaceholder_FormatString(t *testing.T) {
	d := crenoxcontext.Classify("db.go", `dsn := fmt.Sprintf("postgres://%s:%s@%s", user, pass, host)`, "postgres://%s:%s@%s", "high-entropy")
	if d != crenoxcontext.SafePlaceholder {
		t.Errorf("expected SafePlaceholder for format string, got %s", d)
	}
}

func TestClassify_SafeVariableName_FakeTokens(t *testing.T) {
	d := crenoxcontext.Classify("auth.go", `token := "completely_fake_token_xxx"`, "completely_fake_token_xxx", "hex")
	if !crenoxcontext.IsSuppressed(d) {
		t.Errorf("expected completely_fake_token_xxx to be suppressed, got %s", d)
	}
}

func TestClassify_SafeVersionString_Geohash(t *testing.T) {
	d := crenoxcontext.Classify("geo.go", `my_secret := "0123456789bcdefghjkmnpqrstuvwxyz"`, "0123456789bcdefghjkmnpqrstuvwxyz", "hex")
	if d != crenoxcontext.SafeVersionString {
		t.Errorf("expected SafeVersionString for geohash base32, got %s", d)
	}
}
