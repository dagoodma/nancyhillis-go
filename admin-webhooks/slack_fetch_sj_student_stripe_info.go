package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"bitbucket.org/dagoodma/nancyhillis-go/slackwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

// This is for the Slack slash command: /sj_stripe <email address>

var RespondToErrorInChannel = true
var RespondToMessageInChannel = true

var Debug = false // supress extra messages if false

// Note that we will be using our own customer error handler: HandleError()
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		//HandleError(nil, "No email address provided")
		log.Fatalf("No email address provided")
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

	// Find Stripe customer ID in spreadsheet
	stripeId, err := studiojourney.GetStripeIdByEmail(email)
	if err != nil {
		HandleError(w, "%v", err)
		return
	}

	// Get SJ account status
	// TODO support other courses
	status, err := studiojourney.GetAccountStatus(stripeId)
	_ = status
	if err != nil {
		HandleError(w, err.Error())
		return
	}

	// Return result
	msg := fmt.Sprintf("Found student \"%s\".", email)
	a, err := CreateStudentInfoAttachment(status)
	if err != nil {
		HandleError(w, "%v", err)
		return
	}
	slackwrap.RespondMessage(msg, a, RespondToMessageInChannel)
	return
}

func CreateStudentInfoAttachment(status *studiojourney.StudentStatus) (string, error) {
	name := fmt.Sprintf("%s %s", status.FirstName, status.LastName)
	startTime := time.Unix(status.Start, 0)
	createdTime := time.Unix(status.Created, 0)
	startTimeStr := startTime.Format("Mon Jan 2 15:04 2006")
	createdTimeStr := createdTime.Format("Mon Jan 2 15:04 2006")
	r := fmt.Sprintf("Name: %s\n"+
		"Email: %s\n"+
		"Stripe ID: %s\n"+
		"Plan: %s\n"+
		"Status: %s\n"+
		"VAT ID: %s\n"+
		"Trial? %t\n"+
		"Founder? %t\n"+
		"Package? %t\n"+
		"Recurring? %t\n"+
		"Overdue? %t\n"+
		"Delinquent? %t\n"+
		"Cancel at end of period? %t\n"+
		"Refunded? %t\n"+
		"Enrolled time: %s\n"+
		"Billing cycle anchor: %s\n"+
		"Created: %s\n"+
		"Started: %s\n",
		name, status.Email, status.CustomerId, status.PlanHuman, status.StatusHuman, status.BusinessVatId,
		status.IsTrial, status.IsFounder, status.IsPackage, status.IsRecurring, status.IsOverdue,
		status.IsDelinquent, status.CancelAtEndOfPeriod, status.IsRefunded, status.EnrolledDurationHuman,
		status.BillingCycleAnchorHuman, createdTimeStr, startTimeStr)
	if status.IsOverdue {
		r = fmt.Sprintf("%sOverdue grace period days left: %d\n", r, status.GracePeriodDaysLeft)
	}
	if status.CancelAtEndOfPeriod {
		r = fmt.Sprintf("%sCanceling days left: %d\n", r, status.DaysUntilEndOfPeriod)
	}
	if status.IsTrial {
		r = fmt.Sprintf("%sTrial days left: %d\n", r, status.TrialDaysLeft)
	}
	if status.IsRecurring {
		p := float64(status.RecurringPrice) / 100
		priceStr := fmt.Sprintf("$%.2f", p)
		startTime := time.Unix(status.PeriodStart, 0)
		endTime := time.Unix(status.PeriodEnd, 0)
		startTimeStr := startTime.Format("Mon Jan 2 15:04 2006")
		endTimeStr := endTime.Format("Mon Jan 2 15:04 2006")
		r = fmt.Sprintf("%s\n*Recurring Bill Info*:\n-------------\n"+
			"Next bill: %s\n"+
			"Days until next bill: %d\n"+
			"Bill price: %s\n"+
			"Billing schedule: every %d %s\n"+
			"Billing period start: %s\n"+
			"Billing period end: %s\n",
			r, status.NextBillHuman, status.DaysUntilDue, priceStr,
			status.BillingIntervalCount, status.BillingInterval, startTimeStr, endTimeStr)
	}
	return r, nil
}

func HandleError(w *util.WebhookEvent, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	//if Debug {
	//	log.Printf(message)
	//}
	slackwrap.RespondError(message, RespondToErrorInChannel)
	/*
		if w != nil {
			util.ReportWebhookFailure(w, message)
		} else {
			log.Fatalf(message)
		}
	*/
	//util.PrintJsonError(message) // message to Slack
}
