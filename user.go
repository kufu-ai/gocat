package main

import (
	"fmt"
	"strings"

	"github.com/slack-go/slack"
	v1 "k8s.io/api/core/v1"
)

// Role is a type to represent a role of a user.
//
// A user can have multiple roles.
// But we currently assume any Admin users are also Developer users.
// So, a consumer of this type should be able to check if a user has an equal or higher role than
// Developer by checking if the user has the Developer role.
type Role string

const (
	RoleDeveloper Role = "Developer"
	RoleAdmin     Role = "Admin"
)

type User struct {
	SlackUserID      string
	SlackDisplayName string
	GitHubUserName   string
	GitHubNodeID     string
	isDeveloper      bool
	isAdmin          bool
}

func (u User) IsDeveloper() bool {
	return u.isDeveloper
}

// IsAdmin returns a flag to indicate whether the user is an admin or not.
// An admin can force-unlock a project/phase using `@bot unlock` command,
// even if the project/phase is locked by other users.
func (u User) IsAdmin() bool {
	return u.isAdmin
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
	userNamesInGroups := ul.createUserNamesInGroups(rolebindings)

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
		_, user.isDeveloper = userNamesInGroups[RoleDeveloper][user.SlackDisplayName]
		_, user.isAdmin = userNamesInGroups[RoleAdmin][user.SlackDisplayName]
		ul.Items = append(ul.Items, user)
	}
}

func (ul UserList) createUserNamesInGroups(rolebindings *v1.ConfigMapList) map[Role]map[string]struct{} {
	userNamesInGroups := map[Role]map[string]struct{}{
		RoleDeveloper: {},
		RoleAdmin:     {},
	}
	for _, rolebinding := range rolebindings.Items {
		for group, users := range userNamesInGroups {
			raw := rolebinding.Data[string(group)]
			userNames := strings.Split(raw, "\n")
			for _, userName := range userNames {
				if userName != "" {
					users[userName] = struct{}{}
				}
			}
		}
	}
	return userNamesInGroups
}

func (ul UserList) FindBySlackUserID(slackUserID string) User {
	for _, user := range ul.Items {
		if user.SlackUserID == slackUserID {
			return user
		}
	}
	return User{}
}
