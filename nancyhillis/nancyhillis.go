package nancyhillis

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/dagoodma/nancyhillis-go/gsheetwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	"bitbucket.org/dagoodma/nancyhillis-go/util"
	"gopkg.in/Iwark/spreadsheet.v2"
)

// Spreadsheet constants
var SjEnrollmentSpreadsheetId = "1wRHucYoRuGzHav7nK3V5Hv2Z4J67D_vTZN5wjw8aa2k"
var SjEnrollmentSpreadsheetStripeIdRow = 13

// Cancel after overdue grace period ends (as set in Stripe dashboard)
// TODO Load this from Stripe dashboard settings?
var OverdueGracePeriodDays = 21 // 3 weeks

/*
 * Customer status response
 */
// Json Header of customer SJ status response
type SjStudentStatus struct {
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

type _SjStudentStatus SjStudentStatus

var StripeAccountActiveStatuses = []string{"active", "trialing", "unpaid"} // rest are inactive
var AccountStatuses = [5]string{"active", "past_due", "canceling", "canceled", "unknown"}
var AccountStatusesHuman = map[string]string{
	"active":         "Active",
	"past_due":       "Overdue",
	"canceled":       "Canceled",
	"pending_cancel": "Canceling",
	"unknown":        "Unknown",
}

// --- Functions ---
// Find Stripe customer ID in spreadsheet
func GetSjEnrollmentRowByEmail(email string) ([]spreadsheet.Cell, error) {
	row, err := gsheetwrap.SearchForSingleRowWithValue(SjEnrollmentSpreadsheetId, email)
	if err != nil {
		msg := fmt.Sprintf("Error searching for email address '%s': %v", email, err)
		//log.Printf("Error while searching spreadsheet (%s) for '%s'. %v", SpreadsheetId, emailAddress, err)
		return nil, errors.New(msg)
	}
	if row == nil || len(row) < 14 {
		msg := fmt.Sprintf("Failed to find email address: %s", email)
		//log.Printf("Failed to find row in spreadsheet (%s) with '%s'", SpreadsheetId, emailAddress)
		return nil, errors.New(msg)
	}
	return row, nil
}

func GetSjStripeIdByEmail(email string) (string, error) {
	row, err := GetSjEnrollmentRowByEmail(email)
	if err != nil {
		return "", err
	}
	stripeCustomerId := row[SjEnrollmentSpreadsheetStripeIdRow].Value
	return stripeCustomerId, nil
}

func GetSjAccountStatus(stripeId string) (*SjStudentStatus, error) {
	if !stripewrap.CustomerIdLooksValid(stripeId) {
		msg := fmt.Sprintf("Invalid customer ID: %s", stripeId)
		return nil, errors.New(msg)
	}

	// Find Stripe customer ID in SJ spreadsheet
	// TODO determine if we need this or not

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

	// Start building response
	status := &SjStudentStatus{
		Email:         cust.Email,
		FirstName:     cust.Meta["first_name"],
		LastName:      cust.Meta["last_name"],
		IsDelinquent:  cust.Delinquent,
		CustomerId:    stripeId,
		NextBillHuman: "N/A",
	}
	// Fetch Stripe default payment source (card)
	if cust.DefaultSource != nil {
		cardId := cust.DefaultSource.ID
		c2, err := stripewrap.GetCard(cardId, stripeId)
		if err != nil || c2 == nil {
			msg := fmt.Sprintf("Failed retrieving customer card data. %v", err)
			return nil, errors.New(msg)
		}
		status.DefaultCardLastFour = c2.LastFour
		status.DefaultCardBrand = string(c2.Brand)
	}
	if len(cust.BusinessVatID) > 0 {
		status.BusinessVatId = cust.BusinessVatID
	}

	// Get stripe subscriptions
	if len(cust.Subs.Values) > 1 {
		msg := fmt.Sprintf("Found mulitple subscriptions for '%s'", cust.Email)
		return nil, errors.New(msg)
	}
	if len(cust.Subs.Values) == 1 {
		// TODO Add support for payment plan here
		sub := cust.Subs.Values[0]
		status.IsRecurring = true
		status.Created = sub.Created
		status.Start = sub.Start
		status.Ended = sub.Ended
		status.Canceled = sub.Canceled
		status.CancelAtEndOfPeriod = sub.EndCancel
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
		status.PeriodStart = sub.PeriodStart
		status.PeriodEnd = sub.PeriodEnd
		// Calculate days until period end
		et := time.Unix(sub.PeriodEnd, 0)
		ct := time.Now()
		dt := et.Sub(ct)
		status.DaysUntilEndOfPeriod = uint64(util.RoundDown(dt.Hours() / 24))
		// Billing interval and anchor
		status.BillingInterval = string(sub.Plan.Interval)
		status.BillingIntervalCount = sub.Plan.IntervalCount
		status.BillingCycleAnchor = sub.BillingCycleAnchor
		billingCycleTime := time.Unix(sub.BillingCycleAnchor, 0)
		status.BillingCycleAnchorHuman = billingCycleTime.Format("Mon Jan 2 15:04 2006")
		if sub.Billing == "charge_automatically" && status.Canceled == 0 && !status.CancelAtEndOfPeriod {
			status.DaysUntilDue = status.DaysUntilEndOfPeriod
			status.NextBillHuman = et.Format("Jan 2")
		} else {
			// Manual invoice
			status.DaysUntilDue = sub.DaysUntilDue
			status.NextBillHuman = "Manual"
		}
		// Figure out status of subscription
		subStatus := string(sub.Status)
		if util.StringSliceContains(StripeAccountActiveStatuses, subStatus) {
			status.Status = "active"
		} else if subStatus == "canceled" {
			status.Status = "canceled"
		} else if subStatus == "past_due" {
			status.Status = "past_due"
			// Calculate grace period days
			status.IsOverdue = true
			st := time.Unix(sub.PeriodStart, 0)
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
		status.RecurringPrice = sub.Plan.Amount
		status.Plan = sub.Plan.ID
		status.PlanHuman = sub.Plan.Nickname
		status.IsFounder = IsFounderPlan(sub.Plan.ID)
	} else {
		// Must be a package plan
		//l := stripewrap.GetChargeList(stripeId)
		ch, err := stripewrap.GetLastChargeWithPrefix(stripeId, "Studio Journey")
		if err != nil || ch == nil {
			msg := fmt.Sprintf("Failed to find active subscriptions or charges for '%s'. %v", cust.Email, err)
			return nil, errors.New(msg)
		}
		status.IsPackage = true
		status.IsRefunded = ch.Refunded
		status.PlanHuman = ch.Desc
		status.IsFounder = IsFounderPlan(ch.Desc)
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

func IsFounderPlan(planIdOrName string) bool {
	return strings.Contains(planIdOrName, "founder") ||
		strings.Contains(planIdOrName, "Founder")
}
