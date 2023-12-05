package main

import "testing"

func TestGit_FSOS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var o GitOperator
	o.repo = "https://github.com/zaiminc/gocat"
	o.defaultBranch = "master"
	o.gitRoot = t.TempDir()

	if err := o.Clone(); err != nil {
		t.Fatal(err)
	}
}

func TestGit_Mem(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	var o GitOperator
	o.repo = "https://github.com/zaiminc/gocat"
	o.defaultBranch = "master"

	if err := o.Clone(); err != nil {
		t.Fatal(err)
	}
}
