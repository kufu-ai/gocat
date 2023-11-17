package main

import (
	"strings"

	"github.com/nlopes/slack"
)

// DeployUsecase, or alternatively, interactor as well call it in our cocdebase, is an interface that defines the usecases of deploy.
// It is used by slackbot to interact with users.
//
// We have several deploy usecases or interactor implementations, such as Lambda, Kustomize, Combine, and Job.
// See respective interactor_*.go files for more details.
//
// Note that this is different from DeployModel a.k.a model.
// Although both are responsible for managing deployments, DeployUsecase is generally more about how to manage deployment workflows
// including how to request and approve deployments, whereas DeployModel is more about how to actually deploy.
//
// Some DeployUsecase implementations, such as InteractorCombine, InteractorJob, and InteractorLambda,
// do actual deployments by calling DeployModel.
// However an implementation, such as InteractorKustomize, does prepare for deployments by calling DeployModel,
// but does not actually deploy.
type DeployUsecase interface {
	Request(DeployProject, string, string, string, string) (blocks []slack.Block, err error)
	BranchList(DeployProject, string) (blocks []slack.Block, err error)
	BranchListFromRaw(string) (blocks []slack.Block, err error)
	Approve(string, string, string) (blocks []slack.Block, err error)
	Reject(string, string) (blocks []slack.Block, err error)
	SelectBranch(string, string, string, string) (blocks []slack.Block, err error)
}

type InteractorFactory struct {
	kanvas    InteractorGitOps
	kustomize InteractorGitOps
	jenkins   InteractorJenkins
	job       InteractorJob
	lambda    InteractorLambda
	combine   InteractorCombine
}

func NewInteractorFactory(c InteractorContext) InteractorFactory {
	return InteractorFactory{
		kanvas:    NewInteractorKanavs(c),
		kustomize: NewInteractorKustomize(c),
		jenkins:   NewInteractorJenkins(c),
		job:       NewInteractorJob(c),
		lambda:    NewInteractorLambda(c),
		combine:   NewInteractorCombine(c),
	}
}

func (i InteractorFactory) Get(pj DeployProject, phase string) DeployUsecase {
	if p := pj.FindPhase(phase); p.Kind != "" {
		return i.get(p.Kind)
	}
	return i.get(pj.Kind)
}

func (i InteractorFactory) get(kind string) DeployUsecase {
	switch kind {
	case "kanvas":
		return i.kanvas
	case "kustomize":
		return i.kustomize
	case "job":
		return i.job
	case "lambda":
		return i.lambda
	case "combine":
		return i.combine
	default:
		return i.jenkins
	}
}

func (i InteractorFactory) GetByParams(params string) DeployUsecase {
	switch {
	case strings.Contains(params, "kanvas"):
		return i.kanvas
	case strings.Contains(params, "kustomize"):
		return i.kustomize
	case strings.Contains(params, "job"):
		return i.job
	case strings.Contains(params, "lambda"):
		return i.lambda
	case strings.Contains(params, "combine"):
		return i.combine
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
