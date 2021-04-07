package activecampaign

// For ActiveCampaign v3 API

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "math"
    "strings"
    "regexp"

    "github.com/xiam/to"
    "gopkg.in/yaml.v2"

    "bitbucket.org/dagoodma/dagoodma-go/util"
)

var DEBUG = false
var DEBUG_VERBOSE = false
var SAVE_API_KEY = true // whether to save and re-use the API key

const (
    USER_AGENT = "Nancy Hillis Studio Webhook Backend (nancyhillis.com)"
    ADMIN_PREFIX_URL_FORMAT = "https://%s.activehosted.com/app"
)

const (
    // ActiveCampaign APIv3 URL: https://<your-account>.api-us1.com/api/3/
    API_URL_SUFFIX = "/api/3"
    API_URL_SUBSCRIBERS = "/subscribers"
    API_URL_CONTACTS = "/contacts"
    API_URL_NOTES = "/notes"
    API_URL_TAGS = "/tags"
    API_URL_AUTOMATIONS = "/automations"
    API_URL_EVENTS = "/events"
    API_URL_CONTACT_TAGS = "/contactTags"
)

const (
    API_LIMIT_MAXIMUM = 100
    API_LIMIT_MINIMUM = 5
    REQUEST_CONCURRENCY_LIMIT = 4 // maximum number of concurrent requests
)

var RegexOffsetParameter = regexp.MustCompile(`(offset)=(\d+)`)

// Custom field names to IDs
// Fetch these with GET request:
// curl --request GET -H "Api-Token: ..." https://nancyhillis.api-us1.com/api/3/fields | jq '.'
/*
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
*/

var SecretsFilePath = "/var/webhook/secrets/ac_secrets.yml"

type SecretsConfig struct {
    AccountId   string `yaml:"ACCOUNT_ID"`
    ApiUrl      string `yaml:"API_URL"`
    ApiToken    string `yaml:"API_TOKEN"`
}

var SavedSecretsConfig *SecretsConfig

func (c *SecretsConfig) GetSecrets(filePath string) (error) {
    if SAVE_API_KEY && SavedSecretsConfig != nil {
        c.AccountId = SavedSecretsConfig.AccountId
        c.ApiUrl = SavedSecretsConfig.ApiUrl
        c.ApiToken = SavedSecretsConfig.ApiToken
        return nil
    }

    yamlFile, err := ioutil.ReadFile(filePath)
    if err != nil {
        log.Printf("Failed reading screts file: %s ", err)
    }
    err = yaml.Unmarshal(yamlFile, c)
    if err != nil {
        log.Fatalf("Failed unmarshaling secrets file '%s': %s", filePath, err)
        return err
    }

    if SAVE_API_KEY {
        SavedSecretsConfig = c
    }

    return nil
}

func GetApiCredentials() (string, string) {
    var c SecretsConfig
    err := c.GetSecrets(SecretsFilePath)
    if err != nil {
        log.Fatalf("Could not open YAML secrets file: %s", err.Error())
    }

    accountId := c.AccountId
    _ = accountId
    apiUrl := c.ApiUrl
    apiToken := c.ApiToken

    // Add suffix
    url := fmt.Sprintf("%s%s", apiUrl, API_URL_SUFFIX)

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
    req.Header.Set("User-Agent", USER_AGENT)
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

    return nil
}

func BuildRequestUrl(apiUrl string, requestSuffix string, pathParts ...string) (*url.URL, error) {
    urlRaw := fmt.Sprintf("%s%s", apiUrl, requestSuffix)
    for _, v := range pathParts {
        if !strings.HasPrefix(v, "/") {
            v = fmt.Sprintf("/%s", v)
        }
        urlRaw = fmt.Sprintf("%s%s", urlRaw, v)
    }
    u, err := url.ParseRequestURI(urlRaw)
    if err != nil {
        msg := fmt.Sprintf("Failed parsing request url '%s': %s", urlRaw, err)
        return nil, errors.New(msg)
    }
    if u.Host == "" {
        msg := fmt.Sprintf("Request url '%s' is missing host", u)
        return nil, errors.New(msg)
    }
    if u.Scheme == "" {
        msg := fmt.Sprintf("Request url '%s' is missing scheme", u)
        return nil, errors.New(msg)
    }
    return u, nil
}

type QueryParameters struct {
    Email       string
    Id          int
    TagId       int
    TagName     string
    Limit       int
    Offset      int
    FetchAll    bool
    AutomationName string
    ExactMatch  bool
}

func BuildQueryWithParams(params QueryParameters) (*url.Values, error) {
    q := url.Values{}
    if params.Email != "" {
        var email = params.Email
        // Ensure there's an email to lookup
        if len(email) < 1 {
            msg := fmt.Sprintf("No email given to search contacts for.")
            return nil, errors.New(msg)
        }
        if !util.EmailLooksValid(email) {
            msg := fmt.Sprintf("Invalid email given: %s", email)
            return nil, errors.New(msg)
        }

        q.Set("email", email)
    }
    if params.TagId != 0 {
        q.Set("tagid", to.String(params.TagId))
    }
    //if len(params.TagName) > 0 {
    if params.TagName != "" {
        q.Set("search", params.TagName)
    }
    if params.Limit != 0 {
        limit := params.Limit
        if limit > API_LIMIT_MAXIMUM {
            limit = API_LIMIT_MAXIMUM
        }
        q.Set("limit", to.String(limit))
    }
    return &q, nil
}

type ApiRequestResult struct {
    Data []byte
    Error error
}

// Handled a POST request with JSON data
func DoApiRequestPost(requestUrl string, apiToken string, data []byte) (*ApiRequestResult) {
    r := &ApiRequestResult{Data: nil, Error: nil}

    if DEBUG {
        log.Println(fmt.Sprintf("Putting data %d to url: %s", requestUrl))
    }

    client := &http.Client{}
    req, err := http.NewRequest(http.MethodPost, requestUrl, bytes.NewBuffer(data))
    if err != nil {
        r.Error = err
        return r
    }
    req.Header.Set("Content-Type", "application/json; charset=utf-8")
    //req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", USER_AGENT)
    req.Header.Set("Api-Token", apiToken)

    resp, err := client.Do(req)
    if err != nil {
        r.Error = err
        return r
    }

    defer resp.Body.Close()

    //log.Println("response Status:", resp.Status)
    //log.Println("response Headers:", resp.Header)

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        r.Error = err
        return r
    }
    //log.Println("Got body: ", to.String(body))

    if resp.StatusCode != 201 {
        msg := fmt.Sprintf("Got bad response status '%s', expected: %s", resp.StatusCode, 200)
        r.Error = errors.New(msg)
        return r
    }
    r.Data = []byte(body)
    return r
}
// Handled a PUT request with JSON data
func DoApiRequestPut(requestUrl string, apiToken string, data []byte) (*ApiRequestResult) {
    r := &ApiRequestResult{Data: nil, Error: nil}

    if DEBUG {
        log.Println(fmt.Sprintf("Putting data %d to url: %s", requestUrl))
    }

    client := &http.Client{}
    req, err := http.NewRequest(http.MethodPut, requestUrl, bytes.NewBuffer(data))
    if err != nil {
        r.Error = err
        return r
    }
    req.Header.Set("Content-Type", "application/json; charset=utf-8")
    //req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", USER_AGENT)
    req.Header.Set("Api-Token", apiToken)

    resp, err := client.Do(req)
    if err != nil {
        r.Error = err
        return r
    }

    defer resp.Body.Close()

    //log.Println("response Status:", resp.Status)
    //log.Println("response Headers:", resp.Header)

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        r.Error = err
        return r
    }
    //log.Println("Got body: ", to.String(body))

    if resp.StatusCode != 200 {
        msg := fmt.Sprintf("Got bad response status '%s', expected: %s", resp.StatusCode, 200)
        r.Error = errors.New(msg)
        return r
    }
    r.Data = []byte(body)
    return r
}


func DoApiRequestGet(requestUrl string, apiToken string) (*ApiRequestResult) {
    r := &ApiRequestResult{Data: nil, Error: nil}

    if DEBUG {
        log.Println(fmt.Sprintf("Querying url: %s", requestUrl))
    }

    client := &http.Client{}
    req, err := http.NewRequest("GET", requestUrl, nil)
    if err != nil {
        r.Error = err
        return r
    }
    //req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", USER_AGENT)
    req.Header.Set("Api-Token", apiToken)

    resp, err := client.Do(req)
    if err != nil {
        r.Error = err
        return r
    }

    defer resp.Body.Close()

    //log.Println("response Status:", resp.Status)
    //log.Println("response Headers:", resp.Header)

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        r.Error = err
        return r
    }
    //log.Println("Got body: ", to.String(body))

    if resp.StatusCode != 200 {
        msg := fmt.Sprintf("Got bad response status '%s', expected: %s", resp.StatusCode, 200)
        r.Error = errors.New(msg)
        return r
    }
    r.Data = []byte(body)
    return r
}

// r, the ListResponse interface, will be populated, but the list within will be missing data
// Refs:
// - https://guzalexander.com/2013/12/06/golang-channels-tutorial.html
// - https://gist.github.com/montanaflynn/ea4b92ed640f790c4b9cee36046a5383
func FetchAllEndpointDataAsync(u *url.URL, q *url.Values, r ListResponse, apiToken string) ([]ApiRequestResult, error) {
    // Set initial offset and limit for discovery request
    q.Set("offset", "0")
    q.Set("limit", to.String(API_LIMIT_MINIMUM))
    u.RawQuery = q.Encode()
    resp := DoApiRequestGet(u.String(), apiToken)
    if resp.Error != nil {
        return nil, resp.Error
    }

    // Unmarshal the message metedata
    err := json.Unmarshal(resp.Data, r)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }
    //m := result.Metadata
    //total := m.Total
    total := r.totalResults()
    if total < 1 {
        msg := fmt.Sprintf("Could not find any endpoint data with query: %#v", q)
        return nil, errors.New(msg)
    }
    pageTotal := int(math.Ceil(float64(total) / float64(API_LIMIT_MAXIMUM)))

    if DEBUG {
        log.Printf("Fetching %d pages from endpoint '%s' with %d total results.",
            pageTotal, u.Path, total)
    }
    q.Set("limit", to.String(API_LIMIT_MAXIMUM))
    u.RawQuery = q.Encode()

    // Use goroutine to request results concurrently
    // this buffered channel will block at the concurrency limit
    semaphoreChan := make(chan struct{}, REQUEST_CONCURRENCY_LIMIT )
    // this channel will not block and collect the http request results
    resultsChan := make(chan *ApiRequestResult)

    // make sure we close these channels when we're done with them
    defer func() {
        close(semaphoreChan)
        close(resultsChan)
    }()

    for i := 0; i < pageTotal; i++ {
        //log.Printf("Do request for page %d", i)
        go func(page int) {
            // this sends an empty struct into the semaphoreChan which
            // is basically saying add one to the limit, but when the
            // limit has been reached block until there is room
            semaphoreChan <- struct{}{}

            // Build the request URL string
            // (need to work with copies to prevent concurrent map read/write)
            // this code still has concurrency issues, even with copies
            offset := page * API_LIMIT_MAXIMUM
            /*
            uc := *u
            qc := *q
            qc.Set("offset", to.String(offset))
            uc.RawQuery = qc.Encode()
            requestUrl := uc.String()
            */
            requestUrl := RegexOffsetParameter.ReplaceAllString(u.String(),
                "$1=" + to.String(offset))
            //log.Printf("Here doing request for page=%d, offset=%d, url=%s", page, offset, requestUrl)

            // send the request and put the response in a result struct
            // along with the index so we can sort them later along with
            // any error that might have occoured
            result := DoApiRequestGet(requestUrl, apiToken)
            // now we can send the result struct through the resultsChan
            resultsChan <- result

            // once we're done it's we read from the semaphoreChan which
            // has the effect of removing one from the limit and allowing
            // another goroutine to start
            <-semaphoreChan
            //log.Printf("Finished request for page=%d, offset=%d", page, offset)
        }(i)
    }

    // make a slice to hold the results we're expecting
    var results []ApiRequestResult

    //log.Printf("Listening for channel results...")
    // start listening for any results over the resultsChan
    // once we get a result append it to the result slice
    for {
        result := <-resultsChan
        results = append(results, *result)

        // if we've reached the expected amount of urls then stop
        if len(results) == pageTotal {
            break
        }
    }
    return results, nil
}

func GetContactsAsync(params QueryParameters) (*ListContacts, error) {
    if (!params.FetchAll) {
        return GetContacts(params)
    }

    apiUrl, apiToken := GetApiCredentials()
    result := &ListContacts{}
    var resultList []ListContactsContact // Replace result list with empty one later

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_CONTACTS)
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    // Build request query variables 
    q, err := BuildQueryWithParams(params)
    if err != nil {
        return nil, fmt.Errorf("Failed parsing query parameters '%#v': %s", params, err)
    }

    // Fetch all endpoint data asynchronously
    results, err := FetchAllEndpointDataAsync(u, q, result, apiToken)
    if err != nil {
        return nil, fmt.Errorf("Failed fetching all endpoint data asynchronously: %s", err)
    }

    // Receive results from the channel and unmarshal them
    errorCount := 0
    var errorStrings []string
    for _, r := range(results) {
        if r.Error != nil {
            errorCount += 1
            errorStrings = append(errorStrings, r.Error.Error())
            continue
        } else {
            l := &ListContacts{}
            err = json.Unmarshal(r.Data, &l)
            if err != nil {
                msg := fmt.Sprintf("Failed to unmarshal response data: %s", err)
                errorCount += 1
                errorStrings = append(errorStrings, msg)
                continue
            }
            if DEBUG {
                log.Printf("Adding %d contacts to %d contacts...",
                    len(l.Contacts), len(resultList))
            }
            resultList = append(resultList, l.Contacts...)
        }
    }
    if len(errorStrings) > 0 {
        log.Printf("Finished parsing all response data from endpoint '%s'," +
            " but encountered %d errors:", u.String(), len(errorStrings))
        log.Println(errorStrings)
    }
    result.Contacts = resultList
    return result, nil
}

func GetContacts(params QueryParameters) (*ListContacts, error) {
    apiUrl, apiToken := GetApiCredentials()
    _ = apiToken

    r := &ListContacts{}
    var l []ListContactsContact

    fetchMultiple := false
    if (params.FetchAll) {
        fetchMultiple = true
    }

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_CONTACTS)
    if err != nil {
        msg := fmt.Sprintf("Failed building request url: %s", err)
        return nil, errors.New(msg)
    }

    // Build request query variables 
    q, err := BuildQueryWithParams(params)
    if err != nil {
        msg := fmt.Sprintf("Failed parsing query parameters '%#v': %s", params, err)
        return nil, errors.New(msg)
    }

    offset := 0
    pages := 0
    doRequest := true
    for doRequest {
        // Add offset if we're fetching multiple pages
        if offset > 0 && fetchMultiple {
            q.Set("offset", to.String(offset))
        }
        u.RawQuery = q.Encode()
        requestUrl := u.String()
        result := DoApiRequestGet(requestUrl, apiToken)
        if result.Error != nil {
            return nil, result.Error
        }

        // Unmarshal the message
        t := &ListContacts{}
        err = json.Unmarshal(result.Data, &t)
        if err != nil {
            return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
        }
        pages++
        var limit int32 = t.Metadata.PageInput.Limit
        var total uint64 = t.Metadata.Total
        //log.Printf("limit: %d, total: %d", limit, total)
        pageTotal := int(math.Ceil(float64(total) / float64(limit)))
        count := len(t.Contacts)

        if DEBUG {
            pageMsg := ""
            if fetchMultiple {
                pageMsg = fmt.Sprintf("page %d of %d out of ", pages, pageTotal)
            }

            log.Printf("Got %s%d results in contact search with %d total results.",
                pageMsg, count, t.Metadata.Total)
        }

        if t.Metadata.Total < 1 {
            msg := fmt.Sprintf("Could not find any contacts with params: %#v", params)
            return nil, errors.New(msg)
        }
        //log.Printf("Adding %d contacts to %d contacts...", len(l), len(t.Contacts))
        l = append(l, t.Contacts...)
        //log.Printf("List now has %d contacts", len(l))

        if fetchMultiple {
            if pages < pageTotal {
                offset = int(t.Metadata.PageInput.Offset) + int(t.Metadata.PageInput.Limit)
            } else {
                if DEBUG {
                    log.Printf("Finished fetching %d pages with %d total contacts",
                        pages, len(t.Contacts))
                }
                doRequest = false
            }
        } else {
            doRequest = false
        }
        // Copy metadata and score values to result when done
        if !doRequest {
            r.ScoreValues = t.ScoreValues
            r.Metadata = t.Metadata
        }
    } // for doRequest
    r.Contacts = l

    return r, nil
}

func GetContact(params QueryParameters) (*RetrieveContact, error) {
    apiUrl, apiToken := GetApiCredentials()
    _ = apiToken

    if params.Id == 0 {
        return nil, fmt.Errorf("No contact ID specified")
    }

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_CONTACTS, to.String(params.Id))
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    requestUrl := u.String()
    result := DoApiRequestGet(requestUrl, apiToken)
    if result.Error != nil {
        return nil, result.Error
    }

    // Unmarshal the message
    r := &RetrieveContact{}
    err = json.Unmarshal(result.Data, &r)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }

    return r, nil
}


func GetAutomationsAsync(params QueryParameters) (*ListAutomations, error) {
    if (!params.FetchAll) {
        return nil, fmt.Errorf("FetchAll must be set to true for GetAutomationsAsync()")
    }

    apiUrl, apiToken := GetApiCredentials()
    result := &ListAutomations{}
    var resultList []ListAutomationsAutomation // Replace result list with empty one later

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_AUTOMATIONS)
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    // Build request query variables 
    q, err := BuildQueryWithParams(params)
    if err != nil {
        return nil, fmt.Errorf("Failed parsing query parameters '%#v': %s", params, err)
    }

    // Fetch all endpoint data asynchronously
    results, err := FetchAllEndpointDataAsync(u, q, result, apiToken)
    if err != nil {
        return nil, fmt.Errorf("Failed fetching all endpoint data asynchronously: %s", err)
    }

    // Unmarshal and inspect the results
    errorCount := 0
    var errorStrings []string
    for _, r := range(results) {
        if r.Error != nil {
            errorCount += 1
            errorStrings = append(errorStrings, r.Error.Error())
            continue
        } else {
            l := &ListAutomations{}
            err = json.Unmarshal(r.Data, &l)
            if err != nil {
                msg := fmt.Sprintf("Failed to unmarshal GetAutomations response data: %s", err)
                errorCount += 1
                errorStrings = append(errorStrings, msg)
                continue
            }
            if params.AutomationName == "" {
                // Return all automations (no name comparing)
                if DEBUG {
                    log.Printf("Adding all %d automations to %d automations...",
                        len(l.Automations), len(resultList))
                }
                resultList = append(resultList, l.Automations...)
            } else {
                // Only return automations with the given name
                for _, a := range l.Automations {
                    if (params.ExactMatch && a.Name == params.AutomationName) ||
                       (!params.ExactMatch && strings.Contains(a.Name, params.AutomationName)) {
                        resultList = append(resultList, a)
                    }
                }
            }
        }
    }
    if len(errorStrings) > 0 {
        log.Printf("Finished parsing all response data from endpoint '%s'," +
            " but encountered %d errors:", u.String(), len(errorStrings))
        log.Println(errorStrings)
    }
    result.Automations = resultList
    return result, nil
}

// TODO Add ExactMatch support
func GetTagByName(tag string) (*ListTagsTag, error) {
    //if (!params.ExactMatch) {
    //    return nil, fmt.Errorf("ExactMatch must be set to true for GetTagByName()")
    //}

    apiUrl, apiToken := GetApiCredentials()

    result := &ListTags{}
    var p QueryParameters
    p.Limit = API_LIMIT_MAXIMUM
    p.TagName = tag

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_TAGS)
    if err != nil {
        msg := fmt.Sprintf("Failed building request url: %s", err)
        return nil, errors.New(msg)
    }

    // Build request query variables 
    q, err := BuildQueryWithParams(p)
    if err != nil {
        msg := fmt.Sprintf("Failed parsing query parameters '%#v': %s", p, err)
        return nil, errors.New(msg)
    }

    u.RawQuery = q.Encode()
    requestUrl := u.String()
    r := DoApiRequestGet(requestUrl, apiToken)
    if r.Error != nil {
        return nil, r.Error
    }

    // Unmarshal the message metedata
    err = json.Unmarshal(r.Data, &result)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }
    m := result.Metadata
    total := m.Total
    if total < 1 {
        msg := fmt.Sprintf("Could not find any tags with params: %#v", p)
        return nil, errors.New(msg)
    }
    pageTotal := int(math.Ceil(float64(total) / float64(API_LIMIT_MAXIMUM)))
    if pageTotal > 1 {
        return nil, fmt.Errorf("Too many tags found. Got %d and expected %d at most",
            total, API_LIMIT_MAXIMUM)
    }

    // Iterate through the results and find an exact match
    tagNames := ""
    for _, element := range result.Tags {
        if element.Tag == tag {
            return &element, nil
        }
        delim := ""
        if len(tagNames) > 0 {
            delim = ", "
        }
        tagNames = fmt.Sprintf("%s%s%s", tagNames, delim, element.Tag)
    }

    return nil, fmt.Errorf("Found %d tags, but none were an exact match: %s",
        total, tagNames)
}

func GetContactsByTag(tag string) ([]ListContactsContact, error) {
    t, err := GetTagByName(tag)
    if err != nil {
        return nil, err
    }

    var p QueryParameters
    p.TagId = to.Int(t.Id)
    p.FetchAll = true
    p.Limit = API_LIMIT_MAXIMUM

    r, err := GetContactsAsync(p)
    if err != nil {
        msg := fmt.Sprintf("Failed to get contacts by tag '%s': %s", tag, err)
        return nil, errors.New(msg)
    }
    if r.Metadata.Total < 1 || len(r.Contacts) < 1 {
        msg := fmt.Sprintf("No contacts found for tag: %s", tag)
        return nil, errors.New(msg)
    }

    return r.Contacts, nil
}

func GetAutomationContacts(automation *ListAutomationsAutomation) ([]ListContactAutomationsContact, error) {
    _, apiToken := GetApiCredentials()

    result := &ListContactAutomations{}

    // Build request URL
    urlRaw := automation.Links.ContactAutomations
    u, err := url.ParseRequestURI(urlRaw)
    if err != nil {
        msg := fmt.Sprintf("Failed parsing request url '%s': %s", urlRaw, err)
        return nil, errors.New(msg)
    }
    if u.Host == "" {
        msg := fmt.Sprintf("Request url '%s' is missing host", u)
        return nil, errors.New(msg)
    }
    if u.Scheme == "" {
        msg := fmt.Sprintf("Request url '%s' is missing scheme", u)
        return nil, errors.New(msg)
    }

    // Build request query variables 
    q := &url.Values{}
    q.Set("limit", to.String(API_LIMIT_MAXIMUM))
    requestUrl := u.String()
    r := DoApiRequestGet(requestUrl, apiToken)
    if r.Error != nil {
        return nil, r.Error
    }

    // Unmarshal the message metedata
    err = json.Unmarshal(r.Data, &result)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }

    return result.ContactAutomations, nil
}

// This will asynchronously fetch more info on each contact in a ListContactAutomationsContact list
func GetAutomationContactsInfo(contacts []ListContactAutomationsContact) ([]ListContactsContact, error) {
    _, apiToken := GetApiCredentials()

    var result []ListContactsContact

    if DEBUG {
        log.Printf("Fetching %d contacts from a list of contacts in an automation.",
            len(contacts))
    }

    if len(contacts) < 1 {
        if DEBUG {
            log.Printf("No contacts to fetch for empty contact list.")
        }
        return result, nil
    }

    // Use goroutine to request results concurrently
    // this buffered channel will block at the concurrency limit
    semaphoreChan := make(chan struct{}, REQUEST_CONCURRENCY_LIMIT )
    // this channel will not block and collect the http request results
    resultsChan := make(chan *ApiRequestResult)

    // make sure we close these channels when we're done with them
    defer func() {
        close(semaphoreChan)
        close(resultsChan)
    }()

    for _, c := range(contacts) {
        //log.Printf("Do request for contact %d (id=%d)", i, c.Contact)
        go func(contact ListContactAutomationsContact) {
            // this sends an empty struct into the semaphoreChan which
            // is basically saying add one to the limit, but when the
            // limit has been reached block until there is room
            semaphoreChan <- struct{}{}

            // Build the request URL string
            urlRaw := contact.Links.Contact
            u, err := url.ParseRequestURI(urlRaw)
            if err != nil {
                r := &ApiRequestResult{Data: nil,
                    Error: fmt.Errorf("Failed parsing contact link request url '%s': %s", urlRaw, err)}
                resultsChan <- r
                <-semaphoreChan
                return
            }
            if u.Host == "" {
                r := &ApiRequestResult{Data: nil,
                    Error: fmt.Errorf("Contact link request url '%s' is missing host", u)}
                resultsChan <- r
                <-semaphoreChan
                return
            }
            if u.Scheme == "" {
                r := &ApiRequestResult{Data: nil,
                    Error: fmt.Errorf("Contact link request url '%s' is missing scheme", u)}
                resultsChan <- r
                <-semaphoreChan
                return
            }
            requestUrl := u.String()

            // send the request and put the response in a result struct
            // along with the index so we can sort them later along with
            // any error that might have occoured
            r := DoApiRequestGet(requestUrl, apiToken)
            // now we can send the result struct through the resultsChan
            resultsChan <- r

            // once we're done it's we read from the semaphoreChan which
            // has the effect of removing one from the limit and allowing
            // another goroutine to start
            <-semaphoreChan
            //log.Printf("Finished request for page=%d, offset=%d", page, offset)
        }(c)
    }

    // make a slice to hold the results we're expecting
    var results []ApiRequestResult

    //log.Printf("Listening for channel results...")
    // start listening for any results over the resultsChan
    // once we get a result append it to the result slice
    for {
        result := <-resultsChan
        results = append(results, *result)

        // if we've reached the expected amount of urls then stop
        if len(results) == len(contacts) {
            break
        }
    }

    // Read results from the response list and unmarshal them
    errorCount := 0
    var errorStrings []string
    for _, r := range(results) {
        if r.Error != nil {
            errorCount += 1
            errorStrings = append(errorStrings, r.Error.Error())
            continue
        } else {
            c := &ListContactAutomationsContactLinksContact{}
            err := json.Unmarshal(r.Data, &c)
            if err != nil {
                msg := fmt.Sprintf("Failed to unmarshal response data: %s", err)
                errorCount += 1
                errorStrings = append(errorStrings, msg)
                continue
            }
            /*
            if DEBUG {
                log.Printf("Adding %d contacts to %d contacts...",
                    len(l.Contacts), len(resultList))
            }
            */
            result = append(result, c.Contact)
        }
    }
    if len(errorStrings) > 0 {
        log.Printf("Finished parsing all response data from list of contact automation links," +
            " but encountered %d errors:", len(errorStrings))
        log.Println(errorStrings)
    }
    return result, nil
}

func GetAutomationsByName(name string, exactMatch bool) ([]ListAutomationsAutomation, error) {
    var p QueryParameters
    p.AutomationName = name
    p.FetchAll = true
    p.Limit = API_LIMIT_MAXIMUM
    p.ExactMatch = exactMatch

    r, err := GetAutomationsAsync(p)
    if err != nil {
        msg := fmt.Sprintf("Failed to get automations by name '%s': %s", name, err)
        return nil, errors.New(msg)
    }
    if r.Metadata.Total < 1 || len(r.Automations) < 1 {
        msg := fmt.Sprintf("No automations found with name: %s", name)
        return nil, errors.New(msg)
    }

    return r.Automations, nil
}

// This returns the simpler ListContactsContact instead of GetContact (for now)
func GetContactByEmail(email string) (*ListContactsContact, error) {
    var p QueryParameters
    p.Email = email

    r, err := GetContacts(p)
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

func GetContactById(id string) (*RetrieveContact, error) {
    var p QueryParameters
    p.Id = to.Int(id)

    r, err := GetContact(p)
    if err != nil {
        msg := fmt.Sprintf("Failed to get contact %s: %s", id, err)
        return nil, errors.New(msg)
    }

    return r, nil
}

func GetContactProfileUrlById(id string) string {
    var c SecretsConfig
    err := c.GetSecrets(SecretsFilePath)
    if err != nil {
        log.Fatalf("Could not open YAML secrets file: %s", err.Error())
    }
    accountId := c.AccountId

    adminUrlPrefix := fmt.Sprintf(ADMIN_PREFIX_URL_FORMAT, accountId)
    url := fmt.Sprintf("%s%s/%s", adminUrlPrefix, API_URL_CONTACTS, id)
    return url
}

func GetContactProfileUrlByEmail(email string) (string, error) {
    c, err := GetContactByEmail(email)
    if err != nil {
        msg := fmt.Sprintf("Failed to get profile url: %s", err)
        return "", errors.New(msg)
    }
    return GetContactProfileUrlById(c.Id), nil
}

func GetTag(id string) (*RetrieveTag, error) {
    apiUrl, apiToken := GetApiCredentials()

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_TAGS, id)
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    // Build request query variables 
    //q := &url.Values{}
    //q.Set("limit", to.String(API_LIMIT_MAXIMUM))
    requestUrl := u.String()
    r := DoApiRequestGet(requestUrl, apiToken)
    if r.Error != nil {
        return nil, fmt.Errorf("Failed retrieving tag with ID %s: %s", id, r.Error)
    }

    // Unmarshal the message metedata
    t2 := &RetrieveTagContainer{}
    err = json.Unmarshal(r.Data, &t2)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }
    t := t2.Tag

    return &t, nil
}

func GetTagsAsync(ids []string) ([]RetrieveTag, error) {
    apiUrl, apiToken := GetApiCredentials()
    var resultList []RetrieveTag // Replace result list with empty one later

    if DEBUG {
        log.Printf("Fetching %d tags...", len(ids))
    }

    // Use goroutine to request results concurrently
    // this buffered channel will block at the concurrency limit
    semaphoreChan := make(chan struct{}, REQUEST_CONCURRENCY_LIMIT )
    // this channel will not block and collect the http request results
    resultsChan := make(chan *ApiRequestResult)

    // make sure we close these channels when we're done with them
    defer func() {
        close(semaphoreChan)
        close(resultsChan)
    }()

    for _, i := range(ids) {
        //log.Printf("Do request for contact %d (id=%d)", i, c.Contact)
        go func(id string) {
            semaphoreChan <- struct{}{}

            // Build request URL
            u, err := BuildRequestUrl(apiUrl, API_URL_TAGS, id)
            if err != nil {
                r := &ApiRequestResult{Data: nil,
                    Error: fmt.Errorf("Failed building request url: %s",  err)}
                resultsChan <- r
                <-semaphoreChan
                return
            }
            // Do request
            requestUrl := u.String()
            r := DoApiRequestGet(requestUrl, apiToken)
            resultsChan <- r
            <-semaphoreChan
        }(i)
    }

    var results []ApiRequestResult

    if DEBUG {
        log.Printf("Listening for %d channel results...", len(ids))
    }
    for {
        result := <-resultsChan
        results = append(results, *result)

        // if we've reached the expected amount of urls then stop
        if len(results) == len(ids) {
            break
        }
    }

    // Unmarshal and inspect the results
    errorCount := 0
    var errorStrings []string
    for _, r := range(results) {
        if r.Error != nil {
            errorCount += 1
            errorStrings = append(errorStrings, r.Error.Error())
            continue
        } else {

            // Unmarshal the message metedata
            t2 := &RetrieveTagContainer{}
            err := json.Unmarshal(r.Data, &t2)
            if err != nil {
                msg := fmt.Sprintf("Failed to unmarshal response data: %s", err)
                errorCount += 1
                errorStrings = append(errorStrings, msg)
                continue
            }
            t := t2.Tag
            resultList = append(resultList, t)
        }
    }
    if len(errorStrings) > 0 {
        log.Printf("Finished parsing all response data from fetching tags: %+q," +
            " but encountered %d errors:", ids, len(errorStrings))
        log.Println(errorStrings)
    }
    return resultList, nil
}

func GetContactTags(id string) ([]RetrieveTag, error) {
    apiUrl, apiToken := GetApiCredentials()

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_CONTACTS, id, API_URL_CONTACT_TAGS)
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    // Build request query variables 
    //q := &url.Values{}
    //q.Set("limit", to.String(API_LIMIT_MAXIMUM))
    requestUrl := u.String()
    r := DoApiRequestGet(requestUrl, apiToken)
    if r.Error != nil {
        return nil, fmt.Errorf("Failed retrieving list of contact tags" +
            " for contact with ID %s: %s", id, r.Error)
    }

    // Unmarshal the message metedata
    l := &ListContactTags{}
    err = json.Unmarshal(r.Data, &l)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }

    /*
    // Now fetch each tag...
    var tags []*RetrieveTag
    for i, v := range l.ContactTags {
        t, err := GetTag(v.Tag)
        if err != nil {
            return nil, fmt.Errorf("Failed fetching tag %d for contact" +
                " '%s' from list: %s", i, id, err)
        }
        tags = append(tags, t)
    }
    */
    // Fetch all tags
    var ids []string
    for _, v := range l.ContactTags {
        ids = append(ids, v.Tag)
    }
    tags, err := GetTagsAsync(ids)
    if err != nil {
        return nil, fmt.Errorf("Failed fetching tags for contact '%s': %s", id, err)
    }

    return tags, nil
}

func GetTagsString(tags []RetrieveTag) string {
	var strBuffer bytes.Buffer

    for i, v := range tags {
        if i > 0 {
            strBuffer.WriteString(", ")
        }
        strBuffer.WriteString(v.Tag)
    }
    return strBuffer.String()
}

/*
func GetContact(id string) (error {
    apiUrl, apiToken := GetApiCredentials()
    err := AuthenticateWithCredentials(apiUrl, apiToken)
    if err != nil {

    }
}

*/

//func UpdateContact(id string,  
func UpdateContactEmail(id string, newEmail string) error {
    apiUrl, apiToken := GetApiCredentials()
    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_CONTACTS, id)
    if err != nil {
        return fmt.Errorf("Failed building request url: %s", err)
    }

    // Build request data
    var c UpdateContact
    c.Contact.Email = newEmail;
    json, err := json.Marshal(c)
    if err != nil {
        return fmt.Errorf("Failed marshaling update contact request data: %s", err)
    }

    // Send request
    requestUrl := u.String()
    r := DoApiRequestPut(requestUrl, apiToken, json)
    if r.Error != nil {
        return fmt.Errorf("Failed updating data for contact with ID %s: %s",
            id, r.Error)
    }

    // Unmarshal the message metedata
    //l := &ListContactTags{}
    //err = json.Unmarshal(r.Data, &l)
    //if err != nil {
    //    return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    //}
    return nil
}

// TODO create one UpdateContact function, unmarshal response, and all these functions will call it
func UpdateContactCustomField(id string, field string, value string) error {
    apiUrl, apiToken := GetApiCredentials()
    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_CONTACTS, id)
    if err != nil {
        return fmt.Errorf("Failed building request url: %s", err)
    }

    // Build request data
    var c UpdateContact
    f := UpdateContactContactFieldValue {
        Field: "changed_email_to",
        Value: newEmail
    }

    c.Contact.FieldValues = append(c.Contact.FieldValues, f)
    json, err := json.Marshal(c)
    if err != nil {
        return fmt.Errorf("Failed marshaling update contact request data: %s", err)
    }

    // Send request
    requestUrl := u.String()
    r := DoApiRequestPut(requestUrl, apiToken, json)
    if r.Error != nil {
        return fmt.Errorf("Failed updating data for contact with ID %s: %s",
            id, r.Error)
    }

    // Unmarshal the message metedata
    //l := &ListContactTags{}
    //err = json.Unmarshal(r.Data, &l)
    //if err != nil {
    //    return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    //}
    return nil
}

/*
 * Messages and unmarshalers
 */
// Update a contact by ID
type UpdateContact struct {
    Contact     UpdateContactContact    `json:"contact"`
}

type _UpdateContact UpdateContact

type UpdateContactContact struct {
    Email       string      `json:"email"`
    FirstName   string      `json:"firstName"`
    LastName    string      `json:"lastName"`
    Phone       string      `json:"phone"`
    FieldValues []UpdateContactContactFieldValue    `json:"fieldValues"`
}

type UpdateContactContactFieldValue struct {
    Field string    `json:"field"`
    Value string    `json:"value"`
}

func (c *UpdateContact) MarshalJSON() ([]byte, error) {
    json, err := json.Marshal(c)
    if err != nil {
        return nil, err
    }

    return json, nil
}

func AddNoteToContact(id string, note string) error {
    apiUrl, apiToken := GetApiCredentials()
    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_NOTES)
    if err != nil {
        return fmt.Errorf("Failed building request url: %s", err)
    }

    // Build request data
    var n CreateNote
    n.Note.Note = note;
    n.Note.RelativeId = id;
    n.Note.RelativeType = "Subscriber"
    json, err := json.Marshal(n)
    if err != nil {
        return fmt.Errorf("Failed marshaling create note request data: %s", err)
    }

    // Send request
    requestUrl := u.String()
    fmt.Printf("Here with url: %s", requestUrl)
    r := DoApiRequestPost(requestUrl, apiToken, json)
    if r.Error != nil {
        return fmt.Errorf("Failed creating note for contact with ID %s: %s",
            id, r.Error)
    }

    // Unmarshal the message metedata
    //l := &ListContactTags{}
    //err = json.Unmarshal(r.Data, &l)
    //if err != nil {
    //    return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    //}
    return nil
}

/*
 * Messages and unmarshalers
 */
// Update a contact by ID
type CreateNote struct {
    Note     CreateNoteNote    `json:"note"`
}

type CreateNoteNote struct {
    Note            string      `json:"note"`
    RelativeId      string      `json:"relid"`
    RelativeType    string      `json:"reltype"`
}

func (n *CreateNote) MarshalJSON() ([]byte, error) {
    json, err := json.Marshal(n)
    if err != nil {
        return nil, err
    }

    return json, nil
}


// Retrieve a contact by ID
type RetrieveContact struct {
    Automations         []RetrieveContactAutomation        `json:"contactAutomations"`
    Data                []RetrieveContactData              `json:"contactData"`
    Lists               []RetrieveContactList              `json:"contactLists"`
    Deals               []RetrieveContactDeal              `json:"deals"`
    FieldValues         []RetrieveContactFieldValue        `json:"fieldValues"`
    GeographicIps       []RetrieveContactGeographicIp      `json:"geoIps"`
    GeographicAddresses []RetrieveContactGeographicAddress `json:"geoAddresses"`
    AccountContacts     []RetrieveContactAccountContacts   `json:"accountContacts"`
    Contact             RetrieveContactContact             `json:"contact"`
}

type _RetrieveContact RetrieveContact

type RetrieveContactAutomation ListContactAutomationsContact

type RetrieveContactData struct {
    Contact                  string     `json:"contact"`
    Timestamp                string     `json:"tstamp"`
    GeographicTimestamp      string     `json:"geoTstamp"`
    GeographicIpv4           string     `json:"geoIp4"`
    GeographicCountry2       string     `json:"geoCountry2"`
    GeographicCountry        string     `json:"geo_country"`
    GeographicState          string     `json:"geoState"`
    GeographicCity           string     `json:"geoCity"`
    GeographicZipcode        string     `json:"geoZip"`
    GeographicArea           string     `json:"geoArea"`
    GeographicLatitude       string     `json:"geoLat"`
    GeographicLongitude      string     `json:"geoLon"`
    GeographicTimezone       string     `json:"geoTz"`
    GeographicTimezoneOffset string     `json:"geoTzOffset"`
    GaCampaignSource         string     `json:"ga_campaign_source"`
    GaCampaignName           string     `json:"ga_campaign_name"`
    GaCampaignMedium         string     `json:"ga_campaign_medium"`
    GaCampaignTerm           string     `json:"ga_campaign_term"`
    GaCampaignContent        string     `json:"ga_campaign_content"`
    GaCampaignCustomSegment  string     `json:"ga_campaign_customsegment"`
    GaFirstVisit             string     `json:"ga_first_visit"`
    GaTimesVisited           string     `json:"ga_times_visited"`
    FacebookId               string     `json:"fb_id"`
    FacebookName             string     `json:"fb_name"`
    TwitterId                string     `json:"tw_id"`
    CreatedTimestamp         string     `json:"created_timestamp"`
    UpdatedTimestamp         string     `json:"updated_timestamp"`
    CreatedBy                string     `json:"created_by"`
    UpdatedBy                string     `json:"updated_by"`
    Links                    []string   `json:"links"`
    Id                       string     `json:"id"`
}

type RetrieveContactList struct {
    Contact               string    `json:"contact"`
    List                  string    `json:"list"`
    Form                  string    `json:"form"`
    SeriesId              string    `json:"seriesid"`
    SubscribeDate         string    `json:"sdate"`
    UnsubscribeDate       string    `json:"udate"`
    Status                string    `json:"status"`
    Responder             string    `json:"responder"`
    Sync                  string    `json:"sync"`
    UnsubscribeReason     string    `json:"unsubreason"`
    Campaign              string    `json:"campaign"`
    Message               string    `json:"message"`
    FirstName             string    `json:"first_name"`
    LastName              string    `json:"last_name"`
    Ipv4Subscribe         string    `json:"ip4Sub"`
    SourceId              string    `json:"sourceid"`
    AutoSyncLog           string    `json:"autosyncLog"`
    Ipv4Last              string    `json:"ip4_last"`
    Ipv4Unsubscribe       string    `json:"ip4Unsub"`
    CreatedTimestamp      string    `json:"created_timestamp"`
    UpdatedTimestamp      string    `json:"updated_timestamp"`
    CreatedBy             string    `json:"created_by"`
    UpdatedBy             string    `json:"updated_by"`
    UnsubscribeAutomation string    `json:"unsubscribeAutomation"`
    Links                 RetrieveContactListLinks `json:"links"`
    Id                    string    `json:"id"`
    Automation            string    `json:"automation"`
}

type RetrieveContactListLinks struct {
    Automation            string    `json:"automation"`
    List                  string    `json:"list"`
    Contact               string    `json:"contact"`
    Form                  string    `json:"form"`
    AutoSyncLog           string    `json:"autosyncLog"`
    Campaign              string    `json:"campaign"`
    UnsubscribeAutomation string    `json:"unsubscribeAutomation"`
    Message               string    `json:"message"`
    //Extra map[string]interface{}
}

type RetrieveContactDeal struct {
    // Nothing yet
    //Extra map[string]interface{}
}

type RetrieveContactFieldValue struct {
    Contact      string     `json:"contact"`
    Field        string     `json:"field"`
    Value        string     `json:"value"`
    CreationDate string     `json:"cdate"`
    UpdateDate   string     `json:"udate"`
    CreatedBy    string     `json:"created_by"`
    UpdatedBy    string     `json:"updated_by"`
    Links RetrieveContactFieldValueLinks `json:"links"`
    Id           string     `json:"id"`
    Owner        string     `json:"owner"`
}

type RetrieveContactFieldValueLinks struct {
    Owner       string      `json:"owner"`
    Field       string      `json:"field"`
}

type RetrieveContactGeographicAddress struct {
    Ip4         string      `json:"ip4"`
    Country     string      `json:"country"`
    Country2    string      `json:"country2"`
    State       string      `json:"state"`
    City        string      `json:"city"`
    Zip         string      `json:"zip"`
    Area        string      `json:"area"`
    Latitude    string      `json:"lat"`
    Longitude   string      `json:"lon"`
    Timezone    string      `json:"tz"`
    Timestamp   string      `json:"tstamp"`
    Links       []string    `json:"links"`
    Id          string      `json:"id"`
}

type RetrieveContactGeographicIp struct {
    Contact             string  `json:"contact"`
    CampaignId          string  `json:"campaignid"`
    MessageId           string  `json:"messageid"`
    GeographicAddressId string  `json:"geoaddrid"`
    Ip4                 string  `json:"ip4"`
    Timestamp           string  `json:"tstamp"`
    GeographicAddress   string  `json:"geoAddress"`
    Links               RetrieveContactGeographicIpLinks `json:"links"`
    Id                  string  `json:"id"`
}

type RetrieveContactGeographicIpLinks struct {
    GeographicAddress   string  `json:"geoAddress"`
}


type RetrieveContactAccountContacts struct {
    // None for now
}

// Slightly different than: ListContactsContact (maybe?/)
type RetrieveContactContact struct {
    CreatedDate         string      `json:"cdate"`
    Email               string      `json:"email"`
    Phone               string      `json:"phone"`
    FirstName           string      `json:"firstName"`
    LastName            string      `json:"lastName"`
    OrganizationId      string      `json:"orgid"`
    OrganizationName    string      `json:"orgname"`
    SegmentIoId         string      `json:"segmentio_id"`
    BouncedHard         string      `json:"bounced_hard"`
    BouncedSoft         string      `json:"bounced_soft"`
    BouncedDate         string      `json:"bounced_date"`
    Ip                  string      `json:"ip"`
    UserAgent           string      `json:"ua"`
    Hash                string      `json:"hash"`
    SocialDataLastCheck string      `json:"socialdata_lastcheck"`
    EmailLocal          string      `json:"email_local"`
    EmailDomain         string      `json:"email_domain"`
    SentCount           string      `json:"sentcnt"`
    RatingTimestamp     string      `json:"rating_tstamp"`
    Gravatar            string      `json:"gravatar"`
    Deleted             string      `json:"deleted"`
    Anonymized          string      `json:"anonymized"`
    AddedDate           string      `json:"adate"`
    UpdatedDate         string      `json:"udate"`
    EditedDate          string      `json:"edate"`
    DeletedAt           string      `json:"deleted_at"`
    CreatedUtcTimestamp string      `json:"created_utc_timestamp"`
    UpdatedUtcTimestamp string      `json:"updated_utc_timestamp"`
    CreatedTimestamp    string      `json:"created_timestamp"`
    UpdatedTimestamp    string      `json:"updated_timestamp"`
    CreatedBy           string      `json:"created_by"`
    UpdatedBy           string      `json:"updated_by"`
    EmailEmpty          bool        `json:"email_empty"`
    ContactAutomations  []string    `json:"contactAutomations"` // Not in ListContactsContact
    ContactLists        []string    `json:"contactLists"` // Not in ListContactsContact
    ContactData         string      `json:"contactData"` // Not in ListContactsContact
    FieldValues         []string    `json:"fieldValues"` // Not in ListContactsContact
    GeographicIps       []string    `json:"geoIps"` // Not in ListContactsContact
    Deals               []string    `json:"deals"` // Not in ListContactsContact
    AccountCountacts    []string    `json:"accountContacts"`
    Sentiment           string      `json:"sentiment"` // Not in ListContactsContact
    Links               ListContactsContactLinks `json:"links"`
    Id                  string      `json:"id"`
    Organization        string      `json:"organization"`
}

func (l *RetrieveContact) UnmarshalJSON(jsonStr []byte) error {
    l2 := _RetrieveContact{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    *l = RetrieveContact(l2)

    return nil
}

// We'll throw this away after unpacking
type RetrieveTagContainer struct {
    Tag                 RetrieveTag     `json:"tag"`
}

type _RetrieveTagContainer RetrieveTagContainer

type RetrieveTag struct {
    TagType             string  `json:"tagType"`
    Tag                 string  `json:"tag"`
    Description         string  `json:"description"`
    CreationDate        string  `json:"cdate"`
    SubscriberCount     string  `json:"subscriber_count"`
    CreatedTimestamp    string  `json:"created_timestamp"`
    UpdatedTimestamp    string  `json:"updated_timestamp"`
    CreatedBy           string  `json:"created_by"`
    UpdatedBy           string  `json:"updated_by"`
    Links               RetrieveTagLinks  `json:"links"`
    Id                  string  `json:"id"`
}

type RetrieveTagLinks struct {
    ContactGoalTags     string  `json:"contactGoalTags"`
}

func (t *RetrieveTagContainer) UnmarshalJSON(jsonStr []byte) error {
    t2 := _RetrieveTagContainer{}

    err := json.Unmarshal(jsonStr, &t2)
    if err != nil {
        return err
    }

    *t = RetrieveTagContainer(t2)

    return nil
}

// Most list retrieval responses implement this interface
type ListResponse interface {
    totalResults()  uint64
}

// List contact tags
type ListContactTags struct {
    ContactTags         []ListContactTagsTag    `json:"contactTags"`
}

type _ListContactTags ListContactTags

type ListContactTagsTag struct {
    Contact             string      `json:"contact"`
    Tag                 string      `json:"tag"`
    CreationDate        string      `json:"cdate"`
    CreatedTimestamp    string      `json:"created_timestamp"`
    UpdatedTimestamp    string      `json:"updated_timestamp"`
    CreatedBy           string      `json:"created_by"`
    UpdatedBy           string      `json:"updated_by"`
    Links               ListContactTagsTagLinks    `json:"links"`
    Id                  string      `json:"id"`
}

type ListContactTagsTagLinks struct {
    Tag     string      `json:"tag"`
    Contact string      `json:"contact"`
}

func (l *ListContactTags) UnmarshalJSON(jsonStr []byte) error {
    l2 := _ListContactTags{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    *l = ListContactTags(l2)

    return nil
}

func (l *ListContactTags) totalResults() uint64 {
    return uint64(len(l.ContactTags))
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
    OrganizationName    string   `json:"orgname"`
    SegmentIoId         string   `json:"segmentio_id"`
    BouncedHard         string   `json:"bounced_hard"`
    BouncedSoft         string   `json:"bounced_soft"`
    BouncedDate         string   `json:"bounced_date"`
    Ip                  string   `json:"ip"`
    UserAgent           string   `json:"ua"`
    Hash                string   `json:"hash"`
    SocialDateLastCheck string   `json:"socialdata_lastcheck"`
    EmailLocal          string   `json:"email_local"`
    EmailDomain         string   `json:"email_domain"`
    SentCount           string   `json:"sentcnt"`
    RatingTimestamp     string   `json:"rating_tstamp"`
    Gravatar            string   `json:"gravatar"`
    Deleted             string   `json:"deleted"`
    Anonymized          string   `json:"anonymized"`
    AddedDate           string   `json:"adate"`
    UpdatedDate         string   `json:"udate"`
    EditedDate          string   `json:"edate"`
    DeletedAt           string   `json:"deleted_at"`
    CreatedUtcTimestamp string   `json:"created_utc_timestamp"`
    UpdatedUtcTimestamp string   `json:"updated_utc_timestamp"`
    CreatedTimestamp    string   `json:"created_timestamp"`
    UpdatedTimestamp    string   `json:"updated_timestamp"`
    CreatedBy           string   `json:"created_by"`
    UpdatedBy           string   `json:"updated_by"`
    EmailEmpty          bool     `json:"email_empty"`
    ScoreValues         []string `json:"scoreValues"`
    AccountContacts     []string `json:"accountContacts"`
    Links ListContactsContactLinks `json:"links"`
    Id                  string   `json:"id"`
    Organization        string   `json:"organization"`

}

type ContactList []*ListContactsContact

func (l *ContactList) Contains(email string) bool {
    email = strings.ToLower(email)
	for _, c := range *l {
		if strings.ToLower(c.Email) == email {
            return true
		}
	}
	return false
}

func GetContactList(l []ListContactsContact) ContactList {
    var l2 ContactList
    for i := 0; i < len(l); i++ {
        l2 = append(l2, &l[i])
    }
    return l2
}


func (c *ListContactsContact) String() string {
	var strBuffer bytes.Buffer

    strBuffer.WriteString(fmt.Sprintf(
        " - %s:\n" +
        "   id: %s\n" +
        "   name: %s %s\n" +
        "   user_agent: %s\n" +
        "   sent_count: %s\n" +
        "   added_date: %s\n" +
        "   updated_date: %s\n" +
        "   edited_date: %s\n" +
        "   creation_date: %s\n" +
        "   deleted?: %s\n" +
        "   deleted_at: %s\n" +
        "   ip: %s\n" +
        "   bounced: hard=%s / soft=%s / date=%s\n",
        c.Email, c.Id, c.FirstName, c.LastName, c.UserAgent,
        c.SentCount, c.AddedDate, c.UpdatedDate, c.EditedDate,
        c.CreationDate, c.Deleted, c.DeletedAt, c.Ip,
        c.BouncedHard, c.BouncedSoft, c.BouncedDate))

    return strBuffer.String()
}

type ListContactsContactLinks struct {
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
    AccountContacts    string `json:"accountContacts"`
    AutomationEntryCounts string `json:"automationEntryCounts"`
}

type ListContactsMetadata struct {
    TotalRaw  string `json:"total"`
    Total     uint64
    Sortable  bool   `json:"sortable"`
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

func (l *ListContacts) totalResults() uint64 {
    return l.Metadata.Total
}

// List tags
type ListTags struct {
    Tags        []ListTagsTag `json:"tags"`
    Metadata    ListTagsMetadata  `json:"meta"`
}

type _ListTags ListTags

type ListTagsTag struct {
    TagType             string   `json:"tagType"`
    Tag                 string   `json:"tag"`
    Description         string   `json:"description"`
    CreationDate        string   `json:"cdate"`
    SubscriberCount     string   `json:"subscriber_count"`
    CreatedTimestamp    string   `json:"created_timestamp"`
    UpdatedTimestamp    string   `json:"updated_timestamp"`
    CreatedBy           string   `json:"created_by"`
    UpdatedBy           string   `json:"updated_by"`
    Id                  string   `json:"id"`
    Links               ListTagsTagLinks  `json:"links"`
}

type ListTagsTagLinks struct {
    ContactGoalTags         string `json:"contactGoalTags"`
}

type ListTagsMetadata struct {
    TotalRaw  string `json:"total"`
    Total     uint64
}

func (l *ListTags) UnmarshalJSON(jsonStr []byte) error {
    l2 := _ListTags{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    l2.Metadata.Total = to.Uint64(l2.Metadata.TotalRaw)

    *l = ListTags(l2)

    return nil
}

func (l *ListTags) totalResults() uint64 {
    return l.Metadata.Total
}

// List automations
type ListAutomations struct {
    Automations []ListAutomationsAutomation `json:"automations"`
    Metadata    ListAutomationsMetadata  `json:"meta"`
}

type _ListAutomations ListAutomations

type ListAutomationsAutomation struct {
    Name                string   `json:"name"`
    CreatedDate         string   `json:"cdate"`
    ModifiedDate        string   `json:"mdate"`
    UserId              string   `json:"userid"`
    Status              string   `json:"status"`
    Entered             string   `json:"entered"`
    Exited              string   `json:"exited"`
    Hidden              string   `json:"hidden"`
    DefaultScreenshot   string   `json:"defaultscreenshot"`
    Screenshot          string   `json:"screenshot"`
    Id                  string   `json:"id"`
    Links               ListAutomationsAutomationLinks  `json:"links"`
}

type ListAutomationsAutomationLinks struct {
    Campaigns           string `json:"campaigns"`
    ContactGoals        string `json:"contactGoals"`
    ContactAutomations  string `json:"contactAutomations"`
    Blocks              string `json:"blocks"`
    Goals               string `json:"goals"`
    Sms                 string `json:"sms"`
    SiteMessages        string `json:"sitemessages"`
    Triggers            string `json:"triggers"`
}

type ListAutomationsMetadata struct {
    TotalRaw  string `json:"total"`
    Total     uint64
    Starts    []ListAutomationsMetadataStartsStart `json:"starts"`
    Filtered  bool      `json:"filtered"`
    SmsLogs   []string  `json:"smsLogs"`
}

type ListAutomationsMetadataStartsStart struct {
    Id          string  `json:"id"`
    Series      string  `json:"series"`
    Type        string  `json:"type"`
    DealField   bool  `json:"dealfield"`
}

func (l *ListAutomations) UnmarshalJSON(jsonStr []byte) error {
    l2 := _ListAutomations{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    l2.Metadata.Total = to.Uint64(l2.Metadata.TotalRaw)

    *l = ListAutomations(l2)

    return nil
}

func (l *ListAutomations) totalResults() uint64 {
    return l.Metadata.Total
}

type AutomationList []*ListAutomationsAutomation

func (l *AutomationList) String() string {
	var automationListBuffer bytes.Buffer

	for i, a := range *l {
		if i > 0 {
			automationListBuffer.WriteString(", ")
		}
		automationListBuffer.WriteString(a.Name)
	}
	return automationListBuffer.String()
}

func GetAutomationList(l []ListAutomationsAutomation) AutomationList {
    var l2 AutomationList
    for i := 0; i < len(l); i++ {
        l2 = append(l2, &l[i])
    }
    return l2
}

func (a *ListAutomationsAutomation) String() string {
	var strBuffer bytes.Buffer

    strBuffer.WriteString(fmt.Sprintf("Automation: %s\n" +
               "   id: %s\n" +
               "   status: %s\n" +
               "   creation_date: %s\n" +
               "   modified_date: %s\n" +
               "   entered_count: %s\n" +
               "   exited_count: %s\n" +
               "   hidden: %s\n" +
               "   user_id: %s\n",
               a.Name, a.Id, a.Status, a.CreatedDate, a.ModifiedDate,
               a.Entered, a.Exited, a.Hidden, a.UserId))

    return strBuffer.String()
}

// List contact automations
type ListContactAutomations struct {
    ContactAutomations []ListContactAutomationsContact `json:"contactAutomations"`
}

type _ListContactAutomations ListContactAutomations

type ListContactAutomationsContact struct {
    Contact             string  `json:"contact"`
    SeriesId            string  `json:"seriesid"`
    StartId             string  `json:"startid"`
    Status              string  `json:"status"`
    BatchId             string  `json:"batchid"`
    AddDate             string  `json:"addate"`
    RemoveDate          string  `json:"remdate"`
    TimeSpan            string  `json:"timespan"`
    LastBlock           string  `json:"lastblock"`
    LastLogId           string  `json:"lastlogid"`
    LastDate            string  `json:"lastdate"`
    InAls               string  `json:"in_als"`
    CompletedElements   int     `json:"completedElements"`
    TotalElements       int     `json:"totalElements"`
    Completed           int     `json:"completed"`
    CompleteValue       int     `json:"completeValue"`
    Links               ListContactAutomationsContactLinks  `json:"links"`
    Id                  string  `json:"id"`
    Automation          string  `json:"automation"`
    IsCompleted         bool
}

type ListContactAutomationsContactLinks struct {
    Automation      string `json:"automation"`
    Contact         string `json:"contact"`
    ContactGoals    string `json:"contactGoals"`
    AutomationLogs  string `json:"automationLogs"`
}

func (l *ListContactAutomations) UnmarshalJSON(jsonStr []byte) error {
    l2 := _ListContactAutomations{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    for i, c := range l2.ContactAutomations {
        l2.ContactAutomations[i].IsCompleted = c.Completed > 0
    }

    *l = ListContactAutomations(l2)

    return nil
}

// The response for requesting the Contact link of an automation contact
type ListContactAutomationsContactLinksContact struct {
    Contact     ListContactsContact     `json:"contact"`
}

type _ListContactAutomationsContactLinksContact ListContactAutomationsContactLinksContact

func (l *ListContactAutomationsContactLinksContact) UnmarshalJSON(jsonStr []byte) error {
    l2 := _ListContactAutomationsContactLinksContact{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    *l = ListContactAutomationsContactLinksContact(l2)

    return nil
}

/*
func printListOfAutomationContacts(list []ac.ListContactAutomationsContact) {
	var automationContactsListBuffer bytes.Buffer

	for i, a := range list {
		if i > 0 {
			automationContactsListBuffer.WriteString(", ")
		}
        automationContactsListBuffer.WriteString("contact_" + a.Contact)
	}
	fmt.Println(automationContactsListBuffer.String())
}*/

type AutomationContactList []*ListContactAutomationsContact

func (l *AutomationContactList) String() string {
	var automationContactListBuffer bytes.Buffer

	for _, c := range *l {
        automationContactListBuffer.WriteString(fmt.Sprintf(
            " - Contact id: %s\n" +
            "   automation_contact_id: %s\n" +
            "   status: %s\n" +
            "   batchid: %s\n" +
            "   add_date: %s\n" +
            "   remove_date: %s\n" +
            "   last_date: %s\n" +
            "   completed_elements: %d\n" +
            "   total_elements: %d\n" +
            "   completed?: %t\n",
            c.Contact, c.Id, c.Status, c.BatchId, c.AddDate, c.RemoveDate,
            c.LastDate, c.CompletedElements, c.TotalElements, c.Completed > 0))
	}
	return automationContactListBuffer.String()
}

func GetAutomationContactList(l []ListContactAutomationsContact) AutomationContactList {
    var l2 AutomationContactList
    for i := 0; i < len(l); i++ {
        l2 = append(l2, &l[i])
    }
    return l2
}

