package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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
	m := &teachable.StudentEnrolled{}
	err = json.Unmarshal(data, &m)
	if err != nil {
		util.ReportWebhookFailure(w, err.Error())
		return
	}

	// Grab the data
	email := m.Object.User.Email
	id := m.Object.User.Id
	//name := m.Object.User.Name
	ip := m.Object.User.CurrentSignInIp

	// TODO check course id
	// if courseId != SjCourseId {
	//teachable.StudentEnrolledInCourse(email, id, name, ip, EventAction, AddTags, RemoveTags)

	// Notify slack they joined
	message := fmt.Sprintf("Student \"%s\" (%s) (ip: %s) enrolled in a course.\n",
		email, id, ip)
	log.Println(message)
	util.ReportWebhookSuccess(w, message)
}
