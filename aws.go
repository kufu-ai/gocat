package main

import (
	"bytes"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/lambda"
)

type ECRClient struct {
	client *ecr.ECR
}

type ImageTagVars struct {
	Branch string
}

func (self ImageTagVars) Parse(s string) (string, error) {
	b := bytes.NewBuffer([]byte(""))
	tmpl, err := template.New("").Parse(s)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(b, self)
	return b.String(), err
}

func CreateECRInstance() (ECRClient, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1")},
	)
	if err != nil {
		return ECRClient{}, err
	}
	return ECRClient{client: ecr.New(sess, aws.NewConfig().WithRegion("ap-northeast-1"))}, nil
}

func (e ECRClient) FindImageTagByRegexp(repo string, rawFilterRegexp string, rawTargetRegexp string, vars ImageTagVars) (string, error) {
	vars.Branch = strings.Replace(vars.Branch, "/", "_", -1)
	filterRegexp, err := vars.Parse(rawFilterRegexp)
	if err != nil {
		return "", fmt.Errorf("[ERROR] filterRegexp cannot be parsed: %s", rawFilterRegexp)
	}
	targetRegexp, err := vars.Parse(rawTargetRegexp)
	if err != nil {
		return "", fmt.Errorf("[ERROR] targetRegexp cannot be parsed: %s", rawTargetRegexp)
	}
	arr := e.describeImages(&repo, nil)
	for _, v := range arr {
		for _, vv1 := range v.ImageTags {
			if regexp.MustCompile(filterRegexp).FindStringSubmatch(*vv1) == nil {
				continue
			}
			for _, vv2 := range v.ImageTags {
				if regexp.MustCompile(targetRegexp).FindStringSubmatch(*vv2) != nil {
					return *vv2, nil
				}
			}
		}
	}
	return "", fmt.Errorf("[ERROR] NotFound specified image tag")
}

func (e ECRClient) describeImages(repo *string, nextToken *string) []*ecr.ImageDetail {
	input := &ecr.DescribeImagesInput{
		RepositoryName: repo,
		NextToken:      nextToken,
	}
	outputs, err := e.client.DescribeImages(input)
	if err != nil {
		log.Print("Failed describe images, %v", err)
		return []*ecr.ImageDetail{}
	}
	if outputs.NextToken != nil {
		return append(outputs.ImageDetails, e.describeImages(repo, outputs.NextToken)...)
	}
	return outputs.ImageDetails
}

type LambdaClient struct {
	client *lambda.Lambda
}

func CreateLambdaInstance() (LambdaClient, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1")},
	)
	if err != nil {
		return LambdaClient{}, err
	}
	return LambdaClient{client: lambda.New(sess)}, nil
}

func (self LambdaClient) Invoke(funcName string, payload string) (*lambda.InvokeOutput, error) {
	input := &lambda.InvokeInput{
		FunctionName:   &funcName,
		Payload:        []byte(payload),
		InvocationType: aws.String("RequestResponse"),
	}
	res, err := self.client.Invoke(input)
	return res, err
}

type ECSClient struct {
	client *ecs.ECS
}

func CreateECSInstance() (ECSClient, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1")},
	)
	if err != nil {
		return ECSClient{}, err
	}
	return ECSClient{client: ecs.New(sess)}, nil
}

func (self ECSClient) DescribeTaskDefinition(arn string) (*ecs.TaskDefinition, error) {
	o, err := self.client.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{TaskDefinition: &arn})
	return o.TaskDefinition, err
}
