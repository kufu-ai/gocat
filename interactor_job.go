package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"encoding/json"
	"github.com/nlopes/slack"
	batchv1 "k8s.io/api/batch/v1"
	yaml "k8s.io/apimachinery/pkg/util/yaml"
)

type InteractorJob struct {
	InteractorContext
}

func NewInteractorJob(i InteractorContext) (o InteractorJob) {
	o = InteractorJob{i}
	o.kind = "job"
	return
}

func (i InteractorJob) Request(pj DeployProject, phase string, branch string, assigner string) (blocks []slack.Block, err error) {
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

func (i InteractorJob) SelectBranch(params string, branch string, userID string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.Request(pj, p[1], branch, userID)
}

func (i InteractorJob) Approve(params string, userID string, channel string) (blocks []slack.Block, err error) {
	p := strings.Split(params, "_")
	return i.approve(p[0], p[1], p[2], userID, channel)
}

func (i InteractorJob) approve(target string, phase string, branch string, userID string, channel string) (blocks []slack.Block, err error) {
	pj := i.projectList.Find(target)
	p := pj.FindPhase(phase)
	rawFile, err := i.github.GetFile(p.Path)
	if err != nil {
		blockObject := slack.NewTextBlockObject("mrkdwn", err.Error(), false, false)
		blocks = []slack.Block{slack.NewSectionBlock(blockObject, nil, nil)}
		return
	}

	ecr, err := CreateECRInstance()
	if err != nil {
		blocks = i.plainBlock(err.Error())
		return
	}
	tag := ecr.DockerImageTag(pj.ECRRepository(), branch)

	job := batchv1.Job{}
	j, err := yaml.ToJSON(rawFile)
	err = json.Unmarshal(j, &job)
	if err != nil {
		blocks = i.plainBlock(err.Error())
		return
	}

	if job.Namespace == "" {
		job.Namespace = "default"
	}
	job.Name = job.Name + "-" + RandString(10)
	for i, container := range job.Spec.Template.Spec.Containers {
		if container.Image == pj.DockerRepository() {
			job.Spec.Template.Spec.Containers[i].Image = container.Image + ":" + tag
		}
	}

	if err = createJob(&job); err != nil {
		blocks = i.plainBlock(err.Error())
		return
	}

	go i.watchAndNotify(job, channel)

	blocks = i.plainBlocks(
		fmt.Sprintf("*%s Job Started*", pj.ID),
		fmt.Sprintf("*Namespace*: %s", job.Namespace),
		fmt.Sprintf("*Name*: %s", job.Name),
		fmt.Sprintf("*Path*: %s", p.Path),
		fmt.Sprintf("*ImageTag*: %s", tag),
		"by <@"+userID+">",
	)
	return
}

func (i InteractorJob) Reject(params string, userID string) (blocks []slack.Block, err error) {
	return
}

func (i InteractorJob) watchAndNotify(tpl batchv1.Job, channel string) {
	t := time.NewTicker(time.Duration(20) * time.Second)
	for {
		select {
		case <-t.C:
			job, err := getJob(tpl.Name, tpl.Namespace)
			if err != nil {
				fmt.Println("[ERROR] Quit watching job ", job.Name)
				t.Stop()
				return
			}
			fmt.Println("[INFO] Watch job", job.Name)
			if job.Status.Succeeded >= 1 {
				msg := slack.Attachment{Color: "#36a64f", Text: fmt.Sprintf("Succeed %s Job execution", tpl.Name)}
				i.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
				t.Stop()
				return
			}
			if job.Status.Failed >= 1 {
				msg := slack.Attachment{Color: "#e01e5a", Text: fmt.Sprintf("Failed %s execution", tpl.Name)}
				i.client.PostMessage(channel, slack.MsgOptionAttachments(msg))
				t.Stop()
				return
			}
		}
	}
	t.Stop()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
