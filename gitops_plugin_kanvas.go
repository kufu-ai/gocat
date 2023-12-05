package main

import (
	"context"
	"fmt"

	"github.com/davinci-std/kanvas/client"
	"github.com/davinci-std/kanvas/client/cli"
)

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

func (k GitOpsPluginKanvas) Prepare(pj DeployProject, phase string, branch string, assigner User, tag string) (GitOpsPrepareOutput, error) {
	var o GitOpsPrepareOutput

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
		return o, err
	}

	if tag == currentTag {
		o.status = DeployStatusAlready
		return o, nil
	}

	head := fmt.Sprintf("bot/docker-image-tag-%s-%s-%s", pj.ID, ph.Name, tag)
	wt, err := k.git.createAndCheckoutNewBranch(head)
	if err != nil {
		return o, err
	}

	c := cli.New()

	applyOpts := client.ApplyOptions{
		SkippedComponents: map[string]map[string]string{
			"image": {
				"tag": tag,
			},
		},
		PullRequestHead: head,
	}

	if assigner.GitHubNodeID != "" {
		applyOpts.PullRequestAssigneeIDs = []string{assigner.GitHubNodeID}
	}

	r, err := c.Apply(context.Background(), wt.Filesystem.Join("kanvas.yaml"), "production", applyOpts)
	if err != nil {
		return o, err
	}

	var (
		prID  string
		prNum int
	)

	for _, o := range r.Outputs {
		if o.PullRequest != nil {
			prID = fmt.Sprintf("%d", o.PullRequest.ID)
			prNum = o.PullRequest.Number
			break
		}
	}

	o = GitOpsPrepareOutput{
		PullRequestID:     prID,
		PullRequestNumber: prNum,
		Branch:            head,
		status:            DeployStatusSuccess,
	}
	return o, nil
}
