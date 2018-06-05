package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

//var SpreadsheetId = "1wRHucYoRuGzHav7nK3V5Hv2Z4J67D_vTZN5wjw8aa2k"
var Debug = false          // Show/hide debug output
var WebhookIsSilent = true // don't print anything since we return JSON

//var LoggedInUserSecretKey = "RBxEi2rt4Skd4TgKytdusBbdp4A4wtbvH"

// Update card input data
type InputData struct {
	CustomerId  string `json:"customer_id"`
	StripeToken string `json:"stripe_token"`
}

//Extra map[string]interface{}

func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		HandleError("No data provided")
		return
	}
	//log.Println("Entire headers: " + string(header))
	//log.Println("Entire payload: " + string(data))

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
	m := InputData{}
	err := json.Unmarshal(data, &m)
	//m := make(map[string]string)
	//err := json.Unmarshal(data, &m)
	if err != nil {
		HandleError("Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Get and validate the fields (customer_id)
	//customerId, ok := m["customer_id"]
	customerId := m.CustomerId
	if len(customerId) < 1 {
		HandleError("No customer ID provided")
		return
	}
	if !stripewrap.CustomerIdLooksValid(customerId) {
		HandleError("Invalid customer ID: %s", customerId)
		return
	}
	//stripeToken, ok := m["stripe_token"]
	stripeToken := m.StripeToken
	if len(stripeToken) < 1 {
		HandleError("No card data provided")
		return
	}
	// TODO Validate token first?

	// Fetch Stripe customer data
	cust, err := stripewrap.GetCustomer(customerId)
	if err != nil || cust == nil {
		reason := ""
		// Try and parse the Stripe error (to make it less cryptic for user)
		ers := err.Error()
		serr, err2 := stripewrap.UnmarshallErrorResponse([]byte(ers))
		if err2 == nil {
			if serr.HTTPStatusCode == 404 {
				reason = "No such customer ID"
			}
		}
		HandleError("Failed retrieving customer data. %s", reason)
		return
	}

	// Update the customer resource after saving the token as a source
	err = stripewrap.SaveNewDefaultCard(customerId, stripeToken)
	if err != nil {
		reason := "Could not update user account"
		// Try and parse the Stripe error (to make it less cryptic for user)
		ers := err.Error()
		serr, err2 := stripewrap.UnmarshallErrorResponse([]byte(ers))
		if err2 == nil {
			/*
				if serr.HTTPStatusCode >= 400 {
					reason = "No such customer ID"
				}
			*/
			reason = serr.Msg
		}
		HandleError("Error setting default card. %s", reason)
		return
	}

	// Return result
	r := make(map[string]interface{})
	r["result"] = "success"
	util.PrintJsonObject(r)

	// Report to slack
	message := fmt.Sprintf("Customer \"%s\" (%s) updated their default credit card.",
		customerId, cust.Email)
	util.ReportSlackWebhookSuccess(w, message)
	return
}

func HandleError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	util.PrintJsonError(message)
	if Debug {
		log.Printf(message)
	}
}
