package validate

import "dangernoodle.io/terranoodle/internal/config"

// Options configures lint validation behavior.
type Options struct {
	Config *config.LintConfig
	Strict bool
}

// ruleNames maps ErrorKind to config rule names.
var ruleNames = map[ErrorKind]string{
	MissingRequired:         "missing-required",
	ExtraInput:              "extra-inputs",
	TypeMismatch:            "type-mismatch",
	SourceRefSemver:         "source-ref-semver",
	SourceProtocol:          "source-protocol",
	MissingDescription:      "missing-description",
	NonSnakeCase:            "non-snake-case",
	UnusedVariable:          "unused-variables",
	OptionalWithoutDefault:  "optional-without-default",
	MissingIncludeExpose:    "missing-include-expose",
	DisallowedFilename:      "allowed-filenames",
	MissingVersionsTF:       "has-versions-tf",
	MissingTerraformBlock:   "has-versions-tf",
	MissingProviderSource:   "has-versions-tf",
	MissingProviderVersion:  "has-versions-tf",
	DuplicateProvider:       "has-versions-tf",
	NoProviderBlock:         "no-tg-provider-blocks",
	SetStringType:           "set-string-type",
	ProviderConstraintStyle: "provider-constraint-style",
	EmptyOutputsTF:          "empty-outputs-tf",
	VersionsTFNotSymlink:    "versions-tf-symlink",
	MissingValidation:       "missing-validation",
	SensitiveOutput:         "sensitive-output",
	DependencyMergeOrder:    "dependency-merge-order",
}

// filterErrors removes errors for disabled rules.
func filterErrors(errs []Error, opts Options) []Error {
	if opts.Config == nil {
		return errs
	}

	filtered := make([]Error, 0, len(errs))
	for _, e := range errs {
		ruleName, ok := ruleNames[e.Kind]
		if !ok || opts.Config.IsRuleEnabled(ruleName, e.File) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// applySeverity sets error severity based on per-rule config.
// Default severity is warn. Allow-list downgrades are preserved (never upgraded back to error).
func applySeverity(errs []Error, opts Options) []Error {
	if opts.Config == nil {
		return errs
	}
	for i := range errs {
		ruleName, ok := ruleNames[errs[i].Kind]
		if !ok {
			continue
		}
		configured := opts.Config.RuleSeverity(ruleName, errs[i].File)
		var configSev Severity
		if configured == "error" {
			configSev = SeverityError
		} else {
			configSev = SeverityWarning
		}
		// Use max to never upgrade allow-list warnings back to error.
		// SeverityWarning(1) > SeverityError(0), so max preserves warnings.
		if configSev > errs[i].Severity {
			errs[i].Severity = configSev
		}
	}
	return errs
}

// isExcludedDir checks if a directory name matches ExcludeDirs.
func isExcludedDir(name string, opts Options) bool {
	if opts.Config == nil {
		return false
	}
	for _, excl := range opts.Config.ExcludeDirs {
		if name == excl || name+"/" == excl {
			return true
		}
	}
	return false
}

// getStringOption reads a string option from a rule's config.
func getStringOption(opts Options, ruleName, key string) string {
	if opts.Config == nil {
		return ""
	}
	rule, ok := opts.Config.Rules[ruleName]
	if !ok {
		return ""
	}
	raw, ok := rule.Options[key]
	if !ok {
		return ""
	}
	if s, ok := raw.(string); ok {
		return s
	}
	return ""
}

// getEnforceOption reads the "enforce" option from a rule's config.
func getEnforceOption(opts Options, ruleName string) string {
	return getStringOption(opts, ruleName, "enforce")
}

// getListOption reads a string list option from a rule's config.
func getListOption(opts Options, ruleName, key string) []string {
	if opts.Config == nil {
		return nil
	}
	rule, ok := opts.Config.Rules[ruleName]
	if !ok {
		return nil
	}
	raw, ok := rule.Options[key]
	if !ok {
		return nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	patterns := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			patterns = append(patterns, s)
		}
	}
	return patterns
}

// getAllowPatterns reads the "allow" option from a rule's config.
func getAllowPatterns(opts Options, ruleName string) []string {
	return getListOption(opts, ruleName, "allow")
}

// getExcludePatterns reads the "exclude" option from a rule's config.
func getExcludePatterns(opts Options, ruleName string) []string {
	return getListOption(opts, ruleName, "exclude")
}

// autofixRules maps fixable ErrorKind values to their fix hint strings.
var autofixRules = map[ErrorKind]string{
	MissingDescription: "add a description attribute to this block",
	MissingValidation:  "add a validation block with condition and error_message",
	SensitiveOutput:    "add sensitive = true to this output block",
}

// RuleName returns the kebab-case config name for an ErrorKind.
func RuleName(kind ErrorKind) string {
	return ruleNames[kind]
}

// AutofixHint returns the fix hint and true if the rule is auto-fixable.
func AutofixHint(kind ErrorKind) (string, bool) {
	hint, ok := autofixRules[kind]
	return hint, ok
}

// applyAutofix sets the Autofix and Fix fields on errors based on code metadata and config.
func applyAutofix(errs []Error, opts Options) []Error {
	for i := range errs {
		hint, fixable := AutofixHint(errs[i].Kind)
		if !fixable {
			continue
		}
		errs[i].Autofix = true
		errs[i].Fix = hint

		// Check config opt-out
		if opts.Config != nil {
			ruleName := ruleNames[errs[i].Kind]
			if rule, ok := opts.Config.Rules[ruleName]; ok {
				if rule.Autofix != nil && !*rule.Autofix {
					errs[i].Autofix = false
					errs[i].Fix = ""
				}
			}
		}
	}
	return errs
}
