package gsheetwrap

import (
	"testing"
	//"fmt"

	"bitbucket.org/dagoodma/nancyhillis-go/gsheetwrap"
	"bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
)

var SjEnrollmentSpreadsheetId = studiojourney.EnrollmentSpreadsheetId
var SjFounderMigratedSpreadsheetId = studiojourney.FounderMigratedSpreadsheetId
var SjBillingSpreadsheetId = studiojourney.BillingSpreadsheetId
var SjEnrollmentSpreadsheetEmailCol = studiojourney.EnrollmentSpreadsheetEmailCol
var SjEnrollmentSpreadsheetNameCol = studiojourney.EnrollmentSpreadsheetNameCol
var SjFounderMigratedSpreadsheetEmailCol = studiojourney.FounderMigratedSpreadsheetEmailCol
var SjBillingSpreadsheetEmailCol = studiojourney.BillingSpreadseheetEmailCol

func TestFetchSpreadsheet(t *testing.T) {
	cases := []struct {
		sheetId   string
		wantError bool
	}{
		{SjEnrollmentSpreadsheetId, false},
		{SjFounderMigratedSpreadsheetId, false},
		{SjBillingSpreadsheetId, false},
		{"blah_blah_blah", true},
		{"1", true},
		{"", true},
	}
	for _, c := range cases {
		s, err := gsheetwrap.FetchSpreadsheet(c.sheetId)
		var gotSpreadsheet = s != nil
		var gotError = err != nil
		var wantSpreadsheet = !c.wantError
		if gotError && !c.wantError || !gotError && c.wantError || !gotSpreadsheet && wantSpreadsheet {
			t.Errorf("FetchSpreadsheet(%q) == (spreadsheet=%t, error=%t), want (spreadsheet=%t, error=%t)",
				c.sheetId, gotSpreadsheet, gotError, wantSpreadsheet, c.wantError)
		}
	}
}

func TestSearchForSingleRowWithValue(t *testing.T) {
	cases := []struct {
		sheetId   string
		value     string
		wantRow   bool
		wantError bool
	}{
		{SjEnrollmentSpreadsheetId, "davie.goodman.music@gmail.com", true, false},
		{SjEnrollmentSpreadsheetId, "brent.h.bailey@gmail.com", true, false},
		{SjEnrollmentSpreadsheetId, "brelix@gmail.com", false, false},
		{SjEnrollmentSpreadsheetId, "", false, true},
		{SjEnrollmentSpreadsheetId, " ", false, false},
		{SjFounderMigratedSpreadsheetId, "davie.goodman.music@gmail.com", true, false},
		{SjFounderMigratedSpreadsheetId, "brent.h.bailey@gmail.com", false, false},
		{SjFounderMigratedSpreadsheetId, "brelix@gmail.com", false, false},
	}
	for _, c := range cases {
		row, err := gsheetwrap.SearchForSingleRowWithValue(c.sheetId, c.value)
		var gotRow = row != nil
		var gotError = err != nil
		if gotError && !c.wantError || !gotError && c.wantError || !gotRow && c.wantRow || gotRow && !c.wantRow {
			t.Errorf("SearchForSingleRowWithValue(%q, %q) == (spreadsheet=%t, error=%t), want (spreadsheet=%t, error=%t)",
				c.sheetId, c.value, gotRow, gotError, c.wantRow, c.wantError)
		}
		if gotRow && c.wantRow {
			if len(row) < (SjEnrollmentSpreadsheetEmailCol + 1) {
				t.Errorf("SearchForSingleRowWithValue(%q, %q) returned too few columns %d, expected at least %d",
					c.sheetId, c.value, len(row), SjEnrollmentSpreadsheetEmailCol+1)
			}
			gotEmail := row[SjEnrollmentSpreadsheetEmailCol].Value
			if gotEmail != c.value {
				t.Errorf("SearchForSingleRowWithValue(%q, %q) returned row with incorrect email value '%q', expected: %q",
					c.sheetId, c.value, gotEmail, c.value)
			}
		}
	}
}

func TestSearchForSingleRowWithValueInColumn(t *testing.T) {
	cases := []struct {
		sheetId               string
		value                 string
		columnNumberWithValue int
		wantRow               bool
		wantError             bool
	}{
		{SjEnrollmentSpreadsheetId, "davie.goodman.music@gmail.com", SjEnrollmentSpreadsheetEmailCol, true, false},
		{SjEnrollmentSpreadsheetId, "brent.h.bailey@gmail.com", SjEnrollmentSpreadsheetEmailCol, true, false},
		{SjEnrollmentSpreadsheetId, "brelix@gmail.com", SjEnrollmentSpreadsheetEmailCol, false, false},
		{SjEnrollmentSpreadsheetId, "", SjEnrollmentSpreadsheetEmailCol, false, true},
		{SjEnrollmentSpreadsheetId, " ", SjEnrollmentSpreadsheetEmailCol, false, false},
		{SjFounderMigratedSpreadsheetId, "davie.goodman.music@gmail.com", SjEnrollmentSpreadsheetEmailCol, true, false},
		{SjFounderMigratedSpreadsheetId, "brent.h.bailey@gmail.com", SjEnrollmentSpreadsheetEmailCol, false, false},
		{SjFounderMigratedSpreadsheetId, "brelix@gmail.com", SjEnrollmentSpreadsheetEmailCol, false, false},
	}
	for _, c := range cases {
		row, err := gsheetwrap.SearchForSingleRowWithValueInColumn(c.sheetId, c.columnNumberWithValue, c.value)
		var gotRow = row != nil
		var gotError = err != nil
		if gotError && !c.wantError || !gotError && c.wantError || !gotRow && c.wantRow || gotRow && !c.wantRow {
			t.Errorf("SearchForSingleRowWithValueInColumn(%q, %q, %q) == (row=%t, error=%t), want (row=%t, error=%t)",
				c.sheetId, c.columnNumberWithValue, c.value, gotRow, gotError, c.wantRow, c.wantError)
		}
		if gotRow && c.wantRow {
			if len(row) < (SjEnrollmentSpreadsheetEmailCol + 1) {
				t.Errorf("SearchForSingleRowWithValueInColumn(%q, %q, %q) returned too few columns %d, expected at least %d",
					c.sheetId, c.columnNumberWithValue, c.value, len(row), SjEnrollmentSpreadsheetEmailCol+1)
			}
			gotEmail := row[SjEnrollmentSpreadsheetEmailCol].Value
			if gotEmail != c.value {
				t.Errorf("SearchForSingleRowWithValueInColumn(%q, %q, %q) returned row with incorrect email value '%q', expected: %q",
					c.sheetId, c.columnNumberWithValue, c.value, gotEmail, c.value)
			}
		}
	}
}

func TestEnsureNoDuplicateRowByColumnValues(t *testing.T) {
	cases := []struct {
		sheetId      string
		columnNumber int
		wantError    bool
	}{
		{SjEnrollmentSpreadsheetId, SjEnrollmentSpreadsheetEmailCol, false},
		{SjFounderMigratedSpreadsheetId, SjFounderMigratedSpreadsheetEmailCol, false},
		{SjBillingSpreadsheetId, SjBillingSpreadsheetEmailCol, false},
	}
	for _, c := range cases {
		err := gsheetwrap.EnsureNoDuplicateRowByColumnValues(c.sheetId, c.columnNumber)
		var gotError = err != nil
		if gotError && !c.wantError {
			t.Errorf("EnsureNoDuplicateRowByColumnValues(%q) == (error=%t), want (error=%t)",
				c.sheetId, gotError, c.wantError)
		}
	}
}

func TestSearchForAllRowsWithValueInColumn(t *testing.T) {
	cases := []struct {
		sheetId               string
		value                 string
		columnNumberWithValue int
		wantRowCount          int
		wantError             bool
	}{
		{SjEnrollmentSpreadsheetId, "davie.goodman.music@gmail.com", SjEnrollmentSpreadsheetEmailCol, 1, false},
		{SjEnrollmentSpreadsheetId, "brent.h.bailey@gmail.com", SjEnrollmentSpreadsheetEmailCol, 1, false},
		{SjEnrollmentSpreadsheetId, "brelix@gmail.com", SjEnrollmentSpreadsheetEmailCol, 0, false},
		{SjEnrollmentSpreadsheetId, "", SjEnrollmentSpreadsheetEmailCol, 0, true},
		{SjEnrollmentSpreadsheetId, " ", SjEnrollmentSpreadsheetEmailCol, 0, false},
		{SjFounderMigratedSpreadsheetId, "davie.goodman.music@gmail.com", SjEnrollmentSpreadsheetEmailCol, 1, false},
		{SjFounderMigratedSpreadsheetId, "brent.h.bailey@gmail.com", SjEnrollmentSpreadsheetEmailCol, 0, false},
		{SjFounderMigratedSpreadsheetId, "brelix@gmail.com", SjEnrollmentSpreadsheetEmailCol, 0, false},
		{SjEnrollmentSpreadsheetId, "Davie Goodman", SjEnrollmentSpreadsheetNameCol, 2, false},
	}
	for _, c := range cases {
		rows, err := gsheetwrap.SearchForAllRowsWithValueInColumn(c.sheetId, c.columnNumberWithValue, c.value)
		var gotRows = len(rows) > 0
		var gotError = err != nil
		if gotError && !c.wantError || !gotError && c.wantError || !gotRows && c.wantRowCount > 0 || gotRows && c.wantRowCount < 1 {
			t.Errorf("SearchForAllRowsWithValueInColumn(%q, %d, %q) == (rows=%t, error=%t), want (rows=%t, error=%t)",
				c.sheetId, c.columnNumberWithValue, c.value, gotRows, gotError, c.wantRowCount > 0, c.wantError)
		}
		if gotRows && c.wantRowCount > 0 {
			if len(rows[0]) < (SjEnrollmentSpreadsheetEmailCol + 1) {
				t.Errorf("SearchForAllRowsWithValueInColumn(%q, %d, %q) returned too few columns %d, expected at least %d",
					c.sheetId, c.columnNumberWithValue, c.value, len(rows[0]), SjEnrollmentSpreadsheetNameCol+1)
			}
			if len(rows) < c.wantRowCount {
				t.Errorf("SearchForAllRowsWithValueInColumn(%q, %d, %q) returned too few rows %d, expected %d",
					c.sheetId, c.columnNumberWithValue, c.value, len(rows), c.wantRowCount)
			}
			for i, row := range rows {
				gotValue := row[c.columnNumberWithValue].Value
				if gotValue != c.value {
					t.Errorf("SearchForAllRowsWithValueInColumn(%q, %d, %q) returned %d rows and row %d had incorrect value '%q', expected: %q",
						c.sheetId, c.columnNumberWithValue, c.value, len(rows), i, gotValue, c.value)
				}
			}
		}
	}
}
