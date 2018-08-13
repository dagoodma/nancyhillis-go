package main

import (
	//"errors"
	"fmt"
	"log"
	"os"
	//"strings"
	//"time"
	//"gsheetwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/nancyhillis"
	//"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
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

	log.Printf("Getting student status with ID: %s\n", stripeId)

	status, err := nancyhillis.GetSjAccountStatus(stripeId)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	fmt.Printf("Status: %v", status)

	return
}
