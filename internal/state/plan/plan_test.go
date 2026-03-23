package plan

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validPlanJSON = `{
  "format_version": "1.0",
  "resource_changes": [
    {
      "address": "google_project_service.apis",
      "type": "google_project_service",
      "change": {"actions": ["create"]}
    },
    {
      "address": "google_project_service.existing",
      "type": "google_project_service",
      "change": {"actions": ["no-op"]}
    }
  ]
}`

const noCreatesPlanJSON = `{
  "format_version": "1.0",
  "resource_changes": [
    {
      "address": "google_project_service.existing",
      "type": "google_project_service",
      "change": {"actions": ["no-op"]}
    }
  ]
}`

func TestParse_Valid(t *testing.T) {
	p, err := Parse(strings.NewReader(validPlanJSON))
	require.NoError(t, err)
	assert.Len(t, p.ResourceChanges, 2)
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := Parse(strings.NewReader(`not valid json {{{`))
	require.Error(t, err)
}

func TestFilterCreates_OnlyCreates(t *testing.T) {
	p, err := Parse(strings.NewReader(validPlanJSON))
	require.NoError(t, err)

	creates := FilterCreates(p)
	require.Len(t, creates, 1)
	assert.Equal(t, "google_project_service.apis", creates[0].Address)
}

func TestFilterCreates_Empty(t *testing.T) {
	p, err := Parse(strings.NewReader(noCreatesPlanJSON))
	require.NoError(t, err)

	creates := FilterCreates(p)
	assert.Empty(t, creates)
}
