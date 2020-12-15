package main

import (
	//"errors"
	//"fmt"
	"log"
	"os"
	//"strings"
	//"time"
	//"strconv"

	"github.com/davecgh/go-spew/spew"

	//"bitbucket.org/dagoodma/nancyhillis-go/gsheetwrap"
	//"bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	//"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	sj "bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
	//"bitbucket.org/dagoodma/nancyhillis-go/util"
	//"github.com/stripe/stripe-go"
)

var Debug = false

// Student payment data for Studio Journey
type SjStudent struct {
	MmId                   string
	StripeId               string
	Email                  string
	Name                   string
	Phone                  string
	Country                string
	IsFounder              bool
	IsMigratedFounder      bool
	Payments               []SjStudentPayment
	LifeTimeValue          float32
	PaymentCount           int32
	HasActiveSubscription  bool
	ActiveSubscription     *SjStudentSubscription
	ExpectedLifeTimeValue  float32
	RemainingLifeTimeValue float32
	HasPaymentsRemaining   bool
	RemainingPaymentCount  int32
}
type SjStudentSubscription struct {
	Description string
	Amount      float32
	CreatedDate string // not sure in MM
}
type SjStudentPayment struct {
	IsRefund bool
	//Type   string
	Amount      float32
	Date        string
	Description string
}

// This is just for looking up students in Stripe
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 2 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		log.Fatal("No email address provided")
		return
	}

	email := string(argsWithProg[1])

	s, err := sj.GetBillingStatus(email)
	if err != nil {
		log.Fatalf("Failed to fetch billing status for \"%s\". %v", email, err)
	}
	/*
		s := SjStudent{Email: email}

		// Check in Stripe
		log.Printf("Looking for \"%s\" in Stripe...\n", email)
		stripeId, err := studiojourney.GetStripeIdByEmail(email)
		if err == nil && len(stripeId) > 0 {
			s.StripeId = stripeId
			c, err := stripewrap.GetCustomer(stripeId)
			if err == nil && c != nil {
				s.Name = fmt.Sprintf("%s %s",
					strings.TrimSpace(c.Metadata["first_name"]),
					strings.TrimSpace(c.Metadata["last_name"]))
				s.Country = c.Metadata["country"]
				if c.Subscriptions.ListMeta.TotalCount > 0 {
					s.HasActiveSubscription = true
					sub := SjStudentSubscription{}
					foundActiveSub := false
					for _, ss := range c.Subscriptions.Data {
						isActiveSub, _ := stripewrap.IsActiveSubscription(ss)
						if isActiveSub {
							if foundActiveSub {
								log.Println("WARNING: Found more than 1 active subscription!")
							}
							sub.Description = ss.Plan.Nickname
							sub.Amount = float32(ss.Plan.Amount) / 100.00
							sub.CreatedDate = stripewrap.FormatEpochTime(ss.Plan.Created)
							foundActiveSub = true
						}
					}
					s.ActiveSubscription = &sub
				} else {
					s.HasActiveSubscription = false
				}

				l := stripewrap.GetChargeList(stripeId)
				if l != nil {
					if Debug {
						log.Printf("Customer: %v\n", c)
						log.Printf("Charges: %v\n\n", l)
					}
					// Print list of charges
					var idx = 1
					for l.Next() {
						c2 := l.Charge()
						if c2.Paid {
							totalAmount := float32(c2.Amount-c2.AmountRefunded) / 100.00
							p := SjStudentPayment{
								Amount:      totalAmount,
								IsRefund:    totalAmount < 0,
								Date:        stripewrap.FormatEpochTime(c2.Created),
								Description: c2.Description,
							}
							if len(p.Description) < 1 {
								p.Description = c2.StatementDescriptor
							}
							s.LifeTimeValue = s.LifeTimeValue + totalAmount
							if totalAmount > 0 {
								s.PaymentCount = s.PaymentCount + 1
							}
							s.Payments = append(s.Payments, p)
						}
						if Debug {
							log.Printf("%d: %s\nRaw data: %v\n", idx, c2.Description, c2)
						}
						idx = idx + 1
					}
				} else {
					log.Printf("Failed to find any charges for: %s\n", stripeId)
				}
			} else {
				reason := ""
				// Try and parse the Stripe error (to make it less cryptic for user)
				ers := err.Error()
				serr, err2 := stripewrap.UnmarshallErrorResponse([]byte(ers))
				if err2 == nil {
					if serr.HTTPStatusCode == 404 {
						reason = fmt.Sprintf("No such customer with Stripe ID: %s", stripeId)
					}
				}
				log.Printf("Failed retrieving customer data. %s\n", reason)
			}
		} else {
			log.Printf("Failed to find student in Stripe. %s\n", err)
		}

		// Check in membermouse
		log.Printf("Looking for \"%s\" in Membermouse...\n", email)
		m, err := membermouse.GetMemberByEmail(email)
		if err == nil && m != nil {
			s.MmId = fmt.Sprintf("%d", m.MemberId)
			s.IsFounder = true
			// This one might be more accurate
			s.Name = fmt.Sprintf("%s %s",
				strings.TrimSpace(m.FirstName),
				strings.TrimSpace(m.LastName))
			if len(m.Phone) > 0 {
				s.Phone = m.Phone
			}
			s.IsMigratedFounder = m.IsMigrated()
			if m.IsActive() && !m.IsComped() {
				if s.HasActiveSubscription {
					log.Println("WARNING: Found an active subscription in both Stripe and Membermouse!")
				} else {
					s.HasActiveSubscription = true
					sub := SjStudentSubscription{Description: "SJ Founder Membermouse",
						Amount: 29.00, CreatedDate: m.Registered}
					s.ActiveSubscription = &sub
				}
			}

			if Debug {
				log.Printf("Member: %v\n", m)
			}
			//log.Printf("Charges: %v\n\n", l)

			rows, err := gsheetwrap.SearchForAllRowsWithValueInColumn(
				studiojourney.MmTransactionsSpreadsheetId,
				studiojourney.MmTransactionsSpreadsheetEmailCol, email)
			if err == nil && rows != nil && len(rows) > 0 {
				if Debug {
					log.Printf("Transactions: %v\n\n", rows)
				}
				for _, r := range rows {
					isRefund := false
					amount, err := strconv.ParseFloat(r[3].Value, 32)
					if err != nil {
						log.Fatalf("Failed parsing payment amount \"%s\". %v", r[3].Value, err)
					}
					if strings.Contains(strings.ToLower(r[0].Value), "refund") {
						isRefund = true
						amount = -amount
					}
					p := SjStudentPayment{
						Amount:      float32(amount),
						IsRefund:    isRefund,
						Date:        r[1].Value,
						Description: fmt.Sprintf("%s for %s order #%s", r[0].Value, r[9].Value, r[2].Value),
					}
					s.LifeTimeValue = s.LifeTimeValue + float32(amount)
					if amount > 0 {
						s.PaymentCount = s.PaymentCount + 1
					}
					s.Payments = append(s.Payments, p)
				}
			} else {
				log.Printf("Failed to find any transactions for: %s\n", email)
			}
		} else {
			log.Printf("Failed to find student. %s\n", err)
		}

		s.ExpectedLifeTimeValue = s.LifeTimeValue
		s.HasPaymentsRemaining = false
		if s.HasActiveSubscription {
			s.ExpectedLifeTimeValue = 36.00 * 12
			if s.IsFounder {
				s.ExpectedLifeTimeValue = 29.00 * 12
			}
		}
		s.RemainingLifeTimeValue = s.ExpectedLifeTimeValue - s.LifeTimeValue
		if s.HasActiveSubscription && s.RemainingLifeTimeValue > 0 {
			s.HasPaymentsRemaining = true
			s.RemainingPaymentCount = int32(s.RemainingLifeTimeValue / 36.00)
			if s.IsFounder {
				s.RemainingPaymentCount = int32(s.RemainingLifeTimeValue / 29.00)
			}
		}
	*/

	spew.Dump(s)

	return
}
