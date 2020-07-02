package main

import (
	"fmt"
	"github.com/nlopes/slack"
	"strings"
)

type User struct {
	SlackUserID      string
	SlackDisplayName string
	GitHubUserName   string
	GitHubNodeID     string
	isDeveloper      bool
}

func (u User) IsDeveloper() bool {
	return u.isDeveloper
}

type UserList struct {
	Items       []User
	github      GitHub
	slackClient *slack.Client
}

func (ul *UserList) Reload() {
	ul.Items = []User{}

	slackUsers, err := ul.slackClient.GetUsers()
	if err != nil {
		fmt.Println("[ERROR] Cannot load slack users")
		return
	}

	githubUsers, err := ul.github.GetUsers()
	if err != nil {
		fmt.Println("[ERROR] Cannot load GitHub users")
		return
	}

	cml := getConfigMapList("githubuser-mapping")
	rolebindings := getConfigMapList("rolebinding")

	for _, slackUser := range slackUsers {
		if slackUser.IsBot || slackUser.Deleted {
			continue
		}
		user := User{SlackUserID: slackUser.ID, SlackDisplayName: slackUser.Profile.DisplayName}
		for _, cm := range cml.Items {
			if cm.Data[user.SlackDisplayName] != "" {
				user.GitHubUserName = cm.Data[user.SlackDisplayName]
				break
			}
		}
		user.GitHubNodeID = githubUsers[user.GitHubUserName]
		for _, rolebinding := range rolebindings.Items {
			raw := rolebinding.Data["Developer"]
			userNames := strings.Split(raw, "\n")
			for _, userName := range userNames {
				if user.SlackDisplayName == userName {
					user.isDeveloper = true
					break
				}
			}
		}
		ul.Items = append(ul.Items, user)
	}
}

func (ul UserList) FindBySlackUserID(slackUserID string) User {
	for _, user := range ul.Items {
		if user.SlackUserID == slackUserID {
			return user
		}
	}
	return User{}
}
