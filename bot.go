package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/nlopes/slack"
)

func main() {
	config, err := InitConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := slack.New(
		config.SlackOAuthToken,
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)
	github := CreateGitHubInstance(config.GitHubAccessToken, config.ManifestRepositoryOrg, config.ManifestRepositoryName, config.GitHubDefaultBranch)
	git := CreateGitOperatorInstance(config.GitHubUserName, config.GitHubAccessToken, config.ManifestRepository, config.GitHubDefaultBranch)
	userList := UserList{github: github, slackClient: client}
	projectList := NewProjectList()
	interactorContext := InteractorContext{projectList: &projectList, userList: &userList, github: github, git: git, client: client, config: *config}
	interactorFactory := NewInteractorFactory(interactorContext)
	autoDeploy := NewAutoDeploy(client, &github, &git, &projectList)

	log.SetOutput(os.Stdout)
	if config.EnableAutoDeploy {
		autoDeploy.Watch(60)
	}

	http.Handle("/events", SlackListener{
		client:            client,
		verificationToken: config.SlackVerificationToken,
		projectList:       &projectList,
		userList:          &userList,
		interactorFactory: &interactorFactory,
	})
	http.Handle("/interaction", interactionHandler{
		verificationToken: config.SlackVerificationToken,
		client:            client,
		projectList:       &projectList,
		userList:          &userList,
		interactorFactory: &interactorFactory,
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "hello")
	})
	http.ListenAndServe(":3000", nil)
}
