package membermouse

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	//"strings"
	"encoding/json"
	"strconv"

	"github.com/gosexy/to"
	"github.com/gosexy/yaml"
	//"bitbucket.org/dagoodma/nancyhillis-go/util"
)

// Membermouse constants (for API access)
var UserAgent = "Nancy Hillis Studio Webhook Backend (wbhk97.nancyhillis.com)"
var ApiUrlPrefix = "https://www.studiojourney.com/wp-content/plugins/membermouse/api/request.php"
var ApiUrlSuffixGetMember = "?q=/getMember"
var ApiUrlSuffixUpdateMember = "?q=/updateMember"

var ManageMemberUrlPrefix = "https://www.studiojourney.com/wp-admin/admin.php?page=manage_members&module=details_general&user_id="

var SecretsFilePath = "/var/webhook/secrets/membermouse_secrets.yml"

var MigratedFlagId = 1
var MigratedFlagTrueValue = "mm_cb_on"

func GetApiCredentials() (string, string) {
	secrets := yaml.New()
	secrets, err := yaml.Open(SecretsFilePath)
	if err != nil {
		log.Fatalf("Could not open YAML secrets file: %s", err.Error())
	}

	apiKey := to.String(secrets.Get("API_KEY"))
	apiPassword := to.String(secrets.Get("API_PASSWORD"))

	return apiKey, apiPassword
}

/*
 * Member mouse API responses
 */
type GetMemberResponse struct {
	Code    string  `json:"response_code"`
	Message string  `json:"response_message"`
	Data    *Member `json:"response_data"`
}

type UpdateMemberResponse struct {
	Code    string `json:"response_code"`
	Message string `json:"response_message"`
	//Data    map[string]interface{} `json:"response_data"` // ignore this
}

// Get Member response
type Member struct {
	MemberId            uint64        `json:"member_id"`
	FirstName           string        `json:"first_name"`
	LastName            string        `json:"last_name"`
	Registered          string        `json:"registered"`
	CancellationDate    string        `json:"cancellation_date"`
	LastLoggedIn        string        `json:"last_logged_in"`
	LastUpdated         string        `json:"last_updated"`
	DaysAsMember        uint64        `json:"days_as_member"`
	Status              string        `json:"status"`
	StatusName          string        `json:"status_name"`
	IsComplimentary     string        `json:"is_complimentary"`
	MembershipLevel     string        `json:"membership_level"`
	MembershipLevelName string        `json:"membership_level_name"`
	Username            string        `json:"username"`
	Email               string        `json:"email"`
	Phone               string        `json:"phone"`
	CustomFields        []CustomField `json:"custom_fields"`
	BillingCountry      string        `json:"billing_country"`
	//Extra               map[string]interface{} `json:"-"`
}

type CustomField struct {
	Id    uint64 `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (m *Member) GetManageMemberUrl() string {
	url := fmt.Sprintf("%s%d", ManageMemberUrlPrefix, m.MemberId)
	return url
}

func (m *Member) GetStatusCode() uint64 {
	if u, err := strconv.ParseUint(m.Status, 10, 64); err == nil {
		return u
	}
	return 0
}

func (m *Member) IsComped() bool {
	return m.IsComplimentary == "true"
}

func (m *Member) IsActive() bool {
	return m.Status == "1"
}

func (m *Member) IsCanceled() bool {
	return m.Status == "2"
}

func (m *Member) IsOverdue() bool {
	return m.Status == "5"
}

func (m *Member) IsPendingCancel() bool {
	return m.Status == "9"
}

func (m *Member) IsMigrated() bool {
	c := m.GetCustomFieldByName("Migrated")
	if c != nil && c.Value == MigratedFlagTrueValue {
		return true
	}
	return false
}

/*
func (m *Member) SetCustomFieldByName(name string, value string) error {
	c := m.GetCustomFieldByName(name)
	if c == nil {
		msg := fmt.Sprintf("No custom field to set with name: %s", name)
		return errors.New(msg)
	}
	c.Value = value
	return nil
}
*/

func (m *Member) GetCustomFieldByName(name string) *CustomField {
	for _, c := range m.CustomFields {
		if c.Name == name {
			return &c
		}
	}
	return nil
}

func (m *Member) FlagFounderMigrated() error {
	apiKey, apiPassword := GetApiCredentials()

	url := fmt.Sprintf("%s%s", ApiUrlPrefix, ApiUrlSuffixUpdateMember)
	//fmt.Println("URL:>", url)

	var jsonStr = fmt.Sprintf("apikey=%s&apisecret=%s&member_id=%d&custom_field_%d=%s",
		apiKey, apiPassword, m.MemberId, MigratedFlagId, MigratedFlagTrueValue)
	var jsonData = []byte(jsonStr)

	body, err := DoApiRequest(url, jsonData)
	if err != nil {
		return err
	}
	//fmt.Printf("HERE!  with=%v\n", m)
	//fmt.Printf("HERE! %s or err=%v\n", string(body), err)
	//return nil

	r := UpdateMemberResponse{}
	err = json.Unmarshal([]byte(body), &r)
	if err != nil {
		return err
	}

	//if r.Code != "200" || r.Message != "" || r.Data == nil {
	if r.Code != "200" || r.Message != "" {
		msg := fmt.Sprintf("Got error (%s). %v", r.Code, r.Message)
		return errors.New(msg)
	}

	//log.Printf("%v", r)
	return nil
}

func GetMemberByEmail(email string) (*Member, error) {
	apiKey, apiPassword := GetApiCredentials()

	url := fmt.Sprintf("%s%s", ApiUrlPrefix, ApiUrlSuffixGetMember)
	//fmt.Println("URL:>", url)

	var jsonStr = fmt.Sprintf("apikey=%s&apisecret=%s&email=%s", apiKey, apiPassword, email)
	var jsonData = []byte(jsonStr)

	body, err := DoApiRequest(url, jsonData)
	if err != nil {
		return nil, err
	}

	r := GetMemberResponse{}
	err = json.Unmarshal([]byte(body), &r)
	if err != nil {
		return nil, err
	}

	if r.Code != "200" || r.Message != "" || r.Data == nil {
		msg := fmt.Sprintf("Got error (%s). %v", r.Code, r.Message)
		return nil, errors.New(msg)
	}

	//log.Printf("%v", r.Data)
	return r.Data, nil
}

func DoApiRequest(url string, jsonData []byte) ([]byte, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		//log.Fatalln(err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		//log.Fatalln(err)
		return nil, err
	}

	defer resp.Body.Close()

	//log.Println("response Status:", resp.Status)
	//log.Println("response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//log.Fatalln(err)
		return nil, err
	}
	//log.Println("response Body:", string(body))

	return body, nil
}

/*
 * Status response
 */
type MemberStatus struct {
	IsMigrated      bool   `json:"is_migrated"`
	IsComped        bool   `json:"is_comped"`
	IsActive        bool   `json:"is_active"`
	IsCanceled      bool   `json:"is_canceled"`
	IsPendingCancel bool   `json:"is_pending_cancel"`
	IsOverdue       bool   `json:"is_overdue"`
	Status          uint64 `json:"status"`
	StatusHuman     string `json:"status_human"`
	Phone           string `json:"phone"`
	DaysAsMember    uint64 `json:"days_as_member"`
	LastLogin       string `json:"last_logged_in"`
	LastUpdate      string `json:"last_updated"`
	Started         string `json:"started"`
	Canceled        string `json:"canceled"`
	MemberId        uint64 `json:"member_id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Email           string `json:"email"`
	BillingCountry  string `json:"billing_country"`
}

func (m *Member) GetStatus() (*MemberStatus, error) {
	status := MemberStatus{
		IsMigrated:      m.IsMigrated(),
		IsComped:        m.IsComped(),
		IsActive:        m.IsActive(),
		IsCanceled:      m.IsCanceled(),
		IsPendingCancel: m.IsPendingCancel(),
		IsOverdue:       m.IsOverdue(),
		Status:          m.GetStatusCode(),
		StatusHuman:     m.StatusName,
		Phone:           m.Phone,
		DaysAsMember:    m.DaysAsMember,
		LastLogin:       m.LastLoggedIn,
		LastUpdate:      m.LastLoggedIn,
		Started:         m.Registered,
		Canceled:        m.CancellationDate,
		MemberId:        m.MemberId,
		FirstName:       m.FirstName,
		LastName:        m.LastName,
		Email:           m.Username,
		BillingCountry:  m.BillingCountry,
	}

	return &status, nil
}
