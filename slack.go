package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/zaiminc/gocat/deploy"
	"github.com/zaiminc/gocat/slackcmd"
)

// SlackListener is a http.Handler that can handle slack events.
// See https://api.slack.com/apis/connections/events-api for more details about events.
type SlackListener struct {
	client            *slack.Client
	verificationToken string
	projectList       *ProjectList
	userList          *UserList
	interactorFactory *InteractorFactory

	coordinator *deploy.Coordinator

	allowedPhases []string
}

func (s SlackListener) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(r.Body); err != nil {
		fmt.Printf("[ERROR] Failed to read request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	body := buf.String()
	header := r.Header

	if header.Get("X-Slack-Retry-Num") != "" {
		slackRetryNum, _ := strconv.Atoi(header.Get("X-Slack-Retry-Num"))
		if slackRetryNum > 0 {
			return
		}
	}

	eventsAPIEvent, err := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: s.verificationToken}))
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if eventsAPIEvent.Type == slackevents.URLVerification {
		var r *slackevents.ChallengeResponse
		err = json.Unmarshal([]byte(body), &r)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text")
		if _, err := w.Write([]byte(r.Challenge)); err != nil {
			fmt.Printf("[ERROR] Failed to write challenge response: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	if eventsAPIEvent.Type == slackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		switch ev := innerEvent.Data.(type) {
		case *slackevents.AppMentionEvent:
			if err := s.handleMessageEvent(ev); err != nil {
				log.Println("[ERROR] ", err)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}
}

func (s *SlackListener) handleMessageEvent(ev *slackevents.AppMentionEvent) error {
	// Only response mention to bot. Ignore else.
	log.Print(ev.Text)

	cmd, err := slackcmd.Parse(ev.Text)
	if err == nil {
		log.Printf("[INFO] %s command is Called", cmd.Name())
		return s.runCommand(cmd, ev.User, ev.Channel)
	}

	if _, _, err := s.client.PostMessage(ev.Channel, s.errorMessage("Invalid command. Say `@bot help` to see the usage guide")); err != nil {
		log.Println("[INFO] invalid command", ev.Text)
	}

	return nil
}

func (s *SlackListener) helpMessage() slack.MsgOption {
	deployMasterText := slack.NewTextBlockObject("mrkdwn", "*masterのデプロイ*\n`@bot-name deploy api staging`\napiの部分はその他アプリケーションに置換可能です。stagingの部分はproductionやsandboxに置換可能です。\nコマンド入力後にデプロイするかの確認ボタンが出てきます。", false, false)
	deployMasterSection := slack.NewSectionBlock(deployMasterText, nil, nil)

	deployBranchText := slack.NewTextBlockObject("mrkdwn", "*ブランチのデプロイ*\n`@bot-name deploy api staging branch`\napiの部分はその他アプリケーションに置換可能です。stagingの部分はproductionやsandboxに置換可能です。\nブランチを選択するドロップダウンが出てきます。\nブランチ選択後にデプロイするかの確認ボタンが出てきます。", false, false)
	deployBranchSection := slack.NewSectionBlock(deployBranchText, nil, nil)

	deployText := slack.NewTextBlockObject("mrkdwn", "*デプロイ対象の選択をSlackのUIから選択するデプロイ手法*\n`@bot-name deploy staging`\nstagingの部分はproductionやsandboxに置換可能です。\nデプロイ対象の選択後にデプロイするブランチの選択肢が出てきます。", false, false)
	deploySection := slack.NewSectionBlock(deployText, nil, nil)

	lockText := slack.NewTextBlockObject("mrkdwn", "*デプロイロックをとる*\n`@bot-name lock api staging for REASON`\napiの部分はその他アプリケーションに置換可能です。stagingの部分はproductionやsandboxに置換可能です。\nREASON部分にロックする理由を指定する必要があります。", false, false)
	lockSection := slack.NewSectionBlock(lockText, nil, nil)

	unlockText := slack.NewTextBlockObject("mrkdwn", "*デプロイロックを解除する*\n`@bot-name unlock api staging`\napiの部分はその他アプリケーションに置換可能です。stagingの部分はproductionやsandboxに置換可能です。", false, false)
	unlockSection := slack.NewSectionBlock(unlockText, nil, nil)

	describeLocksText := slack.NewTextBlockObject("mrkdwn", "*デプロイロックの状態を確認する*\n`@bot-name describe locks`\nデプロイロックの状態を確認します。", false, false)
	describeLocksSection := slack.NewSectionBlock(describeLocksText, nil, nil)

	return slack.MsgOptionBlocks(
		deployMasterSection,
		deployBranchSection,
		deploySection,
		lockSection,
		unlockSection,
		describeLocksSection,
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
		sections[i+1] = createDeployButtonSection(pj, phase)
	}
	sections[len(sections)-1] = CloseButton()
	return slack.MsgOptionBlocks(sections...)
}

func createDeployButtonSection(pj DeployProject, phaseName string) *slack.SectionBlock {
	action := "branchlist"
	if pj.DisableBranchDeploy {
		action = "request"
	}
	phase := pj.FindPhase(phaseName)
	txt := slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s* (%s)", pj.ID, pj.GitHubRepository()), false, false)
	btnTxt := slack.NewTextBlockObject("plain_text", "Deploy", false, false)
	btn := slack.NewButtonBlockElement("", fmt.Sprintf("deploy_%s_%s|%s_%s", phase.Kind, action, pj.ID, phase.Name), btnTxt)
	section := slack.NewSectionBlock(txt, nil, slack.NewAccessory(btn))
	return section
}

// runCommand runs the given command.
//
// triggeredBy is the ID of the Slack user who triggered the command,
// and replyIn is the ID of the Slack channel to reply to.
func (s *SlackListener) runCommand(cmd slackcmd.Command, triggeredBy string, replyIn string) error {
	var msgOpt slack.MsgOption

	if envCmd, ok := cmd.(slackcmd.EnvCommand); ok && !isPhaseAllowed(s.allowedPhases, envCmd.EnvName()) {
		msgOpt = s.errorMessage(disallowedPhaseError(envCmd.EnvName()).Error())
		if _, _, err := s.client.PostMessage(replyIn, msgOpt); err != nil {
			log.Println("[ERROR] ", err)
		}
		return nil
	}

	switch cmd.(type) {
	case *slackcmd.Help, *slackcmd.ListProjects:
		// do nothing
	default:
		s.projectList.Reload()
		s.userList.Reload()
	}

	switch cmd := cmd.(type) {
	case *slackcmd.Help:
		msgOpt = s.helpMessage()
	case *slackcmd.ListProjects:
		msgOpt = s.projectListMessage()
	case *slackcmd.Reload:
		msgOpt = s.reloadMessage()
	case *slackcmd.Deploy:
		msgOpt = s.deploy(cmd, triggeredBy, replyIn)
	case *slackcmd.DeployBranchList:
		msgOpt = s.deployBranchList(cmd)
	case *slackcmd.DeployTargetSelection:
		msgOpt = s.SelectDeployTarget(cmd.Env)
	case *slackcmd.Lock:
		user := s.userList.FindBySlackUserID(triggeredBy)
		msgOpt = s.lock(cmd, user, replyIn)
	case *slackcmd.Unlock:
		user := s.userList.FindBySlackUserID(triggeredBy)
		msgOpt = s.unlock(cmd, user, replyIn)
	case *slackcmd.DescribeLocks:
		msgOpt = s.describeLocks()
	default:
		return fmt.Errorf("unsupported slack command %T", cmd)
	}

	if _, _, err := s.client.PostMessage(replyIn, msgOpt); err != nil {
		log.Println("[ERROR] ", err)
	}

	return nil
}

func (s *SlackListener) reloadMessage() slack.MsgOption {
	section := slack.NewSectionBlock(slack.NewTextBlockObject("mrkdwn", "Deploy Projects and Users is Reloaded", false, false), nil, nil)
	return slack.MsgOptionBlocks(section)
}

func (s *SlackListener) deploy(cmd *slackcmd.Deploy, triggeredBy string, replyIn string) slack.MsgOption {
	target, err := s.projectList.FindByAlias(cmd.Project)
	if err != nil {
		log.Println("[ERROR] ", err)
		return s.errorMessage(err.Error())
	}

	phase := cmd.Env
	if msg, locked := s.checkDeploymentLock(target.ID, phase, triggeredBy, replyIn); locked {
		return msg
	}

	interactor := s.interactorFactory.Get(target, phase)
	blocks, err := interactor.Request(target, phase, target.DefaultBranch(), triggeredBy, replyIn)
	if err != nil {
		log.Println("[ERROR] ", err)
		return s.errorMessage(err.Error())
	}

	return slack.MsgOptionBlocks(blocks...)
}

func (s *SlackListener) deployBranchList(cmd *slackcmd.DeployBranchList) slack.MsgOption {
	target, err := s.projectList.FindByAlias(cmd.Project)
	if err != nil {
		log.Println("[ERROR] ", err)
		return s.errorMessage(err.Error())
	}

	phase := cmd.Env
	interactor := s.interactorFactory.Get(target, phase)
	blocks, err := interactor.BranchList(target, phase)
	if err != nil {
		log.Println("[ERROR] ", err)
		return s.errorMessage(err.Error())
	}

	return slack.MsgOptionBlocks(blocks...)
}

// lock locks the given project and environment, and replies to the given channel.
func (s *SlackListener) lock(cmd *slackcmd.Lock, triggeredBy User, replyIn string) slack.MsgOption {
	if err := s.validateProjectEnvUser(cmd.Project, cmd.Env, triggeredBy, replyIn); err != nil {
		return s.errorMessage(err.Error())
	}

	if err := s.coordinator.Lock(context.Background(), cmd.Project, cmd.Env, triggeredBy.SlackDisplayName, cmd.Reason); err != nil {
		return s.errorMessage(err.Error())
	}

	return s.infoMessage(fmt.Sprintf("Locked %s %s", cmd.Project, cmd.Env))
}

// unlock unlocks the given project and environment, and replies to the given channel.
func (s *SlackListener) unlock(cmd *slackcmd.Unlock, triggeredBy User, replyIn string) slack.MsgOption {
	if err := s.validateProjectEnvUser(cmd.Project, cmd.Env, triggeredBy, replyIn); err != nil {
		return s.errorMessage(err.Error())
	}

	if err := s.coordinator.Unlock(context.Background(), cmd.Project, cmd.Env, triggeredBy.SlackDisplayName, triggeredBy.IsAdmin()); err != nil {
		return s.errorMessage(err.Error())
	}

	return s.infoMessage(fmt.Sprintf("Unlocked %s %s", cmd.Project, cmd.Env))
}

// describeLocks describes the locks of all projects and environments, and replies to the given channel.
func (s *SlackListener) describeLocks() slack.MsgOption {
	projects, err := s.coordinator.DescribeLocks(context.Background())
	if err != nil {
		return s.errorMessage(err.Error())
	}

	msg := deploy.FormatProjectDescs(projects)

	return s.infoMessage(msg)
}

func (s *SlackListener) checkDeploymentLock(projectID, env string, triggeredBy string, replyIn string) (slack.MsgOption, bool) {
	locks, err := s.coordinator.FetchLocks(context.Background(), projectID, env)
	if err != nil {
		log.Println("[ERROR] ", err)
		if _, _, err := s.client.PostMessage(replyIn, s.infoMessage(err.Error())); err != nil {
			log.Println("[ERROR] ", err)
		}
		return nil, false
	}

	if len(locks) == 0 {
		// Missing lock means it has never been locked.
		return nil, false
	}

	lock, ok := locks[projectID][env]
	if !ok {
		// Missing lock means it has never been locked.
		return nil, false
	}

	user := s.userList.FindBySlackUserID(triggeredBy)

	if lock.Locked && lock.LockHistory[len(lock.LockHistory)-1].User != user.SlackDisplayName {
		cause := fmt.Sprintf("locked by %s", lock.LockHistory[len(lock.LockHistory)-1].User)
		return s.infoMessage(fmt.Sprintf("Deployment failed: %s", cause)), true
	}

	return nil, false
}

func (s *SlackListener) validateProjectEnvUser(projectID, env string, user User, replyIn string) error {
	pj, err := s.projectList.FindByAlias(projectID)
	if err != nil {
		log.Println("[ERROR] ", err)
		if _, _, err := s.client.PostMessage(replyIn, s.errorMessage(err.Error())); err != nil {
			log.Println("[ERROR] ", err)
		}
		return fmt.Errorf("find by alias %q: %w", projectID, err)
	}

	if phase := pj.FindPhase(env); phase.None() {
		err = fmt.Errorf("phase %s is not found", env)
		log.Println("[ERROR] ", err)
		if _, _, err := s.client.PostMessage(replyIn, s.errorMessage(err.Error())); err != nil {
			log.Println("[ERROR] ", err)
		}
		return fmt.Errorf("find phase %q: %w", env, err)
	}

	if !user.IsDeveloper() {
		return fmt.Errorf("you are not allowed to lock/unlock projects: %q is missing the Developer role", user.SlackDisplayName)
	}

	return nil
}

func (s *SlackListener) infoMessage(message string) slack.MsgOption {
	txt := slack.NewTextBlockObject("mrkdwn", message, false, false)
	section := slack.NewSectionBlock(txt, nil, nil)
	return slack.MsgOptionBlocks(section)
}

func (s *SlackListener) errorMessage(message string) slack.MsgOption {
	txt := slack.NewTextBlockObject("mrkdwn", message, false, false)
	section := slack.NewSectionBlock(txt, nil, nil)
	return slack.MsgOptionBlocks(section)
}
