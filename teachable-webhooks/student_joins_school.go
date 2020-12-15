package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/dagoodma-go/util"
	"bitbucket.org/dagoodma/nancyhillis-go/teachable"
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
	m := &teachable.NewStudent{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		util.ReportWebhookFailure(w, err.Error())
		return
	}

	// Grab the data
	email := m.Object.Email
	id := m.Object.Id
	name := m.Object.Name

	/*
		err = teachable.StudentJoinedSchool(id, email, name, AddTags)
		if err != nil {
			util.ReportWebhookFailure(progName, err.Error(), data)
			return
		}
	*/

	// Notify slack they joined
	message := fmt.Sprintf("Student \"%s\" (%s) with name \"%s\" joined your school.\n",
		email, id, name)
	log.Println(message)
	util.ReportWebhookSuccess(w, message)

}
