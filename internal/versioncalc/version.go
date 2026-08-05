// Package versioncalc derives reproducible WorkMuch versions from Git history.
package versioncalc

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const MainBaseTrailer = "WorkMuch-Main-Base"

var (
	tagPattern      = regexp.MustCompile(`^v[0-9]{8}\.[0-9]+\.[0-9]+\+g[0-9a-f]{12}$`)
	fullHashPattern = regexp.MustCompile(`^[0-9a-f]{40,64}$`)
)

type Options struct {
	Head     string
	MainBase string
}

type Result struct {
	Version          string
	Tag              string
	Head             string
	MainRef          string
	MainBase         string
	MainBaseSequence int
	Distance         int
}

func Calculate(ctx context.Context, directory string, options Options) (Result, error) {
	shallow, err := gitOutput(ctx, directory, "rev-parse", "--is-shallow-repository")
	if err != nil {
		return Result{}, fmt.Errorf("inspect Git history: %w", err)
	}
	if shallow == "true" {
		return Result{}, errors.New("cannot calculate a version from shallow Git history")
	}

	mainRef, err := resolveMain(ctx, directory)
	if err != nil {
		return Result{}, err
	}

	headRef := options.Head
	if headRef == "" {
		headRef = "HEAD"
	}
	head, err := resolveCommit(ctx, directory, headRef)
	if err != nil {
		return Result{}, fmt.Errorf("resolve release commit %q: %w", headRef, err)
	}

	var mainBase string
	if options.MainBase == "" {
		mainBase, err = gitOutput(ctx, directory, "merge-base", head, mainRef)
		if err != nil {
			return Result{}, fmt.Errorf("find merge-base of %s and %s: %w", head, mainRef, err)
		}
	} else {
		mainBase, err = resolveCommit(ctx, directory, options.MainBase)
		if err != nil {
			return Result{}, fmt.Errorf("resolve recorded main base %q: %w", options.MainBase, err)
		}
		if err := requireAncestor(ctx, directory, mainBase, mainRef); err != nil {
			return Result{}, fmt.Errorf("recorded main base is not in %s history: %w", mainRef, err)
		}
	}
	if err := requireAncestor(ctx, directory, mainBase, head); err != nil {
		return Result{}, fmt.Errorf("main base is not in release commit history: %w", err)
	}

	committerSecondsText, err := gitOutput(ctx, directory, "show", "-s", "--format=%ct", mainBase)
	if err != nil {
		return Result{}, fmt.Errorf("read main-base committer date: %w", err)
	}
	committerSeconds, err := strconv.ParseInt(committerSecondsText, 10, 64)
	if err != nil {
		return Result{}, fmt.Errorf("parse main-base committer date %q: %w", committerSecondsText, err)
	}
	date := time.Unix(committerSeconds, 0).UTC().Format("20060102")

	sequence, err := gitCount(ctx, directory, "rev-list", "--first-parent", "--count", mainBase)
	if err != nil {
		return Result{}, fmt.Errorf("count first-parent history through main base: %w", err)
	}
	distance, err := gitCount(ctx, directory, "rev-list", "--count", mainBase+".."+head)
	if err != nil {
		return Result{}, fmt.Errorf("count commits from main base to release commit: %w", err)
	}
	if len(head) < 12 {
		return Result{}, fmt.Errorf("release commit hash %q is shorter than 12 characters", head)
	}

	version := fmt.Sprintf("%s.%d.%d+g%s", date, sequence, distance, head[:12])
	return Result{
		Version:          version,
		Tag:              "v" + version,
		Head:             head,
		MainRef:          mainRef,
		MainBase:         mainBase,
		MainBaseSequence: sequence,
		Distance:         distance,
	}, nil
}

func ValidateTag(ctx context.Context, directory string, tag string) (Result, error) {
	if !tagPattern.MatchString(tag) {
		return Result{}, fmt.Errorf("tag %q does not use the WorkMuch version format", tag)
	}

	ref := "refs/tags/" + tag
	objectType, err := gitOutput(ctx, directory, "cat-file", "-t", ref)
	if err != nil {
		return Result{}, fmt.Errorf("read tag %q: %w", tag, err)
	}
	if objectType != "tag" {
		return Result{}, fmt.Errorf("tag %q must be annotated", tag)
	}

	contents, err := gitOutput(ctx, directory, "for-each-ref", "--format=%(contents)", ref)
	if err != nil {
		return Result{}, fmt.Errorf("read annotated tag %q: %w", tag, err)
	}
	mainBase, err := parseMainBase(contents)
	if err != nil {
		return Result{}, fmt.Errorf("validate tag %q: %w", tag, err)
	}

	head, err := resolveCommit(ctx, directory, ref)
	if err != nil {
		return Result{}, fmt.Errorf("resolve commit for tag %q: %w", tag, err)
	}
	result, err := Calculate(ctx, directory, Options{Head: head, MainBase: mainBase})
	if err != nil {
		return Result{}, fmt.Errorf("calculate version for tag %q: %w", tag, err)
	}
	if result.Tag != tag {
		return Result{}, fmt.Errorf("tag %q does not match calculated tag %q", tag, result.Tag)
	}
	return result, nil
}

func parseMainBase(contents string) (string, error) {
	prefix := MainBaseTrailer + ": "
	var found []string
	for _, line := range strings.Split(contents, "\n") {
		if strings.HasPrefix(line, prefix) {
			found = append(found, strings.TrimSpace(strings.TrimPrefix(line, prefix)))
		}
	}
	if len(found) != 1 || !fullHashPattern.MatchString(found[0]) {
		return "", fmt.Errorf("annotated tag must contain exactly one %s: <full-sha> line", MainBaseTrailer)
	}
	return found[0], nil
}

func resolveMain(ctx context.Context, directory string) (string, error) {
	if symbolic, err := gitOutput(ctx, directory, "symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD"); err == nil {
		if _, resolveErr := resolveCommit(ctx, directory, symbolic); resolveErr == nil {
			return symbolic, nil
		}
	}
	for _, candidate := range []string{"origin/main", "origin/master"} {
		if _, err := resolveCommit(ctx, directory, candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("cannot resolve main history from origin/HEAD, origin/main, or origin/master")
}

func resolveCommit(ctx context.Context, directory string, ref string) (string, error) {
	return gitOutput(ctx, directory, "rev-parse", "--verify", ref+"^{commit}")
}

func requireAncestor(ctx context.Context, directory string, ancestor string, descendant string) error {
	cmd := exec.CommandContext(ctx, "git", "merge-base", "--is-ancestor", ancestor, descendant)
	cmd.Dir = directory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git merge-base --is-ancestor %s %s: %w: %s", ancestor, descendant, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func gitCount(ctx context.Context, directory string, args ...string) (int, error) {
	output, err := gitOutput(ctx, directory, args...)
	if err != nil {
		return 0, err
	}
	count, err := strconv.Atoi(output)
	if err != nil {
		return 0, fmt.Errorf("parse count %q: %w", output, err)
	}
	return count, nil
}

func gitOutput(ctx context.Context, directory string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = directory
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}
