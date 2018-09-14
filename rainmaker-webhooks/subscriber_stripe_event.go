package main

import (
	//"errors"
	"fmt"
	"log"
	"os"
	//"strings"

	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
	//"github.com/stripe/stripe-go"
)

var Debug = true // Show/hide debug output
//var WebhookIsSilent = false // don't print anything since we return JSON
var UserAgent = "Nancy Hillis Studio Webhook Backend (wbhk97.nancyhillis.com)"

// These are what we use mapped to human strings:
var EventDescriptions = map[string]string{
	"charge.succeeded": "billed",
	"charge.refunded":  "refunded",
	"charge.failed":    "failed to be billed",
	"customer.created": "created",
}

/* Rainmail -> Zapier settings */
//var SecretsFilePath = "/var/webhook/secrets/rainmaker_secrets.yml"
//var ZapierUrlParentKey = "RAIN_MAIL_LIST_ZAPIER_URL"

/*
func GetUrlSecret(listName string) (string, error) {
	secrets := yaml.New()
	secrets, err := yaml.Open(SecretsFilePath)
	if err != nil {
		return "", err
	}

	url := to.String(secrets.Get(ZapierUrlParentKey, listName))
	if len(url) < 1 {
		// Remove slashes and try again
		listName = strings.Replace(listName, "\\", "", -1)
		url = to.String(secrets.Get(ZapierUrlParentKey, listName))

		if len(url) < 1 {
			msg := fmt.Sprintf("No url found for list: %s", listName)
			return "", errors.New(msg)
		}
	}

	return url, nil
}
*/

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
	//w.Options.IsSilent = WebhookIsSilent
	if Debug {
		util.RecordWebhookStarted(w)
	}
	log.Println("Entire headers: " + string(header))
	log.Println("Entire payload: " + string(data))

	// Unmarshal the input data
	event, err := stripewrap.UnmarshallWebhookEvent(data)
	if err != nil {
		HandleError(w, "Error while parsing input data for Stripe webhook event '%s'. %v", data, err)
		return
	}
	_ = event

	// Get and validate the fields (customer_id)
	//customerId, ok := m["customer_id"]
	/*
		listName := m.List
		if len(listName) < 1 {
			HandleError(w, "No email list name provided")
			return
		}
		email := m.Email
		if !util.EmailLooksValid(email) {
			HandleError(w, "Invalid subscriber email: %s", email)
			return
		}
		id := m.Id
		if len(id) < 1 {
			HandleError(w, "No subscriber ID provided")
			return
		}

		// Get Zapier webhook url secrets
		url, err := GetUrlSecret(listName)
		if err != nil {
			HandleError(w, "Could not handle new subscriber \"%s\": %s", email, err.Error())
			return
		}

		if Debug {
			log.Printf("Forwarding to Zapier webhook: %s\n\t%v\n", url, m)
		}
		rawData, err := json.Marshal(m)
		if err != nil {
			HandleError(w, "Failed marshalling JSON data for webhook: %s", err.Error())
			return
		}
		r, err := ForwardZapierWebhook(url, rawData)
		if err != nil {
			HandleError(w, "Failed forwarding JSON data webhook url: %s. %v", url, err.Error())
			return
		}
		log.Printf("Here with resp: %s\n", r)

		// Unmarshal the output data (response)
		m2 := ZapOutputData{}
		err = json.Unmarshal(r, &m2)
		if err != nil {
			HandleError(w, "Error while parsing output data for (%s) '%s'. %v", url, r, err)
			return
		}
		if m2.Status != "success" {
			HandleError(w, "Expected success forwarding data to webhook url (%s), got: %s. %s", url, m2.Status, r)
			return
		}
	*/

	// Report to slack
	var testStr string
	if !event.Livemode {
		testStr = "(TEST) "
	}
	eventStr := EventDescriptions[event.Type]
	var reasonStr, email, name string
	id := event.GetObjectValue("customer")
	if event.Type == "charge.succeeded" || event.Type == "charge.refunded" ||
		event.Type == "charge.failed" {
		reasonStr = fmt.Sprintf(" for: %s", event.Data.Object["description"])
		email = event.GetObjectValue("receipt_email")
		name = event.GetObjectValue("source", "name")
	} else if event.Type == "customer.created" {
		email = event.GetObjectValue("email")
	} else {
		HandleError(w, "Unknown Stripe webhook event: %s", event.Type)
		return
	}
	// Build the message for slack
	message := fmt.Sprintf("%sSubscriber \"%s\" (name: \"%s\", cus: %s) was %s in Stripe%s",
		testStr, email, name, id, eventStr, reasonStr)
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
