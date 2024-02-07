package main

import (
	"fmt"
	"strings"
)

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

func (k GitOpsPluginKustomize) Prepare(pj DeployProject, phase string, branch string, assigner User, tag string) (o GitOpsPrepareOutput, err error) {
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

	ph := pj.FindPhase(phase)
	currentTag, err := ph.Destination.GetCurrentRevision(GetCurrentRevisionInput{github: k.github})
	if err != nil {
		return
	}

	if tag == currentTag {
		o.status = DeployStatusAlready
		return
	}

	commits, err := k.github.CommitsBetween(GitHubCommitsBetweenInput{
		Repository:    pj.GitHubRepository(),
		Branch:        branch,
		FirstCommitID: currentTag,
		LastCommitID:  tag,
	})
	if err != nil {
		return
	}

	commitlog := "*Commit Log*\n"
	for _, c := range commits {
		m := strings.Replace(c.Message, "\n", " ", -1)
		commitlog = commitlog + "- " + m + "\n"
	}

	prBranch, err := k.git.PushDockerImageTag(pj.ID, ph, tag, pj.DockerRepository())
	if err != nil {
		return
	}

	prID, prNum, err := k.github.CreatePullRequest(prBranch, fmt.Sprintf("Deploy %s %s", pj.ID, branch), commitlog)
	if err != nil {
		return
	}

	if assigner.GitHubNodeID != "" {
		err = k.github.UpdatePullRequest(prID, assigner.GitHubNodeID)
		if err != nil {
			return
		}
	}

	o = GitOpsPrepareOutput{
		PullRequestID:     prID,
		PullRequestNumber: prNum,
		Branch:            prBranch,
		status:            DeployStatusSuccess,
	}
	return
}
