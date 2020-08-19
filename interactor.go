package main

import (
	"strings"

	"github.com/nlopes/slack"
)

type DeployUsecase interface {
	Request(DeployProject, string, string, string, string) (blocks []slack.Block, err error)
	BranchList(DeployProject, string) (blocks []slack.Block, err error)
	BranchListFromRaw(string) (blocks []slack.Block, err error)
	Approve(string, string, string) (blocks []slack.Block, err error)
	Reject(string, string) (blocks []slack.Block, err error)
	SelectBranch(string, string, string, string) (blocks []slack.Block, err error)
}

type InteractorFactory struct {
	kustomize InteractorKustomize
	jenkins   InteractorJenkins
	job       InteractorJob
	lambda    InteractorLambda
}

func NewInteractorFactory(c InteractorContext) InteractorFactory {
	return InteractorFactory{kustomize: NewInteractorKustomize(c), jenkins: NewInteractorJenkins(c), job: NewInteractorJob(c), lambda: NewInteractorLambda(c)}
}

func (i InteractorFactory) Get(pj DeployProject, phase string) DeployUsecase {
	if p := pj.FindPhase(phase); p.Kind != "" {
		return i.get(p.Kind)
	}
	return i.get(pj.Kind)
}

func (i InteractorFactory) get(kind string) DeployUsecase {
	switch kind {
	case "kustomize":
		return i.kustomize
	case "job":
		return i.job
	case "lambda":
		return i.lambda
	default:
		return i.jenkins
	}
}

func (i InteractorFactory) GetByParams(params string) DeployUsecase {
	switch {
	case strings.Contains(params, "kustomize"):
		return i.kustomize
	case strings.Contains(params, "job"):
		return i.job
	case strings.Contains(params, "lambda"):
		return i.lambda
	default:
		return i.jenkins
	}
}

func CloseButton() *slack.ActionBlock {
	closeBtnTxt := slack.NewTextBlockObject("plain_text", "Close", false, false)
	closeBtn := slack.NewButtonBlockElement("", "close", closeBtnTxt)
	section := slack.NewActionBlock("", closeBtn)
	return section
}
