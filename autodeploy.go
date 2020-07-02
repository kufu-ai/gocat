package main

import (
	"fmt"
	"github.com/nlopes/slack"
	"log"
	"strings"
	"time"
)

type AutoDeploy struct {
	client      *slack.Client
	github      GitHub
	git         GitDocAWSOperator
	projectList *ProjectList
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
			k, err := a.github.GetKustomization(phase.Path)
			if err != nil {
				log.Print(err)
				continue
			}
			for _, image := range k.Images {
				if !strings.HasPrefix(image.Name, dp.DockerRepository()) {
					continue
				}
				tag := ecr.DockerImageTag(dp.ECRRepository(), "master")
				if image.NewTag == tag || tag == "" {
					log.Printf("[INFO] Auto Deploy (%s:%s) is skipped", dp.ID, phase.Name)
					continue
				}
				log.Print("[INFO] Auto Deploy is started")
				a.Deploy(dp, phase, tag)
				break
			}
		}
	}
}

func (a AutoDeploy) Deploy(dp DeployProject, phase DeployPhase, tag string) {
	prBranch, err := a.git.PushDockerImageTag(dp.ID, dp.K8SMetadata(), phase.Name, tag, dp.DockerRepository())
	if err != nil {
		return
	}

	prID, _, err := a.github.CreatePullRequest(prBranch, fmt.Sprintf("Deploy %s %s", dp.ID, "master"), "")
	if err != nil {
		return
	}

	if err := a.github.MergePullRequest(prID); err != nil {
		return
	}
	if phase.NotifyChannel != "" {
		a.client.PostMessage(phase.NotifyChannel, a.slackMessage(fmt.Sprintf("%s:%s %s is Deployed", dp.ID, tag, phase.Name)))
	}
}

func (a AutoDeploy) slackMessage(text string) slack.MsgOption {
	listText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	listSection := slack.NewSectionBlock(listText, nil, nil)

	return slack.MsgOptionBlocks(
		listSection,
	)
}
