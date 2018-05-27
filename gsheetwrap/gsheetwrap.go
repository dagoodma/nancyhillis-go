package gsheetwrap

import (
	//"encoding/json"
	//"fmt"
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
)

/* Gsheet settings */
var SecretsFilePath = "/var/webhook/secrets/gsheet_client_secrets.json"

func SearchForSingleRowWithValue(spreadsheetId string, myValue string) ([]spreadsheet.Cell, error) {
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
				if r != nil {
					err := errors.New("Found multiple matches")
					return nil, err
				}
				//p = &row
				copier.Copy(&r, &row)
				//fmt.Println("Found row: %v", row)
				//return row, nil
			}
		}
	}
	//r := *p
	return r, nil
}
