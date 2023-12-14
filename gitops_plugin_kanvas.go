package main

import (
	"context"
	"fmt"
	"path/filepath"

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
	if ph.Name == "" {
		return o, fmt.Errorf("phase %s not found for project %s", phase, pj.ID)
	}

	// The head of the pull request is bot/docker-image-tag-<project_id>-<phase_name>-<tag>.
	// And it's used by kanvas to create a pull request against the master or the main branch of the repository
	// specified in the kanvas.yaml, not the repository that contains kanvas.yaml.
	//
	// Let's say you have two repositories:
	// - myapp
	// - infra
	//
	// myapp contains kanvas.yaml and infra contains the actual deployment configuration files.
	//
	// In this case, the head of the pull request is bot/docker-image-tag-<project_id>-<phase_name>-<tag>
	// in the infra repository, not the myapp repository.
	head := fmt.Sprintf("bot/docker-image-tag-%s-%s-%s", pj.ID, ph.Name, tag)

	// Treat the kanvas.yaml as the way to generate the desired state of the deployment,
	// not the desired state itself.
	// That's why we don't create pull requests against the master or the main branch of the repository
	// that contains kanvas.yaml!
	//
	// We use kanvas.yaml in the master or the main branch to do the deployment.
	//
	// Unlike gocat kustomize model, we don't create pull requests against the master or the main branch of
	// the repository that contains kanvas.yaml.
	//
	// Instead, we let kanvas to create pull requests against the master or the main branch of the repository
	// as defined in the kanvas.yaml.

	wt, err := k.git.createAndCheckoutNewBranch(head)
	if err != nil {
		return o, err
	}

	c := cli.New()

	applyOpts := client.ApplyOptions{
		SkippedComponents: map[string]map[string]string{
			// Any kanvas.yaml that can be used by gocat needs to have
			// "image" component that uses the kanavs's docker provider for building the container image.
			"image": {
				"tag": tag,
			},
		},
		PullRequestHead: head,
	}

	if assigner.GitHubNodeID != "" {
		applyOpts.PullRequestAssigneeIDs = []string{assigner.GitHubNodeID}
	}

	// path is the relative path to the kanvas.yaml from the root of the repository.
	//
	// In case the project's Phases look like this:
	//
	// 	Phases: |
	// 	- name: staging
	// 	  path: path/to/kanvas.yaml
	// 	- name: production
	// 	  path: path/to
	//
	// The path to the config file is path/to/kanvas.yaml in both cases.
	path := ph.Path
	if path == "" {
		path = "kanvas.yaml"
	} else if filepath.Base(path) != "kanvas.yaml" {
		path = filepath.Join(path, "kanvas.yaml")
	}

	// We treat gocat "phase" as kanvas "environment".
	//
	// This means that kanvas.yaml need to have all the environments
	// corresponding to the gocat phases.
	//
	// Let's say you have two gocat phases: staging and production.
	// kanvas.yaml need to have both staging and production environments like this:
	// 	environments:
	//    staging: ...
	//    production: ...
	//    sandbox: ...
	//    local: ...
	// 	components:
	//   ...
	//
	// This Apply call corresponds to the following kanvas command:
	//
	// 	KANVAS_PULLREQUEST_ASSIGNEE_IDS=< assigner.GitHubNodeID > \
	// 	KANVAS_PULLREQUEST_HEAD=< head > \
	// 	 kanvas apply --env <phase> --config <path> --skipped-jobs-outputs '{"image":{"tag":"<tag>"}}'
	//
	r, err := c.Apply(context.Background(), wt.Filesystem.Join(path), phase, applyOpts)
	if err != nil {
		return o, err
	}

	var (
		prCreated bool
		prID      string
		prNum     int
		// prHTMLURL string
	)

	for _, o := range r.Outputs {
		if o.PullRequest != nil {
			prCreated = true

			prID = fmt.Sprintf("%d", o.PullRequest.ID)
			prNum = o.PullRequest.Number
			// prHTMLURL is the URL to the pull request in the config repository,
			// not the repository that contains kanvas.yaml.
			//
			// This is necessary because unlike the kustomize model that creates pull requests
			// against the specified repository, kanvas uses the kanvas.yaml in the repository
			// specified in the gocat configmap, to create pull requests against the repository
			// specified in the kanvas.yaml.
			// prHTMLURL = o.PullRequest.HTMLURL

			break
		}
	}

	if !prCreated {
		// Instead of determining if the desired image tag is already deployed or not by
		// getting the current tag using:
		//
		// 	ph.Destination.GetCurrentRevision(GetCurrentRevisionInput{github: k.github})
		//
		// we determine it by checking if the pull request is created or not.
		//
		// That's possible because, if the image.tag is already deployed, kanvas won't create a pull request.
		o.status = DeployStatusAlready
		return o, nil
	}

	o = GitOpsPrepareOutput{
		PullRequestID:     prID,
		PullRequestNumber: prNum,
		// PullRequestHTMLURL: prHTMLURL,
		Branch: head,
		status: DeployStatusSuccess,
	}
	return o, nil
}
