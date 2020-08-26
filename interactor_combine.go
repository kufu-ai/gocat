package main

import (
	"fmt"
	"strings"

	"github.com/nlopes/slack"
)

type InteractorCombine struct {
	InteractorContext
	model ModelCombine
}

func NewInteractorCombine(i InteractorContext) (o InteractorCombine) {
	o = InteractorCombine{InteractorContext: i, model: NewModelCombine(&i.github, &i.git, i.projectList)}
	o.kind = "combine"
	return
}

func (self InteractorCombine) Request(pj DeployProject, phase string, branch string, assigner string, channel string) ([]slack.Block, error) {
	var txt *slack.TextBlockObject
	txt = slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*\n*%s*\nをデプロイしますか?", pj.ID, phase), false, false)
	btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
	btn := slack.NewButtonBlockElement("", fmt.Sprintf("%s|%s_%s_%s", self.actionHeader("approve"), pj.ID, phase, branch), btnTxt)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn))
	return []slack.Block{section, CloseButton()}, nil
}

func (self InteractorCombine) Approve(params string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	return self.approve(p[0], p[1], p[2], userID, channel)
}

func (self InteractorCombine) approve(target string, phase string, branch string, userID string, channel string) (blocks []slack.Block, err error) {
	pj := self.projectList.Find(target)
	user := self.userList.FindBySlackUserID(userID)

	go func() {
		res, err := self.model.Deploy(pj, phase, DeployOption{Branch: branch, Assigner: user})
		if err != nil || res.Status() == DeployStatusFail {
			fields := []slack.AttachmentField{
				{Title: "user", Value: "<@" + userID + ">"},
				{Title: "error", Value: err.Error()},
			}
			msg := slack.Attachment{Color: "#e01e5a", Title: fmt.Sprintf("Failed to deploy %s %s", pj.ID, phase), Fields: fields}
			self.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
			return
		}

		fields := []slack.AttachmentField{{Title: "user", Value: "<@" + userID + ">"}}
		msg := slack.Attachment{Color: "#36a64f", Title: fmt.Sprintf("Succeed to deploy %s %s", pj.ID, phase), Fields: fields}
		self.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
		return
	}()

	blocks = self.plainBlocks("Now deploying ...")
	userObject := slack.NewTextBlockObject("mrkdwn", "by <@"+userID+">", false, false)
	blocks = append(blocks, slack.NewSectionBlock(userObject, nil, nil))
	return
}

func (self InteractorCombine) Reject(params string, userID string) (blocks []slack.Block, err error) {
	return
}

func (self InteractorCombine) BranchList(pj DeployProject, phase string) ([]slack.Block, error) {
	return self.branchList(pj, phase)
}

func (self InteractorCombine) BranchListFromRaw(params string) (blocks []slack.Block, err error) {
	p := strings.Split(params, "_")
	pj := self.projectList.Find(p[0])
	return self.branchList(pj, p[1])
}

func (self InteractorCombine) SelectBranch(params string, branch string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := self.projectList.Find(p[0])
	return self.Request(pj, p[1], branch, userID, channel)
}
