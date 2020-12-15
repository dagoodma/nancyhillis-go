package main

import (
	//"errors"
	"flag"
	"fmt"
	"log"
	//"strings"
	//"time"

	"github.com/davecgh/go-spew/spew"

	"bitbucket.org/dagoodma/nancyhillis-go/gsheetwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	"bitbucket.org/dagoodma/nancyhillis-go/nancyhillis"
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	//"bitbucket.org/dagoodma/nancyhillis-go/util"
	//"github.com/stripe/stripe-go"
)

var SjMmTransactionsSpreadsheetId = "1sra-kv8f2ZVLmO9QK3MCfE0IIDIIWm61t2HQcMTdCf8"
var SjMmTransactionsSpreadsheetEmailCol = 6

// This is just for looking up students in Stripe
func main() {
	var verboseFlag bool
	flag.BoolVar(&verboseFlag, "v", false, "Verbose dumping of payment info")
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		log.Fatal("No email address provided")
		return
	}
	email := string(args[0])

	// Check in Stripe
	log.Printf("Looking for \"%s\" in Stripe...\n", email)
	stripeId, err := nancyhillis.GetSjStripeIdByEmail(email)
	if err == nil && len(stripeId) > 0 {
		s, err := stripewrap.GetCustomer(stripeId)
		if err == nil && s != nil {
			l := stripewrap.GetChargeList(stripeId)
			if l != nil {
				if !verboseFlag {
					log.Printf("Customer: %v\n", s)
				} else {
					log.Println("Customer: ")
					spew.Dump(s)
					fmt.Println("")
				}
				// Print list of charges
				log.Println("Transactions: ")
				var idx = 1
				for l.Next() {
					c2 := l.Charge()
					if !verboseFlag {
						log.Printf("%d: %s\nRaw data: %v\n", idx, c2.Description, c2)
					} else {
						spew.Dump(c2)
					}
					idx = idx + 1
				}
				fmt.Println("\n")
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
	if err != nil || m == nil {
		log.Printf("Failed to find student. %s\n", err)
	} else {
		if !verboseFlag {
			log.Printf("Member: %v\n", m)
		} else {
			log.Println("Member: ")
			spew.Dump(m)
			fmt.Println("")
		}

		// TODO fix this code
		rows, err := gsheetwrap.SearchForAllRowsWithValue(SjMmTransactionsSpreadsheetId, SjMmTransactionsSpreadsheetEmailCol, email)
		if err == nil && rows != nil && len(rows) > 0 {
			if !verboseFlag {
				log.Printf("Transactions: %v\n\n", rows)
			} else {
				log.Println("Transactions:")
				spew.Dump(rows)
				fmt.Println("")
			}
		} else {
			log.Printf("Failed to find any transactions for: %s\n", email)
		}
	}

	return
}
