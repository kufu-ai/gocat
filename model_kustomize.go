package main

import (
	"fmt"
	"strings"
)

type ModelGitOps struct {
	github *GitHub
	git    *GitOperator
	plugin GitOpsPlugin
}

func NewModelKustomize(github *GitHub, git *GitOperator) ModelGitOps {
	return ModelGitOps{
		github: github,
		git:    git,
		plugin: NewGitOpsPluginKustomize(github, git),
	}
}

type GitOpsPrepareOutput struct {
	PullRequestID     string
	PullRequestNumber int
	Branch            string
	status            DeployStatus
}

func (self GitOpsPrepareOutput) Status() DeployStatus {
	return self.status
}

func (self GitOpsPrepareOutput) Message() string {
	return "Success to deploy"
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

func (self ModelGitOps) Commit(pullRequestID string) error {
	return self.github.MergePullRequest(pullRequestID)

}

func (self ModelGitOps) Deploy(pj DeployProject, phase string, option DeployOption) (do DeployOutput, err error) {
	o, err := self.plugin.Prepare(pj, phase, option.Branch, option.Assigner, option.Tag)
	if err != nil {
		return
	}
	if o.Status() == DeployStatusSuccess {
		err = self.Commit(o.PullRequestID)
		if err != nil {
			return
		}
	}
	return o, nil
}
