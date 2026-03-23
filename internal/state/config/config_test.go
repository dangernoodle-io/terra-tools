package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePick_SimpleField(t *testing.T) {
	field, expr, jqExpr, err := ParsePick("id")
	require.NoError(t, err)
	assert.Equal(t, "id", field)
	assert.Nil(t, expr)
	assert.Equal(t, "", jqExpr)
}

func TestParsePick_JQPipe(t *testing.T) {
	field, expr, jqExpr, err := ParsePick(".items[] | .id")
	require.NoError(t, err)
	assert.Equal(t, "", field)
	assert.Nil(t, expr)
	assert.Equal(t, ".items[] | .id", jqExpr)
}

func TestParsePick_JQArrayIndex(t *testing.T) {
	field, expr, jqExpr, err := ParsePick(".items[0].id")
	require.NoError(t, err)
	assert.Equal(t, "", field)
	assert.Nil(t, expr)
	assert.Equal(t, ".items[0].id", jqExpr)
}

func TestParsePick_JQSelect(t *testing.T) {
	field, expr, jqExpr, err := ParsePick(".[] | select(.active)")
	require.NoError(t, err)
	assert.Equal(t, "", field)
	assert.Nil(t, expr)
	assert.Equal(t, ".[] | select(.active)", jqExpr)
}

func TestParsePick_WhereFieldMap(t *testing.T) {
	raw := map[string]interface{}{
		"where": map[string]interface{}{
			"name": "acme-corp",
		},
		"field": "id",
	}
	field, expr, jqExpr, err := ParsePick(raw)
	require.NoError(t, err)
	assert.Equal(t, "", field)
	assert.NotNil(t, expr)
	assert.Equal(t, "id", expr.Field)
	assert.Equal(t, map[string]string{"name": "acme-corp"}, expr.Where)
	assert.Equal(t, "", jqExpr)
}

func TestParsePick_NilInput(t *testing.T) {
	_, _, _, err := ParsePick(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pick is nil")
}

func TestParsePick_InvalidType(t *testing.T) {
	_, _, _, err := ParsePick(42)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string or where/field map")
}

func TestParsePick_DotNameNotJQ(t *testing.T) {
	// .name starts with . but has no jq markers, so treated as plain field
	field, expr, jqExpr, err := ParsePick(".name")
	require.NoError(t, err)
	assert.Equal(t, ".name", field)
	assert.Nil(t, expr)
	assert.Equal(t, "", jqExpr)
}

func TestMerge_Empty(t *testing.T) {
	cfg := Merge()
	require.NotNil(t, cfg)
	assert.NotNil(t, cfg.Vars)
	assert.NotNil(t, cfg.Types)
	assert.NotNil(t, cfg.Resolvers)
	assert.Len(t, cfg.Vars, 0)
	assert.Len(t, cfg.Types, 0)
	assert.Len(t, cfg.Resolvers, 0)
}

func TestMerge_LaterAPIWins(t *testing.T) {
	cfg1 := &Config{
		API: &API{BaseURL: "https://api1.example.com", TokenEnv: "TOKEN1"},
	}
	cfg2 := &Config{
		API: &API{BaseURL: "https://api2.example.com", TokenEnv: "TOKEN2"},
	}

	merged := Merge(cfg1, cfg2)
	assert.Equal(t, "https://api2.example.com", merged.API.BaseURL)
	assert.Equal(t, "TOKEN2", merged.API.TokenEnv)
}

func TestMerge_VarsOverride(t *testing.T) {
	cfg1 := &Config{
		Vars: map[string]string{"key": "value1"},
	}
	cfg2 := &Config{
		Vars: map[string]string{"key": "value2", "other": "value3"},
	}

	merged := Merge(cfg1, cfg2)
	assert.Equal(t, "value2", merged.Vars["key"])
	assert.Equal(t, "value3", merged.Vars["other"])
}

func TestMerge_TypesOverride(t *testing.T) {
	cfg1 := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "template1"},
		},
	}
	cfg2 := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "template2"},
			"aws_iam_role":  {ID: "arn:aws:iam::.*:role/.*"},
		},
	}

	merged := Merge(cfg1, cfg2)
	assert.Equal(t, "template2", merged.Types["aws_s3_bucket"].ID)
	assert.Equal(t, "arn:aws:iam::.*:role/.*", merged.Types["aws_iam_role"].ID)
}

func TestMerge_NilSkipped(t *testing.T) {
	cfg1 := &Config{
		Vars: map[string]string{"key": "value"},
	}

	merged := Merge(cfg1, nil, cfg1)
	assert.Equal(t, "value", merged.Vars["key"])
	assert.Len(t, merged.Vars, 1)
}

func TestValidate_ValidMinimal(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "bucket-{{.name}}"},
		},
	}

	err := Validate(cfg)
	assert.NoError(t, err)
}

func TestValidate_EmptyTypes(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one type mapping")
}

func TestValidate_EmptyTypeID(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: ""},
		},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "id must not be empty")
}

func TestValidate_UndefinedResolverInType(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id", Use: []string{"undefined_resolver"}},
		},
		Resolvers: map[string]Resolver{},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined resolver")
}

func TestValidate_UndefinedResolverInResolver(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id"},
		},
		Resolvers: map[string]Resolver{
			"res1": {Get: "/api", Pick: "id", Use: []string{"undefined_resolver"}},
		},
		API: &API{BaseURL: "https://api.example.com", TokenEnv: "TOKEN"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "undefined resolver")
}

func TestValidate_MissingGet(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id", Use: []string{"res1"}},
		},
		Resolvers: map[string]Resolver{
			"res1": {Get: "", Pick: "id"},
		},
		API: &API{BaseURL: "https://api.example.com", TokenEnv: "TOKEN"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get must not be empty")
}

func TestValidate_MissingPick(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id", Use: []string{"res1"}},
		},
		Resolvers: map[string]Resolver{
			"res1": {Get: "/api", Pick: nil},
		},
		API: &API{BaseURL: "https://api.example.com", TokenEnv: "TOKEN"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pick must not be empty")
}

func TestValidate_APIRequiredWhenResolversUsed(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id", Use: []string{"res1"}},
		},
		Resolvers: map[string]Resolver{
			"res1": {Get: "/api", Pick: "id"},
		},
		API: nil,
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "api block is required")
}

func TestValidate_APIBaseURLRequired(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id", Use: []string{"res1"}},
		},
		Resolvers: map[string]Resolver{
			"res1": {Get: "/api", Pick: "id"},
		},
		API: &API{BaseURL: "", OpenAPISpec: "", TokenEnv: "TOKEN"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "base_url or api.openapi_spec")
}

func TestValidate_APITokenEnvRequired(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id", Use: []string{"res1"}},
		},
		Resolvers: map[string]Resolver{
			"res1": {Get: "/api", Pick: "id"},
		},
		API: &API{BaseURL: "https://api.example.com", TokenEnv: ""},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "token_env")
}

func TestValidate_ResolverCycleABCycleDetected(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id"},
		},
		Resolvers: map[string]Resolver{
			"A": {Get: "/a", Pick: "id", Use: []string{"B"}},
			"B": {Get: "/b", Pick: "id", Use: []string{"A"}},
		},
		API: &API{BaseURL: "https://api.example.com", TokenEnv: "TOKEN"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
	assert.Contains(t, err.Error(), "A")
	assert.Contains(t, err.Error(), "B")
}

func TestValidate_ResolverCycleTriangleDetected(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id"},
		},
		Resolvers: map[string]Resolver{
			"A": {Get: "/a", Pick: "id", Use: []string{"B"}},
			"B": {Get: "/b", Pick: "id", Use: []string{"C"}},
			"C": {Get: "/c", Pick: "id", Use: []string{"A"}},
		},
		API: &API{BaseURL: "https://api.example.com", TokenEnv: "TOKEN"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestValidate_ResolverCycleDisjointNotMentioned(t *testing.T) {
	cfg := &Config{
		Types: map[string]TypeMapping{
			"aws_s3_bucket": {ID: "id"},
		},
		Resolvers: map[string]Resolver{
			"A": {Get: "/a", Pick: "id", Use: []string{"B"}},
			"B": {Get: "/b", Pick: "id", Use: []string{"A"}},
			"C": {Get: "/c", Pick: "id", Use: []string{}},
		},
		API: &API{BaseURL: "https://api.example.com", TokenEnv: "TOKEN"},
	}

	err := Validate(cfg)
	require.Error(t, err)
	errMsg := err.Error()
	assert.Contains(t, errMsg, "A")
	assert.Contains(t, errMsg, "B")
	assert.NotContains(t, errMsg, "C")
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	require.Error(t, err)
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(configPath, []byte(`invalid: yaml: content:`), 0o644)
	require.NoError(t, err)

	_, err = Load(configPath)
	require.Error(t, err)
}

func TestLoad_ValidRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	yaml := `api:
  base_url: https://api.example.com
  token_env: API_TOKEN
vars:
  region: us-east-1
types:
  aws_s3_bucket:
    id: "{{.name}}"
    use: []
resolvers:
  bucket_info:
    get: /buckets
    pick: id
`
	err := os.WriteFile(configPath, []byte(yaml), 0o644)
	require.NoError(t, err)

	cfg, err := Load(configPath)
	require.NoError(t, err)
	assert.Equal(t, "https://api.example.com", cfg.API.BaseURL)
	assert.Equal(t, "API_TOKEN", cfg.API.TokenEnv)
	assert.Equal(t, "us-east-1", cfg.Vars["region"])
	assert.NotNil(t, cfg.Types["aws_s3_bucket"])
	assert.Equal(t, "{{.name}}", cfg.Types["aws_s3_bucket"].ID)
}
