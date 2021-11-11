package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

type CatConfig struct {
	ManifestRepository     string
	ManifestRepositoryName string
	ManifestRepositoryOrg  string
	GitHubUserName         string // optional (default: gocat)
	GitHubAccessToken      string
	GitHubDefaultBranch    string
	SlackOAuthToken        string
	SlackVerificationToken string
	JenkinsHost            string
	JenkinsBotToken        string
	JenkinsJobToken        string
	ArgoCDHost             string
	EnableAutoDeploy       bool // optional (default: false)
}

var Config CatConfig

func findRepositoryName(repo string) string {
	match := regexp.MustCompile("/([^/]+).git$").FindAllStringSubmatch(repo, -1)
	if match == nil || len(match[0]) < 2 {
		fmt.Printf("[ERROR] Manifest Repository is Invalid. Set like `https://github.com/org/repo.git`")
		return ""
	}
	return match[0][1]
}

func findRepositoryOrg(repo string) string {
	match := regexp.MustCompile("/([^/]+)/[^/]+.git$").FindAllStringSubmatch(repo, -1)
	if match == nil || len(match[0]) < 2 {
		fmt.Printf("[ERROR] Manifest Repository is Invalid. Set like `https://github.com/org/repo.git`")
		return ""
	}
	return match[0][1]
}

func InitConfig() (err error) {
	configEnvs := []string{"CONFIG_MANIFEST_REPOSITORY"}
	for _, configEnv := range configEnvs {
		if os.Getenv(configEnv) == "" {
			return fmt.Errorf("Set %s environment variable", configEnv)
		}
	}
	Config = CatConfig{}
	Config.ManifestRepository = os.Getenv("CONFIG_MANIFEST_REPOSITORY")
	Config.EnableAutoDeploy = os.Getenv("CONFIG_ENABLE_AUTO_DEPLOY") == "true"
	Config.ArgoCDHost = os.Getenv("CONFIG_ARGOCD_HOST")
	Config.JenkinsHost = os.Getenv("CONFIG_JENKINS_HOST")
	Config.GitHubUserName = os.Getenv("CONFIG_GITHUB_USER_NAME")
	Config.GitHubDefaultBranch = os.Getenv("CONFIG_GITHUB_DEFAULT_BRANCH")
	Config.ManifestRepositoryName = findRepositoryName(Config.ManifestRepository)
	Config.ManifestRepositoryOrg = findRepositoryOrg(Config.ManifestRepository)
	if Config.GitHubUserName == "" {
		Config.GitHubUserName = "gocat"
	}

	switch os.Getenv("SECRET_STORE") {
	case "aws/secrets-manager":
		log.Print("Using aws/secrets-manager as secret store. Set SECRET_STORE env if you want to use another secret store")
		if os.Getenv("SECRET_NAME") == "" {
			return fmt.Errorf("Set SECRET_NAME environment variable")
		}
		secret := getSecret(os.Getenv("SECRET_NAME"))
		Config.GitHubAccessToken = secret.GitHubBotUserToken
		Config.SlackOAuthToken = secret.OauthToken
		Config.SlackVerificationToken = secret.VerificationToken
		Config.JenkinsBotToken = secret.JenkinsBotUserToken
		Config.JenkinsJobToken = secret.JenkinsJobToken
		return

	default:
		log.Print("Using env as secret store. Set SECRET_STORE env if you want to use another secret store")
		envs := []string{"CONFIG_GITHUB_ACCESS_TOKEN", "CONFIG_SLACK_OAUTH_TOKEN", "CONFIG_SLACK_VERIFICATION_TOKEN", "CONFIG_JENKINS_BOT_TOKEN", "CONFIG_JENKINS_JOB_TOKEN"}
		for _, env := range envs {
			if os.Getenv(env) == "" {
				log.Printf("[WARNING] %s environment variable is Empty", env)
			}
		}
		Config.GitHubAccessToken = os.Getenv("CONFIG_GITHUB_ACCESS_TOKEN")
		Config.SlackOAuthToken = os.Getenv("CONFIG_SLACK_OAUTH_TOKEN")
		Config.SlackVerificationToken = os.Getenv("CONFIG_SLACK_VERIFICATION_TOKEN")
		Config.JenkinsBotToken = os.Getenv("CONFIG_JENKINS_BOT_TOKEN")
		Config.JenkinsJobToken = os.Getenv("CONFIG_JENKINS_JOB_TOKEN")
		return
	}
}
