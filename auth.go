package main

import (
	"encoding/json"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
)

type slackConfig struct {
	OauthToken          string `json:"SLACK_BOT_OAUTH_TOKEN"`
	VerificationToken   string `json:"SLACK_BOT_API_VERIFICATION_TOKEN"`
	JenkinsBotUserToken string `json:"JENKINS_BOT_USER_TOKEN"`
	JenkinsJobToken     string `json:"JENKINS_JOB_TOKEN"`
	GitHubBotUserToken  string `json:"GITHUB_BOT_USER_TOKEN"`
	ArgoCDHost          string `json:"ARGOCD_HOST"`
}

// getSecret fetches slackConfig from AWS Secrets Manager secret
// denoted by secretName.
// The secret must be a JSON object with the keys defined in slackConfig.
func getSecret(secretName string) (*slackConfig, error) {
	//Create a Secrets Manager client
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-northeast-1")},
	)
	if err != nil {
		log.Print("Error creating session", err)
		return nil, err
	}
	svc := secretsmanager.New(sess)
	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html

	result, err := svc.GetSecretValue(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case secretsmanager.ErrCodeDecryptionFailure:
				// Secrets Manager can't decrypt the protected secret text using the provided KMS key.
				log.Print(secretsmanager.ErrCodeDecryptionFailure, aerr.Error())

			case secretsmanager.ErrCodeInternalServiceError:
				// An error occurred on the server side.
				log.Print(secretsmanager.ErrCodeInternalServiceError, aerr.Error())

			case secretsmanager.ErrCodeInvalidParameterException:
				// You provided an invalid value for a parameter.
				log.Print(secretsmanager.ErrCodeInvalidParameterException, aerr.Error())

			case secretsmanager.ErrCodeInvalidRequestException:
				// You provided a parameter value that is not valid for the current state of the resource.
				log.Print(secretsmanager.ErrCodeInvalidRequestException, aerr.Error())

			case secretsmanager.ErrCodeResourceNotFoundException:
				// We can't find the resource that you asked for.
				log.Print(secretsmanager.ErrCodeResourceNotFoundException, aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Print(err.Error())
		}
		return nil, err
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString string
	if result.SecretString != nil {
		secretString = *result.SecretString
	}

	// Your code goes here.
	var config slackConfig
	if err := json.Unmarshal([]byte(secretString), &config); err != nil {
		log.Print("Base64 Decode Error:", err)
		return nil, err
	}

	return &config, nil
}
