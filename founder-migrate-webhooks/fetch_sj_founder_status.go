package main

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	//"gsheetwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	"bitbucket.org/dagoodma/nancyhillis-go/nancyhillis"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var Debug = false
var WebhookIsSilent = true // We are printing JSON, so we must be quiet

// Json object to hold the result
type StatusResult struct {
	Result *membermouse.MemberStatus `json:"result,string"`
}

// Note that we will be using our own customer error handler: HandleError()
// Because this is a AJAX webhook that must always return a valid JSON response
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		HandleError(nil, "No founder email address provided")
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

	// Find founder by email in membermouse
	m2, err := membermouse.GetMemberByEmail(email)
	if err != nil {
		//HandleError(w, "Failed to find founder with email \"%s\". %v", email, err.Error())
		HandleError(w, "Failed to find founder with email: %s", email)
		return
	}
	//fmt.Println(m2)

	status, err := m2.GetStatus()
	if err != nil {
		HandleError(w, "Failed fetching member status. %v", err.Error())
		return
	}
	// Let's only use MM api for now (Faster and easier to test)
	/*
		status.IsMigrated = false // We will set this if they're in the spreadsheet

		// Find founder by email address in migrated spreadsheet
		isMigrated, err := nancyhillis.GetSjFounderMigratedByEmail(email)
		if err != nil {
			// Not migrated...
			if Debug {
				log.Printf("Not migrated. Failed to find in spreadsheet. %v\n", err)
			}
		} else {
			status.IsMigrated = isMigrated
		}
	*/

	// Check if they signed up as a regular member
	row, err := nancyhillis.GetSjEnrollmentRowByEmail(email)
	if err == nil && row != nil {
		if !status.IsMigrated {
			// We found them as already signed up
			// Uh-oh! They already enrolled as a regular member
			var msg = "already signed up as a regular SJ member!"
			HandleError(w, "You are %s", msg)

			// Tell us in slack about it
			msg = fmt.Sprintf("Founder \"%s\" is %s", email, msg)
			util.ReportWebhookFailure(w, msg)
			return
		}
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
