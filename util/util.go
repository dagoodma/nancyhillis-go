package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"bitbucket.org/dagoodma/nancyhillis-go/slackwrap"
)

/* Util settings */
var SlackChannel = "#teachable-alerts"
var SlackNotifyErrorPrefix = "`Error:` "

// Data types
type WebhookEvent struct {
	Name   string
	Header []byte
	Data   []byte
}

// Functions
func NewWebhookEvent(programName string, header []byte, data []byte) *WebhookEvent {
	w := new(WebhookEvent)
	w.Name = GetWebhookName(programName)
	w.Header = header
	w.Data = data
	return w
}

func GetWebhookName(programName string) string {
	s := strings.Split(programName, "/")
	var webhookName string
	if len(s) > 1 {
		webhookName = s[len(s)-1]
	} else {
		webhookName = programName
	}
	return webhookName
}

// Report failure to slack channel and email
func ReportWebhookFailure(w *WebhookEvent, errorMessage string) {
	log.Printf("%s webhook failed: %s\n", w.Name, errorMessage)
	message := fmt.Sprintf("Webhook *%s* failed: %s", w.Name, errorMessage)
	slackwrap.PostMessage(SlackChannel, SlackNotifyErrorPrefix+message)
	//email.SendMessage(EmailNotifyErrorTo, EmailNotifyErrorSubject, message)
}

// Report success to slack channel
func ReportWebhookSuccess(w *WebhookEvent, successMessage string) {
	log.Printf("%s webhook ran successfully: %s\n", w.Name, successMessage)
	message := fmt.Sprintf("Webhook *%s* ran successfully: %s", w.Name, successMessage)
	slackwrap.PostMessage(SlackChannel, message)
}

// Report success to slack channel
func ReportSlackWebhookSuccess(w *WebhookEvent, successMessage string) {
	message := fmt.Sprintf("Webhook *%s* ran successfully: %s", w.Name, successMessage)
	slackwrap.PostMessage(SlackChannel, message)
}

func RecordWebhookStarted(w *WebhookEvent) {
	log.Printf("Running webhook \"%s\"\n\tHeader: %s\n\tData: %s\n", w.Name, w.Header, w.Data)
}

// Report failure to slack channel and email
func ReportServiceFailure(serviceName string, errorMessage string) {
	log.Printf("%s service failed: %s\n", serviceName, errorMessage)
	message := fmt.Sprintf("Backend service *%s* failed: %s", serviceName, errorMessage)
	slackwrap.PostMessage(SlackChannel, SlackNotifyErrorPrefix+message)
}

// These are for services (command line) instead of webhooks with headers and data
// Report success to slack channel
func ReportServiceSuccess(serviceName string, successMessage string) {
	log.Printf("%s service ran successfully: %s\n", serviceName, successMessage)
	message := fmt.Sprintf("Backend service *%s* ran successfully: %s", serviceName, successMessage)
	slackwrap.PostMessage(SlackChannel, message)
}

func RecordServiceStarted(serviceName string, header []byte, data []byte) {
	log.Printf("Running service \"%s\"\n\tHeader: %s\n\tData: %s\n", serviceName, header, data)
}

func GetFirstAndLastName(fullName string) (string, string) {
	nameParts := strings.Fields(fullName)
	if len(nameParts) < 1 {
		return "", ""
	}
	firstName := nameParts[0]
	var buffer bytes.Buffer

	for i := 1; i < len(nameParts); i++ {
		if i > 1 {
			buffer.WriteString(" ")
		}
		buffer.WriteString(nameParts[i])
	}
	lastName := buffer.String()

	return firstName, lastName
}

func EmailLooksValid(email string) bool {
	if len(email) < 1 {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	user, domain := parts[0], parts[1]
	if len(user) < 1 || len(domain) < 1 {
		return false
	}

	// Ensure domain looks valid
	domainParts := strings.Split(domain, ".")
	if len(domainParts) < 2 {
		return false
	}
	host, ending := domainParts[0], domainParts[1]
	if len(host) < 1 || len(ending) < 1 {
		return false
	}
	return true
}

func PrintJsonData(data []byte) error {
	var parsed interface{}

	err := json.Unmarshal(data, &parsed)
	if err != nil {
		return err
		//log.Fatalln(err)
	}
	b, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return err
		//log.Fatalln(err)
	}
	os.Stdout.Write(b)
	fmt.Printf("\n")
	return nil
}

func UnmarshallJsonData(data []byte, m map[string]interface{}) error {
	if err := json.Unmarshal(data, &m); err != nil {
		log.Printf("Error: %s", err)
		return err
	}
	return nil
}

func GetJsonByteString(o map[string]interface{}) ([]byte, error) {
	jsonString, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return jsonString, nil
}

func PrintJsonObject(o map[string]interface{}) error {
	jsonString, err := GetJsonByteString(o)
	if err != nil {
		log.Printf("Error: %s", err)
		return err
	}
	fmt.Printf("%s", jsonString)
	return nil
}

func PrintJsonError(message string) error {
	m := make(map[string]interface{})
	m["error"] = message
	err := PrintJsonObject(m)
	if err != nil {
		return err
	}
	return nil
}

func StringSliceContains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func RoundDown(val float64) int {
	if val < 0 {
		return int(val - 1.0)
	}
	return int(val)
}

func RoundUp(val float64) int {
	if val > 0 {
		return int(val + 1.0)
	}
	return int(val)
}
