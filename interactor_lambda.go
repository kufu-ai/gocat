package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/slack-go/slack"
)

type InteractorLambda struct {
	InteractorContext
	model ModelLambda
}

func NewInteractorLambda(i InteractorContext) (o InteractorLambda) {
	o = InteractorLambda{InteractorContext: i, model: NewModelLambda()}
	o.kind = "lambda"
	return
}

func (self InteractorLambda) Request(pj DeployProject, phase string, branch string, assigner string, channel string) ([]slack.Block, error) {
	txt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*\n*%s*\n*%s* ブランチをデプロイしますか?", pj.ID, phase, branch), false, false)
	btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
	btn := slack.NewButtonBlockElement("", fmt.Sprintf("%s|%s_%s_%s", self.actionHeader("approve"), pj.ID, phase, branch), btnTxt)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn))
	return []slack.Block{section, CloseButton()}, nil
}

func (self InteractorLambda) Approve(params string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	return self.approve(p[0], p[1], p[2], userID, channel)
}

func (self InteractorLambda) approve(target string, phase string, branch string, userID string, channel string) (blocks []slack.Block, err error) {
	pj := self.projectList.Find(target)

	go func() {
		res, err := self.model.Deploy(pj, phase, DeployOption{Branch: branch})
		if err != nil || res.Status() == DeployStatusFail {
			fields := []slack.AttachmentField{
				{Title: "user", Value: "<@" + userID + ">"},
				{Title: "error", Value: err.Error()},
			}
			msg := slack.Attachment{Color: "#e01e5a", Title: fmt.Sprintf("Failed to deploy %s %s", pj.ID, phase), Fields: fields}
			if _, _, err := self.client.PostMessage(channel, slack.MsgOptionAttachments(msg)); err != nil {
				log.Printf("Failed to post message: %s", err.Error())
			}
			return
		}

		msg := slack.Attachment{Color: "#36a64f", Title: fmt.Sprintf("Succeed to deploy %s %s", pj.ID, phase)}
		msg.Fields = []slack.AttachmentField{
			{Title: "user", Value: "<@" + userID + ">"},
			{Title: "phase", Value: phase},
			{Title: "branch", Value: branch},
		}
		if res.Message() != "" {
			msg.Fields = append(msg.Fields, slack.AttachmentField{Title: "response", Value: res.Message()})
		}
		if _, _, err := self.client.PostMessage(channel, slack.MsgOptionAttachments(msg)); err != nil {
			log.Printf("Failed to post message: %s", err.Error())
		}
	}()

	blocks = self.plainBlocks("Now deploying ...")
	userObject := slack.NewTextBlockObject("mrkdwn", "by <@"+userID+">", false, false)
	blocks = append(blocks, slack.NewSectionBlock(userObject, nil, nil))
	return
}

func (self InteractorLambda) Reject(params string, userID string) (blocks []slack.Block, err error) {
	return
}

func (self InteractorLambda) BranchList(pj DeployProject, phase string) ([]slack.Block, error) {
	return self.branchList(pj, phase)
}

func (self InteractorLambda) BranchListFromRaw(params string) (blocks []slack.Block, err error) {
	p := strings.Split(params, "_")
	pj := self.projectList.Find(p[0])
	return self.branchList(pj, p[1])
}

func (self InteractorLambda) SelectBranch(params string, branch string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := self.projectList.Find(p[0])
	return self.Request(pj, p[1], branch, userID, channel)
}
