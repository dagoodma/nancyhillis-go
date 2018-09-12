package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/teachable"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var AddUnsubscribeTags = []string{"DO_NOT_DISTURB"}

func main() {
	// Get the args
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		log.Fatalf("Not enough arguments, expected %d given: %d",
			3, len(argsWithProg))
	}

	// Create the webhook event
	programName := string(argsWithProg[0])
	header := []byte(argsWithProg[1])
	data := []byte(argsWithProg[2])
	w := util.NewWebhookEvent(programName, header, data)
	util.RecordWebhookStarted(w)

	// Unmarshall the header and ensure its correct
	h := &teachable.WebhookHeader{}
	err := json.Unmarshal(header, &h)
	if err != nil {
		util.ReportWebhookFailure(w, err.Error())
		return
	}
	err = teachable.EnsureValidWebhook(h, data)
	if err != nil {
		util.ReportWebhookFailure(w, err.Error())
		return
	}

	// Unmarshall the message
	m := &teachable.StudentUpdated{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		util.ReportWebhookFailure(w, err.Error())
		return
	}

	// Grab the data
	email := m.Object.Email
	name := m.Object.Name
	id := m.Object.Id

	// Check for updated name
	oldName := m.Object.OldName
	newName := m.Object.NewName
	if m.Object.NameUpdated {
		/*
			err := teachable.StudentUpdatedName(email, oldName, newName)
			if err != nil {
				util.ReportWebhookFailure(w, err.Error())
				return
			}
		*/
		log.Printf("Student %s changed their name from \"%s\" to: %s\n",
			email, oldName, newName)
	}

	// Check for updated email
	oldEmail := m.Object.OldEmail
	newEmail := m.Object.NewEmail
	if m.Object.EmailUpdated {
		/*
			err = teachable.StudentUpdatedEmail(id, oldEmail, newEmail)
			if err != nil {
				util.ReportWebhookFailure(w, err.Error())
				return
			}
		*/
		message := fmt.Sprintf("Student \"%s\" (%s) changed their email from \"%s\" to: %s", name, id, oldEmail, newEmail)
		log.Println(message)
		// Notify slack if they updated
		util.ReportWebhookSuccess(w, message)
	}

	// Check for unsubscribe tag wanted
	if m.Object.UnsubscribeFromMarketingEmails {
		message := fmt.Sprintf("Student %s (name: \"%s\", id: %s) prefers not to receive marketing emails.",
			email, name, id)
		log.Println(message)
		// Notify slack if they updated
		util.ReportWebhookSuccess(w, message)
	}
}
