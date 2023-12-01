package main

func NewModelKustomize(github *GitHub, git *GitOperator) ModelGitOps {
	return ModelGitOps{
		github: github,
		git:    git,
		plugin: NewGitOpsPluginKustomize(github, git),
	}
}
