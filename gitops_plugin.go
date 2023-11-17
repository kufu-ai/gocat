package main

// GitOpsPlugin is the extension point for InteractorGitOps
// It is used to support various GitOps tools.
type GitOpsPlugin interface {
	Prepare(pj DeployProject, phase string, branch string, user User, message string) (o GitOpsPrepareOutput, err error)
}
