package main

import (
	//"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	//"gsheetwrap"
	"bitbucket.org/dagoodma/dagoodma-go/stripewrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"
	"bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
)

var Debug = false

// Json object to hold the result
type StatusResult struct {
	Result *studiojourney.StudentStatus `json:"result,string"`
}

// Note that we will be using our own customer error handler: HandleError()
// Because this is a AJAX webhook that must always return a valid JSON response
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 3 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		HandleError("No customer ID provided")
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
	m := make(map[string]string)
	err := json.Unmarshal(data, &m)
	if err != nil {
		HandleError("Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Get and validate the fields (customer_id)
	// First try email address
	email, ok := m["email"]
	customerId := ""
	if ok && len(email) > 0 {
		if !util.EmailLooksValid(email) {
			HandleError("Invalid email address provided")
			return
		}
		// Get customer ID from email address
		log.Printf("Here with %s\n", email)
		c, err := stripewrap.GetCustomerByEmail(email)
		if err != nil || c == nil {
			HandleError("Could not find customer by email address. %v", err)
			return
		}
		customerId = c.ID
	} else {
		customerId, ok = m["customer_id"]
		if !ok || len(customerId) < 1 {
			HandleError("No customer ID or email provided")
			return
		}
	}
	if len(customerId) < 1 {
		HandleError("Could not find customer ID")
		return
	}

	status, err := studiojourney.GetAccountStatus(customerId)
	if err != nil {
		HandleError(err.Error())
		return
	}

	// Marshall data and return result
	result := StatusResult{Result: status}
	r, err := json.Marshal(result)
	if err != nil {
		HandleError("Failed building status response. %v", err)
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
