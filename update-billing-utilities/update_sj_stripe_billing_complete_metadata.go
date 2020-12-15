package main

import (
	//"bytes"
	//"encoding/json"
	"fmt"
	"os"
	//"io/ioutil"
	"log"
	"path/filepath"
	//"strconv"
	//"strings"

	"github.com/Songmu/prompter"
	"github.com/davecgh/go-spew/spew"
	flag "github.com/spf13/pflag"
	//"github.com/stripe/stripe-go"
	//"gopkg.in/cheggaaa/pb.v1"

	"bitbucket.org/dagoodma/dagoodma-go/stripewrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"

	//mm "bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	sj "bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
)

//var Debug = false // supress extra messages if false
var FetchStripeLimit = 100

var ProgramName = ""

// Override pflag usage
var Usage = func() {
	fullProgramName := os.Args[0]
	dir, ProgramName := filepath.Split(fullProgramName)
	_ = dir
	fmt.Fprintf(os.Stderr, "Usage: %s [Options] student_email\n", ProgramName)
	flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun, cancelCompleted, quiet bool
	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&quiet, "quiet", "q", false, "Supress all unnecessary output")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without updating Stripe customers")
	flag.BoolVarP(&cancelCompleted, "cancel-completed", "c", false, "Cancel student subscription if billing completed")
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		log.Fatalf("No student email address provided.")
	}
	if len(args) > 1 {
		log.Fatalf("Too many arguments given.")
	}
	email := args[0]
	if !util.EmailLooksValid(email) {
		log.Fatalf("Invalid email address given: %s\n", email)
	}

	if !quiet {
		log.Printf("Looking up student \"%s\" in Stripe...\n", email)
	}
	c, err := stripewrap.GetCustomerByEmail(email)
	if err != nil {
		log.Fatalf("Failed to find student \"%s\" in Stripe. %v\n", email, err)
	}

	s, err := sj.GetStripeCustomerAccountStatus(c)
	if err != nil {
		log.Fatalf("Failed looking up account status for Stripe customer \"%s\". %v\n", email, err)
	}

	founderStr := ""
	if s.IsBillingActive && s.IsBillingComplete {
		if s.IsFounder {
			founderStr = "foundation "
		}
		if !quiet {
			log.Printf("Found SJ Stripe %scustomer \"%s\" with completed billing that's still being billed with email: %s.\n", founderStr, c.ID, c.Email)
		}
	}

	// Update them in Stripe
	if !quiet {
		log.Printf("Updating \"%s\" in Stripe...\n", email)
	}
	if cancelCompleted && !quiet {
		log.Printf("Also prompting for canceling completed billing customers.\nWARNING: Be sure that the cancelation webhook is turned off in Zapier.\n")
	}
	dryRunStr := "Dry-run: "
	if !dryRun {
		dryRunStr = ""
	}
	if verbose > 0 {
		log.Printf("Checking on \"%s\" with id \"%s\"...\n", c.Email, c.ID)
	}
	if verbose > 2 {
		spew.Dump(c)
	}
	if verbose > 1 {
		spew.Dump(s)
	}

	// Update metadata
	needsCancel := s.IsBillingComplete && s.IsBillingActive
	updatedMetadata := false
	var newFounderMetadata, newCompleteMetadata string
	if s.IsFounder {
		if val, ok := c.Metadata["sj_founder"]; !ok || val != "true" {
			c.Metadata["sj_founder"] = "true"
			newFounderMetadata = "true"
			updatedMetadata = true
		}
	}
	if s.IsBillingComplete {
		if val, ok := c.Metadata["sj_billing_complete"]; !ok || val != "true" {
			c.Metadata["sj_billing_complete"] = "true"
			newCompleteMetadata = "true"
			updatedMetadata = true
		}
	}
	if updatedMetadata {
		if !dryRun {
			c2, err := stripewrap.UpdateCustomerMetadata(c.ID, c.Metadata)
			_ = c2
			if err != nil {
				log.Fatalf("Failed updating Stripe foundation customer metadata \"%s\". %v",
					c.Email, err)
			}
		}
		if !quiet {
			updatedStr := "Updated"
			if dryRun {
				updatedStr = "\"updating\""
			}
			founderStar := ""
			completeStar := ""
			cancelStar := ""
			if len(newFounderMetadata) > 0 {
				founderStar = "*"
			}
			if len(newCompleteMetadata) > 0 {
				completeStar = "*"
			}
			if cancelCompleted && needsCancel {
				cancelStar = "*"
			}
			log.Printf("%s%s Stripe customer metadata: %sfounder=\"%s\", %scomplete=\"%s\", %sneeds_cancel=%t\n",
				dryRunStr, updatedStr, founderStar, newFounderMetadata,
				completeStar, newCompleteMetadata, cancelStar, needsCancel)
		}
	} else {
		if !quiet && verbose > 0 {
			log.Printf("\tDid not need to update Stripe customer metadata.\n")
		}
	}

	// Using cancel option?
	if cancelCompleted && needsCancel {
		msg := fmt.Sprintf("%sCancel completed %smember \"%s\" (%s) with Stripe ID \"%s\"",
			dryRunStr, founderStr, c.Email, c.Description, c.ID)
		if quiet || prompter.YN(msg, false) {
			if !dryRun {
				s, err := sj.CancelSubscription(c)
				if err != nil {
					log.Fatalf("Failed canceling Studio Journey subscription (%s) for \"%s\". %v\n",
						s.ID, c.Description, err)
				}
			}
		}
	}
}
