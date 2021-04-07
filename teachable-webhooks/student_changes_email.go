package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/dagoodma-go/slackwrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"
	teachable "bitbucket.org/dagoodma/nancyhillis-go/teachable"
	ac "bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
)

// TODO support fetching this and remove hard coded id
var ChangedEmailCustomFieldId = 71

func main() {
	// Get the args
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		log.Fatalf("Not enough arguments, expected %d given: %d",
			3, len(argsWithProg))
	}

    // Local secrets?
    if _, err := os.Stat("slack_secrets.yml"); !os.IsNotExist(err) {
        log.Println("Got local slack secrets.")
        slackwrap.SecretsFilePath = "slack_secrets.yml"
    }
    //slackwrap.DEBUG = false
    if _, err := os.Stat("ac_secrets.yml"); !os.IsNotExist(err) {
        log.Println("Got local AC secrets.")
        ac.SecretsFilePath = "ac_secrets.yml"
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
        util.ReportWebhookFailure(w, fmt.Sprintf("Failed unmarshaling header: %s", err))
		return
	}
	err = teachable.EnsureValidWebhook(h, data)
	if err != nil {
        util.ReportWebhookFailure(w, fmt.Sprintf("Failed validating webhook: %s", err))
		return
	}

	// Unmarshall the message
	m := &teachable.StudentUpdated{}
	err = json.Unmarshal(data, &m)
	if err != nil {
        util.ReportWebhookFailure(w, fmt.Sprintf("Failed unmarshaling update profile: %s",
            err))
		return
	}

	// Grab the data
	//_ := m.Object.Email
	name := m.Object.Name
	//id := m.Object.Id


	// Check for updated email
	oldEmail := m.Object.OldEmail
	newEmail := m.Object.NewEmail
	if m.Object.EmailUpdated {

        // Propagate changes (email) through to system
        // - Active Campaign
        c1, err := ac.GetContactByEmail(oldEmail)
        if err != nil {
            util.ReportWebhookFailure(w, fmt.Sprintf("Failed to update '%s' email address to '%s': Could not find old email in AC",
                    oldEmail, newEmail))
            return
        }
        _, err = ac.GetContactByEmail(newEmail)
        needMerge := false
        if err == nil {
            needMerge = true
            // Found them in AC, can't update their email automatically
            // TODO add them to the automation and set the field
            err = ac.UpdateContactCustomField(c1.Id, ChangedEmailCustomFieldId , newEmail)
            if err != nil {
                util.ReportWebhookFailure(w, fmt.Sprintf("Failed to set custom field to update contact '%s' who needs merge with '%s': %s",
                    oldEmail, newEmail, err))
                return
            }
        }

        var message string
        if !needMerge {
            err = ac.UpdateContactEmail(c1.Id, newEmail)
            if err != nil {
                util.ReportWebhookFailure(w, fmt.Sprintf("Failed to update '%s' (%s) email to '%s': %s",
                    oldEmail, c1.Id, newEmail, err))
                return
            }
            message = fmt.Sprintf("Webhook updated contact '%s' (%s) email from '%s' to: %s",
            name, c1.Id, oldEmail, newEmail)
        } else {
            message = fmt.Sprintf("Webhook found conflict for contact (%s) email" +
                " who changed from '%s' to '%s'. See email notification for instructions.",
                c1.Id, oldEmail, newEmail)
        }

        err = ac.AddNoteToContact(c1.Id, message)
        if err != nil {
            log.Printf("Failed to add note to contact (%s): %s", c1.Id, err)
        }
        log.Println(message)
        util.ReportWebhookSuccess(w, message)
        // - GSheet
        // - Zendesk (TODO: add zendesk api support)

		// Notify slack 
        return
	} // if m.Object.EmailUpdated
    log.Println("Skipped message because of no changed email address.")
}
