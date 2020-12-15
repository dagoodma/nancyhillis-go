package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Songmu/prompter"
	"github.com/davecgh/go-spew/spew"
	flag "github.com/spf13/pflag"
	//"github.com/stripe/stripe-go"

	//"bitbucket.org/dagoodma/nancyhillis-go/gsheetwrap"
	//mm "bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	"bitbucket.org/dagoodma/dagoodma-go/stripewrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"
	sj "bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
)

//var Debug = false // supress extra messages if false
var ProgramName = ""

// Override pflag usage
var Usage = func() {
	fullProgramName := os.Args[0]
	dir, ProgramName := filepath.Split(fullProgramName)
	_ = dir
	fmt.Fprintf(os.Stderr, "Usage: %s [Options] user_email\n", ProgramName)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage
	var verbose int
	var dryRun bool
	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without updating Stripe customers")

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
		log.Fatalf("Invalid email address given: %s", email)
	}

	// Find them in Stripe
	c, err := stripewrap.GetCustomerByEmail(email)
	if err != nil {
		log.Fatalf("Failed to find student with email: %s. %v", email, err)
	}
	if c == nil {
		log.Fatalf("No student found with email: %s", email)
	}
	s, err := sj.GetBillingStatus(email)
	if err != nil {
		log.Fatalf("Failed to find student status with email: %s. %v", email, err)
	}
	if !s.HasActiveSubscription {
		log.Fatalf("Student \"%s\" does not have plan with active billing in Stripe", email)
	}

	dryRunStr := "Dry-run: "
	if !dryRun {
		dryRunStr = ""
	}
	msg := fmt.Sprintf("%sCancel SJ member \"%s\" (%s) with Stripe ID \"%s\". Did you disable Zap?",
		dryRunStr, c.Email, c.Description, c.ID)
	if prompter.YN(msg, false) {
		if !dryRun {
			// TODO add code to cancel them
			s, err := sj.CancelSubscription(c)
			if err != nil {
				log.Fatalf("Failed canceling Studio Journey subscription (%s) for \"%s\". %v\n",
					s.ID, c.Description, err)
			}

			if verbose > 0 {
				spew.Dump(s)
			}
		}
		log.Printf("%sCanceled SJ billing plan for: %s", dryRunStr, email)
	}
}
