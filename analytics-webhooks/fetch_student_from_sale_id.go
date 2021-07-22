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

var Debug = false

// Json object to hold the result
/*
type EmailResult struct {
	Result string `json:"result,string"`
}
*/

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

	// Unmarshal the input data
	m := make(map[string]string)
  err := json.Unmarshal(data, &m)
	if err != nil {
		HandleError("Error while parsing input data for '%s'. %v", data, err)
		return
	}

  // Read the request and validate
  var regexId = regexp.MustCompile(`^\d+$`)
	saleId, ok := m["sale_id"]
	studentEmail := ""
	if ok && len(saleId) > 0 {
    if regexId.FindString(saleId) == "" {
			HandleError("Invalid sale ID provided")
			return
    }
    // Fetch the sale info from Teachable
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
  /*
	result := EmailResult{Result: studentEmail}
	r, err := json.Marshal(result)
	if err != nil {
		HandleError("Failed building student info response. %v", err)
		return
	}
  */

	// Send response to JS client
	fmt.Println(string(r))
	return

	// Return result
	r := make(map[string]interface{})
	r["result"] = studentEmail
	util.PrintJsonObject(r)
	return
}

func HandleError(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	util.PrintJsonError(message)
	if Debug {
		log.Printf(message)
	}
}
