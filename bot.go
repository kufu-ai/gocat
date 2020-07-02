package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/nlopes/slack"
)

func init() {
	err := InitConfig()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	client := slack.New(
		Config.SlackOAuthToken,
		slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags)),
	)
	github := CreateGitHubInstance(Config.GitHubAccessToken, Config.ManifestRepositoryOrg, Config.ManifestRepositoryName)
	git := CreateGitDocAWSOperatorInstance(Config.GitHubAccessToken, Config.ManifestRepository)
	userList := UserList{github: github, slackClient: client}
	projectList := NewProjectList()
	interactorContext := InteractorContext{projectList: &projectList, userList: &userList, github: github, git: git, client: client, config: Config}
	interactorFactory := NewInteractorFactory(interactorContext)
	autoDeploy := AutoDeploy{client, github, git, &projectList}

	log.SetOutput(os.Stdout)
	listener := SlackListener{client: client, botID: os.Getenv("BOT_ID"), projectList: &projectList, userList: &userList, interactorFactory: &interactorFactory}
	if Config.EnableAutoDeploy {
		go autoDeploy.Watch(60)
	}
	go listener.ListenAndResponse()

	http.Handle("/interaction", interactionHandler{
		verificationToken: Config.SlackVerificationToken,
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
