package main

// GitOpsPluginKustomize is a gocat gitops plugin to prepare
// deployments using kustomize and gocat's builtin Git and GitHub support.
// This is used when you want to use gocat as a workflow engine
// with a chatops interface, while using kustomize along with
// the gocat native features as a deployment tool.
type GitOpsPluginKustomize struct {
	github *GitHub
	git    *GitOperator
}

func NewGitOpsPluginKustomize(github *GitHub, git *GitOperator) GitOpsPlugin {
	return &GitOpsPluginKustomize{github: github, git: git}
}
