package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	//"time"
	//"github.com/gosexy/to"

	"bitbucket.org/dagoodma/nancyhillis-go/teachable"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

func main() {
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
	m := &teachable.StudentCancelled{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		util.ReportWebhookFailure(w, err.Error())
		return
	}

	// Grab the data
	email := m.Object.User.Email

	/*
		err = teachable.StudentCancelledCourse(email, EventAction, AddTags, RemoveTags)
		if err != nil {
			util.ReportWebhookFailure(progName, string(err), data)
			return 0
	*/

	// Notify slack they joined
	message := fmt.Sprintf("Student \"%s\" cancelled a course.\n",
		email)
	log.Println(message)
	util.ReportWebhookSuccess(w, message)
}
