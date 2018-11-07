package stripewrap

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"bitbucket.org/dagoodma/nancyhillis-go/util"

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

func IsActiveSubscription(sub *stripe.Subscription) (bool, error) {
	if sub.Status == stripe.SubscriptionStatusActive {
		return true, nil
	}
	return false, nil
}

func FormatEpochTime(timestamp int64) string {
	t := time.Unix(timestamp, 0)
	//timeStr := timestamp.Format("Mon Jan 2 15:04 2006")
	//timeStr := t.Format("11/1/2017 16:29:00")
	timeStr := fmt.Sprintf("%02d/%02d/%d %02d:%02d:%02d",
		t.Month(), t.Day(), t.Year(),
		t.Hour(), t.Minute(), t.Second())
	return timeStr
}

func GetCustomerListIterator() *customer.Iter {
	SetApiKey()
	var p = stripe.CustomerListParams{}
	i := customer.List(&p)
	/*
		var idx = 1
		for i.Next() {
			//$resource$ := i.$Resource$()
			log.Printf("%d: %s\n", idx, i.Customer().Email)
			idx = idx + 1
		}
		return nil
	*/
	return i
}

func GetCustomerListIteratorWithParams(listParams map[string]string) *customer.Iter {
	SetApiKey()
	stripe.LogLevel = 0 // don't print to log

	p := &stripe.CustomerListParams{}
	for k, v := range listParams {
		p.Filters.AddFilter(k, "", v)
	}

	i := customer.List(p)
	return i
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

func GetCustomerByEmail(email string) (*stripe.Customer, error) {
	if !util.EmailLooksValid(email) {
		msg := fmt.Sprintf("Invalid email address given: %s", email)
		return nil, errors.New(msg)
	}
	listParams := map[string]string{"email": email}
	i := GetCustomerListIteratorWithParams(listParams)
	cnt := 0
	var c *stripe.Customer = nil
	for i.Next() {
		if cnt > 0 {
			msg := fmt.Sprintf("Found multiple customers with email address: %s", email)
			return nil, errors.New(msg)
		}

		c = i.Customer()
		cnt += 1
	}
	if cnt < 1 {
		msg := fmt.Sprintf("Failed to find customer with email address: %s", email)
		return nil, errors.New(msg)
	}
	return c, nil
}

func SearchCustomersByName(fullName string) ([]*stripe.Customer, error) {
	fullNameClean := util.StandardizeSpaces(fullName)
	if len(fullNameClean) < 1 {
		msg := fmt.Sprintf("Invalid name given: %s", fullName)
		return nil, errors.New(msg)
	}
	i := GetCustomerListIteratorWithParams(map[string]string{"limit": "100"})
	l := []*stripe.Customer{}
	for i.Next() {
		c := i.Customer()
		descriptionClean := util.StandardizeSpaces(c.Description)
		if strings.EqualFold(descriptionClean, fullNameClean) {
			l = append(l, c)
			continue
		}
		if metaName, ok := c.Metadata["contact_name"]; ok {
			metaNameClean := util.StandardizeSpaces(metaName)
			if strings.EqualFold(metaNameClean, fullNameClean) {
				l = append(l, c)
			}
		}
	}
	if len(l) < 1 {
		msg := fmt.Sprintf("Failed to find any customers with name: %s", fullNameClean)
		return nil, errors.New(msg)
	}
	return l, nil
}

func CreateCustomer(email string, description string) (*stripe.Customer, error) {
	c1, err := GetCustomerByEmail(email)
	if c1 != nil {
		msg := fmt.Sprintf("Customer with email \"%s\" already exists: %s",
			email, c1.ID)
		return nil, errors.New(msg)
	}
	//SetApiKey() // dont need this will call above
	//stripe.LogLevel = 0 // don't print to log

	p := &stripe.CustomerParams{
		Description: stripe.String(description),
		Email:       stripe.String(email),
	}

	c2, err := customer.New(p)
	return c2, err
}

func UpdateCustomerMetadata(customerId string, metadata map[string]string) (*stripe.Customer, error) {
	SetApiKey() // dont need this will call above
	//stripe.LogLevel = 0 // don't print to log

	p := &stripe.CustomerParams{}
	for k, v := range metadata {
		p.AddMetadata(k, v)
	}

	c, err := customer.Update(customerId, p)
	return c, err
}

func UnmarshallWebhookEvent(data []byte) (*stripe.Event, error) {
	var e stripe.Event
	var err = json.Unmarshal(data, &e)
	if err != nil {
		return nil, err
	}
	return &e, nil
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
