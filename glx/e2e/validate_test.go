package e2e

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A git worktree checked out under .claude/worktrees/ is a full copy of the
// archive inside the archive (#1212). Every entity would collide with itself
// if the walk entered it.
func TestValidate_IgnoresArchiveCopyUnderDotDirectory(t *testing.T) {
	archive := copyExample(t, "basic-family")

	clean := runGLX(t, archive, "validate", ".")
	require.Equal(t, 0, clean.exitCode, clean.stdout+clean.stderr)

	copyTreeFollowingSymlinks(t, filepath.Join(examplesDir(t), "basic-family"), filepath.Join(archive, ".claude", "worktrees", "copy"))

	res := runGLX(t, archive, "validate", ".")

	assert.Equal(t, 0, res.exitCode, res.stdout+res.stderr)
	assert.Equal(t, clean.stdout, res.stdout)
	assert.NotContains(t, res.stderr, "conflict")
}
