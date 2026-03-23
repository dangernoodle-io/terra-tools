package scaffold

import (
	"bytes"
	"testing"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_Deduplicates(t *testing.T) {
	resources := []*tfjson.ResourceChange{
		{
			Type: "google_compute_instance",
			Change: &tfjson.Change{
				After: map[string]interface{}{"name": "acme-instance"},
			},
		},
		{
			Type: "google_compute_instance",
			Change: &tfjson.Change{
				After: map[string]interface{}{"name": "another-instance"},
			},
		},
	}

	result := Generate(resources, nil)

	require.Len(t, result, 1)
	assert.Equal(t, "google_compute_instance", result[0].ResourceType)
}

func TestGenerate_CollectsFields(t *testing.T) {
	resources := []*tfjson.ResourceChange{
		{
			Type: "google_compute_instance",
			Change: &tfjson.Change{
				After: map[string]interface{}{
					"project": "acme-project",
					"name":    "acme-instance",
					"zone":    "us-central1-a",
				},
			},
		},
	}

	result := Generate(resources, nil)

	require.Len(t, result, 1)
	assert.Equal(t, "google_compute_instance", result[0].ResourceType)
	assert.Len(t, result[0].Fields, 3)
	assert.Contains(t, result[0].Fields, "project")
	assert.Contains(t, result[0].Fields, "name")
	assert.Contains(t, result[0].Fields, "zone")
	assert.Equal(t, "acme-project", result[0].Fields["project"])
	assert.Equal(t, "acme-instance", result[0].Fields["name"])
	assert.Equal(t, "us-central1-a", result[0].Fields["zone"])
}

func TestGenerate_SkipsNilResource(t *testing.T) {
	resources := []*tfjson.ResourceChange{
		nil,
		{
			Type: "google_compute_instance",
			Change: &tfjson.Change{
				After: map[string]interface{}{"name": "acme-instance"},
			},
		},
	}

	result := Generate(resources, nil)

	require.Len(t, result, 1)
	assert.Equal(t, "google_compute_instance", result[0].ResourceType)
}

func TestGenerate_WithFormat(t *testing.T) {
	resources := []*tfjson.ResourceChange{
		{
			Type: "google_compute_instance",
			Change: &tfjson.Change{
				After: map[string]interface{}{
					"project": "acme-project",
					"zone":    "us-central1-a",
					"name":    "acme-instance",
				},
			},
		},
	}

	formats := map[string]string{
		"google_compute_instance": "projects/{project}/zones/{zone}/instances/{name}",
	}

	result := Generate(resources, formats)

	require.Len(t, result, 1)
	assert.NotEqual(t, "TODO", result[0].IDTemplate)
	assert.Contains(t, result[0].IDTemplate, "{{ .project }}")
	assert.Contains(t, result[0].IDTemplate, "{{ .zone }}")
	assert.Contains(t, result[0].IDTemplate, "{{ .name }}")
}

func TestGenerate_Empty(t *testing.T) {
	result := Generate(nil, nil)
	assert.Empty(t, result)
}

func TestRenderYAML_WritesToBuffer(t *testing.T) {
	types := []TypeInfo{
		{
			ResourceType: "google_compute_instance",
			Fields:       map[string]string{"name": "acme-instance", "project": "acme-project"},
			IDTemplate:   "projects/{{ .project }}/instances/{{ .name }}",
		},
	}

	var buf bytes.Buffer
	err := RenderYAML(&buf, types)

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "google_compute_instance")
	assert.Contains(t, output, "types:")
	assert.Contains(t, output, "Available fields:")
}

func TestRenderYAML_EmptyTypes(t *testing.T) {
	var buf bytes.Buffer
	err := RenderYAML(&buf, nil)

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No resource types")
}

func TestRenderYAML_GroupedByProvider(t *testing.T) {
	types := []TypeInfo{
		{
			ResourceType: "google_compute_instance",
			Fields:       map[string]string{"name": "acme-instance"},
			IDTemplate:   "instance-{{ .name }}",
		},
		{
			ResourceType: "aws_s3_bucket",
			Fields:       map[string]string{"bucket": "acme-bucket"},
			IDTemplate:   "bucket-{{ .bucket }}",
		},
	}

	var buf bytes.Buffer
	err := RenderYAML(&buf, types)

	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "# Provider: aws")
	assert.Contains(t, output, "# Provider: google")
}

func TestProviderNamespace_Known(t *testing.T) {
	result := ProviderNamespace("google")
	assert.Equal(t, "hashicorp", result)
}

func TestProviderNamespace_KnownNonHashicorp(t *testing.T) {
	result := ProviderNamespace("gitlab")
	assert.Equal(t, "gitlabhq", result)
}

func TestProviderNamespace_Unknown(t *testing.T) {
	result := ProviderNamespace("custom")
	assert.Equal(t, "hashicorp", result)
}

func TestResourceSuffix_Normal(t *testing.T) {
	result := ResourceSuffix("google_compute_instance", "google")
	assert.Equal(t, "compute_instance", result)
}

func TestResourceSuffix_NoMatch(t *testing.T) {
	result := ResourceSuffix("custom_resource", "other")
	assert.Equal(t, "custom_resource", result)
}

func TestResourceSuffix_AwsType(t *testing.T) {
	result := ResourceSuffix("aws_s3_bucket", "aws")
	assert.Equal(t, "s3_bucket", result)
}
