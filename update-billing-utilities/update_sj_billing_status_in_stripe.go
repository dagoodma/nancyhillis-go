package main

import (
	"bytes"
	//"encoding/json"
	"fmt"
	//"io/ioutil"
	"log"
	//"path/filepath"
	"strconv"

	"github.com/Songmu/prompter"
	"github.com/davecgh/go-spew/spew"
	flag "github.com/spf13/pflag"
	"github.com/stripe/stripe-go"
	"gopkg.in/cheggaaa/pb.v1"

	"bitbucket.org/dagoodma/dagoodma-go/stripewrap"
	//"bitbucket.org/dagoodma/dagoodma-go/util"

	//mm "bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	sj "bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
)

var Debug = false // supress extra messages if false
var FetchStripeLimit = 100

func main() {
	var verbose, limit int
	var dryRun bool
	var cancelCompleted bool
	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without updating Stripe customers")
	flag.BoolVarP(&cancelCompleted, "cancel-completed", "c", false, "Cancel completed founding members")
	flag.IntVarP(&limit, "limit", "l", -1, "Limit the number Stripe students to fetch")

	flag.Parse()
	args := flag.Args()
	_ = args

	// Iterate all Stripe customers and check if they're SJ members
	var allCustomers, foundationCustomers, activeBillingCustomers,
		activeBillingFoundationCustomers, activeBillingCompletedCustomers,
		foundationIncompleteBillingCustomers, unknownCustomers []*stripe.Customer
	studentStatusById := map[string]*(sj.StudentStatus){}
	haveError := false

	// Limit Stripe customers to fetch?
	lookupCount := stripewrap.GetTotalCustomerCount()
	fetchLimitStr := strconv.Itoa(FetchStripeLimit)
	lookupStr := "all"
	if limit > 0 && limit < int(lookupCount) {
		lookupStr = "limited"
		lookupCount = uint32(limit)
		// Limited below fetch limit?
		if limit < FetchStripeLimit {
			fetchLimitStr = strconv.Itoa(limit)
		}
	}
	i := stripewrap.GetCustomerListIteratorWithParams(map[string]string{"limit": fetchLimitStr})
	log.Printf("Looking up %s %d Stripe customers...\n",
		lookupStr, lookupCount)
	bar := pb.StartNew(int(lookupCount))
	bar.Increment()
	index := 0
	for i.Next() {
		c := i.Customer()
		allCustomers = append(allCustomers, c)

		s, err := sj.GetStripeCustomerAccountStatus(c)
		if err != nil {
			//haveError = true
			unknownCustomers = append(unknownCustomers, c)
			if verbose > 1 {
				log.Printf("Failed looking up account status for Stripe customer. %v\n", err)
			}
			bar.Increment()
			continue
		}
		studentStatusById[c.ID] = s
		/*
			if s.HasPackagePayment || s.HasActiveSubscription {
		*/
		if s.IsFounder {
			foundationCustomers = append(foundationCustomers, c)
		}

		if s.IsBillingActive {
			activeBillingCustomers = append(activeBillingCustomers, c)
			founderStr := ""
			if s.IsFounder {
				founderStr = "foundation "
				activeBillingFoundationCustomers = append(activeBillingFoundationCustomers, c)
			}
			if s.IsBillingComplete {
				activeBillingCompletedCustomers = append(activeBillingCompletedCustomers, c)
				if verbose > 1 {
					log.Printf("Found SJ Stripe %scustomer \"%s\" with completed billing that's still being billed with email: %s.\n", founderStr, c.ID, c.Email)
				}
			}
		}
		bar.Increment()
		index += 1
		if limit > 0 && index >= limit {
			break
		}
	}
	bar.FinishPrint("Finished looking up Stripe customers.")

	// Report
	allCount := len(allCustomers)
	foundersCount := len(foundationCustomers)
	nonFoundersCount := allCount - foundersCount
	foundersNotCompletedCount := len(foundationIncompleteBillingCustomers)
	activeCount := len(activeBillingCustomers)
	nonActiveCount := allCount - len(activeBillingCustomers)
	activeFoundersCount := len(activeBillingFoundationCustomers)
	activeCompletedCount := len(activeBillingCompletedCustomers)
	unknownCount := len(unknownCustomers)

	log.Printf("Found %d customers, with %d founders, %d non-founders, %d active billing customers, %d non-active billing customers, %d active billing founders, %d active billing completed customers, %d founders with incomplete billing, %d unknown customers.\n",
		allCount, foundersCount, nonFoundersCount, activeCount, nonActiveCount,
		activeFoundersCount, activeCompletedCount, foundersNotCompletedCount, unknownCount)

	if verbose > 0 {
		log.Println("\nCustomers with incomplete billing:")
		printListOfCustomersEmails(foundationIncompleteBillingCustomers)

		log.Println("\nCustomers with completed billing who need to be canceled:")
		printListOfCustomersEmails(activeBillingCompletedCustomers)

		log.Println("\nCustomers with unknown status:")
		printListOfCustomersEmails(unknownCustomers)
	}

	if haveError {
		log.Fatalln("Encountered errors looking up members. Exiting...")
		return
	}

	// Update the founders
	log.Printf("Updating %d Stripe SJ customers...\n", foundersCount)
	if cancelCompleted {
		log.Printf("Also prompting for canceling completed SJ customers.\nWARNING: Be sure that the cancelation webhook is turned off in Zapier.\n")
	}
	dryRunStr := "Dry-run: "
	if !dryRun {
		dryRunStr = ""
	}
	var updatedMetadataCustomers, canceledCustomers, failedCanceledCustomers []*stripe.Customer
	//bar = pb.StartNew(int(count))
	//bar.Increment()
	for _, c := range allCustomers {
		s := studentStatusById[c.ID]
		if s == nil {
			if verbose > 1 {
				log.Printf("Skipping updating \"%s\" with id \"%s\" due to missing status.\n",
					c.Email, c.ID)
			}
			continue
		}
		if verbose > 1 {
			log.Printf("Checking on \"%s\" with id \"%s\"...\n",
				c.Email, c.ID)
		}
		if verbose > 3 {
			spew.Dump(c)
		}
		if verbose > 2 {
			spew.Dump(s)
		}

		// Update metadata
		updatedMetadata := false
		if s.IsFounder {
			if val, ok := c.Metadata["sj_founder"]; !ok || val != "true" {
				c.Metadata["sj_founder"] = "true"
				updatedMetadata = true
			}
		}
		if s.IsBillingComplete {
			if val, ok := c.Metadata["sj_billing_complete"]; !ok || val != "true" {
				c.Metadata["sj_billing_complete"] = "true"
				updatedMetadata = true
			}
		}
		if updatedMetadata {
			if !dryRun {
				c2, err := stripewrap.UpdateCustomerMetadata(c.ID, c.Metadata)
				_ = c2
				if err != nil {
					log.Fatalf("Failed updating Stripe SJ customer metadata \"%s\". %v",
						c.Email, err)
				}
			}
			updatedMetadataCustomers = append(updatedMetadataCustomers, c)
			if verbose > 1 {
				updatedStr := "Updated"
				if dryRun {
					updatedStr = "\"updating\""
				}
				log.Printf("\t%s%s Stripe customer metadata.\n",
					dryRunStr, updatedStr)
			}
		} else {
			if verbose > 2 {
				log.Printf("\tDid not need to update Stripe customer metadata.\n")
			}
		}

		// Using cancel option?
		if cancelCompleted && s.IsBillingComplete && s.IsBillingActive {
			msg := fmt.Sprintf("%sCancel completed SJ member \"%s\" (%s) with Stripe ID \"%s\"",
				dryRunStr, c.Email, c.Description, c.ID)
			if prompter.YN(msg, false) {
				if !dryRun {
					s, err := sj.CancelSubscription(c)
					if err != nil {
						failedCanceledCustomers = append(failedCanceledCustomers, c)
						log.Printf("Failed canceling Studio Journey subscription (%s) for \"%s\". %v\n",
							s.ID, c.Description, err)
						continue
					}
				}
				canceledCustomers = append(canceledCustomers, c)
			}

			// TODO this
			//if updateActiveCampaign {
			//}
		}

		//bar.Increment()
	}
	bar.FinishPrint("Finished updating Stripe customers.")

	updatedMetadataCount := len(updatedMetadataCustomers)
	canceledCount := len(canceledCustomers)

	log.Printf("Updated metadata on %d students, and canceled %d students.\n",
		updatedMetadataCount, canceledCount)

	if verbose > 0 {
		if updatedMetadataCount > 0 {
			log.Println("\nStudents with updated metadata:")
			printListOfCustomersEmails(updatedMetadataCustomers)
		}

		if cancelCompleted && canceledCount > 0 {
			log.Println("\nStudents with completed billing canceled:")
			printListOfCustomersEmails(canceledCustomers)
		}
	}
	if len(failedCanceledCustomers) > 0 {
		log.Printf("\nFailed to cancel %d students with completed billing:\n",
			len(failedCanceledCustomers))
		printListOfCustomersEmails(failedCanceledCustomers)
	}
}

func printListOfCustomersEmails(list []*stripe.Customer) {
	var customerListBuffer bytes.Buffer

	for i, c := range list {
		if i > 0 {
			customerListBuffer.WriteString(", ")
		}
		customerListBuffer.WriteString(c.Email)
	}
	fmt.Println(customerListBuffer.String())
}
