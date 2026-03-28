package scaffold

import (
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// ProviderFromType extracts the provider name from a resource type.
// e.g., "google_compute_instance" → "google", "gitlab_project" → "gitlab".
func ProviderFromType(resourceType string) string {
	if idx := strings.Index(resourceType, "_"); idx > 0 {
		return resourceType[:idx]
	}
	return resourceType
}

// providerNamespaces maps provider prefix to GitHub org/namespace.
var providerNamespaces = map[string]string{
	"google":     "hashicorp",
	"aws":        "hashicorp",
	"azurerm":    "hashicorp",
	"azuread":    "hashicorp",
	"gitlab":     "gitlabhq",
	"github":     "integrations",
	"datadog":    "DataDog",
	"kubernetes": "hashicorp",
	"helm":       "hashicorp",
	"vault":      "hashicorp",
	"consul":     "hashicorp",
	"nomad":      "hashicorp",
	"random":     "hashicorp",
	"null":       "hashicorp",
	"local":      "hashicorp",
	"tls":        "hashicorp",
	"dns":        "hashicorp",
	"http":       "hashicorp",
}

// ProviderNamespace returns the GitHub org/namespace for the given provider.
// Defaults to "hashicorp" for unknown providers.
func ProviderNamespace(provider string) string {
	if ns, ok := providerNamespaces[provider]; ok {
		return ns
	}
	return "hashicorp"
}

// ResourceSuffix strips the provider prefix from a resource type.
// e.g., "google_compute_instance" with provider "google" → "compute_instance".
func ResourceSuffix(resourceType, provider string) string {
	prefix := provider + "_"
	if strings.HasPrefix(resourceType, prefix) {
		return resourceType[len(prefix):]
	}
	return resourceType
}

// registryBaseURL is the base URL for fetching provider docs. Override in tests.
var registryBaseURL = "https://raw.githubusercontent.com"

// httpClient is used for registry fetches with a 10s timeout.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// docURLs returns the two doc path variants for a given doc name.
func docURLs(namespace, provider, docName string) []string {
	base := registryBaseURL + "/" + namespace + "/terraform-provider-" + provider + "/main"
	return []string{
		base + "/website/docs/r/" + docName + ".html.markdown",
		base + "/docs/resources/" + docName + ".md",
	}
}

// docAliases returns alternative doc names to try when the primary suffix
// and full resource type don't match. Returns nil if no aliases apply.
var docAliases = func(suffix string) []string {
	// IAM consolidated docs: _iam_member/_iam_binding/_iam_policy → _iam
	for _, variant := range []string{"_iam_member", "_iam_binding", "_iam_policy"} {
		if strings.HasSuffix(suffix, variant) {
			return []string{suffix[:len(suffix)-len(variant)] + "_iam"}
		}
	}
	return nil
}

// FetchImportFormat fetches the import format for resourceType from the
// Terraform provider docs on GitHub. Results are cached in cache.
// Returns empty string on any error or 404 (graceful degradation).
func FetchImportFormat(ctx context.Context, resourceType string, cache map[string]string) string {
	if v, ok := cache[resourceType]; ok {
		return v
	}

	provider := ProviderFromType(resourceType)
	namespace := ProviderNamespace(provider)
	suffix := ResourceSuffix(resourceType, provider)

	var urls []string
	urls = append(urls, docURLs(namespace, provider, suffix)...)
	urls = append(urls, docURLs(namespace, provider, resourceType)...)
	for _, alias := range docAliases(suffix) {
		urls = append(urls, docURLs(namespace, provider, alias)...)
	}

	for _, url := range urls {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			continue
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			continue
		}
		result := ParseImportSection(string(body))
		if result != "" {
			cache[resourceType] = result
			return result
		}
	}

	cache[resourceType] = ""
	return ""
}

// importSectionRe matches ## Import or # Import headings (case-insensitive).
var importSectionRe = regexp.MustCompile(`(?im)^#{1,2}\s+import\s*$`)

// terraformImportLineRe matches a line containing "terraform import <address> <id>".
// Capture group 1 = the import ID portion.
var terraformImportLineRe = regexp.MustCompile(`(?i)terraform\s+import\s+\S+\s+(?:"([^"]+)"|(\S+))`)

// ParseImportSection finds the ## Import or # Import section in markdown and
// extracts the import ID from a "terraform import" command line within it.
// Returns empty string if no import section or import command is found.
func ParseImportSection(markdown string) string {
	loc := importSectionRe.FindStringIndex(markdown)
	if loc == nil {
		return ""
	}
	// Only look in the text after the Import heading.
	section := markdown[loc[1]:]

	// Find the next heading to limit the search scope.
	nextHeading := regexp.MustCompile(`(?m)^#{1,2}\s+\S`)
	if nl := nextHeading.FindStringIndex(section); nl != nil {
		section = section[:nl[0]]
	}

	m := terraformImportLineRe.FindStringSubmatch(section)
	if m == nil {
		return ""
	}
	if m[1] != "" {
		return m[1]
	}
	return m[2]
}

// placeholderRe matches both {{name}} and {name} style placeholders.
var placeholderRe = regexp.MustCompile(`\{\{(\w+)\}\}|\{(\w+)\}`)

// FormatToTemplate converts a registry import format string to a Go template
// string. Placeholders that match available fields become {{ .fieldname }};
// those that don't become TODO(fieldname).
func FormatToTemplate(format string, availableFields map[string]string) string {
	return placeholderRe.ReplaceAllStringFunc(format, func(match string) string {
		// Extract field name from either {{name}} or {name}.
		sub := placeholderRe.FindStringSubmatch(match)
		fieldName := sub[1]
		if fieldName == "" {
			fieldName = sub[2]
		}
		if _, ok := availableFields[fieldName]; ok {
			return "{{ ." + fieldName + " }}"
		}
		return "TODO(" + fieldName + ")"
	})
}
