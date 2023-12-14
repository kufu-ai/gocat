package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewInteractorKustomize(t *testing.T) {
	git := GitOperator{}
	github := GitHub{}
	i := InteractorContext{
		git:    git,
		github: github,
	}
	got := NewInteractorKustomize(i)

	want := InteractorGitOps{}
	want.kind = "kustomize"
	want.git = git
	want.github = github
	want.model = &GitOpsPluginKustomize{github: &github, git: &git}

	require.Equal(t, want, got)
}
