package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetch_LocalPathExists(t *testing.T) {
	catalogPath := t.TempDir()

	// Write a placeholder file so the directory has content.
	placeholder := filepath.Join(catalogPath, "test.txt")
	require.NoError(t, os.WriteFile(placeholder, []byte("test"), 0o644))

	result, cleanup, err := Fetch(catalogPath)

	require.NoError(t, err)
	assert.Equal(t, catalogPath, result)

	// Cleanup should be a no-op for local sources.
	cleanup()
}

func TestFetch_LocalPathNotFound(t *testing.T) {
	nonexistentPath := "/nonexistent/path/to/catalog"

	_, cleanup, err := Fetch(nonexistentPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Even on error, cleanup function should be callable without panic.
	cleanup()
}

func TestFetch_RemoteSourceNoGit(t *testing.T) {
	// Set PATH to empty to ensure git command is not found.
	t.Setenv("PATH", "")

	gitURL := "git::https://example.com/repo.git"

	_, cleanup, err := Fetch(gitURL)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cloning")

	// Cleanup should not panic even on error.
	cleanup()
}
