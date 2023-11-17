package main

func NewModelKanvas(github *GitHub, git *GitOperator) ModelGitOps {
	return ModelGitOps{
		github: github,
		git:    git,
		plugin: NewGitOpsPluginKanvas(github, git),
	}
}
