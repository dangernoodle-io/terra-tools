package importer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePlan_BinaryNotFound(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := GeneratePlan(context.Background(), ".", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terraform")
}

func TestTerragruntGeneratePlan_BinaryNotFound(t *testing.T) {
	t.Setenv("PATH", "")

	_, err := TerragruntGeneratePlan(context.Background(), ".", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "terragrunt")
}
