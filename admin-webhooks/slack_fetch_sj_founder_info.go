package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	"bitbucket.org/dagoodma/nancyhillis-go/slackwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

// This is for the Slack slash command: /student <email address>

var RespondToErrorInChannel = true
var RespondToMessageInChannel = true

var Debug = false // supress extra messages if false

//var LoggedInUserSecretKey = "RBxEi2rt4Skd4TgKytdusBbdp4A4wtbvH"

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

	// Find founder by email in membermouse
	m, err := membermouse.GetMemberByEmail(email)
	if err != nil {
		HandleError(w, "Failed to find founder with email \"%s\". %v", email, err.Error())
		return
	}

	// Get their member status info
	status, err := m.GetStatus()
	if err != nil {
		HandleError(w, "Failed fetching member status. %v", err.Error())
		return
	}

	// Return result
	msg := fmt.Sprintf("Found founder \"%s\".", email)
	a, err := CreateFounderInfoAttachment(m, status)
	if err != nil {
		HandleError(w, "%v", err)
		return
	}
	slackwrap.RespondMessage(msg, a, RespondToMessageInChannel)
	return
}

func CreateFounderInfoAttachment(member *membermouse.Member, status *membermouse.MemberStatus) (string, error) {
	name := fmt.Sprintf("%s %s", status.FirstName, status.LastName)
	//startTime := time.Unix(status.Start, 0)
	//createdTime := time.Unix(status.Created, 0)
	//startTimeStr := startTime.Format("Mon Jan 2 15:04 2006")
	//createdTimeStr := createdTime.Format("Mon Jan 2 15:04 2006")
	r := fmt.Sprintf("Name: %s\n"+
		"Email: %s\n"+
		"Membermouse ID: %d\n"+
		"Migrated? %t\n"+
		"Comped? %t\n"+
		"Overdue? %t\n"+
		"Status: %s\n"+
		"Phone: %s\n"+
		"Days As Member: %d\n"+
		"Last Login: %s\n"+
		"Last Update: %s\n"+
		"Billing Country: %s\n"+
		"MM Admin User Profile URL: %s\n",
		name, status.Email, status.MemberId, status.IsMigrated, status.IsComped, status.IsOverdue,
		status.StatusHuman, status.Phone, status.DaysAsMember, status.LastLogin, status.LastUpdate,
		status.BillingCountry, member.GetManageMemberUrl())
	return r, nil
}

func HandleError(w *util.WebhookEvent, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	slackwrap.RespondError(message, RespondToErrorInChannel)
}
