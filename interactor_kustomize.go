package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
)

type InteractorKustomize struct {
	InteractorContext
	model ModelKustomize
}

func NewInteractorKustomize(i InteractorContext) (o InteractorKustomize) {
	o = InteractorKustomize{InteractorContext: i, model: NewModelKustomize(&o.github, &o.git)}
	o.kind = "kustomize"
	return
}

func (i InteractorKustomize) Request(pj DeployProject, phase string, branch string, assigner string, channel string) (blocks []slack.Block, err error) {
	user := i.userList.FindBySlackUserID(assigner)

	go func() {
		o, err := i.model.Prepare(pj, phase, branch, user, "")
		if err != nil {
			blocks := i.plainBlocks(err.Error())
			i.client.PostMessage(channel, slack.MsgOptionBlocks(blocks...))
			return
		}

		if o.Status() == DeployStatusAlready {
			blocks = i.plainBlocks("Already Deployed in this revision")
			i.client.PostMessage(channel, slack.MsgOptionBlocks(blocks...))
			return
		}

		txt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("<@%s>\n*%s*\n*%s*\n*%s* ブランチをデプロイしますか?\nhttps://github.com/%s/%s/pull/%d", assigner, pj.GitHubRepository(), phase, branch, i.github.org, i.github.repo, o.PullRequestNumber), false, false)
		btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
		btn := slack.NewButtonBlockElement("", fmt.Sprintf("%s|%s_%d", i.actionHeader("approve"), o.PullRequestID, o.PullRequestNumber), btnTxt)
		blocks = append(blocks, slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn)))

		closeBtnTxt := slack.NewTextBlockObject("plain_text", "Close", false, false)
		closeBtn := slack.NewButtonBlockElement("", fmt.Sprintf("%s|%s_%d_%s", i.actionHeader("reject"), o.PullRequestID, o.PullRequestNumber, o.Branch), closeBtnTxt)
		blocks = append(blocks, slack.NewActionBlock("", closeBtn))
		i.client.PostMessage(channel, slack.MsgOptionBlocks(blocks...))
	}()

	return i.plainBlocks("Now creating pull request..."), nil
}

func (i InteractorKustomize) BranchList(pj DeployProject, phase string) ([]slack.Block, error) {
	return i.branchList(pj, phase)
}

func (i InteractorKustomize) BranchListFromRaw(params string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.branchList(pj, p[1])
}

func (i InteractorKustomize) SelectBranch(params string, branch string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.Request(pj, p[1], branch, userID, channel)
}

func (i InteractorKustomize) Approve(params string, userID string, channel string) (blocks []slack.Block, err error) {
	p := strings.Split(params, "_")
	if len(p) != 2 {
		err = fmt.Errorf("Invalid Arguments")
		return
	}
	if err = i.github.MergePullRequest(p[0]); err != nil {
		return
	}

	blockObject := slack.NewTextBlockObject("mrkdwn", i.config.ArgoCDHost+"/applications", false, false)
	blocks = append(blocks, slack.NewSectionBlock(blockObject, nil, nil))

	prMsg := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("merged https://github.com/%s/%s/pull/%s\nby <@%s>", i.github.org, i.github.repo, p[1], userID), false, false)
	blocks = append(blocks, slack.NewSectionBlock(prMsg, nil, nil))

	num, err := strconv.Atoi(p[1])
	if err != nil {
		return blocks, nil
	}

	pr, err := i.github.GetPullRequest(GitHubGetPullRequestInput{Number: num})
	if err != nil {
		return blocks, nil
	}

	commitLogLimit := 5000
	prBody := pr.Body
	if len(pr.Body) >= commitLogLimit {
		tmp := strings.Split(pr.Body[:commitLogLimit], "\n")
		prBody = strings.Join(tmp[0:len(tmp)-1], "\n")
	}
	prDesc := slack.NewTextBlockObject("mrkdwn", prBody, false, false)
	blocks = append(blocks, slack.NewSectionBlock(prDesc, nil, nil))
	return
}

func (i InteractorKustomize) Reject(params string, userID string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	return i.reject(p[0], p[1], p[2], userID)
}

func (i InteractorKustomize) reject(prID string, prNum string, branch string, userID string) (blocks []slack.Block, err error) {
	if err = i.github.ClosePullRequest(prID); err != nil {
		return
	}
	if err = i.github.DeleteBranch(branch); err != nil {
		return
	}

	blockObject := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("closed https://github.com/%s/%s/pull/%s\nby <@%s>", i.github.org, i.github.repo, prNum, userID), false, false)
	blocks = append(blocks, slack.NewSectionBlock(blockObject, nil, nil))
	return
}
