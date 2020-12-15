package studiojourney

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	//"github.com/davecgh/go-spew/spew"
	"github.com/stripe/stripe-go"
	"gopkg.in/Iwark/spreadsheet.v2"

	"bitbucket.org/dagoodma/dagoodma-go/gsheetwrap"
	"bitbucket.org/dagoodma/dagoodma-go/stripewrap"
	"bitbucket.org/dagoodma/dagoodma-go/util"
	// Membermouse is now deactivated
	//"bitbucket.org/dagoodma/nancyhillis-go/membermouse"
)

/*
 * Constants
 */
//var Debug = false

// Spreadsheet constants
var EnrollmentSpreadsheetId = "1wRHucYoRuGzHav7nK3V5Hv2Z4J67D_vTZN5wjw8aa2k"
var FounderMigratedSpreadsheetId = "13xE8UGR03CBEjB0othGm4abv8vDi_d8wD7U1FF2BWUg"
var BillingSpreadsheetId = "1p_tRygUVmhDNK68fPkKkXZnlvq7D2sQjBmpJ6TwDJUs"
var MmTransactionsSpreadsheetId = "1sra-kv8f2ZVLmO9QK3MCfE0IIDIIWm61t2HQcMTdCf8"
var CancellationSpreadsheetId = "1EKg0vqz2eaYqL31W1IqkdxuoqZ1FXtGEVXOC_lyfh5E"
var ChangeEmailSpreadsheetId = "1ZeLSi3-IwVbRiMbPFAvW0et7bjhpwc0yYqTD5xYx8KI"

// Spreadsheet columns (starts from 0)
// Enrollment/Signup spreadsheet
// https://docs.google.com/spreadsheets/d/1wRHucYoRuGzHav7nK3V5Hv2Z4J67D_vTZN5wjw8aa2k/edit?usp=sharing
var EnrollmentSpreadsheetStripeIdCol = 13
var EnrollmentSpreadsheetEmailCol = 2
var EnrollmentSpreadsheetNameCol = 1
var EnrollmentSpreadsheetCancelCol = 18
var EnrollmentSpreadsheetUpgradeCol = 19

var FounderMigratedSpreadsheetEmailCol = 2

// Billing spreadsheet
// https://docs.google.com/spreadsheets/d/1p_tRygUVmhDNK68fPkKkXZnlvq7D2sQjBmpJ6TwDJUs/edit?usp=sharing
var BillingSpreadsheetEndedCol = 0
var BillingSpreadsheetCompleteCol = 1
var BillingSpreadsheetEmailCol = 3
var BillingSpreadsheetPaymentsCol = 8
var BillingSpreadsheetLtvCol = 9
var BillingSpreadsheetFounderCol = 15
var BillingSpreadsheetCancelCol = 16

// Membermouse transactions spreadsheet
// https://docs.google.com/spreadsheets/d/1sra-kv8f2ZVLmO9QK3MCfE0IIDIIWm61t2HQcMTdCf8/edit?usp=sharing
var MmTransactionsSpreadsheetEmailCol = 6
var MmTransactionsSpreadsheetAmountCol = 3

// Cancellation spreadsheet
var CancellationSpreadsheetEmailCol = 2

// Change email spreadsheet
var ChangeEmailSpreadsheetTeachableIdCol = 0
var ChangeEmailSpreadsheetNameCol = 1
var ChangeEmailSpreadsheetOldEmailCol = 2
var ChangeEmailSpreadsheetNewEmailCol = 3
var ChangeEmailSpreadsheetStripeIdCol = 4
var ChangeEmailSpreadsheetAcIdCol = 5
var ChangeEmailSpreadsheetTimestampCol = 6
var ChangeEmailSpreadsheetSourceCol = 7

var MonthlyPrice = float32(36.00)
var FounderMonthlyPrice = float32(29.00)
var ArtBundleCount = 12 // months

// Cancel after overdue grace period ends (as set in Stripe dashboard)
// TODO Load this from Stripe dashboard settings?
// Note this is different now. We retry 3 times and then it goes to
// AC automation for a few days.
var OverdueGracePeriodDays = 21 // 3 weeks

// Stripe account statuses for SJ
var StripeAccountActiveStatuses = []string{"active", "trialing", "unpaid"} // rest are inactive
// These are what stripe uses:
var AccountStatuses = [6]string{"active", "past_due", "canceling", "canceled", "unknown", "complete"}

// These are what we use mapped to human strings:
var AccountStatusesHuman = map[string]string{
	"active":         "Active",
	"past_due":       "Overdue",
	"canceled":       "Canceled",
	"pending_cancel": "Canceling",
	"unknown":        "Unknown",
	"complete":       "Complete",
}

/*
 * Student data structures
 */
// Json Header of customer SJ status response for billing portal
type StudentStatus struct {
	Email                   string `json:"email"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name"`
	Plan                    string `json:"plan"`
	PlanHuman               string `json:"plan_human"`
	Status                  string `json:"status"`
	StatusHuman             string `json:"status_human"`
	IsFounder               bool   `json:"is_founder"`
	IsPackage               bool   `json:"is_package"`
	IsRecurring             bool   `json:"is_recurring"`
	IsPaymentPlan           bool   `json:"is_payment_plan"`
	IsRefunded              bool   `json:"is_refunded"`
	IsOverdue               bool   `json:"is_overdue"`
	IsBillingComplete       bool   `json:"is_billing_complete"`
	IsBillingActive         bool   `json:"is_billing_active"`
	RecurringPrice          uint64 `json:"recurring_price"`
	BillingCycleAnchor      int64  `json:"billing_cycle_anchor"`
	BillingCycleAnchorHuman string `json:"billing_cycle_anchor_human"`
	DaysUntilDue            uint64 `json:"days_until_due"`
	Created                 int64  `json:"created"`
	Start                   int64  `json:"start"`
	Ended                   int64  `json:"ended_at"`
	PeriodStart             int64  `json:"period_start"`
	PeriodEnd               int64  `json:"period_end"`
	DaysUntilEndOfPeriod    uint64 `json:"days_until_end_of_period"`
	NextBillHuman           string `json:"next_bill_human"`
	BillingInterval         string `json:"billing_interval"`
	BillingIntervalCount    uint64 `json:"billing_interval_count"`
	GracePeriodDaysLeft     uint64 `json:"grace_period_days_left"`
	Canceled                int64  `json:"canceled_at"`
	CancelAtEndOfPeriod     bool   `json:"cancel_at_end_of_period"`
	EnrolledDuration        int64  `json:"enrolled_duration"`
	EnrolledDurationHuman   string `json:"enrolled_duration_human"`
	CustomerId              string `json:"customer_id"`
	TrialEnd                int64  `json:"trial_end"`
	TrialStart              int64  `json:"trial_start"`
	IsTrial                 bool   `json:"is_trial"`
	TrialDaysLeft           uint64 `json:"trial_days_left"`
	DefaultCardLastFour     string `json:"default_card_last_four"`
	DefaultCardBrand        string `json:"default_card_brand"`
	IsDelinquent            bool   `json:"is_delinquent"`
	BusinessVatId           string `json:"business_vat_id"`
}

type _StudentStatus StudentStatus

// Student payment data for Studio Journey
type StudentBillingStatus struct {
	MmId                    string  `json:"mm_id"`
	Email                   string  `json:"email"`
	Name                    string  `json:"name"`
	Phone                   string  `json:"phone"`
	Country                 string  `json:"country"`
	StripeId                string  `json:"stripe_id"`
	IsFounder               bool    `json:"is_founder"`
	IsComplete              bool    `json:"is_complete"`
	IsMigratedFounder       bool    `json:"is_migrated_founder"`
	LifeTimeValue           float32 `json:"ltv"`
	ExpectedLifeTimeValue   float32 `json:"expected_ltv"`
	RemainingLifeTimeValue  float32 `json:"remaining_ltv"`
	PaymentCount            int32   `json:"payment_count"`
	RemainingPaymentCount   int32   `json:"remaining_payment_count"`
	HasPackagePayment       bool    `json:"has_package_payment"`
	HasPaymentsRemaining    bool    `json:"has_payments_remaining"`
	HasActiveSubscription   bool    `json:"has_active_subscription"`
	HasCanceledSubscription bool    `json:"has_canceled_subscription"`

	Payments           []StudentBillingPayment     `json:"payments"`
	ActiveSubscription *StudentBillingSubscription `json:"active_subscription"`
}

type StudentBillingSubscription struct {
	Amount      float32 `json:"amount"`
	Description string  `json:"description"`
	CreatedDate string  `json:"created_date"` // MM is just an estimate
}

type StudentBillingPayment struct {
	Amount      float32 `json:"amount"`
	IsRefund    bool    `json:"is_refund"`
	Description string  `json:"description"`
	Date        string  `json:"date"`
	Source      string  `json:"source"`
}

/*
 * Functions
 */
func GetSpreadsheet(sheetId string, sheetNum int, name string) (*spreadsheet.Sheet, error) {
	if sheetNum < 0 {
		msg := fmt.Sprintf("Sheet number must be non-negative. Got: %d", sheetNum)
		return nil, errors.New(msg)
	}
	s, err := gsheetwrap.FetchSpreadsheet(sheetId)
	if err != nil {
		msg := fmt.Sprintf("Could not open %s spreadsheet \"%s\". %v",
			name, sheetId, err)
		return nil, errors.New(msg)
	}
	sheet, err := s.SheetByIndex(uint(sheetNum))
	if err != nil {
		msg := fmt.Sprintf("Could not retrieve sheet %d from %s spreadsheet \"%s\". %v",
			sheetNum, name, sheetId, err)
		return nil, errors.New(msg)
	}
	return sheet, nil
}

func GetEnrollmentSpreadsheet() (*spreadsheet.Sheet, error) {
	return GetSpreadsheet(EnrollmentSpreadsheetId, 0, "enrollment")
}

func GetBillingSpreadsheet() (*spreadsheet.Sheet, error) {
	return GetSpreadsheet(BillingSpreadsheetId, 0, "billing")
}

func GetMmTransactionsSpreadsheet() (*spreadsheet.Sheet, error) {
	return GetSpreadsheet(MmTransactionsSpreadsheetId, 0, "mm transaction")
}

func GetCancellationSpreadsheet() (*spreadsheet.Sheet, error) {
	return GetSpreadsheet(CancellationSpreadsheetId, 0, "cancellation")
}

func GetChangeEmailSpreadsheet() (*spreadsheet.Sheet, error) {
	return GetSpreadsheet(ChangeEmailSpreadsheetId, 0, "change email")
}

func GetFounderMigratedSpreadsheet() (*spreadsheet.Sheet, error) {
	return GetSpreadsheet(FounderMigratedSpreadsheetId, 0, "founder migrated")
}

// Note: These are read-only functions since they don't return the sheet.
// Find Stripe customer ID in SJ_Student_Signups spreadsheet
func GetEnrollmentRowByEmail(email string) ([]spreadsheet.Cell, error) {
	sheet, err := GetEnrollmentSpreadsheet()
	if err != nil {
		return nil, err
	}

	row, err := gsheetwrap.SearchForSingleRowWithValueInColumn(sheet, EnrollmentSpreadsheetEmailCol, email)
	if err != nil {
		msg := fmt.Sprintf("Error searching for email address '%s': %v", email, err)
		//log.Printf("Error while searching spreadsheet (%s) for '%s'. %v", SpreadsheetId, emailAddress, err)
		return nil, errors.New(msg)
	}
	if row == nil || len(row) < (EnrollmentSpreadsheetStripeIdCol+1) {
		msg := fmt.Sprintf("Failed to find email address: %s", email)
		//log.Printf("Failed to find row in spreadsheet (%s) with '%s'", SpreadsheetId, emailAddress)
		return nil, errors.New(msg)
	}
	return row, nil
}

func GetBillingRowByEmail(email string) ([]spreadsheet.Cell, error) {
	sheet, err := GetBillingSpreadsheet()
	if err != nil {
		return nil, err
	}

	row, err := gsheetwrap.SearchForSingleRowWithValueInColumn(sheet, BillingSpreadsheetEmailCol, email)
	if err != nil {
		msg := fmt.Sprintf("Error checking billing status for email address '%s': %v", email, err)
		return nil, errors.New(msg)
	}
	if row == nil || len(row) < (BillingSpreadsheetEmailCol+1) {
		msg := fmt.Sprintf("Failed to find billing status email address: %s", email)
		return nil, errors.New(msg)
	}
	return row, nil
}

// This doesn't work. We only have stripe customer id
// TODO figure a good method for accessing teachable data
/*
func GetTeachableIdByEmail(email string) (string, error) {
	row, err := GetEnrollmentRowByEmail(email)
	if err != nil {
		return "", err
	}
	stripeCustomerId := row[EnrollmentSpreadsheetStripeIdCol].Value
	return stripeCustomerId, nil
}*/

func GetStripeIdByEmail(email string) (string, error) {
	/*
		fmt.Printf("Here with bobby: %s\n", email)
		row, err := GetEnrollmentRowByEmail(email)
		if err != nil {
			fmt.Printf("Got error: %v\n", err)
			return "", err
		}
		stripeCustomerId := row[EnrollmentSpreadsheetStripeIdCol].Value
		fmt.Printf("Got him: %s\n", stripeCustomerId)
		return stripeCustomerId, nil
	*/
	c, err := stripewrap.GetCustomerByEmail(email)

	if err != nil {
		return "", err
	}
	return c.ID, nil
}

// Find founder by email address in migrated spreadsheet
func GetFounderMigratedByEmail(email string) (bool, error) {
	s, err := GetFounderMigratedSpreadsheet()
	if err != nil {
		msg := fmt.Sprintf("Failed checking if founder migrated (email: %s). %v",
			email, err)
		return false, errors.New(msg)
	}
	row, err := gsheetwrap.SearchForSingleRowWithValue(s, email)
	if err != nil {
		msg := fmt.Sprintf("Error searching for founder email address '%s': %v", email, err)
		return false, errors.New(msg)
	}
	if row == nil || len(row) < (FounderMigratedSpreadsheetEmailCol+1) ||
		!strings.EqualFold(row[FounderMigratedSpreadsheetEmailCol].Value, email) {
		msg := fmt.Sprintf("Failed to find founder email address: %s", email)
		return false, errors.New(msg)
	}
	return true, nil
}

func GetAccountStatus(stripeId string) (*StudentStatus, error) {
	if !stripewrap.CustomerIdLooksValid(stripeId) {
		msg := fmt.Sprintf("Invalid customer ID: %s", stripeId)
		return nil, errors.New(msg)
	}

	// Fetch Stripe customer data
	cust, err := stripewrap.GetCustomer(stripeId)
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
		return nil, errors.New(msg)
	}
	//fmt.Printf("%v", c)
	status, err := GetStripeCustomerAccountStatus(cust)
	return status, err
}

func GetStripeCustomerAccountStatus(c *stripe.Customer) (*StudentStatus, error) {
	status := &StudentStatus{
		Email:         c.Email,
		FirstName:     c.Metadata["first_name"],
		LastName:      c.Metadata["last_name"],
		IsDelinquent:  c.Delinquent,
		CustomerId:    c.ID,
		NextBillHuman: "N/A",
	}
	// Fetch Stripe default payment source (card)
	if c.DefaultSource != nil {
		cardId := c.DefaultSource.ID
		c2, err := stripewrap.GetCard(cardId, c.ID)
		if err != nil || c2 == nil {
			msg := fmt.Sprintf("Failed retrieving customer card data. %v", err)
			return nil, errors.New(msg)
		}
		status.DefaultCardLastFour = c2.Last4
		status.DefaultCardBrand = string(c2.Brand)
	}
	if len(c.BusinessVATID) > 0 {
		status.BusinessVatId = c.BusinessVATID
	}

	// Get stripe canceled subscriptions
	sc, scErr := stripewrap.GetLastCanceledSubWithPrefix(c.ID, "sj-")
	// Check this below if there's no active subscriptions

	// Get stripe subscriptions
	if len(c.Subscriptions.Data) > 1 {
		msg := fmt.Sprintf("Found mulitple subscriptions for '%s'", c.Email)
		return nil, errors.New(msg)
	}
	/*
		// Check if canceled subs exist
		if sc != nil && scErr == nil {
			msg := fmt.Sprintf("Found a canceled Studio Journey subscription for '%s'. Please contact us to reinstate your subscription.", c.Email)
			return nil, errors.New(msg)
		}
	*/
	// First check if they have a package plan charge
	// Which has a description on the payment/charge, unlike subscription charges
	ch, chErr := stripewrap.GetLastChargeWithPrefix(c.ID, "Studio Journey")

	// First check if they're billing is complete in the SJ_Student_Billings spreadsheet
	// TODO support lookup of free year access and renew account status
	bi, biErr := GetBillingRowByEmail(c.Email)
	isComplete := false
	if biErr == nil {
		//log.Printf("Here with: %v\n", bi)
		isComplete = strings.EqualFold(bi[BillingSpreadsheetCompleteCol].Value, "yes")
	} else {
		// TODO create a log entry so we can find this person later
		//log.Printf("Error: %v\n", biErr)
	}

	// Only proceed if they have an active or canceled subscription, and
	// they don't have a package charge AND a cancelled subscription.
	// Otherwise we will treat them as a package plan subscriber.
	isMonthly := len(c.Subscriptions.Data) == 1 || (sc != nil && scErr == nil && ch == nil)

	if isComplete {
		// TODO something about free year, and maybe check if still being billed for warning too
		status.NextBillHuman = "Billing Complete"
		status.Status = "complete"
		status.Plan = "none"
		status.PlanHuman = "N/A"
		status.IsFounder = strings.EqualFold(bi[BillingSpreadsheetFounderCol].Value, "yes")
		status.StatusHuman = AccountStatusesHuman[status.Status]
		status.IsBillingComplete = true
		status.IsBillingActive = HasActiveSubscription(c)
	} else if isMonthly {
		// TODO Add support for payment plan here
		var sub *stripe.Subscription = nil
		if len(c.Subscriptions.Data) > 0 {
			sub = c.Subscriptions.Data[0]
		} else if sc != nil {
			sub = sc
		} else {
			msg := fmt.Sprintf("Found a Studio Journey subscription for '%s', but encountered an error analyzing it.", c.Email)
			return nil, errors.New(msg)
		}
		status.IsRecurring = true
		status.Created = sub.Created
		status.Start = sub.Start
		status.Ended = sub.EndedAt
		status.Canceled = sub.CanceledAt
		status.CancelAtEndOfPeriod = sub.CancelAtPeriodEnd
		status.TrialStart = sub.TrialStart
		status.TrialEnd = sub.TrialEnd
		if sub.TrialStart > 0 {
			// Determine if trial is active
			et := time.Unix(sub.TrialEnd, 0)
			ct := time.Now()
			status.IsTrial = et.After(ct)
			// Calculate trial days let
			if status.IsTrial {
				dd := time.Until(et)
				status.TrialDaysLeft = uint64(util.RoundDown(dd.Hours() / 24))
			}
		}
		status.PeriodStart = sub.CurrentPeriodStart
		status.PeriodEnd = sub.CurrentPeriodEnd
		// Calculate days until period end
		et := time.Unix(sub.CurrentPeriodEnd, 0)
		ct := time.Now()
		dt := et.Sub(ct)
		status.DaysUntilEndOfPeriod = uint64(util.RoundDown(dt.Hours() / 24))
		// Billing interval and anchor
		status.BillingInterval = string(sub.Plan.Interval)
		status.BillingIntervalCount = uint64(sub.Plan.IntervalCount)
		status.BillingCycleAnchor = sub.BillingCycleAnchor
		billingCycleTime := time.Unix(sub.BillingCycleAnchor, 0)
		status.BillingCycleAnchorHuman = billingCycleTime.Format("Mon Jan 2 15:04 2006")
		if sub.Billing == "charge_automatically" && status.Canceled == 0 &&
			!status.CancelAtEndOfPeriod {
			status.DaysUntilDue = status.DaysUntilEndOfPeriod
			status.NextBillHuman = et.Format("Jan 2")
		} else {
			// Manual invoice
			status.DaysUntilDue = uint64(sub.DaysUntilDue)
			status.NextBillHuman = "Manual"
		}
		// Figure out status of subscription
		subStatus := string(sub.Status)
		if util.StringSliceContains(StripeAccountActiveStatuses, subStatus) {
			status.Status = "active"
			status.IsBillingActive = true
		} else if subStatus == "canceled" {
			status.Status = "canceled"
		} else if subStatus == "past_due" {
			status.Status = "past_due"
			status.IsBillingActive = true
			// Calculate grace period days
			status.IsOverdue = true
			st := time.Unix(sub.CurrentPeriodStart, 0)
			gs := fmt.Sprintf("%dh", uint64(24*OverdueGracePeriodDays))
			gd, err := time.ParseDuration(gs)
			if err == nil {
				et := st.Add(gd)
				dd := time.Until(et)
				status.GracePeriodDaysLeft = uint64(util.RoundDown(dd.Hours() / 24))
			}
		} else {
			status.Status = "unknown"
		}
		// Pending cancellation
		if status.Status == "active" && status.CancelAtEndOfPeriod {
			status.Status = "pending_cancel"
		}
		status.StatusHuman = AccountStatusesHuman[status.Status]
		// Get plan info
		status.RecurringPrice = uint64(sub.Plan.Amount)
		status.Plan = sub.Plan.ID
		status.PlanHuman = sub.Plan.Nickname
		status.IsFounder = IsFounderPlan(sub.Plan.ID)
		// if len(c.Subscriptions.Data) == 1 {
	} else {
		// Must be a package plan
		if chErr != nil || ch == nil {
			msg := fmt.Sprintf("Failed to find active subscriptions or charges for '%s'. %v", c.Email, chErr)
			return nil, errors.New(msg)
		}
		status.IsPackage = true
		status.IsRefunded = ch.Refunded
		status.PlanHuman = ch.Description
		status.IsFounder = IsFounderPlan(ch.Description)
		status.Created = ch.Created
		status.Start = ch.Created
		if ch.Paid || ch.Status != "failed" {
			status.Status = "active"
		} else {
			status.Status = "past_due"
		}
		status.StatusHuman = AccountStatusesHuman[status.Status]
		// TODO determine when package should expire
		//status.Ended = sub.Ended
		//status.Canceled = sub.Ended
	}

	// Calculate time enrolled
	if status.Status == "active" {
		st := time.Unix(status.Created, 0)
		ct := time.Now()
		var dur = ct.Sub(st)
		status.EnrolledDuration = int64(dur)
		var enrolledHours = dur.Hours()
		var enrolledDays = enrolledHours / 24
		var months = util.RoundDown(enrolledDays / 31)

		if months > 0 {
			var days = util.RoundDown(enrolledDays - float64(months*31))
			status.EnrolledDurationHuman = fmt.Sprintf("%d months, %d days", months, days)
		} else {
			var days = util.RoundDown(enrolledDays)
			status.EnrolledDurationHuman = fmt.Sprintf("%d days", days)
		}
	}

	return status, nil
}

func GetBillingStatus(email string) (*StudentBillingStatus, error) {
	if !util.EmailLooksValid(email) {
		msg := fmt.Sprintf("Invalid customer email: %s", email)
		return nil, errors.New(msg)
	}
	s := StudentBillingStatus{Email: email}

	// Check in Stripe
	//log.Printf("Looking for \"%s\" in Stripe...\n", email)
	stripeId, err := GetStripeIdByEmail(email)
	var c *stripe.Customer
	if err == nil && len(stripeId) > 0 {
		s.StripeId = stripeId
		c, err = stripewrap.GetCustomer(stripeId)
		s.IsComplete = IsBillingComplete(c)
		if err == nil && c != nil {
			s.Name = fmt.Sprintf("%s %s",
				strings.TrimSpace(c.Metadata["first_name"]),
				strings.TrimSpace(c.Metadata["last_name"]))
			s.Country = c.Metadata["country"]
			if c.Subscriptions != nil && len(c.Subscriptions.Data) > 0 {
				s.HasActiveSubscription = true
				sub := StudentBillingSubscription{}
				foundActiveSub := false
				for _, ss := range c.Subscriptions.Data {
					isActiveSub, _ := stripewrap.IsActiveSubscription(ss)
					if isActiveSub {
						if foundActiveSub {
							//log.Println("WARNING: Found more than 1 active subscription!")
							msg := fmt.Sprintf("Found more than 1 active stripe subscription for: %s", email)
							return nil, errors.New(msg)
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
				// Look for canceled billings
				sc, scErr := stripewrap.GetLastCanceledSubWithPrefix(c.ID, "sj-")
				if scErr == nil && sc != nil {
					s.HasCanceledSubscription = true
				}
			}

			l := stripewrap.GetChargeList(stripeId)
			if l != nil {
				/*
					if Debug {
						log.Printf("Customer: %v\n", c)
						log.Printf("Charges: %v\n\n", l)
					}
				*/
				// Build list of charges as payments
				var idx = 1
				for l.Next() {
					c2 := l.Charge()
					if c2.Paid {
						totalAmount := float32(c2.Amount-c2.AmountRefunded) / 100.00
						p := StudentBillingPayment{
							Amount:      totalAmount,
							IsRefund:    totalAmount < 0,
							Date:        stripewrap.FormatEpochTime(c2.Created),
							Description: c2.Description,
							Source:      "Stripe",
						}
						if len(p.Description) < 1 {
							p.Description = c2.StatementDescriptor
						}
						s.LifeTimeValue = s.LifeTimeValue + totalAmount
						if totalAmount > 0 {
							s.PaymentCount = s.PaymentCount + 1
						}
						if strings.Contains(strings.ToLower(c2.Description), "package") {
							s.HasPackagePayment = true
						}
						s.Payments = append(s.Payments, p)
					}
					/*
						if Debug {
							log.Printf("%d: %s\nRaw data: %v\n", idx, c2.Description, c2)
						}
					*/
					idx = idx + 1
				}
			} else {
				//log.Printf("Failed to find any charges for: %s\n", stripeId)
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
			} else {
				reason = fmt.Sprintf("(%s) %s", serr.Code, serr.Msg)
			}
			msg := fmt.Sprintf("Failed retrieving customer Stripe data. %s\n", reason)
			return nil, errors.New(msg)
		}
	} else {
		msg := fmt.Sprintf("Could not find student in Stripe. %s\n", err)
		return nil, errors.New(msg)
	}

	// Check in membermouse
	// Membermouse is down, so this no longer works - David 2018-12-06
	//log.Printf("Looking for \"%s\" in Membermouse...\n", email)
	/*
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
					//log.Println("WARNING: Found an active subscription in both Stripe and Membermouse!")
					msg := fmt.Sprintf("Found an active subscription in Stripe and Membermouse")
					return nil, errors.New(msg)
				} else {
					s.HasActiveSubscription = true
					sub := StudentBillingSubscription{Description: "SJ Founder Membermouse",
						Amount: 29.00, CreatedDate: m.Registered}
					s.ActiveSubscription = &sub
				}
			}
		} // m, err := membermouse.GetMemberByEmail(email)
		//log.Printf("Charges: %v\n\n", l)
	*/

	isFounder := IsFounder(c)
	if isFounder {
		sheet, err := GetMmTransactionsSpreadsheet()
		if err != nil {
			log.Printf("%v", err)
		} else {
			rows, err := gsheetwrap.SearchForAllRowsWithValueInColumn(sheet,
				MmTransactionsSpreadsheetEmailCol, email)
			if err == nil && rows != nil && len(rows) > 0 {
				/*
					if Debug {
						log.Printf("Transactions: %v\n\n", rows)
					}
				*/
				s.IsMigratedFounder = true // all founders are migrated now
				s.IsFounder = true
				for _, r := range rows {
					isRefund := false
					amount, err := strconv.ParseFloat(r[MmTransactionsSpreadsheetAmountCol].Value, 32)
					if err != nil {
						//log.Fatalf("Failed parsing payment amount \"%s\". %v", r[3].Value, err)
						msg := fmt.Sprintf("Failed parsing \"%s\"'s Membermouse payment amount \"%s\". %v",
							email, r[MmTransactionsSpreadsheetAmountCol].Value, err)
						return nil, errors.New(msg)
					}
					if strings.Contains(strings.ToLower(r[0].Value), "refund") {
						isRefund = true
						amount = -amount
					}
					p := StudentBillingPayment{
						Amount:      float32(amount),
						IsRefund:    isRefund,
						Date:        r[1].Value,
						Description: fmt.Sprintf("%s for %s order #%s", r[0].Value, r[9].Value, r[2].Value),
						Source:      "Mm",
					}
					s.LifeTimeValue = s.LifeTimeValue + float32(amount)
					if amount > 0 {
						s.PaymentCount = s.PaymentCount + 1
					}
					s.Payments = append(s.Payments, p)
				}
			} else {
				log.Printf("Failed to find any Membermouse transactions for founder: %s\n", email)
			}
		}
	}

	s.ExpectedLifeTimeValue = s.LifeTimeValue
	s.HasPaymentsRemaining = false
	if s.HasActiveSubscription {
		s.ExpectedLifeTimeValue = MonthlyPrice * float32(ArtBundleCount)
		if s.IsFounder {
			s.ExpectedLifeTimeValue = FounderMonthlyPrice * float32(ArtBundleCount)
		}
	}

	// If they never bought a package plan, then calculate remaining LTV and payments
	if !s.HasPackagePayment {
		s.RemainingLifeTimeValue = s.ExpectedLifeTimeValue - s.LifeTimeValue
		if s.HasActiveSubscription && s.RemainingLifeTimeValue > 0 {
			s.HasPaymentsRemaining = true
			s.RemainingPaymentCount = int32(s.RemainingLifeTimeValue / MonthlyPrice)
			if s.IsFounder {
				s.RemainingPaymentCount = int32(s.RemainingLifeTimeValue / FounderMonthlyPrice)
			}
		}
	}
	/*
		if Debug {
			spew.Dump(s)
		}
	*/

	return &s, nil
}

func ChangeStudentEmail(oldEmail string, newEmail string, teachableId string) error {
	failMsgPrefix := fmt.Sprintf("Failed changing student email from \"%s\" to \"%s\" (tid: %s)",
		oldEmail, newEmail, teachableId)
	// Change email in signup spreadsheet
	s1, err := GetEnrollmentSpreadsheet()
	if err != nil {
		msg := fmt.Sprintf("%s. %v", failMsgPrefix, err)
		return errors.New(msg)
	}
	r1, err := gsheetwrap.SearchForSingleRowWithValueInColumn(s1, EnrollmentSpreadsheetEmailCol, oldEmail)
	if err != nil {
		msg := fmt.Sprintf("%s. %v", failMsgPrefix, err)
		return errors.New(msg)
	}
	s1.Update(int(r1[0].Row), EnrollmentSpreadsheetEmailCol, newEmail)

	// Change email in billing spreadsheet
	s2, err := GetBillingSpreadsheet()
	if err != nil {
		msg := fmt.Sprintf("%s. %v", failMsgPrefix, err)
		return errors.New(msg)
	}
	r2, err := gsheetwrap.SearchForSingleRowWithValueInColumn(s2, BillingSpreadsheetEmailCol, oldEmail)
	if err != nil {
		msg := fmt.Sprintf("%s. %v", failMsgPrefix, err)
		return errors.New(msg)
	}
	s2.Update(int(r2[0].Row), BillingSpreadsheetEmailCol, newEmail)

	// Change email in cancellation spreadsheet
	s3, err := GetCancellationSpreadsheet()
	if err != nil {
		msg := fmt.Sprintf("%s. %v", failMsgPrefix, err)
		return errors.New(msg)
	}
	r3, err := gsheetwrap.SearchForSingleRowWithValueInColumn(s3, CancellationSpreadsheetEmailCol, oldEmail)
	if err != nil {
		msg := fmt.Sprintf("%s. %v", failMsgPrefix, err)
		return errors.New(msg)
	}
	s3.Update(int(r3[0].Row), CancellationSpreadsheetEmailCol, newEmail)

	// Change email in mm transaction spreadsheet
	s4, err := GetMmTransactionsSpreadsheet()
	if err != nil {
		msg := fmt.Sprintf("%s. %v", failMsgPrefix, err)
		return errors.New(msg)
	}
	r4, err := gsheetwrap.SearchForSingleRowWithValueInColumn(s4, MmTransactionsSpreadsheetEmailCol, oldEmail)
	if err != nil {
		msg := fmt.Sprintf("%s. %v", failMsgPrefix, err)
		return errors.New(msg)
	}
	s4.Update(int(r4[0].Row), MmTransactionsSpreadsheetEmailCol, newEmail)

	// Change email in Stripe
	// Change email in Active Campaign
	// Add email to change email spreadsheet
	//err = AddChangeStudentEmailRow(teachableId, name string, oldEmail string, newEmail string,
	//		stripeId string, acId string, timestamp string, source string)

	return nil
}

func AddChangeStudentEmailRow(teachableId string, name string, oldEmail string, newEmail string, stripeId string, acId string, timestamp string, source string) error {
	/*
			var ChangeEmailSpreadsheetTeachableIdCol = 0
			var ChangeEmailSpreadsheetNameCol = 1
			var ChangeEmailSpreadsheetOldEmailCol = 2
			var ChangeEmailSpreadsheetNewEmailCol = 3
			var ChangeEmailSpreadsheetStripeIdCol = 4
			var ChangeEmailSpreadsheetAcIdCol = 5
			var ChangeEmailSpreadsheetTimestampCol = 6
			var ChangeEmailSpreadsheetSourceCol = 7
		gsheetwrap.AddRow(ChangeEmailSpreadsheetId, values []interface{
			teachableId, r1.}) error {
	*/
	return nil

}

func CancelSubscription(c *stripe.Customer) (*stripe.Subscription, error) {
	var sub *stripe.Subscription
	if len(c.Subscriptions.Data) > 0 {
		sub = c.Subscriptions.Data[0]
	} else {
		sub = nil
		msg := fmt.Sprintf("Could not find a subscription for \"%s\"",
			c.Email)
		return nil, errors.New(msg)
	}
	return stripewrap.CancelSubscription(sub.ID, false)
}

// Checks the 'sj_founder' metadata field in Stripe
func IsFounder(c *stripe.Customer) bool {
	if val, ok := c.Metadata["sj_founder"]; ok && val == "true" {
		return true
	}
	return false
}

// Checks the 'sj_billing_complete' metadata field in Stripe
func IsBillingComplete(c *stripe.Customer) bool {
	if val, ok := c.Metadata["sj_billing_complete"]; ok && val == "true" {
		return true
	}
	return false
}

func IsFounderPlan(planIdOrName string) bool {
	return strings.Contains(planIdOrName, "founder") ||
		strings.Contains(planIdOrName, "Founder")
}

/*
func (status *StudentStatus) IsBillingComplete() bool {
	return status.Status == "complete"
}
*/

func HasActiveSubscription(c *stripe.Customer) bool {
	if len(c.Subscriptions.Data) > 0 {
		sub := c.Subscriptions.Data[0]
		if sub.Billing == "charge_automatically" && sub.CanceledAt == 0 &&
			!sub.CancelAtPeriodEnd {
			return true
		}
	}
	return false
}
