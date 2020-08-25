package main

import (
	"fmt"
	"strings"

	"github.com/nlopes/slack"
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
	var txt *slack.TextBlockObject
	txt = slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*\n*%s*\nをデプロイしますか?", pj.ID, phase), false, false)
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
			self.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
			return
		}

		msg := slack.Attachment{Color: "#36a64f", Title: fmt.Sprintf("Succeed to deploy %s %s", pj.ID, phase)}
		msg.Fields = []slack.AttachmentField{{Title: "user", Value: "<@" + userID + ">"}}
		if res.Message() != "" {
			msg.Fields = append(msg.Fields, slack.AttachmentField{Title: "response", Value: res.Message()})
		}
		self.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
		return
	}()

	blocks = self.plainBlocks("Now deploying ...")
	userObject := slack.NewTextBlockObject("mrkdwn", "by <@"+userID+">", false, false)
	blocks = append(blocks, slack.NewSectionBlock(userObject, nil, nil))
	return
}

func (self InteractorLambda) Reject(params string, userID string) (blocks []slack.Block, err error) {
	return
}

func (self InteractorLambda) BranchList(DeployProject, string) (blocks []slack.Block, err error) {
	return self.plainBlocks("ブランチデプロイには対応していません"), nil
}

func (self InteractorLambda) BranchListFromRaw(string) (blocks []slack.Block, err error) {
	return self.plainBlocks("ブランチデプロイには対応していません"), nil
}

func (self InteractorLambda) SelectBranch(string, string, string, string) (blocks []slack.Block, err error) {
	return self.plainBlocks("ブランチデプロイには対応していません"), nil
}
