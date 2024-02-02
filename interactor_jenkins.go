package main

import (
	"fmt"
	"strings"

	"net/http"

	"github.com/slack-go/slack"
)

type InteractorJenkins struct {
	InteractorContext
}

func NewInteractorJenkins(i InteractorContext) (o InteractorJenkins) {
	o = InteractorJenkins{i}
	o.kind = "jenkins"
	return
}

func (i InteractorJenkins) Request(pj DeployProject, phase string, branch string, assigner string, channel string) ([]slack.Block, error) {
	var txt *slack.TextBlockObject
	if phase == "production" && branch != pj.DefaultBranch() {
		txt = slack.NewTextBlockObject(
			"mrkdwn",
			fmt.Sprintf(
				":star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star:\n本番環境に master ブランチ以外をデプロイしようとしています\n:star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star::star:\n*%s*\n*%s*\n*%s* ブランチ\nをデプロイしますか?",
				pj.GitHubRepository(),
				phase,
				branch,
			),
			false,
			false,
		)
	} else {
		txt = slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*\n*%s*\n*%s* ブランチ\nをデプロイしますか?", pj.GitHubRepository(), phase, branch), false, false)
	}
	btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
	btn := slack.NewButtonBlockElement("", fmt.Sprintf("%s|%s_%s_%s", i.actionHeader("approve"), pj.ID, phase, branch), btnTxt)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn))
	return []slack.Block{section, CloseButton()}, nil
}

func (i InteractorJenkins) BranchList(pj DeployProject, phase string) ([]slack.Block, error) {
	return i.branchList(pj, phase)
}

func (i InteractorJenkins) BranchListFromRaw(params string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.branchList(pj, p[1])
}

func (i InteractorJenkins) SelectBranch(params string, branch string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.Request(pj, p[1], branch, userID, channel)
}

func (i InteractorJenkins) Approve(params string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	return i.approve(p[0], p[1], p[2], userID)
}

func (i InteractorJenkins) approve(target string, phase string, branch string, userID string) (blocks []slack.Block, err error) {
	pj := i.projectList.Find(target)
	jobName := pj.JenkinsJob()
	url := fmt.Sprintf("https://bot:%s@%s/job/%s/buildWithParameters?token=%s&cause=slack-bot&ENV=%s&BRANCH=%s", i.config.JenkinsBotToken, i.config.JenkinsHost, jobName, i.config.JenkinsJobToken, phase, branch)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	res := fmt.Sprintf("Execute https://%s/job/%s/ \n selected branch: %s", i.config.JenkinsHost, jobName, branch)
	if err != nil {
		res = err.Error()
	}
	if resp.StatusCode != 201 {
		res = jobName + " Request failed. responsed " + fmt.Sprint(resp.StatusCode)
	}

	blockObject := slack.NewTextBlockObject("mrkdwn", res, false, false)
	blocks = append(blocks, slack.NewSectionBlock(blockObject, nil, nil))

	userObject := slack.NewTextBlockObject("mrkdwn", "by <@"+userID+">", false, false)
	blocks = append(blocks, slack.NewSectionBlock(userObject, nil, nil))
	return
}

func (i InteractorJenkins) Reject(params string, userID string) (blocks []slack.Block, err error) {
	return
}
