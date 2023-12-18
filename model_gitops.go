package main

type ModelGitOps struct {
	github *GitHub
	git    *GitOperator
	plugin GitOpsPlugin
}

type GitOpsPrepareOutput struct {
	// PullRequestID is the node ID of the pull request.
	// It's used to merge and close the pull request using GitHub GraphQL API.
	//
	// Note that this is not the pull request ID, which is an integer,
	// and used by non-GraphQL API to identify a pull request.
	PullRequestID      string
	PullRequestNumber  int
	PullRequestHTMLURL string
	Branch             string
	status             DeployStatus
}

func (self GitOpsPrepareOutput) Status() DeployStatus {
	return self.status
}

func (self GitOpsPrepareOutput) Message() string {
	return "Success to deploy"
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
