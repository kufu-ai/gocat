package main

import (
	"fmt"
	"strings"
)

type ModelKustomize struct {
	github *GitHub
	git    *GitOperator
}

func NewModelKustomize(github *GitHub, git *GitOperator) ModelKustomize {
	return ModelKustomize{github: github, git: git}
}

type ModelKustomizePrepareOutput struct {
	PullRequestID     string
	PullRequestNumber int
	Branch            string
	status            DeployStatus
}

func (self ModelKustomizePrepareOutput) Status() DeployStatus {
	return self.status
}

func (self ModelKustomizePrepareOutput) Message() string {
	return "Success to deploy"
}

func (self ModelKustomize) Prepare(pj DeployProject, phase string, branch string, assigner User) (o ModelKustomizePrepareOutput, err error) {
	o.status = DeployStatusFail
	ecr, err := CreateECRInstance()
	if err != nil {
		return
	}
	tag, err := ecr.FindImageTagByRegexp(pj.ECRRepository(), pj.FilterRegexp(), pj.TargetRegexp(), ImageTagVars{Branch: branch, Phase: phase})
	if err != nil {
		return
	}

	ph := pj.FindPhase(phase)
	currentTag, err := ph.Destination.GetCurrentRevision(GetCurrentRevisionInput{github: self.github})
	if err != nil {
		return
	}

	if tag == currentTag {
		o.status = DeployStatusAlready
		return
	}

	commits, err := self.github.CommitsBetween(GitHubCommitsBetweenInput{
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

	prBranch, err := self.git.PushDockerImageTag(pj.ID, ph, tag, pj.DockerRepository())
	if err != nil {
		return
	}

	prID, prNum, err := self.github.CreatePullRequest(prBranch, fmt.Sprintf("Deploy %s %s", pj.ID, branch), commitlog)
	if err != nil {
		return
	}

	if assigner.GitHubNodeID != "" {
		err = self.github.UpdatePullRequest(prID, assigner.GitHubNodeID)
		if err != nil {
			return
		}
	}

	o = ModelKustomizePrepareOutput{
		PullRequestID:     prID,
		PullRequestNumber: prNum,
		Branch:            prBranch,
		status:            DeployStatusSuccess,
	}
	return
}

func (self ModelKustomize) Commit(pullRequestID string) error {
	return self.github.MergePullRequest(pullRequestID)

}

func (self ModelKustomize) Deploy(pj DeployProject, phase string, option DeployOption) (do DeployOutput, err error) {
	o, err := self.Prepare(pj, phase, option.Branch, option.Assigner)
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
