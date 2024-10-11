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

	// This is used as the GitHub organization of the repository included in each project configmap.
	// Let's say we have a gitops config repository at a/b and a project under c/d where the project configmap
	// for c/d says the `GitHubRepository: d`.
	// WITHOUT this field, gocat looks for a/d for the app repository, which is incorrect.
	// WITH this field set to `c`, gocat looks for c/d for the app repository, which is correct.
	AppRepositoryOrg string

	// This is used for calling GitHub API for app repositories, where the only usecase is
	// to list all the branches of the repository.
	//
	// If not set, gocat uses the same GitHub access token as the manifest repository, which is
	// specified in `GitHubAccessToken`.
	// Put another way, this is specified only when AppRepositoryOrg is different from ManifestRepositoryOrg.
	AppRepositoryGitHubAccessToken string

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

	// For deploy.Coordinator
	Namespace          string
	LocksConfigMapName string
}

func (c *CatConfig) GetAppRepositoryOrg() string {
	if c.AppRepositoryOrg == "" {
		return c.ManifestRepositoryOrg
	}
	return c.AppRepositoryOrg
}

func (c *CatConfig) GetAppRepositoryGitHubAccessToken() string {
	if c.AppRepositoryGitHubAccessToken == "" {
		return c.GitHubAccessToken
	}
	return c.AppRepositoryGitHubAccessToken
}

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

func InitConfig() (*CatConfig, error) {
	configEnvs := []string{"CONFIG_MANIFEST_REPOSITORY"}
	for _, configEnv := range configEnvs {
		if os.Getenv(configEnv) == "" {
			return nil, fmt.Errorf("Set %s environment variable", configEnv)
		}
	}
	var Config = &CatConfig{}
	Config.ManifestRepository = os.Getenv("CONFIG_MANIFEST_REPOSITORY")
	Config.EnableAutoDeploy = os.Getenv("CONFIG_ENABLE_AUTO_DEPLOY") == "true"
	Config.ArgoCDHost = os.Getenv("CONFIG_ARGOCD_HOST")
	Config.JenkinsHost = os.Getenv("CONFIG_JENKINS_HOST")
	Config.GitHubUserName = os.Getenv("CONFIG_GITHUB_USER_NAME")
	Config.GitHubDefaultBranch = os.Getenv("CONFIG_GITHUB_DEFAULT_BRANCH")
	Config.ManifestRepositoryName = findRepositoryName(Config.ManifestRepository)
	Config.ManifestRepositoryOrg = findRepositoryOrg(Config.ManifestRepository)
	Config.AppRepositoryOrg = os.Getenv("CONFIG_APP_REPOSITORY_ORG")
	if Config.GitHubUserName == "" {
		Config.GitHubUserName = "gocat"
	}

	Config.Namespace = os.Getenv("CONFIG_NAMESPACE")
	if Config.Namespace == "" {
		log.Printf("[WARNING] CONFIG_NAMESPACE environment variable is not set. Lock-related features will not work.")
	}
	Config.LocksConfigMapName = os.Getenv("CONFIG_LOCKS_CONFIGMAP_NAME")
	if Config.LocksConfigMapName == "" {
		log.Printf("[WARNING] CONFIG_LOCKS_CONFIGMAP_NAME environment variable is not set. Lock-related features will not work.")
	}

	switch os.Getenv("SECRET_STORE") {
	case "aws/secrets-manager":
		log.Print("Using aws/secrets-manager as secret store. Set SECRET_STORE env if you want to use another secret store")
		if os.Getenv("SECRET_NAME") == "" {
			return nil, fmt.Errorf("Set SECRET_NAME environment variable")
		}
		secret, err := getSecret(os.Getenv("SECRET_NAME"))
		if err != nil {
			return nil, fmt.Errorf("unable to get secret: %w", err)
		}
		Config.GitHubAccessToken = secret.GitHubBotUserToken
		Config.AppRepositoryGitHubAccessToken = secret.AppRepositoryGitHubAccessToken
		Config.SlackOAuthToken = secret.OauthToken
		Config.SlackVerificationToken = secret.VerificationToken
		Config.JenkinsBotToken = secret.JenkinsBotUserToken
		Config.JenkinsJobToken = secret.JenkinsJobToken
		return Config, nil

	default:
		log.Print("Using env as secret store. Set SECRET_STORE env if you want to use another secret store")
		envs := []string{"CONFIG_GITHUB_ACCESS_TOKEN", "CONFIG_SLACK_OAUTH_TOKEN", "CONFIG_SLACK_VERIFICATION_TOKEN", "CONFIG_JENKINS_BOT_TOKEN", "CONFIG_JENKINS_JOB_TOKEN"}
		for _, env := range envs {
			if os.Getenv(env) == "" {
				log.Printf("[WARNING] %s environment variable is Empty", env)
			}
		}
		Config.GitHubAccessToken = os.Getenv("CONFIG_GITHUB_ACCESS_TOKEN")
		Config.AppRepositoryGitHubAccessToken = os.Getenv("CONFIG_APP_REPOSITORY_GITHUB_ACCESS_TOKEN")
		Config.SlackOAuthToken = os.Getenv("CONFIG_SLACK_OAUTH_TOKEN")
		Config.SlackVerificationToken = os.Getenv("CONFIG_SLACK_VERIFICATION_TOKEN")
		Config.JenkinsBotToken = os.Getenv("CONFIG_JENKINS_BOT_TOKEN")
		Config.JenkinsJobToken = os.Getenv("CONFIG_JENKINS_JOB_TOKEN")
		return Config, nil
	}
}
