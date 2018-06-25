package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/nancyhillis"
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var Debug = false              // Show/hide debug output
var WebhookIsSilent = true     // don't print anything since we return JSON
var CancelAtEndOfPeriod = true // don't immediately cancel them

var CancelRequestSpreadsheetId = "1cO-eRbvtKUte5-knEqqjncZ22m5zuwbUWk3mjFuTwAE"

// Update card input data
type InputData struct {
	CustomerId     string `json:"customer_id"`
	SubscriptionId string `json:"subscription_id"`
	Reason         string `json:"reason"`
}

//Extra map[string]interface{}

func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		HandleError(nil, "No data provided")
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
	//log.Println("Entire headers: " + string(header))
	//log.Println("Entire payload: " + string(data))

	// Unmarshal the input data
	m := InputData{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		HandleError(w, "Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Get and validate the fields (customer_id)
	customerId := m.CustomerId
	if len(customerId) < 1 {
		HandleError(w, "No customer ID provided")
		return
	}
	if !stripewrap.CustomerIdLooksValid(customerId) {
		HandleError(w, "Invalid customer ID: %s", customerId)
		return
	}

	subId := m.SubscriptionId
	if len(subId) < 1 {
		HandleError(w, "No subscription ID provided")
		return
	}
	if !stripewrap.SubscriptionIdLooksValid(subId) {
		HandleError(w, "Invalid subscription ID: %s", subId)
		return
	}

	reason := m.Reason
	if len(reason) < 1 {
		reason = "unknown"
		//HandleError(w, "No reason given")
		//return
	}

	// Lookup the customer account info
	status, err := nancyhillis.GetSjAccountStatus(customerId)
	if err != nil {
		HandleError(w, err.Error())
		return
	}

	// check if they are already pending cancelation
	if status.Status == "pending_cancel" {
		HandleError(w, "Customer \"%s\" (%s) subscription (%s) is already pending cancelation.",
			status.Email, status.CustomerId, subId)
		return
	}
	// Ensure they have an active subscription that we can cancel
	if status.Status != "active" || status.IsRecurring != true {
		HandleError(w, "No active subscription to cancel for customer: %s (%s)",
			status.Email, status.CustomerId)
		return
	}

	// Cancel it
	// TODO move this to NancyHillis package
	//err = nancyhillis.CancelSjSubscription(customerId, subId)
	err = stripewrap.CancelSubscription(subId, CancelAtEndOfPeriod)
	if err != nil {
		HandleError(w, err.Error())
		return
	}

	// Return result
	r := make(map[string]interface{})
	r["result"] = "success"
	util.PrintJsonObject(r)

	// Report to slack
	message := fmt.Sprintf("Customer \"%s\" (%s) subscription was canceled by: %s",
		status.Email, customerId, reason)
	util.ReportWebhookSuccess(w, message)
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
