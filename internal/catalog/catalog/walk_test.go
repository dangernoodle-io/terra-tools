package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWalk_MinimalCatalog(t *testing.T) {
	dir := t.TempDir()

	// Create root/terragrunt-root.hcl
	rootDir := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "terragrunt-root.hcl"), []byte(`# root`), 0o644))

	// Create project/cloud-run/terragrunt.hcl
	projectDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "cloud-run"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "cloud-run", "terragrunt.hcl"),
		[]byte(`dependency "redis" { config_path = "../redis" }`),
		0o644,
	))

	layout, err := Walk(dir)

	require.NoError(t, err)
	require.Len(t, layout.Services, 1)
	assert.Contains(t, layout.Services, "cloud-run")
	assert.Equal(t, "cloud-run", layout.Services["cloud-run"].Path)
	assert.False(t, layout.Services["cloud-run"].IsRegion)
}

func TestWalk_RegionalService(t *testing.T) {
	dir := t.TempDir()

	// Create root/terragrunt-root.hcl
	rootDir := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "terragrunt-root.hcl"), []byte(`# root`), 0o644))

	// Create project/region/redis/terragrunt.hcl
	projectDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "region", "redis"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "region", "redis", "terragrunt.hcl"),
		[]byte(`# service config`),
		0o644,
	))

	layout, err := Walk(dir)

	require.NoError(t, err)
	require.Len(t, layout.Services, 1)
	assert.Contains(t, layout.Services, "region/redis")
	assert.Equal(t, "region/redis", layout.Services["region/redis"].Path)
	assert.True(t, layout.Services["region/redis"].IsRegion)
}

func TestWalk_MissingRootConfig(t *testing.T) {
	dir := t.TempDir()

	// Create project/ but no root/terragrunt-root.hcl
	projectDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0o755))

	_, err := Walk(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing root")
}

func TestWalk_MissingProjectDir(t *testing.T) {
	dir := t.TempDir()

	// Create root/terragrunt-root.hcl but no project/
	rootDir := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "terragrunt-root.hcl"), []byte(`# root`), 0o644))

	_, err := Walk(dir)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing project")
}

func TestWalk_SkipsDirsWithoutTerragrunt(t *testing.T) {
	dir := t.TempDir()

	// Create root/terragrunt-root.hcl
	rootDir := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "terragrunt-root.hcl"), []byte(`# root`), 0o644))

	// Create project/empty-dir/ (no terragrunt.hcl)
	projectDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "empty-dir"), 0o755))

	// Create project/cloud-run/terragrunt.hcl
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "cloud-run"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "cloud-run", "terragrunt.hcl"),
		[]byte(`# service`),
		0o644,
	))

	layout, err := Walk(dir)

	require.NoError(t, err)
	require.Len(t, layout.Services, 1)
	assert.Contains(t, layout.Services, "cloud-run")
	assert.NotContains(t, layout.Services, "empty-dir")
}

func TestWalk_ParsesDependencies(t *testing.T) {
	dir := t.TempDir()

	// Create root/terragrunt-root.hcl
	rootDir := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "terragrunt-root.hcl"), []byte(`# root`), 0o644))

	// Create project/cloud-run/terragrunt.hcl with dependencies
	projectDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "cloud-run"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "cloud-run", "terragrunt.hcl"),
		[]byte(`
dependency "redis" {
  config_path = "../redis"
}

dependency "firestore" {
  config_path = "../firestore"
}
`),
		0o644,
	))

	layout, err := Walk(dir)

	require.NoError(t, err)
	require.Len(t, layout.Services, 1)
	svc := layout.Services["cloud-run"]
	require.Len(t, svc.Dependencies, 2)
	assert.Contains(t, svc.Dependencies, "redis")
	assert.Contains(t, svc.Dependencies, "firestore")
}

func TestWalk_MultipleServices(t *testing.T) {
	dir := t.TempDir()

	// Create root/terragrunt-root.hcl
	rootDir := filepath.Join(dir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "terragrunt-root.hcl"), []byte(`# root`), 0o644))

	// Create project/cloud-run/terragrunt.hcl
	projectDir := filepath.Join(dir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "cloud-run"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "cloud-run", "terragrunt.hcl"),
		[]byte(`# cloud-run`),
		0o644,
	))

	// Create project/redis/terragrunt.hcl
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "redis"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "redis", "terragrunt.hcl"),
		[]byte(`# redis`),
		0o644,
	))

	// Create project/region/firestore/terragrunt.hcl
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, "region", "firestore"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(projectDir, "region", "firestore", "terragrunt.hcl"),
		[]byte(`# firestore`),
		0o644,
	))

	layout, err := Walk(dir)

	require.NoError(t, err)
	require.Len(t, layout.Services, 3)
	assert.Contains(t, layout.Services, "cloud-run")
	assert.Contains(t, layout.Services, "redis")
	assert.Contains(t, layout.Services, "region/firestore")
	assert.False(t, layout.Services["cloud-run"].IsRegion)
	assert.False(t, layout.Services["redis"].IsRegion)
	assert.True(t, layout.Services["region/firestore"].IsRegion)
}
