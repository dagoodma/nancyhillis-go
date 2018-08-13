package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	//"strings"
	//"time"
	//"gsheetwrap"
	//"bitbucket.org/dagoodma/nancyhillis-go/nancyhillis"
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	//"bitbucket.org/dagoodma/nancyhillis-go/util"
	//"github.com/stripe/stripe-go"
)

// This is just for looking up students in Stripe
func main() {
	argsWithProg := os.Args
	if len(argsWithProg) < 2 {
		//log.Fatalf("Not enough arguments, expected %d given: %d",
		//	1, len(argsWithProg))
		log.Fatal("No customer ID provided")
		return
	}

	stripeId := string(argsWithProg[1])

	log.Printf("Looking up student with ID: %s\n", stripeId)

	err := DumpSjCustomerInfo(stripeId)
	if err != nil {
		log.Fatal(err)
	}

	return
}

func DumpSjCustomerInfo(customerId string) error {
	// Fetch Stripe customer data
	cust, err := stripewrap.GetCustomer(customerId)
	if err != nil || cust == nil {
		reason := ""
		// Try and parse the Stripe error (to make it less cryptic for user)
		ers := err.Error()
		serr, err2 := stripewrap.UnmarshallErrorResponse([]byte(ers))
		if err2 == nil {
			if serr.HTTPStatusCode == 404 {
				reason = "No such customer ID"
			}
		}
		msg := fmt.Sprintf("Failed retrieving customer data. %s", reason)
		return errors.New(msg)
	}

	/*
		ch, err := stripewrap.GetLastChargeWithPrefix(customerId, "Studio Journey")
		if err != nil || ch == nil {
			msg := fmt.Sprintf("Failed to find active subscriptions or charges for '%s'. %v", cust.Email, err)
			return errors.New(msg)
		}
	*/
	l := stripewrap.GetChargeList(customerId)
	if l == nil {
		msg := fmt.Sprintf("Failed to find any charged for: %s", customerId)
		return errors.New(msg)
	}

	fmt.Printf("Cust: %v\n", cust)
	fmt.Printf("Charges: %v\n\n", l)

	fmt.Println("List of charges:")
	// Print list of charges
	var idx = 1
	//var cnt = 0
	for l.Next() {
		c2 := l.Charge()
		log.Printf("%d: %s\nRaw data: %v\n", idx, c2.Description, c2)
		idx = idx + 1
	}

	return nil
}
