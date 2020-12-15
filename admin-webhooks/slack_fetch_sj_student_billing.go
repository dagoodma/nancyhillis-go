package main

import (
	"encoding/json"
	"fmt"
	//"log"
	"bytes"
	"os"

	"bitbucket.org/dagoodma/dagoodma-go/slackwrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"
	"bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
)

var RespondToErrorInChannel = true
var RespondToMessageInChannel = true

var Debug = false // supress extra messages if false

// Note that we will be using our own customer error handler: HandleError()
func main() {
	slackwrap.RespondMessageTextOnly("okay, got that")

	argsWithProg := os.Args
	/*
		if len(argsWithProg) < 2 {
			HandleError(nil, "No email address provided")
			return
		}
	*/

	// Create the webhook event
	// Headers was making the command load too slow
	if len(argsWithProg) < 3 {
		HandleError(nil, "No email address provided")
		return
	}
	programName := string(argsWithProg[0])
	header := []byte(argsWithProg[1])
	data := []byte(argsWithProg[2])
	w := util.NewWebhookEvent(programName, header, data)
	/*
		programName := string(argsWithProg[0])
		data := []byte(argsWithProg[1])
		w := util.NewWebhookEvent(programName, nil, data)

	*/
	if Debug {
		util.RecordWebhookStarted(w)
	}

	// Unmarshal the input data
	c := slackwrap.SlackCommandRequest{}

	err := json.Unmarshal(data, &c)
	if err != nil {
		HandleError(w, "Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Validate command token
	err = slackwrap.ValidateCommandRequest(w.Name, c.Token)
	if err != nil {
		HandleError(w, "Failed validating slash command request for '%s'. Security token mismatch.", w.Name)
		return
	}

	// Get the fields: email is only argument to command
	email := c.Text
	if len(email) < 1 {
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

	// Return result
	msg := fmt.Sprintf("Found billing info for \"%s\".", email)
	a, err := CreateBillingInfoAttachment(status)
	if err != nil {
		HandleError(w, "%v", err)
		return
	}
	slackwrap.RespondMessageToUrl(msg, a, c.ResponseUrl)
	//slackwrap.RespondMessage(msg, a, RespondToMessageInChannel)
	return
}

func CreateBillingInfoAttachment(status *studiojourney.StudentBillingStatus) (string, error) {
	var phoneBuffer bytes.Buffer
	if len(status.Phone) > 0 {
		sx := fmt.Sprintf("Phone: %s\n", status.Phone)
		phoneBuffer.WriteString(sx)
	}
	var countryBuffer bytes.Buffer
	if len(status.Country) > 0 {
		sx := fmt.Sprintf("Country: %s\n", status.Country)
		countryBuffer.WriteString(sx)
	}
	var mmBuffer bytes.Buffer
	if len(status.MmId) > 0 {
		sx := fmt.Sprintf("Membermouse ID: %s\n"+
			"Migrated? %t\n", status.MmId, status.IsMigratedFounder)
		mmBuffer.WriteString(sx)

	}
	var subBuffer bytes.Buffer
	subBuffer.WriteString("none")
	if status.HasActiveSubscription {
		subBuffer.Reset()
		sx := fmt.Sprintf("$%.2f %s (%s)",
			status.ActiveSubscription.Amount,
			status.ActiveSubscription.Description,
			status.ActiveSubscription.CreatedDate)
		subBuffer.WriteString(sx)
	}
	var paymentsBuffer bytes.Buffer
	paymentsBuffer.WriteString("none")
	if len(status.Payments) > 0 {
		paymentsBuffer.Reset()
		for _, p := range status.Payments {
			var refundBuffer bytes.Buffer
			if p.IsRefund {
				refundBuffer.WriteString("(REFUND) ")
			}
			sx := fmt.Sprintf("\n\t%s$%.2f (%s) \"%s\" [%s]",
				refundBuffer.String(), p.Amount, p.Date, p.Description,
				p.Source)
			paymentsBuffer.WriteString(sx)
		}
	}
	r := fmt.Sprintf("Name: %s\n"+
		"Email: %s\n"+
		"Stripe ID: %s\n"+
		"%s%s"+ // phone + country
		"Founder? %t\n"+
		"%s"+ // membermouse info string
		"Active subscription? %t\n"+
		"Payment count: %d\n"+
		"Payments left: %d\n"+
		"LTV: %.2f\n"+
		"Expected remaining value: %.2f\n"+
		"Subscription: %s\n"+
		"Payments: %s\n",
		status.Name, status.Email, status.StripeId, phoneBuffer.String(), countryBuffer.String(),
		status.IsFounder, mmBuffer.String(), status.HasActiveSubscription, status.PaymentCount,
		status.RemainingPaymentCount, status.LifeTimeValue, status.RemainingLifeTimeValue,
		subBuffer.String(), paymentsBuffer.String())

	return r, nil
}

func HandleError(w *util.WebhookEvent, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	slackwrap.RespondError(message, RespondToErrorInChannel)
}
