package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/slackwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

// This is for the Slack slash command: /sj <email address>

var RespondToErrorInChannel = true
var RespondToMessageInChannel = true

var Debug = false // supress extra messages if false

// Note that we will be using our own customer error handler: HandleError()
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		HandleError(nil, "No email address provided")
		return
	}

	// Create the webhook event
	programName := string(argsWithProg[0])
	header := []byte(argsWithProg[1])
	data := []byte(argsWithProg[2])
	w := util.NewWebhookEvent(programName, header, data)
	if Debug {
		util.RecordWebhookStarted(w)
	}

	// Unmarshal the input data
	c := slackwrap.SlackCommandRequest{}

	err := json.Unmarshal(data, &c)
	if err != nil {
		HandleError(w, "Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Validate command token
	err = slackwrap.ValidateCommandRequest(w.Name, c.Token)
	if err != nil {
		HandleError(w, "Failed validating slash command request for '%s'. Security token mismatch.", w.Name)
		return
	}

	// Get the fields: email is only argument to command
	email := c.Text
	if len(email) < 1 {
		HandleError(w, "No email address provided")
		return
	}
	if !util.EmailLooksValid(email) {
		HandleError(w, "Invalid email address: %s", email)
		return
	}

	// Find founder by email in membermouse
	m, err := membermouse.GetMemberByEmail(email)
	if err != nil {
		HandleError(w, "Failed to find founder with email \"%s\". %v", email, err.Error())
		return
	}

	// Get their member status info
	status, err := m.GetStatus()
	if err != nil {
		HandleError(w, "Failed fetching member status. %v", err.Error())
		return
	}

	// Return result
	msg := fmt.Sprintf("Found founder \"%s\".", email)
	a, err := CreateFounderInfoAttachment(m, status)
	if err != nil {
		HandleError(w, "%v", err)
		return
	}
	slackwrap.RespondMessage(msg, a, RespondToMessageInChannel)
	return
}

func HandleError(w *util.WebhookEvent, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	slackwrap.RespondError(message, RespondToErrorInChannel)
}
