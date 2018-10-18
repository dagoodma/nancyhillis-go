package main

import (
	"encoding/json"
	"fmt"
	"log"
	//"net/url"
	"os"
	//"strings"

	"bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
	"bitbucket.org/dagoodma/nancyhillis-go/slackwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

// This is for the Slack slash command: /ac <email address>

var RespondToErrorInChannel = true
var RespondToMessageInChannel = true

var Debug = false // supress extra messages if false

// Note that we will be using our own customer error handler: HandleError()
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		//HandleError(nil, "No email address provided")
		log.Fatalf("No email address provided")
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
		HandleError(w, "Invalid student email: %s", email)
		return
	}
	contact, err := activecampaign.GetContactByEmail(email)
	if err != nil {
		HandleError(w, "Failed fetching AC subscriber: %s", err)
		return
	}

	// Return result
	msg := fmt.Sprintf("Found AC subscriber \"%s\".", email)
	a, err := CreateSubscriberInfoAttachment(contact)
	if err != nil {
		HandleError(w, "%v", err)
		return
	}
	slackwrap.RespondMessage(msg, a, RespondToMessageInChannel)
	return
}

func CreateSubscriberInfoAttachment(contact *activecampaign.ListContactsContact) (string, error) {
	name := fmt.Sprintf("%s %s", contact.FirstName, contact.LastName)
	url := activecampaign.GetContactProfileUrlById(contact.Id)
	r := fmt.Sprintf("Name: %s\n"+
		"Email: %s\n"+
		"ID: %s\n"+
		"Created: %s\n"+
		"Updated: %s\n"+
		"Profile URL: %s\n",
		name, contact.Email, contact.Id, contact.CreationDate, contact.UpdatedUtcTimestamp,
		url)
	return r, nil
}
func HandleError(w *util.WebhookEvent, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	//if Debug {
	//	log.Printf(message)
	//}
	slackwrap.RespondError(message, RespondToErrorInChannel)
	/*
		if w != nil {
			util.ReportWebhookFailure(w, message)
		} else {
			log.Fatalf(message)
		}
	*/
	//util.PrintJsonError(message) // message to Slack
}
