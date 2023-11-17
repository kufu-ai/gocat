package main

// GitOpsPluginKanvas is a gocat gitops plugin to prepare
// deployments using kanvas.
// This is used when you want to use gocat as a workflow engine
// with a chatops interface, while using kanvas as a deployment tool.
//
// Unlike GitOpsPluginKustomize which uses gocat's builtin Git and GitHub support,
// GitOpsPluginKanvas uses kanvas's Git and GitHub support.
type GitOpsPluginKanvas struct {
	github *GitHub
	git    *GitOperator
}

func NewGitOpsPluginKanvas(github *GitHub, git *GitOperator) GitOpsPlugin {
	return &GitOpsPluginKanvas{github: github, git: git}
}

func (k GitOpsPluginKanvas) Prepare(pj DeployProject, phase string, branch string, assigner User, tag string) (o GitOpsPrepareOutput, err error) {
	o.status = DeployStatusFail
	if tag == "" {
		ecr, err := CreateECRInstance()
		if err != nil {
			return o, err
		}
		tag, err = ecr.FindImageTagByRegexp(pj.ECRRegistryId(), pj.ECRRepository(), pj.ImageTagRegexp(), pj.TargetRegexp(), ImageTagVars{Branch: branch, Phase: phase})
		if err != nil {
			return o, err
		}
	}

	// TODO Enhance kanvas to create a git commit and branch from the image tag
	_ = tag

	// TODO Do kanvas deployment using specific PR title and description

	// TODO if the result code is "No difference", we should not create a pull request.
	// if tag == currentTag {
	// 	o.status = DeployStatusAlready
	// 	return
	// }

	// TODO Enhance kanvas to support setting assigner for the pull request
	// TODO Enhance kanvas to returning:
	// - pull request id
	// - pull request number
	// - pull request head branch

	o = GitOpsPrepareOutput{
		// PullRequestID:     prID,
		// PullRequestNumber: prNum,
		// Branch:            prBranch,
		// status:            DeployStatusSuccess,
	}
	return
}
