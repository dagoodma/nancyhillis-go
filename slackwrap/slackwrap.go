package slackwrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gosexy/to"
	"github.com/gosexy/yaml"
	"github.com/nlopes/slack"
)

/* Slack settings */
var SecretsFilePath = "/var/webhook/secrets/slack_secrets.yml"

// Json message for slash command request
type SlackCommandRequest struct {
	ChannelId   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Command     string `json:"command"`
	ResponseUrl string `json:"response_url"`
	TeamDomain  string `json:"team_domain"`
	TeamId      string `json:"team_id"`
	Text        string `json:"text"`
	Token       string `json:"token"`
	TriggerId   string `json:"trigger_id"`
	UserId      string `json:"user_id"`
	UserName    string `json:"user_name"`
}

// Slash command response
type SlackResponse struct {
	Text        string                    `json:"text"`
	Attachments []SlackResponseAttachment `json:"attachments"`
}

type SlackResponseAttachment struct {
	Text string `json:"text"`
}

// Slash command error response
type SlackErrorResponse struct {
	Text         string `json:"text"`
	ResponseType string `json:"response_type"`
}

// Returns the slack channel name for alerts
func GetAlertsChannel() string {
	secrets := yaml.New()
	secrets, err := yaml.Open(SecretsFilePath)
	if err != nil {
		log.Fatalf("Could not open YAML secrets file: %s", err.Error())
	}

	alertsChannel := to.String(secrets.Get("ALERTS_CHANNEL"))

	return alertsChannel
}

// Returns the secret Slack API token
func GetApiToken() string {
	secrets := yaml.New()
	secrets, err := yaml.Open(SecretsFilePath)
	if err != nil {
		log.Fatalf("Could not open YAML secrets file: %s", err.Error())
	}

	apiToken := to.String(secrets.Get("API_TOKEN"))

	return apiToken
}

// For validating slash command  request tokens
func GetCommandTokens() map[string]string {
	secrets := yaml.New()
	secrets, err := yaml.Open(SecretsFilePath)
	if err != nil {
		log.Fatalf("Could not open YAML secrets file: %s", err.Error())
	}

	var l = 0
	commandTokens := make(map[string]string)
	c := secrets.Get("COMMAND_TOKENS")
	if c == nil {
		log.Fatalf("Could not find \"COMMAND_TOKENS\" in YAML secrets file: %s", SecretsFilePath)
	}

	c2, ok := c.(map[interface{}]interface{})
	if !ok || c2 == nil {
		log.Fatalf("Invalid \"COMMAND_TOKENS\" in YAML secrets file: %s", SecretsFilePath)
	}
	for k, v := range c2 {
		kStr := to.String(k)
		vStr := to.String(v)
		commandTokens[kStr] = vStr
		l = l + 1
	}
	if l < 1 {
		log.Fatalf("Entry for \"COMMAND_TOKENS\" was empty in YAML secrets file: %s", SecretsFilePath)
	}

	return commandTokens
}

// Post an in-channel message as the test API bot without printing to log
func PostMessageSilent(channelId string, message string) (string, error) {
	apiToken := GetApiToken()

	api := slack.New(apiToken)

	params := slack.PostMessageParameters{
		Markdown:  true,
		LinkNames: 1,
		Parse:     "full",
	}
	channelPostedId, timestamp, err := api.PostMessage(channelId, message, params)
	_ = channelPostedId
	if err != nil {
		log.Fatalln(err)
	}
	return timestamp, err
}

// Post an in-channel message as the test API bot
func PostMessage(channelId string, message string) {
	// TODO handle errors
	timestamp, _ := PostMessageSilent(channelId, message)
	//log.Printf("Slack message successfully sent to channel %s at %s\n", channelID, timestamp)
	log.Printf("Slack message successfully sent to channel %s at %s\n", channelId, timestamp)
}

// When responding to a slash command
func RespondMessage(text string, attachment string, inChannel bool) {
	// Marshall data and return result
	a := SlackResponseAttachment{Text: attachment}
	response := SlackResponse{Text: text, Attachments: []SlackResponseAttachment{a}}
	r, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Failed building Slack response. %v\n", err)
		return
	}

	// Send response as HTTP response
	fmt.Println(string(r))
	return
}

// When responding to a slash command
func RespondMessageToUrl(text string, attachment string, responseUrl string) (*http.Response, error) {
	// Marshall data and return result
	a := SlackResponseAttachment{Text: attachment}
	response := SlackResponse{Text: text, Attachments: []SlackResponseAttachment{a}}
	jsonStr, err := json.Marshal(response)
	if err != nil {
		msg := fmt.Sprintf("Failed building Slack response for url \"%s\". %v", responseUrl, err)
		return nil, errors.New(msg)
	}

	// Build a request
	client := &http.Client{}
	//var jsonStr = []byte(`{"events":[{"email":"dagoodma@gmail.com","action":"TEst event"}]}`)
	req, err := http.NewRequest("POST", responseUrl, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	/*
		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Api-Token", apiToken)
	*/

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
	_ = body

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Got bad response status '%s', expected: %s", resp.StatusCode, 200)
		return nil, errors.New(msg)
	}
	//log.Println("response Body:", string(body))
	//data := []byte(body)

	// Unmarshall the message
	/*
		l := &ListContacts{}
		err = json.Unmarshal(data, &l)
		if err != nil {
			msg := fmt.Sprintf("Failed to unmarshal GetContacts response data: %s", err)
			return nil, errors.New(msg)
		}
	*/

	//log.Printf("Got %d results in contact search.", l.Metadata.Total)
	return resp, nil
}

// When responding to a slash command with text only (no attachment)
func RespondMessageTextOnly(text string) {
	response := SlackResponse{Text: text}
	r, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Failed building Slack acknowledgement response. %v\n", err)
		return
	}

	// Send response as HTTP response
	fmt.Println(string(r))
	return
}

// When responding to a slash command error
func RespondError(reason string, inChannel bool) {
	// Marshall data and return result
	txt := fmt.Sprintf("Sorry, that didn't work. %s", reason)
	response := SlackErrorResponse{Text: txt, ResponseType: "ephemeral"}
	r, err := json.Marshal(response)
	if err != nil {
		log.Fatalf("Failed building Slack error response. %v\n", err)
		return
	}

	// Send response as HTTP response
	fmt.Println(string(r))
	return
}

// Compares the request token key to the secret entry for the command
func ValidateCommandRequest(name string, requestToken string) error {
	commandTokens := GetCommandTokens()
	t, ok := commandTokens[name]
	if !ok || len(t) < 1 {
		msg := fmt.Sprintf("No such command with token: %s", name)
		return errors.New(msg)
	}
	if strings.Compare(requestToken, t) != 0 {
		msg := fmt.Sprintf("Command token did not match: %s", name)
		return errors.New(msg)
	}
	return nil
}
