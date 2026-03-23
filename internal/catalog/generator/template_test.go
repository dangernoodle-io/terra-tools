package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

func TestResolveTemplate_ValuesReplaced(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "terragrunt.hcl")

	// Write a template with values.xxx references.
	content := []byte(`inputs = {
  env  = values.env
  name = values.name
}
`)
	require.NoError(t, os.WriteFile(templatePath, content, 0o644))

	// Create a values object with env and name.
	values := cty.ObjectVal(map[string]cty.Value{
		"env":  cty.StringVal("staging"),
		"name": cty.StringVal("acme-svc"),
	})

	result, warnings, err := ResolveTemplate(templatePath, values)

	require.NoError(t, err)
	require.Empty(t, warnings)

	// Check that the values were replaced literally.
	resultStr := string(result)
	assert.Contains(t, resultStr, "staging")
	assert.Contains(t, resultStr, "acme-svc")
	assert.NotContains(t, resultStr, "values.env")
	assert.NotContains(t, resultStr, "values.name")
}

func TestResolveTemplate_NoValuesRefs(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "terragrunt.hcl")

	// Write a template with no values references.
	content := []byte(`include "root" {
  path = find_in_parent_folders("terragrunt-root.hcl")
}
`)
	require.NoError(t, os.WriteFile(templatePath, content, 0o644))

	values := cty.EmptyObjectVal

	result, warnings, err := ResolveTemplate(templatePath, values)

	require.NoError(t, err)
	require.Empty(t, warnings)
	assert.Equal(t, content, result)
}

func TestResolveTemplate_UnresolvableRef(t *testing.T) {
	tmpDir := t.TempDir()
	templatePath := filepath.Join(tmpDir, "terragrunt.hcl")

	// Write a template with an unresolvable reference.
	content := []byte(`inputs = {
  missing = values.missing
}
`)
	require.NoError(t, os.WriteFile(templatePath, content, 0o644))

	// Create a values object without the missing attribute.
	values := cty.ObjectVal(map[string]cty.Value{
		"other": cty.StringVal("value"),
	})

	result, warnings, err := ResolveTemplate(templatePath, values)

	require.NoError(t, err)
	require.NotEmpty(t, warnings)
	assert.Len(t, warnings, 1)
	assert.Contains(t, warnings[0], "unresolved")
	assert.Contains(t, warnings[0], "values.missing")
	// The template should still be returned with the unresolved reference intact.
	assert.Contains(t, string(result), "values.missing")
}

func TestResolveTemplate_MissingFile(t *testing.T) {
	templatePath := "/nonexistent/path/terragrunt.hcl"
	values := cty.EmptyObjectVal

	_, _, err := ResolveTemplate(templatePath, values)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading template")
}

func TestLookupTraversal_SimpleAttr(t *testing.T) {
	// Build a simple object with an "env" attribute.
	root := cty.ObjectVal(map[string]cty.Value{
		"env": cty.StringVal("staging"),
	})

	// Build a traversal: values.env
	trav := hcl.Traversal{
		hcl.TraverseRoot{Name: "values"},
		hcl.TraverseAttr{Name: "env"},
	}

	result, err := lookupTraversal(root, trav)

	require.NoError(t, err)
	assert.Equal(t, "staging", result.AsString())
}

func TestLookupTraversal_MissingAttr(t *testing.T) {
	// Build a root object without "missing" attribute.
	root := cty.ObjectVal(map[string]cty.Value{
		"other": cty.StringVal("value"),
	})

	// Build a traversal: values.missing
	trav := hcl.Traversal{
		hcl.TraverseRoot{Name: "values"},
		hcl.TraverseAttr{Name: "missing"},
	}

	_, err := lookupTraversal(root, trav)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no attribute")
}

func TestLookupTraversal_JustRoot(t *testing.T) {
	// Build a root object.
	root := cty.ObjectVal(map[string]cty.Value{
		"key": cty.StringVal("value"),
	})

	// Build a traversal with only the root (no attributes).
	trav := hcl.Traversal{
		hcl.TraverseRoot{Name: "values"},
	}

	result, err := lookupTraversal(root, trav)

	require.NoError(t, err)
	assert.Equal(t, root, result)
}

func TestIndentLiteral_SingleLine(t *testing.T) {
	content := []byte("  foo = ")
	literal := []byte("bar")
	insertPos := len(content) - 1

	result := indentLiteral(literal, insertPos, content)

	assert.Equal(t, literal, result)
}

func TestIndentLiteral_MultiLine(t *testing.T) {
	// Build a line with 2 spaces of indentation.
	content := []byte("  foo = ")
	// A multi-line literal value.
	literal := []byte("{\n  bar = 1\n}")
	insertPos := len(content) - 1

	result := indentLiteral(literal, insertPos, content)

	// The first line should be unchanged, but subsequent lines should be indented
	// by 2 spaces to match the line's indentation.
	resultStr := string(result)
	// Count the leading spaces on the second line (after the opening brace and newline).
	assert.Contains(t, resultStr, "{\n  ")
	assert.Contains(t, resultStr, "bar = 1\n  }")
}

func TestIndentLiteral_NoIndent(t *testing.T) {
	// Build a line with no leading whitespace.
	content := []byte("foo = ")
	literal := []byte("{\n  bar = 1\n}")
	insertPos := len(content) - 1

	result := indentLiteral(literal, insertPos, content)

	// Since there's no leading whitespace, the literal should be unchanged.
	assert.Equal(t, literal, result)
}
