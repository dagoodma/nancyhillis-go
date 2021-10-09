package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
  "regexp"
  "time"

	"bitbucket.org/dagoodma/dagoodma-go/slackwrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"
	"bitbucket.org/dagoodma/dagoodma-go/gsheetwrap"
	//"gopkg.in/Iwark/spreadsheet.v2"
	//"google.golang.org/api/sheets/v4"
)

var Debug = false

var GoogleSheetSleepTime, _ = time.ParseDuration("0.5s")

var ConversionSpreadsheetId = "1Azq9IHETxibYE8rzLK-DqJVSmNP3Oswoycr-V6hAuLc"

// Json object to hold the result
/*
type NotifyResult struct {
	Result bool `json:"result,bool"`
}
*/

func main() {
	// Get the args
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		log.Fatalf("Not enough arguments, expected %d given: %d",
			3, len(argsWithProg))
	}

  // Uncomment for testing:
  //gsheetwrap.SecretsFilePath = "gsheet_client_secrets.json"
  // Local secrets?
  if _, err := os.Stat("slack_secrets.yml"); !os.IsNotExist(err) {
      log.Println("Got local slack secrets.")
      slackwrap.SecretsFilePath = "slack_secrets.yml"
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

	// Unmarshal the header data
	h := make(map[string]string)
  err = json.Unmarshal(header, &h)
	if err != nil {
		HandleError("Error while parsing header for '%s'. %v", header, err)
		return
	}
	ua, ok := h["User-Agent"]

  // Read the request and validate
  var regexKey = regexp.MustCompile(`^[a-zA-Z\d_]*$`)
  var regexPath = regexp.MustCompile(`^\/.+$`)
	key, ok := m["key"]
	if !ok || len(key) <= 0 {
    HandleError("No key provided")
  }
  if regexKey.FindString(key) == "" {
    HandleError("Invalid key provided")
    return
  }

  name, ok := m["name"]
  if !ok && len(name) <= 0 {
    HandleError("No name provided")
    return
  }

  category, ok := m["category"]
  if !ok && len(category) <= 0 {
    HandleError("No category provided")
    return
  }

  path, ok := m["path"]
  if !ok && len(path) <= 0 {
    HandleError("No path provided")
    return
  }
  if regexPath.FindString(path) == "" {
    HandleError("Invalid path provided")
    return
  }

  source, ok := m["source"]
  notes, ok := m["notes"]

  currentTime := time.Now()
  var ts = currentTime.Format("2006-01-02 15:04:05")

  ss, err := gsheetwrap.FetchSpreadsheet(ConversionSpreadsheetId)
  if err != nil {
		HandleError("Failed to fetch sheet. %v", err)
    return
  }
  sheet, err := ss.SheetByIndex(0)
  err = gsheetwrap.AddRowToSheet(sheet, []string{ts, key, category, name, path, source, ua, notes})
  if err != nil {
		HandleError("Error while adding row to spreadsheet. %v", err)
    return
  }

	// Marshall data and return result
  /*
	result := NotifyResult{Result: true}
	r, err := json.Marshal(result)
	if err != nil {
		HandleError("Failed building notify response. %v", err)
		return
	}
	// Send response to JS client
	fmt.Println(string(r))
	return
  */

	// Return result
	r := make(map[string]interface{})
	r["result"] = true
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
