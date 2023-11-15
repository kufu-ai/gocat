package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectFind(t *testing.T) {
	pl := &ProjectList{
		Items: []DeployProject{
			{
				ID:   "testid",
				Kind: "testkind",
			},
		},
	}
	want := DeployProject{
		ID:   "test",
		Kind: "testkind",
	}
	got := pl.Find("testid")
	require.Equal(t, want, got)
}
