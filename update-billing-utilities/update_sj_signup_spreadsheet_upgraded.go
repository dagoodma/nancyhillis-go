package main

import (
	"fmt"
	"log"
	"os"
	//"time"
	"path/filepath"
	//"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	flag "github.com/spf13/pflag"
	"gopkg.in/cheggaaa/pb.v1"

	"bitbucket.org/dagoodma/dagoodma-go/gsheetwrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"
	sj "bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
)

//var Debug = false // supress extra messages if false
var GoogleSheetSleepTime, _ = time.ParseDuration("1s")
var ProgramName = ""

// Override pflag usage
var Usage = func() {
	fullProgramName := os.Args[0]
	dir, ProgramName := filepath.Split(fullProgramName)
	_ = dir
	fmt.Fprintf(os.Stderr, "Usage: %s [Options]\n", ProgramName)
	flag.PrintDefaults()
}

func main() {
	flag.Usage = Usage
	var dryRun bool
	var verbose, offset, limit int
	flag.CountVarP(&verbose, "verbose", "v", "Verbose dumping of payment info")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without updating billing spreadsheet")
	flag.IntVarP(&offset, "offset", "o", -1, "Skip to a certain row in the billing spreadsheet to start (index starts from 1)")
	flag.IntVarP(&limit, "limit", "l", -1, "Limit the number of rows to process by this number")
	flag.Parse()
	args := flag.Args()
	_ = args
	dryRunStr := ""
	if dryRun {
		dryRunStr = "Dry run: "
	}

	// Get the billing spreadsheet
	if verbose > 0 {
		log.Printf("Opening billing spreadsheet \"%s\"...\n", sj.EnrollmentSpreadsheetId)
	}
	s, err := gsheetwrap.FetchSpreadsheet(sj.EnrollmentSpreadsheetId)
	if err != nil {
		log.Fatalf("Failed to open billing spreadsheet \"%s\". %v", sj.EnrollmentSpreadsheetId, err)
	}
	sheet, err := s.SheetByIndex(0)
	if err != nil {
		log.Fatalf("Failed to open billing spreadsheet \"%s\". %v", sj.EnrollmentSpreadsheetId, err)
	}
	count := len(sheet.Rows) - 1
	if limit > 0 {
		count = limit
	}
	if verbose > 0 {
		log.Printf("Opened spreadsheet. Processing %d rows...\n", count)
	}
	bar := pb.StartNew(count)
	// Start after header row, or from offset
	startRow := 1
	if offset > 1 {
		if offset > len(sheet.Rows) {
			log.Fatalf("Cannot start from offset row %d, because there are only %d rows in the spreadsheet.",
				offset, len(sheet.Rows))
		}
		startRow = offset - 1 // offset index begins from 1, but loop uses 0 index
		for i := 1; i < offset; i++ {
			bar.Increment()
		}
	}
	rowsProcessedCount := 0
	emailsOfCustomersMarkedAsUpgraded := []string{}
	emailsOfCustomersMarkedAsCanceled := []string{}
	emailsOfCustomersMarkedAsNotCanceled := []string{}
	for i, row := range sheet.Rows[startRow:] {
		rowNumber := startRow + i     // must add start row since range always starts with 0
		rowNumberStr := rowNumber + 1 // corresponds to row in spreadsheet gui
		// Get their email
		email := row[sj.EnrollmentSpreadsheetEmailCol].Value
		if len(email) < 1 {
			if verbose > 0 {
				log.Printf("No email address in row %d\n", rowNumberStr)
			}
			time.Sleep(GoogleSheetSleepTime)
			continue
		}
		if !util.EmailLooksValid(email) {
			log.Printf("Invalid email address (row=%d): %s\n", rowNumberStr, email)
			rowsProcessedCount += 1
			if limit > 0 && rowsProcessedCount >= limit {
				break
			}
			time.Sleep(GoogleSheetSleepTime)
			continue
		}
		canceled := strings.EqualFold(row[sj.EnrollmentSpreadsheetCancelCol].Value, "yes")
		upgraded := strings.EqualFold(row[sj.EnrollmentSpreadsheetUpgradeCol].Value, "yes")
		if verbose > 1 {
			log.Printf("Found \"%s\" with cancel=%t, upgrade=%t.\n",
				email, canceled, upgraded)
		}

		status, err := sj.GetBillingStatus(email)
		if err != nil {
			log.Printf("Failed to fetch billing status for \"%s\". %v\n", email, err)
			rowsProcessedCount += 1
			if limit > 0 && rowsProcessedCount >= limit {
				break
			}
			time.Sleep(GoogleSheetSleepTime)
			continue
		}
		if verbose > 2 {
			log.Println("Billing status:")
			spew.Dump(status)
		}

		hasPackage := status.HasPackagePayment
		hasActiveSubscription := status.HasActiveSubscription
		hasCanceledSubscription := status.HasCanceledSubscription
		hasMultiplePayments := status.PaymentCount > 1
		if verbose > 1 {
			log.Printf("Got \"%s\" with hasPackage=%t, hasActiveSubscription=%t, hasCanceledSubscription=%t, hasMultiplePayments=%t.\n",
				email, hasPackage, hasActiveSubscription, hasCanceledSubscription, hasMultiplePayments)
		}
		wronglyCanceled := canceled && !hasCanceledSubscription && (hasPackage || hasActiveSubscription)
		missingCanceled := hasCanceledSubscription && !hasPackage && !canceled && !status.IsComplete
		missingUpgraded := hasPackage && hasMultiplePayments && !upgraded && !status.IsFounder
		rowNeedsUpdate := missingCanceled || missingUpgraded || wronglyCanceled
		if verbose > 0 && rowNeedsUpdate {
			log.Printf("Found error in spreadsheet for student \"%s\" with wronglyCanceled=%t, missingCanceled=%t, missingUpgraded=%t.\n",
				email, wronglyCanceled, missingCanceled, missingUpgraded)
		}

		newCanceled := ""
		newUpgraded := ""
		if missingCanceled {
			newCanceled = "yes"
			emailsOfCustomersMarkedAsCanceled = append(emailsOfCustomersMarkedAsCanceled, email)
		}
		if wronglyCanceled {
			emailsOfCustomersMarkedAsNotCanceled = append(emailsOfCustomersMarkedAsNotCanceled, email)
		}
		if missingUpgraded {
			newUpgraded = "yes"
			emailsOfCustomersMarkedAsUpgraded = append(emailsOfCustomersMarkedAsUpgraded, email)
		}

		if rowNeedsUpdate {
			// If not dryRun, then up payment count, LTV, ended billing, canceled billing, completed billing
			if !dryRun {
				sheet.Update(rowNumber, sj.EnrollmentSpreadsheetCancelCol, newCanceled)
				sheet.Update(rowNumber, sj.EnrollmentSpreadsheetUpgradeCol, newUpgraded)
				// Apply the changes
				err = sheet.Synchronize()
				if err != nil {
					log.Fatalf("Failed updating billing spreadsheet \"%s\" (row=%d). %v",
						sj.EnrollmentSpreadsheetId, rowNumberStr, err)
				}
			} else {
				// Put a star if field was updated
				canceledStar := ""
				upgradedStar := ""
				if missingUpgraded {
					upgradedStar = "*"
				}
				if wronglyCanceled || missingCanceled {
					canceledStar = "*"
				}
				log.Printf("%supdate row %d for \"%s\" with %scanceled=\"%s\", %supgraded=\"%s\"\n",
					dryRunStr, rowNumberStr, email, canceledStar, newCanceled, upgradedStar, newUpgraded)
			}
		}

		bar.Increment()
		rowsProcessedCount += 1

		if limit > 0 && rowsProcessedCount >= limit {
			break
		}
		time.Sleep(GoogleSheetSleepTime)
	}
	msg := fmt.Sprintf("%sFinished updating spreadsheet!", dryRunStr)
	bar.FinishPrint(msg)

	if len(emailsOfCustomersMarkedAsUpgraded) > 0 {
		log.Printf("\n%sUpdated %d students as upgraded:\n %v",
			dryRunStr, len(emailsOfCustomersMarkedAsUpgraded), emailsOfCustomersMarkedAsUpgraded)
	} else {
		log.Println("No missing upgraded students found.")
	}

	if len(emailsOfCustomersMarkedAsCanceled) > 0 {
		log.Printf("\nUpdated %d students as missing canceled:\n %v",
			len(emailsOfCustomersMarkedAsCanceled), emailsOfCustomersMarkedAsCanceled)
	}
	if len(emailsOfCustomersMarkedAsNotCanceled) > 0 {
		log.Printf("\nUpdated %d students as not canceled:\n %v",
			len(emailsOfCustomersMarkedAsNotCanceled), emailsOfCustomersMarkedAsNotCanceled)
	}

	return
}
