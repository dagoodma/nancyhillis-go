package gsheetwrap

import (
	//"encoding/json"
	"fmt"
	"io/ioutil"
	//"log"
	//"net/http"
	//"os"
	"errors"
	"strings"

	"github.com/jinzhu/copier"
	"golang.org/x/net/context"
	//"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	//"google.golang.org/api/sheets/v4"
	"gopkg.in/Iwark/spreadsheet.v2"

	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

/* Gsheet settings */
var SecretsFilePath = "/var/webhook/secrets/gsheet_client_secrets.json"

var ErrorOnMultipleMatches = false

func FetchSpreadsheet(spreadsheetId string) (*spreadsheet.Spreadsheet, error) {
	data, err := ioutil.ReadFile(SecretsFilePath)
	if err != nil {
		return nil, err
	}

	conf, err := google.JWTConfigFromJSON(data, spreadsheet.Scope)
	if err != nil {
		return nil, err
	}

	client := conf.Client(context.TODO())
	service := spreadsheet.NewServiceWithClient(client)

	s, err := service.FetchSpreadsheet(spreadsheetId)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func EnsureNoDuplicateRowByColumnValues(spreadsheetId string, columnNumber int) error {
	s, err := FetchSpreadsheet(spreadsheetId)
	if err != nil {
		return err
	}
	sheet, err := s.SheetByIndex(0)
	if err != nil {
		return err
	}

	// Find the value
	var columnValueToRows = map[string][]int{}
	var duplicateColumnValues []string
	for i, row := range sheet.Rows {
		for j, cell := range row {
			val := cell.Value
			if j == columnNumber && len(val) > 0 {
				if _, ok := columnValueToRows[val]; ok {
					if !util.StringSliceContains(duplicateColumnValues, val) {
						duplicateColumnValues = append(duplicateColumnValues, val)
					}
				} else {
					columnValueToRows[val] = make([]int, 1)
				}
				columnValueToRows[val] = append(columnValueToRows[val], i)
			}
		}
	}
	// TODO print duplicate column numbers for each value
	if len(duplicateColumnValues) > 0 {
		msg := fmt.Sprintf("Found rows with duplicate column '%d': %q", columnNumber, duplicateColumnValues)
		fmt.Println(msg)
		err := errors.New(msg)
		return err
	}
	//r := *p
	return nil
}

// Returns the last matching row that has the given value
func SearchForSingleRowWithValue(spreadsheetId string, myValue string) ([]spreadsheet.Cell, error) {
	if len(myValue) < 1 {
		return nil, errors.New("No value given to search for")
	}
	s, err := FetchSpreadsheet(spreadsheetId)
	if err != nil {
		return nil, err
	}
	sheet, err := s.SheetByIndex(0)
	if err != nil {
		return nil, err
	}

	// Find the value
	//var p *[]spreadsheet.Cell = nil
	var r []spreadsheet.Cell = nil
	for _, row := range sheet.Rows {
		for _, cell := range row {
			if strings.EqualFold(cell.Value, myValue) {
				if ErrorOnMultipleMatches {
					if r != nil {
						err := errors.New("Found multiple matches")
						return nil, err
					}
				}
				// Otherwise keep going and just get the last match (highest row/col)
				copier.Copy(&r, &row)
				//fmt.Println("Found row: %v", row)
				//return row, nil
			}
		}
	}
	//r := *p
	return r, nil
}

// Returns the last matching row where the specified column has the given value
func SearchForSingleRowWithValueInColumn(spreadsheetId string, columnNumber int, myValue string) ([]spreadsheet.Cell, error) {
	if len(myValue) < 1 {
		return nil, errors.New("No value given to search for")
	}
	s, err := FetchSpreadsheet(spreadsheetId)
	if err != nil {
		return nil, err
	}
	sheet, err := s.SheetByIndex(0)
	if err != nil {
		return nil, err
	}

	// Find the value
	var r []spreadsheet.Cell = nil
	for _, row := range sheet.Rows {
		if len(row) < columnNumber {
			msg := fmt.Sprintf("The sheet has only %d columns, expected at least %d", len(row), columnNumber)
			return nil, errors.New(msg)
		}
		val := row[columnNumber].Value
		if strings.EqualFold(val, myValue) {
			if ErrorOnMultipleMatches {
				if r != nil {
					err := errors.New("Found multiple matches")
					return nil, err
				}
			}
			// Otherwise keep going and just get the last match (highest row/col)
			copier.Copy(&r, &row)
		}
	}
	return r, nil
}

// Returns the matching rows with the given value
func SearchForAllRowsWithValueInColumn(spreadsheetId string, columnNumber int, myValue string) ([][]spreadsheet.Cell, error) {
	if len(myValue) < 1 {
		return nil, errors.New("No value given to search for")
	}
	s, err := FetchSpreadsheet(spreadsheetId)
	if err != nil {
		return nil, err
	}
	sheet, err := s.SheetByIndex(0)
	if err != nil {
		return nil, err
	}

	// Find the value
	var rows [][]spreadsheet.Cell
	for i, row := range sheet.Rows {
		if len(row) < columnNumber {
			msg := fmt.Sprintf("The sheet has only %d columns, expected at least %d", len(row), columnNumber)
			return nil, errors.New(msg)
		}
		_ = i
		val := row[columnNumber].Value
		if strings.EqualFold(val, myValue) {
			// Otherwise keep going and just get the last match (highest row/col)
			//fmt.Printf("HERE! [%d][%d]: %q\n", i, columnNumber, val)
			var r []spreadsheet.Cell = nil
			copier.Copy(&r, &row)
			rows = append(rows, r)
		}
	}
	return rows, nil
}

/*
// Returns all matching rows with the given value
func SearchForAllRowsWithValue(spreadsheetId string, columnNumber int, myValue string) ([][]spreadsheet.Cell, error) {
	if len(myValue) < 1 {
		return nil, errors.New("No value given to search for")
	}
	s, err := FetchSpreadsheet(spreadsheetId)
	if err != nil {
		return nil, err
	}
	sheet, err := s.SheetByIndex(0)
	if err != nil {
		return nil, err
	}

	// Find the value
	//var p *[]spreadsheet.Cell = nil
	var rows [][]spreadsheet.Cell
	for _, row := range sheet.Rows {
		if len(row) < columnNumber {
			msg := fmt.Sprintf("The sheet has only %d columns, expected at least %d", len(row), columnNumber)
			return nil, errors.New(msg)
		}
		val := row[columnNumber].Value
		if strings.EqualFold(val, myValue) {
			// Otherwise keep going and just get the last match (highest row/col)
			var r []spreadsheet.Cell = nil
			copier.Copy(&r, &row)
			rows = append(rows, r)
		}
	}
	return rows, nil
}
*/
