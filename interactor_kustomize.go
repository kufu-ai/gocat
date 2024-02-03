package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/slack-go/slack"
)

type InteractorGitOps struct {
	InteractorContext
	model GitOpsPlugin
}

func NewInteractorKustomize(i InteractorContext) (o InteractorGitOps) {
	o = InteractorGitOps{
		InteractorContext: i,
		model:             NewGitOpsPluginKustomize(&o.github, &o.git),
	}
	o.kind = "kustomize"
	return
}

func (i InteractorGitOps) Request(pj DeployProject, phase string, branch string, assigner string, channel string) (blocks []slack.Block, err error) {
	user := i.userList.FindBySlackUserID(assigner)

	go func() {
		defer func() {
			log.Printf("[INFO] Exiting the goroutine for Prepare")
		}()

		log.Printf("[INFO] Preparing to deploy %s %s %s", pj.ID, phase, branch)

		o, err := i.model.Prepare(pj, phase, branch, user, "")
		if err != nil {
			log.Printf("[ERROR] %s", err.Error())

			blocks := i.plainBlocks(err.Error())
			if _, _, err := i.client.PostMessage(channel, slack.MsgOptionBlocks(blocks...)); err != nil {
				log.Printf("Failed to post message: %s", err)
			}
			return
		}

		if o.Status() == DeployStatusAlready {
			log.Printf("[INFO] Already Deployed in this revision: %s %s %s", pj.ID, phase, branch)

			blocks = i.plainBlocks("Already Deployed in this revision")
			if _, _, err := i.client.PostMessage(channel, slack.MsgOptionBlocks(blocks...)); err != nil {
				log.Printf("Failed to post message: %s", err)
			}
			return
		}

		log.Printf("[INFO] Prepared to deploy %s %s %s", pj.ID, phase, branch)

		prHTMLURL := o.PullRequestHTMLURL
		if prHTMLURL == "" {
			prHTMLURL = fmt.Sprintf("https://github.com/%s/%s/pull/%d", i.github.org, i.github.repo, o.PullRequestNumber)
		}

		txt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("<@%s>\n*%s*\n*%s*\n*%s* ブランチをデプロイしますか?\n%s", assigner, pj.GitHubRepository(), phase, branch, prHTMLURL), false, false)
		btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
		btn := slack.NewButtonBlockElement("", fmt.Sprintf("%s|%s_%d", i.actionHeader("approve"), o.PullRequestID, o.PullRequestNumber), btnTxt)
		blocks = append(blocks, slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn)))

		closeBtnTxt := slack.NewTextBlockObject("plain_text", "Close", false, false)
		closeBtn := slack.NewButtonBlockElement("", fmt.Sprintf("%s|%s_%d_%s", i.actionHeader("reject"), o.PullRequestID, o.PullRequestNumber, o.Branch), closeBtnTxt)
		blocks = append(blocks, slack.NewActionBlock("", closeBtn))
		if _, _, err := i.client.PostMessage(channel, slack.MsgOptionBlocks(blocks...)); err != nil {
			log.Printf("Failed to post message: %s", err)
		}
	}()

	return i.plainBlocks("Now creating pull request..."), nil
}

func (i InteractorGitOps) BranchList(pj DeployProject, phase string) ([]slack.Block, error) {
	return i.branchList(pj, phase)
}

func (i InteractorGitOps) BranchListFromRaw(params string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.branchList(pj, p[1])
}

func (i InteractorGitOps) SelectBranch(params string, branch string, userID string, channel string) ([]slack.Block, error) {
	p := strings.Split(params, "_")
	pj := i.projectList.Find(p[0])
	return i.Request(pj, p[1], branch, userID, channel)
}

func (i InteractorGitOps) Approve(params string, userID string, channel string) (blocks []slack.Block, err error) {
	prID := ""
	prNumber := ""
	p := strings.Split(params, "_")
	if len(p) == 2 {
		prID = p[0]
		prNumber = p[1]
	} else if strings.HasPrefix(params, "PR") {
		// The format of IDs for GitHub PullReques has changed
		// e.g. PR_hogehoge_fugafuga
		prID = strings.Join(p[:len(p)-1], "_")
		prNumber = p[len(p)-1]
	} else {
		err = fmt.Errorf("Invalid Arguments")
		return
	}
	if err = i.github.MergePullRequest(prID); err != nil {
		return
	}

	blockObject := slack.NewTextBlockObject("mrkdwn", i.config.ArgoCDHost+"/applications", false, false)
	blocks = append(blocks, slack.NewSectionBlock(blockObject, nil, nil))

	prMsg := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("merged https://github.com/%s/%s/pull/%s\nby <@%s>", i.github.org, i.github.repo, prNumber, userID), false, false)
	blocks = append(blocks, slack.NewSectionBlock(prMsg, nil, nil))

	num, err := strconv.Atoi(prNumber)
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

func (i InteractorGitOps) Reject(params string, userID string) ([]slack.Block, error) {
	p := strings.Split(params, "_")

	if strings.HasPrefix(params, "PR") {
		// The format of IDs for GitHub PullReques has changed
		// e.g. PR_hogehoge as a whole needs to be passed as the ID,
		// instead of "PR" or "hogehoge" only.
		//
		// Let's say you have a pull request with the ID "PR_hogehoge", you'll see the ActionValue like:
		//
		// 	deploy_kustomize_reject|PR_hogehoge_2_bot/docker-image-tag-project-foo-staging-14e308b
		//
		// If we used the "PR" as the PR ID, we'll see the following error:
		//
		// 	Could not resolve to a node with the global id of 'PR'
		// 	[ERROR] Internal Server Error
		//
		// Similarly, if you used "hogehoge" as the PR ID, you'll see the following error:
		//
		// 	Could not resolve to a node with the global id of 'hogehoge'
		// 	[ERROR] Internal Server Error
		//
		// See https://docs.github.com/en/graphql/guides/migrating-graphql-global-node-ids#determining-if-you-need-to-take-action
		var a []string
		a = append(a, p[0]+"_"+p[1])
		a = append(a, p[2:]...)
		p = a
	}

	return i.reject(p[0], p[1], p[2], userID)
}

func (i InteractorGitOps) reject(prID string, prNum string, branch string, userID string) (blocks []slack.Block, err error) {
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
