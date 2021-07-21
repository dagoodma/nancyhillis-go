package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
  "regexp"

	"bitbucket.org/dagoodma/dagoodma-go/slackwrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"
	teachable "bitbucket.org/dagoodma/nancyhillis-go/teachable"
)

var Debug = true

// Json object to hold the result
type EmailResult struct {
	Result string `json:"result,string"`
}

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
  if _, err := os.Stat("teachable_secrets.yml"); !os.IsNotExist(err) {
      log.Println("Got local Teachable secrets.")
      teachable.SecretsFilePath = "teachable_secrets.yml"
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

	// Unmarshal the input data
	m := make(map[string]string)
	err = json.Unmarshal(data, &m)
	if err != nil {
		HandleError("Error while parsing input data for '%s'. %v", data, err)
		return
	}

  // Read the request
  var regexId = regexp.MustCompile(`^\d+$`)
	saleId, ok := m["sale_id"]
	studentEmail := ""
	if ok && len(saleId) > 0 {
    if regexId.FindString(saleId) == "" {
			HandleError("Invalid sale ID provided")
			return
    }
    sale, err := teachable.GetSaleById(saleId)
    if err != nil {
			HandleError("Error retrieving sale with ID %s: %s\n", saleId, err)
      return
    }
    studentEmail = sale.User.Email
  } else {
			HandleError("No sale ID provided")
			return
  }

	// Marshall data and return result
	result := EmailResult{Result: studentEmail}
	r, err := json.Marshal(result)
	if err != nil {
		HandleError("Failed building student info response. %v", err)
		return
	}

	// Send response to JS client
	fmt.Println(string(r))
	return
}

func HandleError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	util.PrintJsonError(message)
	if Debug {
		log.Printf(message)
	}
}
