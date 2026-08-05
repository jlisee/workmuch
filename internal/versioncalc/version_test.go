package versioncalc

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type gitRepository struct {
	dir string
}

func newGitRepository(t *testing.T) *gitRepository {
	t.Helper()

	repo := &gitRepository{dir: t.TempDir()}
	repo.git(t, "init", "--initial-branch=main")
	repo.git(t, "config", "user.name", "WorkMuch Tests")
	repo.git(t, "config", "user.email", "workmuch@example.test")
	return repo
}

func (r *gitRepository) git(t *testing.T, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %s failed:\n%s", strings.Join(args, " "), output)
	return strings.TrimSpace(string(output))
}

func (r *gitRepository) commit(t *testing.T, name string, timestamp string) string {
	t.Helper()

	path := filepath.Join(r.dir, "history")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	require.NoError(t, err)
	_, err = file.WriteString(name + "\n")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	r.git(t, "add", "history")

	cmd := exec.Command("git", "commit", "-m", name)
	cmd.Dir = r.dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_DATE="+timestamp,
		"GIT_COMMITTER_DATE="+timestamp,
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git commit failed:\n%s", output)
	return r.git(t, "rev-parse", "HEAD")
}

func (r *gitRepository) setOriginMain(t *testing.T, branch string, head string, symbolicHead bool) {
	t.Helper()
	r.git(t, "update-ref", "refs/remotes/origin/"+branch, head)
	if symbolicHead {
		r.git(t, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/"+branch)
	}
}

func TestCalculateMainVersion(t *testing.T) {
	repo := newGitRepository(t)
	repo.commit(t, "root", "2026-08-01T12:00:00Z")
	head := repo.commit(t, "main", "2026-08-05T12:00:00Z")
	repo.setOriginMain(t, "main", head, true)

	result, err := Calculate(context.Background(), repo.dir, Options{})
	require.NoError(t, err)

	assert.Equal(t, head, result.Head)
	assert.Equal(t, head, result.MainBase)
	assert.Equal(t, "origin/main", result.MainRef)
	assert.Equal(t, 2, result.MainBaseSequence)
	assert.Equal(t, 0, result.Distance)
	assert.Equal(t, "20260805.2.0+g"+head[:12], result.Version)
	assert.Equal(t, "v"+result.Version, result.Tag)
}

func TestCalculateDivergentBranchVersion(t *testing.T) {
	repo := newGitRepository(t)
	repo.commit(t, "root", "2026-08-01T12:00:00Z")
	base := repo.commit(t, "main", "2026-08-02T12:00:00Z")
	repo.setOriginMain(t, "main", base, true)
	repo.git(t, "switch", "-c", "feature")
	repo.commit(t, "feature one", "2026-08-03T12:00:00Z")
	head := repo.commit(t, "feature two", "2026-08-04T12:00:00Z")

	result, err := Calculate(context.Background(), repo.dir, Options{})
	require.NoError(t, err)

	assert.Equal(t, base, result.MainBase)
	assert.Equal(t, 2, result.MainBaseSequence)
	assert.Equal(t, 2, result.Distance)
	assert.Equal(t, "20260802.2.2+g"+head[:12], result.Version)
}

func TestCalculateUsesUTCCommitterDate(t *testing.T) {
	repo := newGitRepository(t)
	head := repo.commit(t, "root", "2026-08-05T23:30:00-07:00")
	repo.setOriginMain(t, "main", head, true)

	result, err := Calculate(context.Background(), repo.dir, Options{})
	require.NoError(t, err)

	assert.Equal(t, "20260806.1.0+g"+head[:12], result.Version)
}

func TestCalculateFallsBackToOriginMain(t *testing.T) {
	repo := newGitRepository(t)
	head := repo.commit(t, "root", "2026-08-05T12:00:00Z")
	repo.setOriginMain(t, "main", head, false)

	result, err := Calculate(context.Background(), repo.dir, Options{})
	require.NoError(t, err)
	assert.Equal(t, "origin/main", result.MainRef)
}

func TestCalculateFallsBackToOriginMaster(t *testing.T) {
	repo := newGitRepository(t)
	head := repo.commit(t, "root", "2026-08-05T12:00:00Z")
	repo.setOriginMain(t, "master", head, false)

	result, err := Calculate(context.Background(), repo.dir, Options{})
	require.NoError(t, err)
	assert.Equal(t, "origin/master", result.MainRef)
}

func TestCalculateRejectsMissingMainHistory(t *testing.T) {
	repo := newGitRepository(t)
	repo.commit(t, "root", "2026-08-05T12:00:00Z")

	_, err := Calculate(context.Background(), repo.dir, Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "origin/HEAD, origin/main, or origin/master")
}

func TestCalculateRejectsShallowHistory(t *testing.T) {
	source := newGitRepository(t)
	head := source.commit(t, "root", "2026-08-05T12:00:00Z")
	source.setOriginMain(t, "main", head, true)

	cloneDir := filepath.Join(t.TempDir(), "clone")
	cmd := exec.Command("git", "clone", "--depth=1", "file://"+source.dir, cloneDir)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git clone failed:\n%s", output)

	_, err = Calculate(context.Background(), cloneDir, Options{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "shallow")
}

func TestCalculateWithRecordedBaseDoesNotFollowAdvancedMain(t *testing.T) {
	repo := newGitRepository(t)
	repo.commit(t, "root", "2026-08-01T12:00:00Z")
	base := repo.commit(t, "base", "2026-08-02T12:00:00Z")
	repo.git(t, "switch", "-c", "release")
	head := repo.commit(t, "release", "2026-08-03T12:00:00Z")
	repo.git(t, "switch", "main")
	advancedMain := repo.commit(t, "advanced main", "2026-08-04T12:00:00Z")
	repo.setOriginMain(t, "main", advancedMain, true)

	result, err := Calculate(context.Background(), repo.dir, Options{Head: head, MainBase: base})
	require.NoError(t, err)

	assert.Equal(t, base, result.MainBase)
	assert.Equal(t, 1, result.Distance)
	assert.Equal(t, "20260802.2.1+g"+head[:12], result.Version)
}

func TestValidateAnnotatedTag(t *testing.T) {
	repo := newGitRepository(t)
	repo.commit(t, "root", "2026-08-01T12:00:00Z")
	base := repo.commit(t, "base", "2026-08-02T12:00:00Z")
	repo.setOriginMain(t, "main", base, true)
	repo.git(t, "switch", "-c", "release")
	head := repo.commit(t, "release", "2026-08-03T12:00:00Z")

	result, err := Calculate(context.Background(), repo.dir, Options{})
	require.NoError(t, err)
	message := "Release " + result.Tag + "\n\n" + MainBaseTrailer + ": " + base
	repo.git(t, "tag", "-a", result.Tag, "-m", message, head)

	validated, err := ValidateTag(context.Background(), repo.dir, result.Tag)
	require.NoError(t, err)
	assert.Equal(t, result, validated)
}

func TestValidateTagRejectsLightweightTag(t *testing.T) {
	repo := newGitRepository(t)
	head := repo.commit(t, "root", "2026-08-05T12:00:00Z")
	repo.setOriginMain(t, "main", head, true)
	result, err := Calculate(context.Background(), repo.dir, Options{})
	require.NoError(t, err)
	repo.git(t, "tag", result.Tag)

	_, err = ValidateTag(context.Background(), repo.dir, result.Tag)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "annotated")
}

func TestValidateTagRejectsVersionMismatch(t *testing.T) {
	repo := newGitRepository(t)
	head := repo.commit(t, "root", "2026-08-05T12:00:00Z")
	repo.setOriginMain(t, "main", head, true)
	message := "Release mismatch\n\n" + MainBaseTrailer + ": " + head
	repo.git(t, "tag", "-a", "v20260805.1.1+g"+head[:12], "-m", message)

	_, err := ValidateTag(context.Background(), repo.dir, "v20260805.1.1+g"+head[:12])
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match calculated tag")
}

func TestValidateTagRejectsInvalidRecordedBase(t *testing.T) {
	repo := newGitRepository(t)
	head := repo.commit(t, "root", "2026-08-05T12:00:00Z")
	repo.setOriginMain(t, "main", head, true)
	result, err := Calculate(context.Background(), repo.dir, Options{})
	require.NoError(t, err)
	repo.git(t, "tag", "-a", result.Tag, "-m", "Release without metadata")

	_, err = ValidateTag(context.Background(), repo.dir, result.Tag)
	require.Error(t, err)
	assert.Contains(t, err.Error(), MainBaseTrailer)
}
