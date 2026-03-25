package validate

import (
	"testing"

	"dangernoodle.io/terranoodle/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestGetAllowPatterns(t *testing.T) {
	// nil config
	assert.Nil(t, getAllowPatterns(Options{}, "source-ref-semver"))

	// no rule
	cfg := &config.LintConfig{Rules: map[string]config.RuleConfig{}}
	assert.Nil(t, getAllowPatterns(Options{Config: cfg}, "source-ref-semver"))

	// no allow option
	cfg = &config.LintConfig{Rules: map[string]config.RuleConfig{
		"source-ref-semver": {Enabled: true},
	}}
	assert.Nil(t, getAllowPatterns(Options{Config: cfg}, "source-ref-semver"))

	// with allow
	cfg = &config.LintConfig{Rules: map[string]config.RuleConfig{
		"source-ref-semver": {Enabled: true, Options: map[string]interface{}{
			"allow": []interface{}{"jae/*", "feature/*"},
		}},
	}}
	patterns := getAllowPatterns(Options{Config: cfg}, "source-ref-semver")
	assert.Equal(t, []string{"jae/*", "feature/*"}, patterns)
}

func TestApplySeverity_DefaultWarn(t *testing.T) {
	cfg := &config.LintConfig{
		Rules: map[string]config.RuleConfig{
			"missing-required": {Enabled: true},
		},
	}
	errs := []Error{{Kind: MissingRequired, File: "/test", Severity: SeverityError}}
	result := applySeverity(errs, Options{Config: cfg})
	assert.Equal(t, SeverityWarning, result[0].Severity)
}

func TestApplySeverity_ExplicitError(t *testing.T) {
	cfg := &config.LintConfig{
		Rules: map[string]config.RuleConfig{
			"missing-required": {Enabled: true, Severity: "error"},
		},
	}
	errs := []Error{{Kind: MissingRequired, File: "/test", Severity: SeverityError}}
	result := applySeverity(errs, Options{Config: cfg})
	assert.Equal(t, SeverityError, result[0].Severity)
}

func TestApplySeverity_AllowListPreserved(t *testing.T) {
	cfg := &config.LintConfig{
		Rules: map[string]config.RuleConfig{
			"extra-inputs": {Enabled: true, Severity: "error"},
		},
	}
	// Allow-list already downgraded this to warning
	errs := []Error{{Kind: ExtraInput, File: "/test", Severity: SeverityWarning}}
	result := applySeverity(errs, Options{Config: cfg})
	assert.Equal(t, SeverityWarning, result[0].Severity)
}

func TestApplySeverity_NilConfig(t *testing.T) {
	errs := []Error{{Kind: MissingRequired, File: "/test", Severity: SeverityError}}
	result := applySeverity(errs, Options{})
	assert.Equal(t, SeverityError, result[0].Severity)
}
