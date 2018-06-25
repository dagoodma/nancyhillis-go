package main

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	"bitbucket.org/dagoodma/nancyhillis-go/nancyhillis"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var Debug = false
var WebhookIsSilent = true // We are printing JSON, so we must be quiet

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
		HandleError(w, "Failed to find founder with email \"%s\". %v", email, err.Error())
		return
	}
	//fmt.Println(m2)

	status, err := m2.GetStatus()
	if err != nil {
		HandleError(w, "Failed fetching founder \"%s\" status. %v", email, err.Error())
		return
	}
	isFlaggedMigrated := m2.IsMigrated()

	// Find founder by email address in migrated spreadsheet
	isMigrated, err := nancyhillis.GetSjFounderMigratedByEmail(email)
	if err != nil { // same as isMigrated == true
		// Not migrated...
		if Debug {
			log.Println("Not migrated. Failed to find in spreadsheet. %v", err)
		}
	}

	var wantFlagMigrated bool = false
	if isMigrated && isFlaggedMigrated {
		HandleError(w, "Founder \"%s\" is already migrated.", email)
		return
	} else if isMigrated && !isFlaggedMigrated {
		// Founder is migrated but needs to be flagged in membermouse
		wantFlagMigrated = true
	} else if !isMigrated && isFlaggedMigrated {
		// Not migrated, but flagged migrated
		HandleError(w, "Founder \"%s\" is already flagged but never migrated!", email)
		return
	} else {
		// Never migrated or flagged
		wantFlagMigrated = true
		// Tell us in slack about it
		msg := fmt.Sprintf("Founder \"%s\" was never migrated, but will be flagged in membermouse.", email)
		util.ReportWebhookFailure(w, msg)
	}

	if wantFlagMigrated {
		err := m2.FlagFounderMigrated()
		if err != nil {
			HandleError(w, "Failed to flag founder \"%s\" as migrated. %v", email, err)
			return
		}
		var instStr = ""
		if !status.IsComped {
			instStr = "Founder is still being billed. Please ask David to safely cancel their billing."
		}
		// Report to slack
		message := fmt.Sprintf("Founder \"%s\" was marked as migrated. %s\nManage member URL: %s",
			status.Email, instStr, m2.GetManageMemberUrl())
		util.ReportWebhookSuccess(w, message)
	}

	// Send response to JS client
	r := make(map[string]interface{})
	r["result"] = "success"
	util.PrintJsonObject(r)
	return
}

func HandleError(w *util.WebhookEvent, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	util.PrintJsonError(message)
	if Debug {
		log.Printf(message)
	}
	if w != nil {
		util.ReportWebhookFailure(w, message)
	}
}
