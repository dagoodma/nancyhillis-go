package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var Debug = true // Show/hide debug output

// Student identifies on website (Rainmaker or Teachable) input data
type InputData struct {
	WebsiteName string `json:"site_name"`
	WebsiteUrl  string `json:"site_url"`
	Email       string `json:"email"`
	UserId      string `json:"id"`
	AnalyticsId string `json:"uaid"`
	ClientId    string `json:"cid"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Device      string `json:"device"`
}

type OutputData struct {
	Status       string `json:"status"`
	GlobalUserId string `json:"uid"` // For GA user ID
}

// Valid website names
var ValidSiteNames = []string{
	"rainmaker",
	"teachable",
}

// Site names to custom field name in AC for site user ID
var SiteIdFieldName = map[string]string{
	"rainmaker": "rid",
	"teachable": "tid",
}

// Note that we will be using our own customer error handler: HandleError()
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		HandleError("No input data provided")
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
	m := InputData{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		HandleError("Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Get the fields
	// Website name
	siteName := m.WebsiteName
	if len(siteName) < 1 {
		HandleError(w, "No website name provided")
		return
	}
	if !StringSliceContains(ValidSiteNames, siteName) {
		HandleError(w, "Invalid website name given: %s", siteName)
		return
	}
	// Website URL
	siteUrl := m.WebsiteUrl
	if !strings.HasPrefix(siteUrl, "http") {
		HandleError(w, "Invalid site url: %s", siteUrl)
		return
	}
	// Email
	email := m.Email
	if !util.EmailLooksValid(email) {
		HandleError(w, "Invalid student email: %s", email)
		return
	}
	// Site User ID
	userId := m.UserId
	if len(userId) < 1 {
		HandleError(w, "No site user ID provided")
		return
	}
	// GA Client ID
	id := m.Id
	if len(id) < 1 {
		HandleError(w, "No site user ID provided")
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
