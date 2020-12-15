package main

import (
	"fmt"
	"log"
	"os"
	//"time"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	flag "github.com/spf13/pflag"
	//"gopkg.in/cheggaaa/pb.v1"

	"bitbucket.org/dagoodma/dagoodma-go/gsheetwrap"
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
	var dryRun bool
	var verbose int
	flag.CountVarP(&verbose, "verbose", "v", "Verbose dumping of payment info")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without updating billing spreadsheet")
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

	// Get the billing spreadsheet
	if verbose > 0 {
		log.Printf("Opening billing spreadsheet \"%s\"...\n", sj.BillingSpreadsheetId)
	}
	s, err := gsheetwrap.FetchSpreadsheet(sj.BillingSpreadsheetId)
	if err != nil {
		log.Fatalf("Failed to open billing spreadsheet \"%s\". %v", sj.BillingSpreadsheetId, err)
	}
	sheet, err := s.SheetByIndex(0)
	if err != nil {
		log.Fatalf("Failed to open billing spreadsheet \"%s\". %v", sj.BillingSpreadsheetId, err)
	}
	if verbose > 0 {
		log.Printf("Opened spreadsheet. Looking for student with email \"%s\"...\n", email)
	}

	// Start after header row, or from offset
	row, err := gsheetwrap.SearchForSingleRowWithValue(s, email)
	rowNumber := int(row[0].Row)
	if err != nil || len(row) <= sj.BillingSpreadsheetEmailCol {
		log.Fatalf("Failed looking for \"%s\" in billing spreadsheet. %v", email, err)
	}

	billingComplete := strings.EqualFold(row[sj.BillingSpreadsheetCompleteCol].Value, "yes")
	endedBilling := strings.EqualFold(row[sj.BillingSpreadsheetEndedCol].Value, "true")
	if verbose > 0 {
		log.Printf("Found \"%s\" with complete=%t, ended=%t.\n",
			email, billingComplete, endedBilling)
	}
	if verbose > 0 {
		spew.Dump(row)
	}

	status, err := sj.GetBillingStatus(email)
	if err != nil {
		log.Fatalf("Failed to fetch billing status for \"%s\". %v\n", email, err)
	}
	if verbose > 1 {
		log.Println("Billing status:")
		spew.Dump(status)
	}

	canceledBilling := status.HasCanceledSubscription
	paymentCount := strconv.FormatInt(int64(status.PaymentCount), 10)
	lifeTimeValue := strconv.FormatFloat(float64(status.LifeTimeValue), 'g', -1, 32)
	newBillingComplete := status.HasPackagePayment ||
		!endedBilling && (status.RemainingLifeTimeValue < sj.FounderMonthlyPrice-1) &&
			!canceledBilling

	// Update payment count and LTV
	sheet.Update(rowNumber, sj.BillingSpreadsheetPaymentsCol, paymentCount)
	sheet.Update(rowNumber, sj.BillingSpreadsheetLtvCol, lifeTimeValue)

	// Mark complete
	if endedBilling && status.HasActiveSubscription {
		log.Printf("Warning! Found \"%s\" in row=%d with ended_billing=%t active_sub=%t!\n",
			email, rowNumber, endedBilling, status.HasActiveSubscription)
		sheet.Update(rowNumber, sj.BillingSpreadsheetEndedCol, "FALSE")
	}
	if !endedBilling && canceledBilling {
		sheet.Update(rowNumber, sj.BillingSpreadsheetEndedCol, "TRUE")
		if row[sj.BillingSpreadsheetCompleteCol].Value != "yes" {
			sheet.Update(rowNumber, sj.BillingSpreadsheetCancelCol, "yes")
		}

		log.Printf("Found \"%s\" with canceled subscription without ended billing.\"",
			email)
	}
	if newBillingComplete && !billingComplete {
		sheet.Update(rowNumber, sj.BillingSpreadsheetCompleteCol, "yes")
	}

	// Apply the changes
	sheet.Update(rowNumber, sj.BillingSpreadsheetLtvCol, lifeTimeValue)
	if newBillingComplete && !billingComplete {
		sheet.Update(rowNumber, sj.BillingSpreadsheetCompleteCol, "yes")
	}
	if !dryRun {
		err = sheet.Synchronize()
		if err != nil {
			log.Fatalf("Failed updating billing spreadsheet \"%s\" (row=%d). %v",
				sj.BillingSpreadsheetId, rowNumber, err)
		}
	}

	dryRunStr := ""
	if dryRun {
		dryRunStr = "Dry run: "
	}

	log.Printf("%sUpdated row %d for \"%s\" in billing spreadsheet with payments=%s, ltv=%s, ended=%t, complete=%t\n",
		dryRunStr, rowNumber, email, paymentCount, lifeTimeValue, endedBilling && !status.HasActiveSubscription, newBillingComplete)
}
