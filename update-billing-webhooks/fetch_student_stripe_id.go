package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	"bitbucket.org/dagoodma/nancyhillis-go/nancyhillis"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var Debug = false // Show/hide debug output

//var LoggedInUserSecretKey = "RBxEi2rt4Skd4TgKytdusBbdp4A4wtbvH"

// Note that we will be using our own customer error handler: HandleError()
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		HandleError("No email address provided")
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
	m := make(map[string]string)
	err := json.Unmarshal(data, &m)
	if err != nil {
		HandleError("Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Get the fields (email)
	email, ok := m["email"]
	if !ok || len(email) < 1 {
		HandleError("No email address provided")
		return
	}
	if !util.EmailLooksValid(email) {
		HandleError("Invalid email address: %s", email)
		return
	}

	// Find Stripe customer ID in spreadsheet
	stripeId, err := nancyhillis.GetSjStripeIdByEmail(email)
	if err != nil {
		// Check if they're a founder who never migrated
		if IsFounderNeverMigrated(email) {
			HandleError("Your account is still in our old billing system and still needs to me moved over")
		} else {
			HandleError("%v", err)
		}
		return
	}

	// Return result
	r := make(map[string]interface{})
	r["result"] = stripeId
	util.PrintJsonObject(r)
	return
}

// Check if this person is in membermouse and never migrated
func IsFounderNeverMigrated(email string) bool {
	// Find founder by email in membermouse
	m, err := membermouse.GetMemberByEmail(email)
	if err != nil {
		return false
	}

	isMigrated := m.IsMigrated()
	if isMigrated {
		return false
	}

	return true
}

func HandleError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	util.PrintJsonError(message)
	if Debug {
		log.Printf(message)
	}
}
