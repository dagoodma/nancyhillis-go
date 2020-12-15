package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var Debug = false          // supress extra messages if false
var WebhookIsSilent = true // don't print anything since we return JSON

// Json object to hold the result
type StatusResult struct {
	Result *studiojourney.StudentBillingStatus `json:"result,string"`
}

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
	w.Options.IsSilent = WebhookIsSilent
	if Debug {
		util.RecordWebhookStarted(w)
	}

	// Unmarshal the input data
	m := make(map[string]string)
	err := json.Unmarshal(data, &m)
	if err != nil {
		HandleError(w, "Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Get the fields (email)
	email, ok := m["email"]
	if !ok || len(email) < 1 {
		HandleError(w, "No email address provided")
		return
	}
	if !util.EmailLooksValid(email) {
		HandleError(w, "Invalid email address: %s", email)
		return
	}

	// Get SJ billing status
	status, err := studiojourney.GetBillingStatus(email)
	_ = status
	if err != nil {
		HandleError(w, err.Error())
		return
	}

	// Marshall data and return result
	result := StatusResult{Result: status}
	r, err := json.Marshal(result)
	if err != nil {
		HandleError(w, "Failed building status response. %v", err)
		return
	}

	// Send response to JS client
	fmt.Println(string(r))
	return
}

func HandleError(w *util.WebhookEvent, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	util.PrintJsonError(message)
	if Debug {
		log.Printf(message)
	}
}
