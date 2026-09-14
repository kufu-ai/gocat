package main

import "testing"

func TestInteractorContextAssigner(t *testing.T) {
	i := InteractorContext{userList: &UserList{Items: []User{
		{SlackUserID: "U123", SlackDisplayName: "user1"},
	}}}

	if got := i.assigner("U123"); got.SlackDisplayName != "user1" {
		t.Errorf("registered user: got %+v", got)
	}
	if got := i.assigner("U999"); got.SlackUserID != "U999" || got.SlackDisplayName != "" {
		t.Errorf("unknown user should keep the Slack user ID: got %+v", got)
	}
}
