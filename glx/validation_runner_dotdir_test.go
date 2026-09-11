package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountGLXFiles_SkipsDotDirectories(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "persons"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "persons", "a.glx"), []byte("a"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".claude", "worktrees", "copy", "persons"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".claude", "worktrees", "copy", "persons", "a.glx"), []byte("a"), 0o644))

	assert.Equal(t, 1, countGLXFiles(dir))
}
