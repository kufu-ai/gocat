package main

import (
	"github.com/nlopes/slack"
	"log"
	"time"
)

type AutoDeploy struct {
	client      *slack.Client
	github      *GitHub
	git         *GitOperator
	projectList *ProjectList
	modelList   *DeployModelList
}

func NewAutoDeploy(client *slack.Client, github *GitHub, git *GitOperator, projectList *ProjectList) AutoDeploy {
	ml := NewDeployModelList(github, git, projectList)
	return AutoDeploy{client, github, git, projectList, ml}
}

func (a AutoDeploy) Watch(sec int64) {
	log.Printf("[INFO] AutoDeploy Watcher is started. Interval is %d seconds.", sec)
	for _, dp := range a.projectList.Items {
		for _, phase := range dp.Phases {
			if !phase.AutoDeploy {
				continue
			}
			go a.CheckAndDeploy(sec, dp, phase)
		}
	}
}

func (a AutoDeploy) CheckAndDeploy(sec int64, dp DeployProject, phase DeployPhase) {
	t := time.NewTicker(time.Duration(sec) * time.Second)
	for {
		select {
		case <-t.C:
			a.checkAndDeploy(dp, phase)
		}
	}
	t.Stop()
}

func (a AutoDeploy) checkAndDeploy(dp DeployProject, phase DeployPhase) {
	ecr, err := CreateECRInstance()
	if err != nil {
		log.Print(err)
		return
	}

	currentTag, err := phase.Destination.GetCurrentRevision(GetCurrentRevisionInput{github: a.github})
	if err != nil {
		log.Print(err)
		return
	}
	tag, err := ecr.FindImageTagByRegexp(dp.ECRRegistryId(), dp.ECRRepository(), dp.ImageTagRegexp(), dp.TargetRegexp(), ImageTagVars{Branch: dp.DefaultBranch()})
	if currentTag == tag || err != nil {
		log.Printf("[INFO] Auto Deploy (%s:%s) is skipped", dp.ID, phase.Name)
		return
	}

	log.Printf("[INFO] Auto Deploy (%s:%s) is started", dp.ID, phase.Name)
	model, err := a.modelList.Find(phase.Kind)
	if err != nil {
		log.Print(err)
		return
	}
	_, err = model.Deploy(dp, phase.Name, DeployOption{Branch: dp.DefaultBranch(), Wait: true})
	if err != nil {
		log.Print(err)
		return
	}
	if phase.NotifyChannel != "" {
		fields := []slack.AttachmentField{
			{Title: "Project", Value: dp.ID, Short: true},
			{Title: "Phase", Value: phase.Name, Short: true},
			{Title: "Tag", Value: tag, Short: true},
		}
		msg := slack.Attachment{Color: "#36a64f", Title: ":white_check_mark: Succeed to auto deploy", Fields: fields}
		a.client.PostMessage(phase.NotifyChannel, slack.MsgOptionAttachments(msg))
	}
}

func (a AutoDeploy) slackMessage(text string) slack.MsgOption {
	listText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	listSection := slack.NewSectionBlock(listText, nil, nil)

	return slack.MsgOptionBlocks(
		listSection,
	)
}
