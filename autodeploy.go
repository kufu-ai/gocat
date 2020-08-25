package main

import (
	"fmt"
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
	t := time.NewTicker(time.Duration(sec) * time.Second)
	for {
		select {
		case <-t.C:
			a.CheckAndDeploy()
		}
	}
	t.Stop()
}

func (a AutoDeploy) CheckAndDeploy() {
	ecr, err := CreateECRInstance()
	if err != nil {
		log.Print(err)
		return
	}
	for _, dp := range a.projectList.Items {
		for _, phase := range dp.Phases {
			if !phase.AutoDeploy {
				continue
			}
			currentTag, err := phase.Destination.GetCurrentRevision(GetCurrentRevisionInput{github: a.github})
			if err != nil {
				log.Print(err)
				continue
			}
			tag, err := ecr.FindImageTagByRegexp(dp.ECRRepository(), dp.FilterRegexp(), dp.TargetRegexp(), ImageTagVars{Branch: "master"})
			if currentTag == tag || err != nil {
				log.Printf("[INFO] Auto Deploy (%s:%s) is skipped", dp.ID, phase.Name)
				continue
			}
			log.Print("[INFO] Auto Deploy is started")
			model, err := a.modelList.Find(phase.Kind)
			if err != nil {
				log.Print(err)
				continue
			}
			go func() {
				_, err = model.Deploy(dp, phase.Name, DeployOption{Branch: "master"})
				if err != nil {
					log.Print(err)
					return
				}
				if phase.NotifyChannel != "" {
					a.client.PostMessage(phase.NotifyChannel, a.slackMessage(fmt.Sprintf("%s:%s %s is Deployed", dp.ID, tag, phase.Name)))
				}
			}()
		}
	}
}

func (a AutoDeploy) slackMessage(text string) slack.MsgOption {
	listText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	listSection := slack.NewSectionBlock(listText, nil, nil)

	return slack.MsgOptionBlocks(
		listSection,
	)
}
