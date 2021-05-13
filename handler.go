package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/nlopes/slack"
)

type interactionHandler struct {
	verificationToken string
	client            *slack.Client
	projectList       *ProjectList
	userList          *UserList
	interactorFactory *InteractorFactory
}

func getSlackError(system, msg string, user string) []byte {
	respoonse := slack.Message{
		Msg: slack.Msg{
			ResponseType: "in_channel",
			Text:         fmt.Sprintf("%s: %s actioned by <@%s>", system, msg, user),
		},
	}

	respoonse.ReplaceOriginal = true

	responseBytes, _ := json.Marshal(respoonse)

	return responseBytes
}

func (h interactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Parse input from request
	if err := r.ParseForm(); err != nil {
		// getSlackError is a helper to quickly render errors back to slack
		responseBytes := getSlackError("Server Error", "An unknown error occurred", "unkown")
		w.Write(responseBytes) // not display message on slack
		return
	}

	interactionRequest := slack.InteractionCallback{}
	json.Unmarshal([]byte(r.PostForm.Get("payload")), &interactionRequest)

	// Get the action from the request, it'll always be the first one provided in my case
	var actionValue string
	switch interactionRequest.ActionCallback.BlockActions[0].Type {
	case "button":
		actionValue = interactionRequest.ActionCallback.BlockActions[0].Value
	case "static_select":
		actionValue = interactionRequest.ActionCallback.BlockActions[0].SelectedOption.Value
	}
	userID := interactionRequest.User.ID
	// Handle close action
	if strings.Contains(actionValue, "close") {

		// Found this on stack overflow, unsure if this exists in the package
		closeStr := fmt.Sprintf(`{
		'response_type': 'in_channel',
		'text': 'closed by <@%s>',
		'replace_original': true,
		'delete_original': true
		}`, userID)

		// Post close json back to response URL to close the message
		http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer([]byte(closeStr)))
		return
	}
	log.Printf("[INFO] Action Value: %s", actionValue)
	if strings.HasPrefix(actionValue, "deploy") {
		h.Deploy(w, interactionRequest)
		return
	}

	log.Print("[ERROR] An unknown error occurred")
	responseBytes := getSlackError("Server Error", "An unknown error occurred", userID)
	http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer([]byte(responseBytes)))
}

func (h interactionHandler) Deploy(w http.ResponseWriter, interactionRequest slack.InteractionCallback) {
	actionValue := interactionRequest.ActionCallback.BlockActions[0].Value
	if actionValue == "" {
		actionValue = interactionRequest.ActionCallback.BlockActions[0].SelectedOption.Value
	}
	userID := interactionRequest.User.ID
	user := h.userList.FindBySlackUserID(userID)
	if !user.IsDeveloper() {
		h.postForbiddenError(interactionRequest.ResponseURL, userID)
		return
	}
	params := strings.Split(actionValue, "|")
	if len(params) != 2 {
		h.postInternalServerError(interactionRequest.ResponseURL, userID)
		return
	}
	interactor := h.interactorFactory.GetByParams(params[0])
	var blocks []slack.Block
	var err error
	switch {
	case strings.Contains(params[0], "request"):
		p := strings.Split(params[1], "_")
		if len(p) != 2 {
			err = fmt.Errorf("Invalid Arguments")
			break
		}
		pj := h.projectList.Find(p[0])
		blocks, err = interactor.Request(pj, p[1], pj.DefaultBranch(), userID, interactionRequest.Channel.ID)
	case strings.Contains(params[0], "approve"):
		blocks, err = interactor.Approve(params[1], userID, interactionRequest.Channel.ID)
	case strings.Contains(params[0], "reject"):
		blocks, err = interactor.Reject(params[1], userID)
	case strings.Contains(params[0], "selectbranch"):
		blocks, err = interactor.SelectBranch(params[1], interactionRequest.ActionCallback.BlockActions[0].SelectedOption.Text.Text, userID, interactionRequest.Channel.ID)
	case strings.Contains(params[0], "branchlist"):
		blocks, err = interactor.BranchListFromRaw(params[1])
	default:
		h.postInternalServerError(interactionRequest.ResponseURL, userID)
		return
	}
	if err != nil {
		log.Print(err)
		h.postInternalServerError(interactionRequest.ResponseURL, userID)
		return
	}
	responseData := slack.NewBlockMessage(blocks...)
	responseData.ReplaceOriginal = true
	responseBytes, _ := json.Marshal(responseData)
	http.Post(interactionRequest.ResponseURL, "application/json", bytes.NewBuffer(responseBytes))
}

func (h interactionHandler) postForbiddenError(responseURL string, userID string) {
	log.Print("[ERROR] Forbidden Error")
	responseBytes := getSlackError("Forbidden Error", "Please contact admin.", userID)
	http.Post(responseURL, "application/json", bytes.NewBuffer(responseBytes))
}

func (h interactionHandler) postInternalServerError(responseURL string, userID string) {
	log.Print("[ERROR] Internal Server Error")
	responseBytes := getSlackError("Internal Server Error", "Please contact admin.", userID)
	http.Post(responseURL, "application/json", bytes.NewBuffer(responseBytes))
}
