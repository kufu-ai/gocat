package main

import (
	"fmt"
	"strings"
)

type ModelCombine struct {
	modelList   *DeployModelList
	projectList *ProjectList
}

func NewModelCombine(github *GitHub, git *GitOperator, pl *ProjectList) ModelCombine {
	return ModelCombine{modelList: NewDeployModelListWithoutCombine(github, git), projectList: pl}
}

type ModelCombineOutput struct {
	status  DeployStatus
	outputs []DeployOutput
}

func (self ModelCombineOutput) Status() DeployStatus {
	if self.status == DeployStatusFail {
		return self.status
	}
	for _, d := range self.outputs {
		if d.Status() == DeployStatusFail {
			return DeployStatusFail
		}
	}
	return DeployStatusSuccess
}

func (self ModelCombineOutput) Message() string {
	messages := []string{}
	for _, d := range self.outputs {
		messages = append(messages, d.Message())
	}
	return strings.Join(messages, "\n")
}

func (self ModelCombine) Deploy(pj DeployProject, phase string, option DeployOption) (DeployOutput, error) {
	o := ModelCombineOutput{}
	ecr, err := CreateECRInstance()
	if err != nil {
		return o, err
	}
	option.Tag, err = ecr.FindImageTagByRegexp(pj.ECRRegistryId(), pj.ECRRepository(), pj.FilterRegexp(), pj.TargetRegexp(), ImageTagVars{Branch: option.Branch, Phase: phase})
	if err != nil {
		return o, err
	}

	steps := self.projectList.FindAll(pj.Steps())
	for i, step := range steps {
		p := step.FindPhase(phase)
		model, err := self.modelList.Find(p.Kind)
		if err != nil {
			o.status = DeployStatusFail
			return o, err
		}

		res, err := model.Deploy(step, phase, option)
		o.outputs = append(o.outputs, res)
		if err != nil {
			return o, fmt.Errorf("[ERROR] Failed to deploy while deploying %s(%d/%d):\n\t%s", step.ID, i+1, len(steps), err)
		}
	}
	return o, nil
}
