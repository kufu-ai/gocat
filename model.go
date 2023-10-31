package main

import (
	"fmt"
)

// DeployModel, or more simply, a deploy model, is a model that can be deployed.
//
// A model can be deployed by calling the Deploy method.
// It's called by the auto deployer which we call AutoDeploy.
//
// We have several deploy models, such as Lambda, Kustomize, Combine, and Job.
// See respective model_*.go files for more details.
//
// Note that this is different from DeployUsecase a.k.a interactor.
// Although both are responsible for managing deployments, DeployUsecase is used by slackbot to interact with users,
// whereas DeployModel is used by AutoDeploy to deploy.
// See DeployUsecase for more details.
type DeployModel interface {
	Deploy(pj DeployProject, phase string, option DeployOption) (o DeployOutput, err error)
}

type DeployOption struct {
	Branch   string
	Assigner User
	Tag      string
	Wait     bool
}

type DeployStatus uint

const (
	DeployStatusSuccess DeployStatus = iota
	DeployStatusFail
	DeployStatusAlready
)

type DeployOutput interface {
	Status() DeployStatus
	Message() string
}

// DeployModelList is a list of deploy models.
//
// We currently have only two variants of DeployModelList:
// - DeployModelList
// - DeployModelListWithoutCombine
// See respective NewDeployModelList* functions for more details.
type DeployModelList map[string]DeployModel

func NewDeployModelList(github *GitHub, git *GitOperator, projectList *ProjectList) *DeployModelList {
	return &DeployModelList{
		"lambda":    NewModelLambda(),
		"kustomize": NewModelKustomize(github, git),
		"combine":   NewModelCombine(github, git, projectList),
		"job":       NewModelJob(github),
	}
}

func NewDeployModelListWithoutCombine(github *GitHub, git *GitOperator) *DeployModelList {
	return &DeployModelList{
		"lambda":    NewModelLambda(),
		"kustomize": NewModelKustomize(github, git),
		"job":       NewModelJob(github),
	}
}

func (self DeployModelList) Find(kind string) (DeployModel, error) {
	if self[kind] != nil {
		return self[kind], nil
	}
	return nil, fmt.Errorf("[ERROR] NotFound deploy kind: %s", kind)
}
