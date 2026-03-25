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

// Helper to create a bool pointer.
func boolPtr(b bool) *bool {
	return &b
}

// Test applyAutofix - MissingDescription is fixable.
func TestApplyAutofix_MissingDescriptionFixable(t *testing.T) {
	errs := []Error{{Kind: MissingDescription, File: "/test/variables.tf", Variable: "my_var", Severity: SeverityWarning}}
	result := applyAutofix(errs, Options{})
	assert.Len(t, result, 1)
	assert.Equal(t, true, result[0].Autofix)
	assert.Equal(t, "add a description attribute to this block", result[0].Fix)
}

// Test applyAutofix - MissingValidation is fixable.
func TestApplyAutofix_MissingValidationFixable(t *testing.T) {
	errs := []Error{{Kind: MissingValidation, File: "/test/variables.tf", Variable: "my_var", Severity: SeverityWarning}}
	result := applyAutofix(errs, Options{})
	assert.Len(t, result, 1)
	assert.Equal(t, true, result[0].Autofix)
	assert.Equal(t, "add a validation block with condition and error_message", result[0].Fix)
}

// Test applyAutofix - SensitiveOutput is fixable.
func TestApplyAutofix_SensitiveOutputFixable(t *testing.T) {
	errs := []Error{{Kind: SensitiveOutput, File: "/test/outputs.tf", Variable: "my_output", Severity: SeverityWarning}}
	result := applyAutofix(errs, Options{})
	assert.Len(t, result, 1)
	assert.Equal(t, true, result[0].Autofix)
	assert.Equal(t, "add sensitive = true to this output block", result[0].Fix)
}

// Test applyAutofix - ExtraInput is not fixable.
func TestApplyAutofix_NonFixable(t *testing.T) {
	errs := []Error{{Kind: ExtraInput, File: "/test/main.tf", Variable: "extra_var", Severity: SeverityWarning}}
	result := applyAutofix(errs, Options{})
	assert.Len(t, result, 1)
	assert.Equal(t, false, result[0].Autofix)
	assert.Equal(t, "", result[0].Fix)
}

// Test applyAutofix - config opt-out.
func TestApplyAutofix_ConfigOptOut(t *testing.T) {
	cfg := &config.LintConfig{
		Rules: map[string]config.RuleConfig{
			"missing-description": {
				Enabled: true,
				Autofix: boolPtr(false),
			},
		},
	}
	errs := []Error{{Kind: MissingDescription, File: "/test/variables.tf", Variable: "my_var", Severity: SeverityWarning}}
	result := applyAutofix(errs, Options{Config: cfg})
	assert.Len(t, result, 1)
	assert.Equal(t, false, result[0].Autofix)
	assert.Equal(t, "", result[0].Fix)
}

// Test applyAutofix - nil config.
func TestApplyAutofix_NilConfig(t *testing.T) {
	errs := []Error{{Kind: MissingValidation, File: "/test/variables.tf", Variable: "my_var", Severity: SeverityWarning}}
	result := applyAutofix(errs, Options{})
	assert.Len(t, result, 1)
	assert.Equal(t, true, result[0].Autofix)
	assert.Equal(t, "add a validation block with condition and error_message", result[0].Fix)
}

// Test RuleName function.
func TestRuleName(t *testing.T) {
	assert.Equal(t, "missing-description", RuleName(MissingDescription))
	assert.Equal(t, "missing-validation", RuleName(MissingValidation))
	assert.Equal(t, "sensitive-output", RuleName(SensitiveOutput))
	assert.Equal(t, "missing-required", RuleName(MissingRequired))
}

// Test AutofixHint function.
func TestAutofixHint(t *testing.T) {
	hint, fixable := AutofixHint(MissingDescription)
	assert.Equal(t, true, fixable)
	assert.Equal(t, "add a description attribute to this block", hint)

	hint, fixable = AutofixHint(ExtraInput)
	assert.Equal(t, false, fixable)
	assert.Equal(t, "", hint)
}
