package main

import (
	"fmt"
	"strings"

	"github.com/nlopes/slack"
	batchv1 "k8s.io/api/batch/v1"
)

type InteractorJob struct {
	InteractorContext
	model ModelJob
}

func NewInteractorJob(i InteractorContext) (o InteractorJob) {
	o = InteractorJob{InteractorContext: i, model: NewModelJob(&i.github)}
	o.kind = "job"
	return
}

func (i InteractorJob) Request(pj DeployProject, phase string, branch string, assigner string, channel string) (blocks []slack.Block, err error) {
	var txt *slack.TextBlockObject
	p := pj.FindPhase(phase)
	txt = slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*\n*%s*\n*%s* ブランチ\nをデプロイしますか?", p.Path, phase, branch), false, false)
	btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
	btn := slack.NewButtonBlockElement("", fmt.Sprintf("%s|%s_%s_%s", i.actionHeader("approve"), pj.ID, phase, branch), btnTxt)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn))
	return []slack.Block{section, CloseButton()}, nil
}

func (i InteractorJob) BranchList(pj DeployProject, phase string) ([]slack.Block, error) {
	return i.branchList(pj, phase)
}

func (i InteractorJob) BranchListFromRaw(params string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.branchList(pj, p[1])
}

func (i InteractorJob) SelectBranch(params string, branch string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.Request(pj, p[1], branch, userID, channel)
}

func (i InteractorJob) Approve(params string, userID string, channel string) (blocks []slack.Block, err error) {
	p := strings.Split(params, "_")
	return i.approve(p[0], p[1], p[2], userID, channel)
}

func (i InteractorJob) approve(target string, phase string, branch string, userID string, channel string) (blocks []slack.Block, err error) {
	pj := i.projectList.Find(target)

	res, err := i.model.Deploy(pj, phase, DeployOption{Branch: branch})
	if err != nil {
		fields := []slack.AttachmentField{
			{Title: "user", Value: "<@" + userID + ">"},
			{Title: "error", Value: err.Error()},
		}
		msg := slack.Attachment{Color: "#e01e5a", Title: fmt.Sprintf("Failed to deploy %s %s", pj.ID, phase), Fields: fields}
		i.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
		return
	}

	switch do := res.(type) {
	case ModelJobDeployOutput:
		go i.model.Watch(do.Name, do.Namespace, channel, func(job batchv1.Job) {
			fields := []slack.AttachmentField{{Title: "user", Value: "<@" + userID + ">"}}
			if job.Status.Succeeded >= 1 {
				msg := slack.Attachment{Color: "#36a64f", Title: fmt.Sprintf("Succeed %s Job execution", job.Name), Fields: fields}
				i.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
				return
			}
			if job.Status.Failed >= 1 {
				msg := slack.Attachment{Color: "#e01e5a", Title: fmt.Sprintf("Failed %s execution", job.Name), Fields: fields}
				i.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
				return
			}
		})
	}

	blocks = i.plainBlocks(
		res.Message(),
		"by <@"+userID+">",
	)
	return
}

func (i InteractorJob) Reject(params string, userID string) (blocks []slack.Block, err error) {
	return
}
