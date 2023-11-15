package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/lambda"
)

type ModelLambda struct{}

func NewModelLambda() ModelLambda {
	return ModelLambda{}
}

type ModelLambdaDeployOutput lambda.InvokeOutput

func (self ModelLambdaDeployOutput) Status() DeployStatus {
	if *self.StatusCode != 200 {
		return DeployStatusFail
	}
	return DeployStatusSuccess
}

func (self ModelLambdaDeployOutput) Message() string {
	return string(self.Payload)
}

func (self ModelLambda) Deploy(pj DeployProject, phase string, option DeployOption) (o DeployOutput, err error) {
	lambda, err := CreateLambdaInstance()
	if err != nil {
		return
	}
	tag := option.Tag
	if tag == "" {
		ecr, err := CreateECRInstance()
		if err != nil {
			return o, err
		}
		tag, err = ecr.FindImageTagByRegexp(pj.ECRRegistryId(), pj.ECRRepository(), pj.ImageTagRegexp(), pj.TargetRegexp(), ImageTagVars{Branch: option.Branch, Phase: phase})
		if err != nil {
			return o, err
		}
	}
	ph := pj.FindPhase(phase)
	payload, err := PayloadVars{Tag: tag}.Parse(ph.Payload)
	if err != nil {
		return
	}

	res, err := lambda.Invoke(pj.FuncName(), payload)
	if res.FunctionError != nil {
		return o, fmt.Errorf(string(res.Payload))
	}

	return ModelLambdaDeployOutput(*res), err
}
