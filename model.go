package main

import (
	"fmt"
)

type DeployModel interface {
	Deploy(pj DeployProject, phase string, option DeployOption) (o DeployOutput, err error)
}

type DeployOption struct {
	Branch   string
	Assigner User
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

type DeployModelList map[string]DeployModel

func NewDeployModelList(github *GitHub, git *GitOperator, projectList *ProjectList) *DeployModelList {
	return &DeployModelList{
		"lambda":    NewModelLambda(),
		"kustomize": NewModelKustomize(github, git),
		"combine":   NewModelCombine(github, git, projectList),
	}
}

func NewDeployModelListWithoutCombine(github *GitHub, git *GitOperator) *DeployModelList {
	return &DeployModelList{
		"lambda":    NewModelLambda(),
		"kustomize": NewModelKustomize(github, git),
	}
}

func (self DeployModelList) Find(kind string) (DeployModel, error) {
	if self[kind] != nil {
		return self[kind], nil
	}
	return nil, fmt.Errorf("[ERROR] NotFound deploy kind: %s", kind)
}
