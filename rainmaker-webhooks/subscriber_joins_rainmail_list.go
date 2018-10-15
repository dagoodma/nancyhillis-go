package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"bitbucket.org/dagoodma/nancyhillis-go/util"

	"github.com/gosexy/to"
	"github.com/gosexy/yaml"
)

var Debug = true // Show/hide debug output
//var WebhookIsSilent = false // don't print anything since we return JSON
var UserAgent = "Nancy Hillis Studio Webhook Backend (wbhk97.nancyhillis.com)"

// Subscriber joins rain mail list input data
type InputData struct {
	List  string `json:"list"`
	Id    string `json:"id"`
	Email string `json:"email"`
}

// Zapier webhook output data
type ZapOutputData struct {
	Status    string `json:"status"`
	Attempt   string `json:"attempt"`
	Id        string `json:"id"`
	RequestId string `json:"request_id"`
}

/* Rainmail -> Zapier settings */
var SecretsFilePath = "/var/webhook/secrets/rainmaker_secrets.yml"
var ZapierUrlParentKey = "RAIN_MAIL_LIST_ZAPIER_URL"

var ListsWithNoSlackAlert = []string{
	"The Artists Journey Resource Library",
}

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
	if Debug {
		util.RecordWebhookStarted(w)
	}
	log.Println("Entire headers: " + string(header))
	log.Println("Entire payload: " + string(data))

	// Unmarshal the input data
	m := InputData{}
	err := json.Unmarshal(data, &m)
	//m := make(map[string]string)
	//err := json.Unmarshal(data, &m)
	if err != nil {
		HandleError(w, "Error while parsing input data for '%s'. %v", data, err)
		return
	}

	// Get and validate the fields (customer_id)
	//customerId, ok := m["customer_id"]
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

	// Report to slack
	// If not in list
	for _, el := range ListsWithNoSlackAlert {
		if el == listName || strings.Replace(listName, "\\", "", -1) == el {
			w.Options.WantSlackSuccessAlert = false
		}
	}
	message := fmt.Sprintf("Subscriber \"%s\" (%s) joined a list \"%s\" and was forwarded to a Zapier webhook: %s",
		email, id, listName, url)
	util.ReportWebhookSuccess(w, message)
	return
}

func ForwardZapierWebhook(url string, jsonData []byte) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		//log.Fatalln(err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		//log.Fatalln(err)
		return nil, err
	}

	defer resp.Body.Close()

	//log.Println("response Status:", resp.Status)
	//log.Println("response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//log.Fatalln(err)
		return nil, err
	}
	//log.Println("response Body:", string(body))

	return body, nil
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
