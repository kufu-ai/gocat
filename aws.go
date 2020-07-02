package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"log"
	"regexp"
	"strings"
)

type ECRClient struct {
	client *ecr.ECR
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

func (e ECRClient) DockerImageTag(repo string, branch string) string {
	imageTag := strings.Replace(branch, "/", "_", -1)
	arr := e.describeImages(&repo, nil)
	for _, v := range arr {
		for _, vv1 := range v.ImageTags {
			if *vv1 == imageTag {
				for _, vv2 := range v.ImageTags {
					// git hash (sha1) regex
					if regexp.MustCompile(`\b[0-9a-f]{5,40}\b`).FindStringSubmatch(*vv2) != nil {
						return *vv2
					}
				}
			}
		}
	}
	return ""
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
