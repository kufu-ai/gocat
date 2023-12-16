package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewInteractorKanvas(t *testing.T) {
	git := GitOperator{}
	github := GitHub{}
	i := InteractorContext{
		git:    git,
		github: github,
	}
	got := NewInteractorKanavs(i)

	i.kind = "kanvas"
	want := InteractorGitOps{}
	want.kind = "kanvas"
	want.git = git
	want.github = github
	want.model = &GitOpsPluginKanvas{github: &github, git: &git}

	require.Equal(t, want, got)
}
