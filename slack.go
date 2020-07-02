package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/nlopes/slack"
)

type SlackListener struct {
	client            *slack.Client
	botID             string
	projectList       *ProjectList
	userList          *UserList
	interactorFactory *InteractorFactory
}

// ListenAndResponse RTMイベントの待ち受け
func (s *SlackListener) ListenAndResponse() {
	// Start listening slack events
	rtm := s.client.NewRTM()
	go rtm.ManageConnection()

	// Handle slack events
	for msg := range rtm.IncomingEvents {
		fmt.Println("Event Reveived: ")
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			if err := s.handleMessageEvent(ev); err != nil {
				log.Printf("[ERROR] Failed to handle message: %s", err)
			}
			log.Print("[INFO] call")
		}
	}
}

func (s *SlackListener) handleMessageEvent(ev *slack.MessageEvent) error {
	// Only response mention to bot. Ignore else.
	log.Print(ev.Msg.Text)
	if !strings.HasPrefix(ev.Msg.Text, fmt.Sprintf("<@%s> ", s.botID)) {
		return nil
	}
	if regexp.MustCompile(`help`).MatchString(ev.Msg.Text) {
		s.client.PostMessage(ev.Msg.Channel, s.helpMessage())
		return nil
	}
	if regexp.MustCompile(`ls`).MatchString(ev.Msg.Text) {
		s.client.PostMessage(ev.Msg.Channel, s.projectListMessage())
		return nil
	}
	if regexp.MustCompile(`reload`).MatchString(ev.Msg.Text) {
		s.projectList.Reload()
		s.userList.Reload()
		section := slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "Deploy Projects and Users is Reloaded", false, false), nil, nil)
		s.client.PostMessage(ev.Msg.Channel, slack.MsgOptionBlocks(section))
		return nil
	}

	s.projectList.Reload()
	s.userList.Reload()
	if match := regexp.MustCompile(`deploy ([0-9a-zA-Z-]+) (staging|production|stg|pro|prd) branch`).FindAllStringSubmatch(ev.Msg.Text, -1); match != nil {
		log.Println("[INFO] Deploy command is Called")
		commands := strings.Split(match[0][0], " ")
		target, err := s.projectList.FindByAlias(commands[1])
		if err != nil {
			log.Println("[ERROR] ", err)
			s.client.PostMessage(ev.Msg.Channel, s.errorMessage(err.Error()))
			return nil
		}

		phase := s.toPhase(commands[2])
		interactor := s.interactorFactory.Get(target, phase)
		blocks, err := interactor.BranchList(target, phase)
		if err != nil {
			log.Println("[ERROR] ", err)
			s.client.PostMessage(ev.Msg.Channel, s.errorMessage(err.Error()))
			return nil
		}

		s.client.PostMessage(ev.Msg.Channel, slack.MsgOptionBlocks(blocks...))
		return nil
	}
	if match := regexp.MustCompile(`deploy ([0-9a-zA-Z-]+) (staging|production|stg|pro|prd)`).FindAllStringSubmatch(ev.Msg.Text, -1); match != nil {
		log.Println("[INFO] Deploy command is Called")
		commands := strings.Split(match[0][0], " ")
		target, err := s.projectList.FindByAlias(commands[1])
		if err != nil {
			log.Println("[ERROR] ", err)
			s.client.PostMessage(ev.Msg.Channel, s.errorMessage(err.Error()))
			return nil
		}

		phase := s.toPhase(commands[2])
		interactor := s.interactorFactory.Get(target, phase)
		blocks, err := interactor.Request(target, phase, "master", ev.User)
		if err != nil {
			log.Println("[ERROR] ", err)
			s.client.PostMessage(ev.Msg.Channel, s.errorMessage(err.Error()))
			return nil
		}

		s.client.PostMessage(ev.Msg.Channel, slack.MsgOptionBlocks(blocks...))
		return nil
	}
	if regexp.MustCompile(`deploy staging`).MatchString(ev.Msg.Text) {
		msgOpt := s.SelectDeployTarget("staging")
		s.client.PostMessage(ev.Msg.Channel, msgOpt)
		return nil
	}
	if regexp.MustCompile(`deploy production`).MatchString(ev.Msg.Text) {
		msgOpt := s.SelectDeployTarget("production")
		s.client.PostMessage(ev.Msg.Channel, msgOpt)
		return nil
	}
	return nil
}

func (s *SlackListener) helpMessage() slack.MsgOption {
	headerText := slack.NewTextBlockObject("mrkdwn", ":zaim:", false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)

	deployMasterText := slack.NewTextBlockObject("mrkdwn", "*masterのデプロイ*\n`@zaim-cat deploy api staging`\napiの部分はその他アプリケーションに置換可能です。stagingの部分はproductionに置換可能です。\nコマンド入力後にデプロイするかの確認ボタンが出てきます。", false, false)
	deployMasterSection := slack.NewSectionBlock(deployMasterText, nil, nil)

	deployBranchText := slack.NewTextBlockObject("mrkdwn", "*ブランチのデプロイ*\n`@zaim-cat deploy api staging branch`\napiの部分はその他アプリケーションに置換可能です。stagingの部分はproductionに置換可能です。\nブランチを選択するドロップダウンが出てきます。\nブランチ選択後にデプロイするかの確認ボタンが出てきます。", false, false)
	deployBranchSection := slack.NewSectionBlock(deployBranchText, nil, nil)

	deployText := slack.NewTextBlockObject("mrkdwn", "*デプロイ対象の選択をSlackのUIから選択するデプロイ手法*\n`@zaim-cat deploy staging`\nstagingの部分はproductionに置換可能です。\nデプロイ対象の選択後にデプロイするブランチの選択肢が出てきます。", false, false)
	deploySection := slack.NewSectionBlock(deployText, nil, nil)

	return slack.MsgOptionBlocks(
		headerSection,
		deployMasterSection,
		deployBranchSection,
		deploySection,
		CloseButton(),
	)
}

func (s *SlackListener) projectListMessage() slack.MsgOption {
	text := ""
	for _, pj := range s.projectList.Items {
		text = text + fmt.Sprintf("*%s* (%s)\n", pj.ID, pj.GitHubRepository())
	}

	listText := slack.NewTextBlockObject("mrkdwn", text, false, false)
	listSection := slack.NewSectionBlock(listText, nil, nil)

	return slack.MsgOptionBlocks(
		listSection,
		CloseButton(),
	)
}

// SelectDeployTarget デプロイ対象を選択するボタンを表示する
func (s *SlackListener) SelectDeployTarget(phase string) slack.MsgOption {
	headerText := slack.NewTextBlockObject("mrkdwn", ":cat:", false, false)
	headerSection := slack.NewSectionBlock(headerText, nil, nil)
	sections := make([]slack.Block, len(s.projectList.Items)+2)
	sections[0] = headerSection
	for i, pj := range s.projectList.Items {
		sections[i+1] = createDeployButtonSection(pj.GitHubRepository(), pj.ID, phase)
	}
	sections[len(sections)-1] = CloseButton()
	return slack.MsgOptionBlocks(sections...)
}

func createDeployButtonSection(repo string, target string, phase string) *slack.SectionBlock {
	txt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* (%s)", target, repo), false, false)
	btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
	btn := slack.NewButtonBlockElement("", fmt.Sprintf("deploy_target_branchlist|%s_%s", target, phase), btnTxt)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn))
	return section
}

func (s *SlackListener) errorMessage(message string) slack.MsgOption {
	txt := slack.NewTextBlockObject("mrkdwn", message, false, false)
	section := slack.NewSectionBlock(txt, nil, nil)
	return slack.MsgOptionBlocks(section)
}

func (s *SlackListener) toPhase(str string) string {
	switch str {
	case "pro", "prd", "production":
		return "production"
	case "stg", "staging":
		return "staging"
	default:
		return "staging"
	}
}
