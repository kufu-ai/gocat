package main

import (
	"fmt"
	"log"

	"github.com/slack-go/slack"
)

type InteractorContext struct {
	kind        string
	projectList *ProjectList
	userList    *UserList
	github      GitHub
	git         GitOperator
	client      *slack.Client
	config      CatConfig
}

func (i InteractorContext) actionHeader(nextFunc string) string {
	return fmt.Sprintf("deploy_%s_%s", i.kind, nextFunc)
}

func (i InteractorContext) branchList(pj DeployProject, phase string) ([]slack.Block, error) {
	repo := pj.GitHubRepository()
	arr, err := i.github.ListBranch(repo)
	if err != nil {
		log.Print("Failed to list branch" + err.Error())
		return []slack.Block{}, err
	}
	var opts []*slack.OptionBlockObject
	for n, v := range arr {
		txt := slack.NewTextBlockObject("plain_text", v, false, false)
		opt := slack.NewOptionBlockObject(fmt.Sprintf("%s|%s_%s_%d", i.actionHeader("selectbranch"), pj.ID, phase, n), txt, nil)
		opts = append(opts, opt)
	}
	txt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* branch list", repo), false, false)
	availableOption := slack.NewOptionsSelectBlockElement("static_select", nil, "", opts...)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(availableOption))
	fmt.Printf("[INFO] %#v", section)
	return []slack.Block{section, CloseButton()}, nil
}

func (i InteractorContext) plainBlocks(texts ...string) (blocks []slack.Block) {
	for _, text := range texts {
		block := slack.NewTextBlockObject("mrkdwn", text, false, false)
		blocks = append(blocks, slack.NewSectionBlock(block, nil, nil))
	}
	return
}
