package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
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

//var StripeSleepSeconds = 0.05
var GoogleSheetSleepTime, _ = time.ParseDuration("0.5s")

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
	emailsOfNewlyCompletedCustomers := []string{}
	emailsOfNewlyCanceledCustomers := []string{}
	emailsOfWronglyCanceledCustomers := []string{}
	emailsOfMissingCustomers := []string{}
	emailsOfCompletedStillNeedCancelCustomers := []string{}
	for i, row := range sheet.Rows[startRow:] {
		rowNumber := startRow + i     // must add start row since range always starts with 0
		rowNumberStr := rowNumber + 1 // corresponds to row in spreadsheet gui
		// Get their email
		email := row[sj.BillingSpreadsheetEmailCol].Value
		if len(email) < 1 {
			if verbose > 0 {
				log.Printf("No email address in row %d\n", rowNumberStr)
			}
			bar.Increment()
			rowsProcessedCount += 1
			if limit > 0 && rowsProcessedCount >= limit {
				break
			}
			time.Sleep(GoogleSheetSleepTime)
			continue
		}
		if !util.EmailLooksValid(email) {
			log.Printf("Invalid email address (row=%d): %s\n", rowNumberStr, email)
			bar.Increment()
			rowsProcessedCount += 1
			if limit > 0 && rowsProcessedCount >= limit {
				break
			}
			time.Sleep(GoogleSheetSleepTime)
			continue
		}
		billingComplete := strings.EqualFold(row[sj.BillingSpreadsheetCompleteCol].Value, "yes")
		endedBilling := strings.EqualFold(row[sj.BillingSpreadsheetEndedCol].Value, "true")
		canceledBilling := strings.EqualFold(row[sj.BillingSpreadsheetCancelCol].Value, "yes")
		if verbose > 0 {
			log.Printf("Found \"%s\" with complete=%t, ended=%t, canceled=%t.\n",
				email, billingComplete, endedBilling, canceledBilling)
		}

		status, err := sj.GetBillingStatus(email)
		if err != nil {
			emailsOfMissingCustomers = append(emailsOfMissingCustomers, email)
			log.Printf("Failed to fetch billing status for \"%s\". %v\n", email, err)
			bar.Increment()
			rowsProcessedCount += 1
			if limit > 0 && rowsProcessedCount >= limit {
				break
			}
			time.Sleep(GoogleSheetSleepTime)
			continue
		}
		if verbose > 1 {
			log.Println("Billing status:")
			spew.Dump(status)
		}

		hasPackage := status.HasPackagePayment
		hasCompletedBilling := status.IsComplete
		hasCanceledBilling := status.HasCanceledSubscription && !status.HasActiveSubscription &&
			!hasCompletedBilling && !hasPackage // This will make sure completed & upgraded customers are not marked as canceled
		paymentCount := strconv.FormatInt(int64(status.PaymentCount), 10)
		lifeTimeValue := strconv.FormatFloat(float64(status.LifeTimeValue), 'g', -1, 32)
		//newBillingComplete := !endedBilling && (status.RemainingLifeTimeValue < sj.FounderMonthlyPrice-1)
		newBillingComplete := hasPackage ||
			!endedBilling && (status.RemainingLifeTimeValue < sj.FounderMonthlyPrice-1) &&
				!hasCanceledBilling
		rowNeedsUpdate := newBillingComplete != billingComplete ||
			endedBilling != !status.HasActiveSubscription ||
			row[sj.BillingSpreadsheetPaymentsCol].Value != paymentCount ||
			row[sj.BillingSpreadsheetLtvCol].Value != lifeTimeValue ||
			canceledBilling != hasCanceledBilling

		if verbose > 2 {
			log.Printf("Here with (complete: stripe=%t != sheet=%t) OR (ended: sheet=%t != stripe=%t) OR (cancel: sheet=%t != stripe=%t) OR (payments: sheet=%s != stripe=%s) OR (ltv: sheet=%s != stripe=%s)\n",
				newBillingComplete, billingComplete,
				endedBilling, !status.HasActiveSubscription,
				canceledBilling, hasCanceledBilling,
				row[sj.BillingSpreadsheetPaymentsCol].Value, paymentCount,
				row[sj.BillingSpreadsheetLtvCol].Value, lifeTimeValue)
		}

		if billingComplete && !endedBilling && !rowNeedsUpdate {
			emailsOfCompletedStillNeedCancelCustomers = append(emailsOfCompletedStillNeedCancelCustomers, email)
		}

		if rowNeedsUpdate {
			if canceledBilling != hasCanceledBilling {
				if hasCanceledBilling {
					emailsOfNewlyCanceledCustomers = append(emailsOfNewlyCanceledCustomers, email)
				} else {
					emailsOfWronglyCanceledCustomers = append(emailsOfWronglyCanceledCustomers, email)
				}
			}
			// If not dryRun, then up payment count, LTV, ended billing, canceled billing, completed billing
			if !dryRun {
				// Update payment count and LTV
				sheet.Update(rowNumber, sj.BillingSpreadsheetPaymentsCol, paymentCount)
				sheet.Update(rowNumber, sj.BillingSpreadsheetLtvCol, lifeTimeValue)

				// Fix wrong canceled value in spreadsheet
				if canceledBilling != hasCanceledBilling {
					log.Printf("Warning! Found \"%s\" in row=%d with canceled_billing=%t sheet_cancel_col=%t!\n",
						email, rowNumberStr, hasCanceledBilling, canceledBilling)
					newCanceledBilling := ""
					if hasCanceledBilling {
						newCanceledBilling = "yes"
					}
					sheet.Update(rowNumber, sj.BillingSpreadsheetCancelCol, newCanceledBilling)
					canceledBilling = hasCanceledBilling
				}
				// Mark complete
				if endedBilling && status.HasActiveSubscription {
					log.Printf("Warning! Found \"%s\" in row=%d with ended_billing=%t active_sub=%t!\n",
						email, rowNumberStr, endedBilling, status.HasActiveSubscription)
					sheet.Update(rowNumber, sj.BillingSpreadsheetEndedCol, "FALSE")
				} else if !endedBilling && !status.HasActiveSubscription && !canceledBilling {
					sheet.Update(rowNumber, sj.BillingSpreadsheetEndedCol, "TRUE")
				}
				if !endedBilling && canceledBilling {
					sheet.Update(rowNumber, sj.BillingSpreadsheetEndedCol, "TRUE")
					sheet.Update(rowNumber, sj.BillingSpreadsheetCancelCol, "yes")

					log.Printf("Found \"%s\" with canceled subscription without ended billing.\"",
						email)
				}
				if newBillingComplete && !billingComplete {
					sheet.Update(rowNumber, sj.BillingSpreadsheetCompleteCol, "yes")
				}

				// Apply the changes
				err = sheet.Synchronize()
				if err != nil {
					log.Fatalf("Failed updating billing spreadsheet \"%s\" (row=%d). %v",
						sj.BillingSpreadsheetId, rowNumberStr, err)
				}
			} else {
				log.Printf("Dry run: update row %d with payments=%s, ltv=%s, ended=%t, complete=%t, cancel=%t\n",
					rowNumberStr, paymentCount, lifeTimeValue, endedBilling && !status.HasActiveSubscription,
					newBillingComplete, hasCanceledBilling)
			}
			if newBillingComplete && !billingComplete {
				sheet.Update(rowNumber, sj.BillingSpreadsheetCompleteCol, "yes")
				emailsOfNewlyCompletedCustomers = append(emailsOfNewlyCompletedCustomers, email)
			}
		}

		bar.Increment()
		rowsProcessedCount += 1

		if limit > 0 && rowsProcessedCount >= limit {
			break
		}
		time.Sleep(GoogleSheetSleepTime)
	}
	bar.FinishPrint("Finished updating spreadsheet!\n")
	dryRunStr := ""
	if dryRun {
		dryRunStr = "Dry run: "
	}

	if len(emailsOfNewlyCompletedCustomers) > 0 {
		log.Printf("%sUpdated %d students as completed billing:\n %v",
			dryRunStr, len(emailsOfNewlyCompletedCustomers), emailsOfNewlyCompletedCustomers)
	} else {
		log.Println("No newly completed customers found.")
	}

	if len(emailsOfNewlyCanceledCustomers) > 0 {
		log.Printf("%sUpdated %d students as newly canceled:\n %v",
			dryRunStr, len(emailsOfNewlyCanceledCustomers), emailsOfNewlyCanceledCustomers)
	}
	if len(emailsOfWronglyCanceledCustomers) > 0 {
		log.Printf("%sUpdated %d students as wrongly (no longer) canceled:\n %v",
			dryRunStr, len(emailsOfWronglyCanceledCustomers), emailsOfWronglyCanceledCustomers)
	}
	if len(emailsOfMissingCustomers) > 0 {
		log.Printf("Failed to find %d students who were missing from Stripe:\n %v",
			len(emailsOfMissingCustomers), emailsOfMissingCustomers)
	}
	if len(emailsOfCompletedStillNeedCancelCustomers) > 0 {
		log.Printf("Found %d students with completed billing who still need to be canceled:\n %v",
			len(emailsOfCompletedStillNeedCancelCustomers), emailsOfCompletedStillNeedCancelCustomers)
	}

	return
}
