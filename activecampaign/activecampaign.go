package activecampaign

// For ActiveCampaign v3 API

import (
	//"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/gosexy/to"
	"github.com/gosexy/yaml"

	"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var UserAgent = "Nancy Hillis Studio Webhook Backend (nancyhillis.com)"
var AdminUrlPrefix = "https://nancyhillis.activehosted.com/app"

// ActiveCampaign APIv3 URL: https://<your-account>.api-us1.com/api/3/
var UrlApiSuffix = "/api/3"
var UrlSuffixSubscribers = "/subscribers"
var UrlSuffixEvents = "/events"
var UrlSuffixContacts = "/contacts"

// Custom field names to IDs
// Fetch these with GET request:
// curl --request GET -H "Api-Token: ..." https://nancyhillis.api-us1.com/api/3/fields | jq '.'
var CustomFields = map[string]int32{
	"cid":     1,  // GA Client ID
	"rid":     8,  // Rainmaker ID
	"tid":     1,  // Teachable ID
	"sid":     9,  // Stripe ID
	"taj_csm": 11, // TAJ Campaign|Source_Medium
	"sj_csm":  -1, // SJ ...^...
	"ewc_csm": -1, // EWC ...^...
	"atc_csm": -1, // ATC ...^...
}

var SecretsFilePath = "/var/webhook/secrets/ac_secrets.yml"

func GetApiCredentials() (string, string) {
	secrets := yaml.New()
	secrets, err := yaml.Open(SecretsFilePath)
	if err != nil {
		log.Fatalf("Could not open YAML secrets file: %s", err.Error())
	}

	//accountId := to.String(secrets.Get("ACCOUNT_ID"))
	apiUrl := to.String(secrets.Get("API_URL"))
	apiToken := to.String(secrets.Get("API_TOKEN"))

	// Add suffix
	url := fmt.Sprintf("%s%s", apiUrl, UrlApiSuffix)

	return url, apiToken
}

func AuthenticateWithCredentials(url string, apiToken string) error {
	//fmt.Println("Auth URL:>", url)

	client := &http.Client{}
	//var jsonStr = []byte(`{"events":[{"email":"dagoodma@gmail.com","action":"TEst event"}]}`)
	//req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		//log.Fatalln(err)
		return err
	}
	//req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Api-Token", apiToken)

	resp, err := client.Do(req)
	if err != nil {
		//log.Fatalln(err)
		return err
	}

	defer resp.Body.Close()

	//log.Println("response Status:", resp.Status)
	//log.Println("response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		//log.Fatalln(err)
		return err
	}
	_ = body

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Got bad response status '%s', expected: %s", resp.StatusCode, 200)
		return errors.New(msg)
	}

	//log.Println("response Body:", string(body))
	return nil
}

func GetContactsByEmail(email string) (*ListContacts, error) {
	// Ensure there's an email to lookup
	if len(email) < 1 {
		msg := fmt.Sprintf("No email given to search contacts for.")
		return nil, errors.New(msg)
	}
	if !util.EmailLooksValid(email) {
		msg := fmt.Sprintf("Invalid email given: %s", email)
		return nil, errors.New(msg)
	}

	apiUrl, apiToken := GetApiCredentials()
	_ = apiToken
	// Build request URL
	urlBase := fmt.Sprintf("%s%s", apiUrl, UrlSuffixContacts)
	u, err := url.Parse(urlBase)
	if err != nil {
		msg := fmt.Sprintf("Failed parsing request base url '%s': %s", urlBase, err)
		return nil, errors.New(msg)
	}
	q := u.Query()
	q.Set("email", email)

	url := fmt.Sprintf("%s?%s", u.String(), q.Encode())

	//fmt.Println("GetContacts URL:>", url)

	client := &http.Client{}
	//var jsonStr = []byte(`{"events":[{"email":"dagoodma@gmail.com","action":"TEst event"}]}`)
	//req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		//log.Fatalln(err)
		return nil, err
	}
	//req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Api-Token", apiToken)

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
	_ = body

	if resp.StatusCode != 200 {
		msg := fmt.Sprintf("Got bad response status '%s', expected: %s", resp.StatusCode, 200)
		return nil, errors.New(msg)
	}
	//log.Println("response Body:", string(body))
	data := []byte(body)

	// Unmarshall the message
	l := &ListContacts{}
	err = json.Unmarshal(data, &l)
	if err != nil {
		msg := fmt.Sprintf("Failed to unmarshal GetContacts response data: %s", err)
		return nil, errors.New(msg)
	}

	//log.Printf("Got %d results in contact search.", l.Metadata.Total)

	/*
		if l.Metadata.Total < 1 {
			msg := fmt.Sprintf("Could not find any contacts with email: %s", email)
			return nil, errors.New(msg)
		}
	*/

	return l, nil
}

// This returns the simpler ListContactsContact instead of GetContact (for now)
func GetContactByEmail(email string) (*ListContactsContact, error) {
	r, err := GetContactsByEmail(email)
	if err != nil {
		msg := fmt.Sprintf("Failed to get contact '%s': %s", email, err)
		return nil, errors.New(msg)
	}
	if r.Metadata.Total < 1 || len(r.Contacts) < 1 {
		msg := fmt.Sprintf("No contacts found for: %s", email)
		return nil, errors.New(msg)
	}
	if r.Metadata.Total > 1 || len(r.Contacts) > 1 {
		msg := fmt.Sprintf("Found multiple contacts for: %s", email)
		return nil, errors.New(msg)
	}
	s := r.Contacts[0]

	return &s, nil
}

func GetContactProfileUrlById(id string) string {
	url := fmt.Sprintf("%s%s/%s", AdminUrlPrefix, UrlSuffixContacts, id)
	return url
}

func GetContactProfileUrlByEmail(email string) (string, error) {
	c, err := GetContactByEmail(email)
	if err != nil {
		msg := fmt.Sprintf("Failed to get profile url: %s", err)
		return "", errors.New(msg)
	}
	return GetContactProfileUrlById(c.Id), nil
	/*
		fmt.Printf("Here with '%s' (id=%s) and url='%s' contact: %v", email, c.Id, url, c)
		return url, nil
	*/
}

/*
func GetContact(id string) (error {
	apiUrl, apiToken := GetApiCredentials()
	err := AuthenticateWithCredentials(apiUrl, apiToken)
	if err != nil {

	}
}

*/

/*
 * Messages and unmarshallers
 */
// Retrieve a contact by ID
type RetrieveContact struct {
	Automations         []RetrieveContactAutomation        `json:"contactAutomations"`
	Data                []RetrieveContactData              `json:"contactData"`
	Lists               []RetrieveContactList              `json:"contactLists"`
	Deals               []RetrieveContactDeal              `json:"deals"`
	FieldValues         []RetrieveContactFieldValue        `json:"fieldValues"`
	GeographicIps       []RetrieveContactGeographicIp      `json:"geoIps"`
	GeographicAddresses []RetrieveContactGeographicAddress `json:"geoAddresses"`
	Contact             []RetrieveContactContact           `json:"contact"`
}

type _RetrieveContact RetrieveContact

type RetrieveContactAutomation struct {
	Contact           string                         `json:"contact"`
	SeriesId          string                         `json:"seriesid"`
	StartId           string                         `json:"startid"`
	Status            string                         `json:"status"`
	BatchId           string                         `json:"batchid"`
	AddDate           string                         `json:"adddate"`
	RemoveDate        string                         `json:"remdate"`
	TimeSpan          string                         `json:"timespan"`
	LastBlock         string                         `json:"lastblock"`
	LastLogId         string                         `json:"lastlogid"`
	LastDate          string                         `json:"lastdate"`
	CompletedElements string                         `json:"completedElements"`
	TotalElements     string                         `json:"totalElements"`
	Completed         string                         `json:"completed"`
	CompletedValue    string                         `json:"completeValue"`
	Links             RetrieveContactAutomationLinks `json:"links"`
	Id                string                         `json:"id"`
	Automation        string                         `json:"automation"`
}
type RetrieveContactAutomationLinks struct {
	Automation   string `json:"automation"`
	Contact      string `json:"contact"`
	ContactGoals string `json:"contactGoals"`
}

type RetrieveContactData struct {
	Contact                  string                 `json:"contact"`
	Timestamp                string                 `json:"tstamp"`
	GeographicTimestamp      string                 `json:"geoTstamp"`
	GeographicIpv4           string                 `json:"geoIp4"`
	GeographicCountry2       string                 `json:"geoCountry2"`
	GeographicCountry        string                 `json:"geo_country"`
	GeographicState          string                 `json:"geoState"`
	GeographicCity           string                 `json:"geoCity"`
	GeographicZipcode        string                 `json:"geoZip"`
	GeographicArea           string                 `json:"geoArea"`
	GeographicLatitude       string                 `json:"geoLat"`
	GeographicLongitude      string                 `json:"geoLon"`
	GeographicTimezone       string                 `json:"geoTz"`
	GeographicTimezoneOffset string                 `json:"geoTzOffset"`
	GaCampaignSource         string                 `json:"ga_campaign_source"`
	GaCampaignName           string                 `json:"ga_campaign_name"`
	GaCampaignMedium         string                 `json:"ga_campaign_medium"`
	GaCampaignTerm           string                 `json:"ga_campaign_term"`
	GaCampaignContent        string                 `json:"ga_campaign_content"`
	GaCampaignCustomSegment  string                 `json:"ga_campaign_customsegment"`
	GaFirstVisit             string                 `json:"ga_first_visit"`
	GaTimesVisited           string                 `json:"ga_times_visited"`
	FacebookId               string                 `json:"fb_id"`
	FacebookName             string                 `json:"fb_name"`
	TwitterId                string                 `json:"tw_id"`
	Links                    map[string]interface{} `json:"links"`
	Id                       string                 `json:"id"`
}

type RetrieveContactList struct {
	Contact               string                   `json:"contact"`
	List                  string                   `json:"list"`
	Form                  string                   `json:"form"`
	SeriesId              string                   `json:"seriesid"`
	SubscribeDate         string                   `json:"sdate"`
	UnsubscribeDate       string                   `json:"udate"`
	Status                string                   `json:"status"`
	Responder             string                   `json:"responder"`
	Sync                  string                   `json:"sync"`
	UnsubscribeReason     string                   `json:"unsubreason"`
	Campaign              string                   `json:"campaign"`
	Message               string                   `json:"message"`
	FirstName             string                   `json:"first_name"`
	LastName              string                   `json:"last_name"`
	Ipv4Subscribe         string                   `json:"ip4Sub"`
	SourceId              string                   `json:"sourceid"`
	AutoSyncLog           string                   `json:"autosyncLog"`
	Ipv4Last              string                   `json:"ip4_last"`
	Ipv4Unsubscribe       string                   `json:"ip4Unsub"`
	UnsubscribeAutomation string                   `json:"unsubscribeAutomation"`
	Links                 RetrieveContactListLinks `json:"links"`
	Id                    string                   `json:"id"`
	Automation            string                   `json:"automation"`
}

type RetrieveContactListLinks struct {
	Automation            string `json:"automation"`
	List                  string `json:"list"`
	Contact               string `json:"contact"`
	Form                  string `json:"form"`
	AutoSyncLog           string `json:"autosyncLog"`
	Campaign              string `json:"campaign"`
	UnsubscribeAutomation string `json:"unsubscribeAutomation"`
	Message               string `json:"message"`
	//Extra map[string]interface{}
}

type RetrieveContactDeal struct {
	// Nothing yet
	//Extra map[string]interface{}
}

type RetrieveContactFieldValue struct {
	Contact      string `json:"contact"`
	Field        string `json:"field"`
	Value        string `json:"value"`
	CreationDate string `json:"cdate"`
	UpdateDate   string `json:"udate"`

	Links RetrieveContactFieldValueLinks `json:"links"`
}

type RetrieveContactFieldValueLinks struct {
	Owner string `json:"owner"`
	Field string `json:"field"`
}

type RetrieveContactGeographicAddress struct {
}

type RetrieveContactGeographicIp struct {
}

type RetrieveContactContact struct {
}

// List all contacts
type ListContacts struct {
	ScoreValues []string              `json:"scoreValues"`
	Contacts    []ListContactsContact `json:"contacts"`
	Metadata    ListContactsMetadata  `json:"meta"`
}

type _ListContacts ListContacts

type ListContactsContact struct {
	CreationDate        string   `json:"cdate"`
	Email               string   `json:"email"`
	Phone               string   `json:"phone"`
	FirstName           string   `json:"firstName"`
	LastName            string   `json:"lastName"`
	OrganizationId      string   `json:"orgid"`
	SegmentIoId         string   `json:"segmentio_id"`
	BouncedHard         string   `json:"bounced_hard"`
	BouncedSoft         string   `json:"bounced_soft"`
	BouncedDate         string   `json:"bounced_date"`
	Ip                  string   `json:"ip"`
	Ua                  string   `json:"ua"`
	Hash                string   `json:"hash"`
	SocialDateLastCheck string   `json:"socialdata_lastcheck"`
	EmailLocal          string   `json:"email_local"`
	EmailDomain         string   `json:"email_domain"`
	SentCount           string   `json:"sentcnt"`
	RatingTimestamp     string   `json:"rating_tstamp"`
	Gravatar            string   `json:"gravatar"`
	Deleted             string   `json:"deleted"`
	Anonymized          string   `json:"anonymized"`
	AddDate             string   `json:"adate"`
	UpdateDate          string   `json:"udate"`
	EditDate            string   `json:"edate"`
	DeletedAt           string   `json:"deleted_at"`
	CreatedUtcTimestamp string   `json:"created_utc_timestamp"`
	UpdatedUtcTimestamp string   `json:"updated_utc_timestamp"`
	Id                  string   `json:"id"`
	ScoreValues         []string `json:"scoreValues"`
	Organization        string   `json:"organization"`

	Links ListContactContactLinks `json:"links"`
}

type ListContactContactLinks struct {
	BounceLogs         string `json:"bounceLogs"`
	ContactAutomations string `json:"contactAutomations"`
	ContactData        string `json:"contactData"`
	ContactGoals       string `json:"contactGoals"`
	ContactLists       string `json:"contactLists"`
	ContactLogs        string `json:"contactLogs"`
	ContactTags        string `json:"contactTags"`
	ContactDeals       string `json:"contactDeals"`
	Deals              string `json:"deals"`
	FieldValues        string `json:"fieldValues"`
	GeographicIps      string `json:"geoIps"`
	Notes              string `json:"notes"`
	Organization       string `json:"organization"`
	PlusAppend         string `json:"plusAppend"`
	TrackingLogs       string `json:"trackingLogs"`
	ScoreValues        string `json:"scoreValues"`
}

type ListContactsMetadata struct {
	TotalRaw  string `json:"total"`
	Total     uint64
	PageInput ListContactsMetadataPageInput `json:"page_input"`
}

type ListContactsMetadataPageInput struct {
	SegmentId  int32  `json:"segmentid"`
	FormId     int32  `json:"formid"`
	ListId     int32  `json:"listid"`
	TagId      int32  `json:"tagid"`
	Limit      int32  `json:"limit"`
	Offset     int32  `json:"offset"`
	Search     string `json:"search"`
	Sort       string `json:"sort"`
	SeriesId   int32  `json:"seriesid"`
	WaitId     int32  `json:"waitid"`
	Status     int32  `json:"status"`
	ForceQuery int32  `json:"forceQuery"`
	CacheId    string `json:"cacheid"`
	Email      string `json:"email"`
}

func (l *ListContacts) UnmarshalJSON(jsonStr []byte) error {
	l2 := _ListContacts{}

	err := json.Unmarshal(jsonStr, &l2)
	if err != nil {
		return err
	}

	l2.Metadata.Total = to.Uint64(l2.Metadata.TotalRaw)

	*l = ListContacts(l2)

	return nil
}
