package stripewrap

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"encoding/json"
	"github.com/stripe/stripe-go"
	"github.com/stripe/stripe-go/card"
	"github.com/stripe/stripe-go/charge"
	"github.com/stripe/stripe-go/customer"
	"github.com/stripe/stripe-go/sub"

	"github.com/gosexy/to"
	"github.com/gosexy/yaml"
)

/* Slack settings */
var SecretsFilePath = "/var/webhook/secrets/stripe_secrets.yml"

func GetApiKey() (string, string) {
	secrets := yaml.New()
	secrets, err := yaml.Open(SecretsFilePath)
	if err != nil {
		log.Fatalf("Could not open YAML secrets file: %s", err.Error())
	}

	publicKey := to.String(secrets.Get("PUBLIC_KEY"))
	secretKey := to.String(secrets.Get("SECRET_KEY"))

	return publicKey, secretKey
}

func SetApiKey() {
	_, secretKey := GetApiKey()
	stripe.Key = secretKey
}

func SubscriptionIdLooksValid(subId string) bool {
	if len(subId) < 1 {
		return false
	}
	if !strings.HasPrefix(subId, "sub_") {
		return false
	}
	return true
}

func CustomerIdLooksValid(customerId string) bool {
	if len(customerId) < 1 {
		return false
	}
	if !strings.HasPrefix(customerId, "cus_") {
		return false
	}
	return true
}

func TokenIdLooksValid(tokenId string) bool {
	if len(tokenId) < 1 {
		return false
	}
	if !strings.HasPrefix(tokenId, "tok_") {
		return false
	}
	return true
}

func GetCustomerList() *stripe.CustomerList {
	SetApiKey()
	var p = stripe.CustomerListParams{}
	i := customer.List(&p)
	var idx = 1
	for i.Next() {
		//$resource$ := i.$Resource$()
		log.Printf("%d: %s\n", idx, i.Customer().Email)
		idx = idx + 1
	}
	return nil
}

func GetCustomer(customerId string) (*stripe.Customer, error) {
	SetApiKey()
	stripe.LogLevel = 0             // don't print to log
	var p = stripe.CustomerParams{} // no params
	c, err := customer.Get(customerId, &p)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func UnmarshallErrorResponse(response []byte) (*stripe.Error, error) {
	var m stripe.Error
	err := json.Unmarshal(response, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func GetCard(cardId string, customerId string) (*stripe.Card, error) {
	SetApiKey()
	stripe.LogLevel = 0                              // don't print to log
	var p = stripe.CardParams{Customer: &customerId} // need account, customer, or recipient
	c, err := card.Get(cardId, &p)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func GetCanceledSubList(customerId string) *sub.Iter {
	SetApiKey()
	stripe.LogLevel = 0 // don't print to log
	var p = stripe.SubscriptionListParams{
		Customer: customerId,
		Status:   string(stripe.SubscriptionStatusCanceled),
	}
	l := sub.List(&p)

	return l
}

func GetChargeList(customerId string) *charge.Iter {
	SetApiKey()
	stripe.LogLevel = 0 // don't print to log
	var p = stripe.ChargeListParams{Customer: &customerId}
	l := charge.List(&p)
	//return l
	/*
		var idx = 1
		for l.Next() {
			//$resource$ := l.$Resource$()
			log.Printf("%d: %s\n", idx, l.Charge())
			idx = idx + 1
		}
	*/
	return l
}

func GetLastChargeWithPrefix(customerId string, descriptionPrefix string) (*stripe.Charge, error) {
	SetApiKey()
	stripe.LogLevel = 0 // don't print to log
	l := GetChargeList(customerId)
	var c *stripe.Charge = nil
	var idx = 1
	var cnt = 0
	for l.Next() {
		c2 := l.Charge()
		if strings.HasPrefix(c2.Description, descriptionPrefix) {
			cnt = cnt + 1
			// Is it newer?
			if c == nil || time.Unix(c2.Created, 0).After(time.Unix(c.Created, 0)) {
				c = c2
			}
		}
		//log.Printf("%d: %s\n", idx, l.Charge())
		idx = idx + 1
	}
	if cnt < 1 || c == nil {
		msg := fmt.Sprintf("No charges starting with: %s", descriptionPrefix)
		return nil, errors.New(msg)
	}
	return c, nil
}

func GetLastCanceledSubWithPrefix(customerId string, idPrefix string) (*stripe.Subscription, error) {
	SetApiKey()
	stripe.LogLevel = 0 // don't print to log
	l := GetCanceledSubList(customerId)
	var s *stripe.Subscription = nil
	var idx = 1
	var cnt = 0
	for l.Next() {
		s2 := l.Subscription()
		p := s2.Plan
		if strings.HasPrefix(p.ID, idPrefix) {
			cnt = cnt + 1
			// Is it newer?
			if s == nil || time.Unix(s2.Created, 0).After(time.Unix(s.Created, 0)) {
				s = s2
			}
		}
		//log.Printf("%d: %s\n", idx, l.Charge())
		idx = idx + 1
	}
	if cnt < 1 || s == nil {
		msg := fmt.Sprintf("No canceled subscriptions starting with: %s", idPrefix)
		return nil, errors.New(msg)
	}
	return s, nil
}

func SaveNewDefaultCard(customerId string, tokenId string) error {
	SetApiKey()
	stripe.LogLevel = 0 // don't print to log
	customerParams := &stripe.CustomerParams{}
	customerParams.SetSource(tokenId)
	_, err := customer.Update(customerId, customerParams)
	if err != nil {
		return err
	}
	return nil
}

func CancelSubscription(subId string, atPeriodEnd bool) error {
	SetApiKey()
	stripe.LogLevel = 0 // don't print to log
	subParams := &stripe.SubscriptionCancelParams{
		AtPeriodEnd: &atPeriodEnd,
	}
	_, err := sub.Cancel(subId, subParams)
	if err != nil {
		return err
	}
	return nil
}
