package teachable

// For Teachable v1 API

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "net/http"
    "net/url"
    //"math"
    "strings"
    "regexp"

    "github.com/xiam/to"
    "gopkg.in/yaml.v2"
    "github.com/gocarina/gocsv"

    "bitbucket.org/dagoodma/dagoodma-go/util"
)

var DEBUG = true
var DEBUG_VERBOSE = false
var SAVE_API_KEY = true // whether to save and re-use the API key

const (
    USER_AGENT = "Nancy Hillis Studio Webhook Backend (nancyhillis.com)"
    API_URL_FORMAT = "https://%s.teachable.com/"
)

const (
    // Teachable APIv1 URL: https://<your-account>.teachable.com/api/v1/
    API_URL_SUFFIX = "/api/v1"
    API_URL_USERS = "/users"
    API_URL_COURSES = "/courses"
    API_URL_ENROLLMENTS = "/enrollments"
    API_URL_SALES = "/sales"
    API_URL_HOOKS = "/hooks"
    API_URL_ADMIN = "/admin"
    API_URL_INFORMATION = "/information"
    //API_PARAM_ENROLLED_IN = "enrolled_in_specific%5B%5D"
    API_PARAM_ENROLLED_IN = "enrolled_in_specific[]"

)

const (
    REQUEST_CONCURRENCY_LIMIT = 4 // maximum number of concurrent requests
)

var RegexPageParameter = regexp.MustCompile(`(page)=(\d+)`)

/*
 * Settings
 */
var SecretsFilePath = "/var/webhook/secrets/teachable_secrets.yml"

type SecretsConfig struct {
    RelicId     string `yaml:"RELIC_ID"`
    ApiUrl      string `yaml:"API_URL"`
    ApiUser     string `yaml:"API_USER"`
    ApiPassword string `yaml:"API_PASSWORD"`
}
var SavedSecretsConfig *SecretsConfig

func (c *SecretsConfig) GetSecrets(filePath string) (error) {
    if SAVE_API_KEY && SavedSecretsConfig != nil {
        c.RelicId = SavedSecretsConfig.RelicId
        c.ApiUrl = SavedSecretsConfig.ApiUrl
        c.ApiUser = SavedSecretsConfig.ApiUser
        c.ApiPassword = SavedSecretsConfig.ApiPassword
        return nil
    }

    yamlFile, err := ioutil.ReadFile(filePath)
    if err != nil {
        log.Fatalf("Failed reading secrets file: %s ", err)
        return err
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

type ApiLoginCredentials struct {
    User        string
    Password    string
}

// Returns API URL, user login, and password
func GetApiCredentials() (string, *ApiLoginCredentials) {
    var c SecretsConfig
    err := c.GetSecrets(SecretsFilePath)
    if err != nil {
        log.Fatalf("Could not open YAML secrets file: %s", err.Error())
    }

    apiUrl := c.ApiUrl
    apiUser := c.ApiUser
    apiPassword := c.ApiPassword

    credentials := ApiLoginCredentials{User: apiUser, Password: apiPassword}
    //log.Printf("Here with apiCred: %#v", credentials)

    // Add suffix to url
    url := fmt.Sprintf("%s%s", apiUrl, API_URL_SUFFIX)

    return url, &credentials
}

// Returns raw API URL only (with no suffix)
func GetRawApiUrl() string {
    var c SecretsConfig
    err := c.GetSecrets(SecretsFilePath)
    if err != nil {
        log.Fatalf("Could not open YAML secrets file: %s", err.Error())
    }

    apiUrl := c.ApiUrl

    return apiUrl
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
    Page        int
    Id          uint64
    CourseId    uint64
    FetchAll    bool
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
    if params.Page > 0 {
        q.Set("page", to.String(params.Page))
    }
    if params.CourseId > 0 {
        // Test adding this to fix queries returning duplicate results
        q.Set("name_or_email_cont", "___")
        // 
        q.Set(API_PARAM_ENROLLED_IN, to.String(params.CourseId))
    }
    return &q, nil
}

type ApiRequestResult struct {
    Data []byte
    Error error
    Url string
}

func DoApiRequest(requestUrl string, apiCredentials *ApiLoginCredentials) (*ApiRequestResult) {
    r := &ApiRequestResult{Data: nil, Error: nil, Url: requestUrl}

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
    //req.Header.Set("Api-Token", apiToken)

    req.SetBasicAuth(apiCredentials.User, apiCredentials.Password)

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
        log.Printf("%#v", resp)
        r.Error = fmt.Errorf("Got '%s' response with status code: %d, expected: %d", resp.Status, resp.StatusCode, 200)
        return r
    }
    r.Data = []byte(body)
    return r
}

/* This function will asynchronously fetch all pages of data from the
 * endpoint. The list response interface, r, given as a parameter will
 * have it's metadata populated, but the list member within will not be
 * poulated and must be filled using the API result array returned.
 */
// Refs:
// - https://guzalexander.com/2013/12/06/golang-channels-tutorial.html
// - https://gist.github.com/montanaflynn/ea4b92ed640f790c4b9cee36046a5383
func FetchAllEndpointDataAsync(u *url.URL, q *url.Values, r ListResponse,
    apiCredentials *ApiLoginCredentials) ([]ApiRequestResult, error) {
    // Set initial offset and limit for discovery request
    q.Set("page", "1")
    u.RawQuery = q.Encode()
    resp := DoApiRequest(u.String(), apiCredentials)
    if resp.Error != nil {
        return nil, resp.Error
    }

    // Unmarshal the message metedata
    err := json.Unmarshal(resp.Data, r)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }
    total := r.TotalResults()
    if total < 1 {
        msg := fmt.Sprintf("Could not find any endpoint data with query: %#v", q)
        return nil, errors.New(msg)
    }
    totalPages := r.TotalPages()

    // make a slice to hold the results we're expecting
    var results []ApiRequestResult
    //results = append(results, *resp) // add initial page 1 result first

    if DEBUG {
        log.Printf("Fetching %d pages from endpoint '%s' with %d total results.",
            totalPages, u.Path, total)
    }
    q.Set("page", "2")
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

    for i := 2; i <= totalPages; i++ {
        //log.Printf("Do request for page %d", i)
        go func(page int) {
            // this sends an empty struct into the semaphoreChan which
            // is basically saying add one to the limit, but when the
            // limit has been reached block until there is room
            semaphoreChan <- struct{}{}

            // Build the request URL string
            requestUrl := RegexPageParameter.ReplaceAllString(u.String(),
                "$1=" + to.String(page))
            if DEBUG && DEBUG_VERBOSE {
                log.Printf("Here doing request for page=%d, url=%s", page, requestUrl)
            }

            // send the request and put the response in a result struct
            // along with the index so we can sort them later along with
            // any error that might have occoured
            result := DoApiRequest(requestUrl, apiCredentials)
            // now we can send the result struct through the resultsChan
            resultsChan <- result

            // once we're done it's we read from the semaphoreChan which
            // has the effect of removing one from the limit and allowing
            // another goroutine to start
            <-semaphoreChan
            //log.Printf("Finished request for page=%d, offset=%d", page, offset)
        }(i)
    }

    if DEBUG {
        log.Printf("Listening for %d channel results...", totalPages)
    }
    // start listening for any results over the resultsChan
    // once we get a result append it to the result slice
    results = append(results, *resp) // first add the initial result
    for {
        // if we've reached the expected amount of urls then stop
        if len(results) >= totalPages {
            break
        }
        result := <-resultsChan
        results = append(results, *result)
    }
    return results, nil
}

func GetAllUsers() ([]ListUsersUser, error) {
    var p QueryParameters
    l, err := GetUsersAsync(p)
    if err != nil {
        return nil, fmt.Errorf("Failed fetching all users: %#v", err)
    }
    return l.Users, nil
}

func GetCourseStudents(courseId uint64) ([]ListUsersUser, error) {
    var p QueryParameters
    p.CourseId = courseId
    l, err := GetUsersAsync(p)
    if err != nil {
        return nil, fmt.Errorf("Failed fetching students in course %d: %#v", courseId, err)
    }
    return l.Users, nil
}

func GetUsersAsync(params QueryParameters) (*ListUsers, error) {
    /*
    if (!params.FetchAll) {
        return GetUsers(params)
    }
    */

    apiUrl, apiCredentials := GetApiCredentials()
    result := &ListUsers{}
    var resultList []ListUsersUser

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_USERS)
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    // Build request query variables 
    q, err := BuildQueryWithParams(params)
    if err != nil {
        return nil, fmt.Errorf("Failed parsing query parameters '%#v': %s", params, err)
    }

    // Fetch all endpoint data asynchronously
    results, err := FetchAllEndpointDataAsync(u, q, result, apiCredentials)
    if err != nil {
        return nil, fmt.Errorf("Failed fetching all endpoint data asynchronously: %s", err)
    }

    // Receive results from the channel and unmarshal them
    //log.Printf("Unpacking all %d responses...", len(results))
    errorCount := 0
    studentUrlByEmail := make(map[string]string) // checking for errors
    var errorStrings []string
    for _, r := range(results) {
        if r.Error != nil {
            errorCount += 1
            errorStrings = append(errorStrings, r.Error.Error())
            continue
        } else {
            l := &ListUsers{}
            err = json.Unmarshal(r.Data, &l)
            if err != nil {
                msg := fmt.Sprintf("\nFailed to unmarshal response data from '%s': %s", r.Url, err)
                errorCount += 1
                errorStrings = append(errorStrings, msg)
                continue
            }
            if DEBUG {
                log.Printf("Adding %d users to %d users...",
                    len(l.Users), len(resultList))
            }
            // Check for duplicates
            for _, s := range(l.Users) {
                if _, ok := studentUrlByEmail[s.Email]; !ok {
                    studentUrlByEmail[s.Email] = r.Url
                } else {
                    log.Printf("Error: found duplicate student '%s' in query result: %s;" +
                               " was originally from query: %s", s.Email, r.Url,
                               studentUrlByEmail[s.Email])
                }
            }
            resultList = append(resultList, l.Users...)
        }
    }
    if len(errorStrings) > 0 {
        requestUrl := RegexPageParameter.ReplaceAllString(u.String(), "$1=*")
        log.Printf("Finished parsing all response data from endpoint '%s'," +
            " but encountered %d errors:%s", requestUrl, len(errorStrings), errorStrings)
    }
    result.Users = resultList
    return result, nil
}

func GetUser(params QueryParameters) (*ListUsersUser, error) {
    apiUrl, apiCredentials := GetApiCredentials()

    if params.Id == 0 {
        return nil, fmt.Errorf("No user ID specified")
    }

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_USERS, to.String(params.Id))
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    requestUrl := u.String()
    result := DoApiRequest(requestUrl, apiCredentials)
    if result.Error != nil {
        return nil, result.Error
    }

    // Unmarshal the message
    r := &ListUsersUser{}
    err = json.Unmarshal(result.Data, &r)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }

    return r, nil
}

func GetUserByEmail(email string) (*ListUsersUser, error) {
    var p QueryParameters
    p.Email = strings.ToLower(email)

    r, err := GetUsersAsync(p)
    if err != nil {
        msg := fmt.Sprintf("Failed to get user '%s': %s", email, err)
        return nil, errors.New(msg)
    }
    if r.Metadata.Total < 1 || len(r.Users) < 1 {
        msg := fmt.Sprintf("No users found for: %s", email)
        return nil, errors.New(msg)
    }
    if r.Metadata.Total > 1 || len(r.Users) > 1 {
        msg := fmt.Sprintf("Found multiple users for: %s", email)
        return nil, errors.New(msg)
    }
    s := r.Users[0]

    return &s, nil
}

func GetUserById(id uint64) (*ListUsersUser, error) {
    var p QueryParameters
    p.Id = to.Uint64(id)

    r, err := GetUser(p)
    if err != nil {
        msg := fmt.Sprintf("Failed to get user %s: %s", id, err)
        return nil, errors.New(msg)
    }

    return r, nil
}

func GetUserProfileUrlById(id uint64) string {
    apiUrl := GetRawApiUrl()
    url := fmt.Sprintf("%s%s%s/%s%s", apiUrl, API_URL_ADMIN, API_URL_USERS,
        to.String(id), API_URL_INFORMATION)
    return url
}

func GetUserProfileUrlByEmail(email string) (string, error) {
    c, err := GetUserByEmail(email)
    if err != nil {
        msg := fmt.Sprintf("Failed to get profile url: %s", err)
        return "", errors.New(msg)
    }
    return GetUserProfileUrlById(c.Id), nil
}

func GetUserEnrollments(id uint64) ([]ListEnrollmentsEnrollment, error) {
    apiUrl, apiCredentials := GetApiCredentials()

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_USERS, to.String(id),
        API_URL_ENROLLMENTS)
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    requestUrl := u.String()
    result := DoApiRequest(requestUrl, apiCredentials)
    if result.Error != nil {
        return nil, result.Error
    }

    // Unmarshal the message
    r := &ListEnrollments{}
    err = json.Unmarshal(result.Data, &r)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }

    return r.Enrollments, nil
}

func GetCourse(id string) (*RetrieveCourse, error) {
    apiUrl, apiCredentials := GetApiCredentials()

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_COURSES, to.String(id))
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    requestUrl := u.String()
    result := DoApiRequest(requestUrl, apiCredentials)
    if result.Error != nil {
        return nil, result.Error
    }

    // Unmarshal the message
    r := &RetrieveCourse{}
    err = json.Unmarshal(result.Data, &r)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }

    return r, nil
}

func GetAllCourses() ([]ListCoursesCourse, error) {
    apiUrl, apiCredentials := GetApiCredentials()

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_COURSES)
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    requestUrl := u.String()
    result := DoApiRequest(requestUrl, apiCredentials)
    if result.Error != nil {
        return nil, result.Error
    }

    // Unmarshal the message
    r := &ListCourses{}
    err = json.Unmarshal(result.Data, &r)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }

    return r.Courses, nil
}

func IsValidCourseAcronym(acronym string) bool {
    var courseAcronymRegex = regexp.MustCompile(`^(TAJC|TAJM|EWC|SJC|SJM|ATC|LYS|BUNDLE_TAJC-EWC|TAP_CHALLENGE|TAPCIP)$`)
    //courseAcronym := strings.ToUpper(acronym)

    if courseAcronymRegex.FindString(acronym) == "" {
        return false
    }
    return true
}


func GetCourseStudentsCsv(csvFile string) ([]ListUsersUser, error) {
  recordFile, err := os.Open(csvFile)
  if err != nil {
      return nil, fmt.Errorf("Failed reading CSV file: %s", err)
  }

  defer recordFile.Close()

  students := []ListUsersUser{}

  if err := gocsv.UnmarshalFile(recordFile, &students); err != nil {
      return nil, fmt.Errorf("Error unmarshaling CSV file '%s': %s", csvFile, err)
  }

  return students, nil

}

func GetSaleById(id string) (*RetrieveSale, error) {
    apiUrl, apiCredentials := GetApiCredentials()
    var p QueryParameters
    p.Id = to.Uint64(id)

    // Build request URL
    u, err := BuildRequestUrl(apiUrl, API_URL_SALES, to.String(p.Id))
    if err != nil {
        return nil, fmt.Errorf("Failed building request url: %s", err)
    }

    requestUrl := u.String()
    result := DoApiRequest(requestUrl, apiCredentials)
    if result.Error != nil {
        return nil, result.Error
    }

    // Unmarshal the message
    r := &RetrieveSale{}
    err = json.Unmarshal(result.Data, &r)
    if err != nil {
        return nil, fmt.Errorf("Failed to unmarshal response data: %s", err)
    }

    return r, nil
}


// ---------------------- Structs --------------------------

/*
type TeachableStudent struct {
    User            *ListUsersUser
    Enrollments     []*ListEnrollmentsEnrollment
    Courses         []*TeachableCourse
    InactiveCourses []*TeachableCourse
}

type TeachableCourse struct {
    FriendlyUrl     string
    Path            string
    Name            string
    Id              uint64
    ParentCourse    *TeachableCourse
    ChildCourses    []*TeachableCourse
    IsOpen          bool
    IsPublished     bool
}
*/

type CourseAcronym int

const (
    TAJC CourseAcronym = iota
    TAJM
    SJC
    SJM
    LYS
    ATC
    EWC
    BUNDLE_TAJC_EWC
    TAP_CHALLENGE
    TAPCIP
)

func (ct CourseAcronym) String() string {
    if ct < 0 {
        return "unknown"
    }
    acronymFriendlyUrls := [...]string{"the-artists-journey",
    "the-artists-journey-masterclass",
    "sjc-course",
    "studio-journey-masterclass",
    "light-your-creative-studio-like-a-pro",
    "activating-the-canvas",
    "experimenting-with-color",
    "bundle-the-artists-journey-experimenting-with-color",
    "creativity-challenge",
    "creativity-immersion"}
    if int(ct) > len(acronymFriendlyUrls) {
        return "unknown"
    }
    return acronymFriendlyUrls[ct]
}

// Returns error or nil if the ct is valid
func (ct CourseAcronym) EnsureValid() error {
    if ct.String() != "unknown" {
        return nil
    }
    return fmt.Errorf("Invalid course acronym: %s", ct)
}

func GetCourseAcronym(a string) (CourseAcronym, error) {
    a2 := strings.ToUpper(strings.ReplaceAll(a, "-", "_"))
    switch a2 {
    case "TAJC": return TAJC, nil
    case "TAJM": return TAJM, nil
    case "SJC": return SJC, nil
    case "SJM": return SJM, nil
    case "LYS": return LYS, nil
    case "ATC": return ATC, nil
    case "EWC": return EWC, nil
    case "BUNDLE_TAJC_EWC": return BUNDLE_TAJC_EWC, nil
    case "TAP_CHALLENGE": return TAP_CHALLENGE, nil
    case "TAPCIP": return TAPCIP, nil
    }

    switch a {
    case "the-artists-journey": return TAJC, nil
    case "the-artists-journey-masterclass": return TAJM, nil
    case "sjc-course": return SJC, nil
    case "studio-journey-masterclass": return SJM, nil
    case "light-your-creative-studio-like-a-pro": return LYS, nil
    case "activating-the-canvas": return ATC, nil
    case "experimenting-with-color": return EWC, nil
    case "bundle-the-artists-journey-experimenting-with-color": return BUNDLE_TAJC_EWC, nil
    case "creativity-challenge": return TAP_CHALLENGE, nil
    case "creativity-immersion": return TAPCIP, nil
    }
    return -1, fmt.Errorf("Cannot build course acronym from unknown acronym string: %s", a)
}

/*
func (s *TeachableStudent) IsInCourse(c CourseAcronym) bool {
    err := c.EnsureValid()
    if err != nil {
        panic(fmt.Sprintf("%s", err))
    }
}

func (s *TeachableStudent) WasInCourse(c CourseAcronym) bool {
   err := c.EnsureValid()
}
*/


// --------------------------------------------
// Messages

// Most list retrieval responses implement this interface
type ListResponse interface {
    TotalResults()  uint64
    TotalPages() int
}

// ListUsers response from endpoint: https://<account_id>.teachable.com/api/v1/users
type ListUsers struct {
    Users       []ListUsersUser     `json:"users"`
    Metadata    ListUsersMetadata   `json:"meta"`
}

type _ListUsers ListUsers

type ListUsersUser struct {
    CreatedAt               string      `json:"created_at"`
    Role                    string      `json:"role"`
    SanitizedName           string      `json:"sanitized_name"`
    GravatarUrl             string      `json:"gravatar_url"`
    IsTeachableAccount      bool        `json:"is_teachable_account"`
    //AuthorBio               string      `json:"author_bio"`
    AuthorBioId             uint64      `json:"author_bio_id"`
    CustomRole              string      `json:"custom_role"`
    CustomRoleName          string      `json:"custom_role_name"`
    PrimaryOwner            bool        `json:"primary_owner?"`
    ShowCustomRoleUpgrade   bool        `json:"show_custom_role_upgrade?"`
    HasZoomCredential       bool        `json:"has_zoom_credential?"`
    Email                   string      `json:"email" csv:"email"`
    Notes                   string      `json:"notes"`
    AffiliateCode           string      `json:"affiliate_code" csv:"affiliate_code"`
    Name                    string      `json:"name" csv:"fullname"`
    IsOwner                 bool        `json:"is_owner"`
    SignInCount             uint32      `json:"sign_in_count" csv:"sign_in_count"`
    IsStudent               bool        `json:"is_student"`
    IsAffiliate             bool        `json:"is_affiliate"`
    IsAuthor                bool        `json:"is_author"`
    Source                  string      `json:"src" csv:"src"`
    CurrentSignInAt         string      `json:"current_sign_in_at"`
    ConfirmedAt             string      `json:"confirmed_at"`
    LastFour                string      `json:"last_four"`
    PaypalEmail             string      `json:"paypal_email"`
    AffiliateRevenueShare   float32     `json:"affiliate_revenue_share"`
    JoinedAt                string      `json:"joined_at" csv:"joined_at"`
    SignedUpAffiliateCode   string      `json:"signed_up_affiliate_code"`
    AuthorRevenueShare      float32     `json:"author_revenue_share"`
    LastSignInIp            string      `json:"last_sign_in_ip"`
    CurrentSignInIp         string      `json:"current_sign_in_ip"`
    Id                      uint64      `json:"id" csv:"userid"`
    SchoolId                uint64      `json:"school_id"`
    TeachableAccountId      uint64      `json:"teachable_account_id"`
    IsStaffUser             bool        `json:"is_staff_user"`
    AgreeUpdatedPrivacyPolicy       bool    `json:"agree_updated_privacy_policy"`
    UnsubscribeFromMarketingEmails  bool    `json:"unsubscribe_from_marketing_emails" csv:"unsubscribe_from_marketing_emails"`
    Metadata                ListUsersUserMetadata   `json:"meta"`
    TransactionsGross       uint64      `json:"transactions_gross"`
    ShippingAddress         ListUsersUserShippingAddress    `json:"shipping_address"`
}

type ListUsersUserMetadata struct {
    Class           string  `json:"class"`
    Url             string  `json:"url"`
    Name            string  `json:"name"`
    Description     string  `json:"description"`
    ImageUrl        string  `json:"image_url"`
    Status          string  `json:"status"`
}

type ListUsersUserShippingAddress struct {
    Id              uint64  `json:"id"`
    Line1           string  `json:"line1"`
    Line2           string  `json:"line2"`
    City            string  `json:"city"`
    Region          string  `json:"region"`
    PostalCode      string  `json:"postal_code"`
    Country         string  `json:"country"`
}

type ListUsersMetadata struct {
    Page            uint32  `json:"page"`
    Total           uint64  `json:"total"`
    NumberOfPages   uint32  `json:"number_of_pages"`
    From            uint64  `json:"from"`
    To              uint64  `json:"to"`
}

func (l *ListUsers) UnmarshalJSON(jsonStr []byte) error {
    l2 := _ListUsers{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    *l = ListUsers(l2)

    return nil
}

type UserSlice []ListUsersUser

func (l UserSlice) String() string {
	var userListBuffer bytes.Buffer

	for i, u := range l {
		if i > 0 {
			userListBuffer.WriteString(", ")
		}
        str := fmt.Sprintf("%s (ID=%d)", u.Email, u.Id)
		userListBuffer.WriteString(str)
	}
	return userListBuffer.String()
}

func (l *ListUsers) TotalResults() uint64 {
    return l.Metadata.Total
}

func (l *ListUsers) TotalPages() int {
    return to.Int(l.Metadata.NumberOfPages)
}

func (u *ListUsersUser) String() string {
	var strBuffer bytes.Buffer

    shippingAddressStr := "nil"
    if u.ShippingAddress.Id > 0 {
        shippingAddressStr = fmt.Sprintf("%+v", u.ShippingAddress)
    }
    strBuffer.WriteString(fmt.Sprintf(" - %s:\n" +
               "   name: %s\n" +
               "   id: %d\n" +
               "   created_at: %s\n"+
               "   joined_at: %s\n" +
               "   confirmed_at: %s\n" +
               "   current_signin_at: %s\n" +
               "   unsubscribe_marketing?: %t\n" +
               "   is_primary_owner: %t\n" +
               "   is_owner: %t\n" +
               "   is_author: %t\n" +
               "   is_affiliate: %t\n" +
               "   is_student: %t\n" +
               "   is_staff: %t\n" +
               "   is_teachable_account: %t\n" +
               "   agree_new_privacy_policy: %t\n" +
               "   shipping_address: %s\n" +
               "   role: %s\n" +
               "   author_revenue_share: %f\n" +
               "   affiliate_revenue_share: %f\n" +
               "   transaction_gross: %d\n" +
               "   notes: %s\n\n",
               u.Email, u.Name, u.Id, u.CreatedAt, u.JoinedAt, u.ConfirmedAt,
               u.CurrentSignInAt, u.UnsubscribeFromMarketingEmails, u.PrimaryOwner,
               u.IsOwner, u.IsAuthor, u.IsAffiliate, u.IsStudent, u.IsStaffUser,
               u.IsTeachableAccount, u.AgreeUpdatedPrivacyPolicy, shippingAddressStr,
               u.Role, u.AuthorRevenueShare, u.AffiliateRevenueShare,
               u.TransactionsGross, u.Notes))
    return strBuffer.String()
}

// ListEnrollments response from endpoint: https://<account_id>.teachable.com/api/v1/users/<user_id>/enrollments
type ListEnrollments struct {
    Enrollments       []ListEnrollmentsEnrollment   `json:"enrollments"`
    Metadata          ListEnrollmentsMetadata       `json:"meta"`
}

type _ListEnrollments ListEnrollments

type ListEnrollmentsEnrollment struct {
    CreatedAt               string      `json:"created_at"`
    Coupon                  string      `json:"coupon"`
    UserId                  uint64      `json:"user_id"`
    CourseId                uint64      `json:"course_id"`
    PrimaryCourseId         uint64      `json:"primary_course_id"`
    SaleId                  uint64      `json:"sale_id"`
    IsActive                bool        `json:"is_active"`
    EnrolledAt              string      `json:"enrolled_at"`
    PercentComplete         float32     `json:"percent_complete"`
    HasFullAccess           bool        `json:"has_full_access"`
    Id                      uint64      `json:"id"`
    CourseProgressId        uint64      `json:"course_progress_id?"`
    UpdatedAt               string      `json:"updated_at"`
    Metadata                ListEnrollmentsEnrollmentMetadata   `json:"meta"`
    Course                  ListEnrollmentsEnrollmentCourse     `json:"course"`
}

type ListEnrollmentsMetadata ListUsersMetadata

type ListEnrollmentsEnrollmentMetadata ListUsersUserMetadata

type ListEnrollmentsEnrollmentCourse struct {
    CreatedAt               string      `json:"created_at"`
    Path                    string      `json:"path"`
    SafeImageUrl            string      `json:"safe_image_url"`
    HasPublishedLecture     bool        `json:"has_published_lecture"`
    HasPublishedProduct     bool        `json:"has_published_product"`
    //PromoVideo              ListEnrollmentsEnrollmentCoursePromoVideo   `json:"promo_video"`
    Name                    string      `json:"name"`
    Heading                 string      `json:"heading"`
    PageTitle               string      `json:"page_title"`
    MetaDescription         string      `json:"meta_description"`
    FriendlyUrl             string      `json:"friendly_url"`
    Description             string      `json:"description"`
    AuthorBiographyId       uint64      `json:"author_bio_id"`
    Position                uint32      `json:"position"`
    //ConversionPixels        
    //ClosingLetter
    //ImageUrl
    //HeroImageUrl
    IsPublished             bool        `json:"is_published"`
    BundledCoursesCount     uint32      `json:"bundled_courses_count"`
    ChildCourseIds          []uint64    `json:"child_course_ids"`
    //PreEnrollmentCallToAction   
    IsOpen                  bool        `json:"is_open"`
    IsAcceptingPreEnrollments bool      `json:"is_accepting_preenrollments"`
    //UseOldCoursePage
    //SkipThankYouPage
    OnboardedAt             string      `json:"onboarded_at"`
    Id                      uint64      `json:"id"`
    DefaultPageId           uint64      `json:"default_page_id"`
    //CheckoutSidebarPageId
    //CertificatePageId
    //PurchaseRedirectUrl
    //MobileThumbnailUrl
    //MobileHeroImageUrl
    PostPurchasePageId      uint64      `json:"post_purchase_page_id"`
    //IssueCertificatesManually
    //IsVideoCompletionEnforced
    //IsMinimumQuizScoreEnforced
    //MinimumQuizScore
    //MaximumQuizRetakes
    Metadata                ListEnrollmentsEnrollmentCourseMetadata   `json:"meta"`
    //Categories
    //AuthorBio             ListEnrollmentsEnrollmentCourseAuthorBiography
    // Custom fields related to bundles
    HaveBundleData          bool // whether we can trust the following variables
    IsBundle                bool // whether bundle parent or child
    IsBundleParent          bool
    IsBundleChild           bool
    BundleParentId          uint64
    BundleParentName        string
    BundleParentEnrollment  *ListEnrollmentsEnrollment
    BundleChildrenIds       []uint64 // identical to ChildCourseIds
    BundleChildrenNames     []string
    BundleChildrenEnrollments  []*ListEnrollmentsEnrollment
    // Custom fields for acronym
    Acronym                 CourseAcronym
}

type _ListEnrollmentsEnrollmentCourse ListEnrollmentsEnrollmentCourse

type ListEnrollmentsEnrollmentCourseMetadata ListUsersUserMetadata

func (l *ListEnrollments) UnmarshalJSON(jsonStr []byte) error {
    l2 := _ListEnrollments{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    // Build maps and fill in custom fields related to bundles
    var bundleParentsMap map[uint64]*ListEnrollmentsEnrollment
    bundleParentsMap = make(map[uint64]*ListEnrollmentsEnrollment)
    var bundleChildrenMap map[uint64]*ListEnrollmentsEnrollment
    bundleChildrenMap = make(map[uint64]*ListEnrollmentsEnrollment)

    for i := range l2.Enrollments {
        e := &l2.Enrollments[i]
        //log.Printf("Here with enrollment: %p", e)
        if e.PrimaryCourseId == e.CourseId && e.Course.BundledCoursesCount > 0 {
            e.Course.IsBundleParent = true
            e.Course.IsBundle = true
            bundleParentsMap[e.CourseId] = e
        } else if e.PrimaryCourseId != e.CourseId {
            e.Course.IsBundleChild = true
            e.Course.IsBundle = true
            bundleChildrenMap[e.CourseId] = e
        }
        e.Course.HaveBundleData = true // this is for print to work later
    }

    //log.Printf("Primary enrollment map: %+v", bundleParentsMap)
    // Add extra fields to bundle children
    for k, _ := range bundleChildrenMap {
        c := bundleChildrenMap[k]
        parentId := c.PrimaryCourseId
        if p, ok := bundleParentsMap[parentId]; ok {
            c.Course.BundleParentId = parentId
            c.Course.BundleParentName = p.Course.Name
            c.Course.BundleParentEnrollment = p
        } else {
            log.Printf("Error: Couldn't find bundle parent for child course '%s' (id=%d)",
                c.Course.Name, c.Course.Id)
        }
    }

    for k, _ := range bundleParentsMap {
        p := bundleParentsMap[k]
        p.Course.BundleChildrenIds = p.Course.ChildCourseIds // identical copy of original list ID list
        for _, v := range p.Course.BundleChildrenIds {
            if c, ok := bundleChildrenMap[v]; ok {
                p.Course.BundleChildrenNames = append(p.Course.BundleChildrenNames, c.Course.Name)
                p.Course.BundleChildrenEnrollments = append(p.Course.BundleChildrenEnrollments, c)
            } else {
                log.Printf("Error: Couldn't find bundle child %d for parent course '%s' (id=%d)",
                    v, p.Course.Name, p.Course.Id)
            }
        }
    }

    *l = ListEnrollments(l2)

    return nil
}

func (c *ListEnrollmentsEnrollmentCourse) UnmarshalJSON(jsonStr []byte) error {
    c2 := _ListEnrollmentsEnrollmentCourse{}

    err := json.Unmarshal(jsonStr, &c2)
    if err != nil {
        return err
    }

    // Course acronym
    a, err := GetCourseAcronym(c2.FriendlyUrl)
    if err != nil {
        log.Printf("No acronym for course '%s' (id=%d, friendly_url=%s): %s",
            c2.Name, c2.Id, c2.FriendlyUrl, err)
    } else {
        c2.Acronym = a
    }

    *c = ListEnrollmentsEnrollmentCourse(c2)

    return nil
}

func (l *ListEnrollments) TotalResults() uint64 {
    return l.Metadata.Total
}

func (l *ListEnrollments) TotalPages() int {
    return to.Int(l.Metadata.NumberOfPages)
}

func (e *ListEnrollmentsEnrollment) String() string {
	var strBuffer bytes.Buffer

    strBuffer.WriteString(fmt.Sprintf(" - %s:\n" +
               "   pointer_address: %p\n" +
               "   friendly_url: %s\n" +
               "   course_id: %d\n" +
               "   primary_course_id: %d\n" +
               "   sale_id: %d\n" +
               "   enrollment_id: %d\n" +
               "   enrolled_at: %s\n" +
               "   is_active: %t\n" +
               "   coupon: %s\n" +
               "   path: %s\n" +
               "   is_pubished: %t\n" +
               "   is_open: %t\n" +
               "   is_bundle: %t\n",
               e.Course.Name, e, e.Course.FriendlyUrl, e.CourseId,
               e.PrimaryCourseId, e.SaleId, e.Id, e.EnrolledAt,
               e.IsActive, e.Coupon, e.Course.Path,
               e.Course.IsPublished, e.Course.IsOpen, e.Course.IsBundle))

    if e.Course.IsBundleChild {
        strBuffer.WriteString(fmt.Sprintf("   is_bundle_child: %t\n" +
                   "   bundle_parent_id: %d\n" +
                   "   bundle_parent_name: %s\n" +
                   "   bundle_parent_pointer: %p\n",
                   e.Course.IsBundleChild, e.Course.BundleParentId,
                   e.Course.BundleParentName, e.Course.BundleParentEnrollment))
    } else if e.Course.IsBundleParent {
        strBuffer.WriteString(fmt.Sprintf("   bundled_courses_count: %d\n" +
                   "   is_bundle_parent: %t\n" +
                   "   bundle_children_ids: %d\n" +
                   "   bundle_children_names: %+q\n" +
                   "   bundle_children_pointers: %p\n",
                   e.Course.BundledCoursesCount, e.Course.IsBundleParent,
                   e.Course.BundleChildrenIds, e.Course.BundleChildrenNames,
                   e.Course.BundleChildrenEnrollments))
    }
    return strBuffer.String()
}

// ListCourses response from endpoint: https://<account_id>.teachable.com/api/v1/courses

type ListCourses struct {
    Courses     []ListCoursesCourse     `json:"courses"`
    Metadata    ListCoursesMetadata   `json:"meta"`
}

type ListCoursesCourse struct {
    Name                string      `json:"name"`
    Id                  uint64      `json:"id"`
    ImageUrl            string      `json:"image_url"`
    IsPublished         bool        `json:"is_published"`
    AuthorBiographyId   uint64      `json:"author_bio_id"`
    Position            uint32      `json:"position"`
    SafeImageUrl        string      `json:"safe_image_url"`
    BundledCoursesCount uint32      `json:"bundled_courses_count"`
    ChildCoursesCount   uint32      `json:"child_courses_count"`
    DefaultPageId       uint64      `json:"default_page_id"`
    CertificatePageId   uint64      `json:"certificate_page_id"`
}

type ListCoursesMetadata ListUsersMetadata

type _ListCourses ListCourses

func (l *ListCourses) UnmarshalJSON(jsonStr []byte) error {
    l2 := _ListCourses{}

    err := json.Unmarshal(jsonStr, &l2)
    if err != nil {
        return err
    }

    *l = ListCourses(l2)

    return nil
}

func (c *ListCoursesCourse) String() string {
    var strBuffer bytes.Buffer

    strBuffer.WriteString(fmt.Sprintf(" - %s:\n" +
               "   pointer_address: %p\n" +
               "   course_id: %d\n" +
               "   image_url: %s\n" +
               "   safe_image_url: %s\n" +
               "   bundled_courses_count: %d\n" +
               "   child_courses_count: %d\n" +
               "   is_pubished: %t\n",
               c.Name, c, c.Id, c.ImageUrl, c.SafeImageUrl,
               c.BundledCoursesCount, c.ChildCoursesCount,
               c.IsPublished))

    return strBuffer.String()
}

// RetrieveCourse response from endpoint: https://<account_id>.teachable.com/api/v1/courses/<course_id>

type RetrieveCourse ListEnrollmentsEnrollmentCourse

type _RetrieveCourse RetrieveCourse

func (c *RetrieveCourse) UnmarshalJSON(jsonStr []byte) error {
    c2 := _RetrieveCourse{}

    err := json.Unmarshal(jsonStr, &c2)
    if err != nil {
        return err
    }

    // Course acronym
    a, err := GetCourseAcronym(c2.FriendlyUrl)
    if err != nil {
        if DEBUG && DEBUG_VERBOSE {
            log.Printf("No acronym for course '%s' (id=%d, friendly_url=%s): %s",
                c2.Name, c2.Id, c2.FriendlyUrl, err)
        }
        c2.Acronym = CourseAcronym(-1)
    } else {
        c2.Acronym = a
    }

    *c = RetrieveCourse(c2)

    return nil
}

func (c *RetrieveCourse) IsAcronym(a string) bool {
    err := c.Acronym.EnsureValid()
    if err != nil {
        //log.Printf("Comparing course with invalid acronym '%s': %s", c.Acronym.String(), err)
        return false
    }
    a2, err := GetCourseAcronym(a)
    if err != nil {
        if DEBUG {
            log.Printf("Invalid course acronym given '%s': %s", a, err)
        }
        return false
    }
    //log.Printf("Comparing course '%s' (id=%d, friendly_url=%s, acronym=%s) with acronym: %s=%s",
        //c.Name, c.Id, c.FriendlyUrl, c.Acronym.String(), a, a2.String())
    return c.Acronym == a2
}


func (c *RetrieveCourse) String() string {
	var strBuffer bytes.Buffer

    strBuffer.WriteString(fmt.Sprintf(" - %s:\n" +
               "   pointer_address: %p\n" +
               "   friendly_url: %s\n" +
               "   course_id: %d\n" +
               "   path: %s\n" +
               "   is_pubished: %t\n" +
               "   is_open: %t\n",
               c.Name, c, c.FriendlyUrl, c.Id, c.Path,
               c.IsPublished, c.IsOpen))

    if c.BundledCoursesCount > 0 {
        strBuffer.WriteString(fmt.Sprintf("   bundled_courses_count: %d\n" +
                   "   bundle_children_ids: %d\n",
                   c.BundledCoursesCount, c.ChildCourseIds))
    }
    return strBuffer.String()
}

// Get sale 
type RetrieveSale struct {
    CreatedAt               string      `json:"created_at"`
    VatRate                 float32      `json:"vat_rate"`
    SubscriptionAmountToBill  uint32    `json:"subscription_amount_to_bill"`
    SubscriptionStatus      string      `json:"subscription_status"`
    ProductId               uint64      `json:"product_id"`
    UserId                  uint64      `json:"user_id"`
    IsActive                bool        `json:"is_active"`
    //CurrentPeriodStart      string      `json:"current_period_start"`
    //CurrentPeriodEnd        string      `json:"current_period_end"`
    IsRecurring             bool        `json:"is_recurring"`
    Currency                string      `json:"currency"`
    PaymentMethod           string      `json:"payment_method"`
    PurchasedAt             string      `json:"purchased_at"`
    Country                 string      `json:"country"`
    NextPeriodStart         string      `json:"next_period_start"`
    NumberOfPaymentRequired uint32      `json:"num_payments_required"`
    FullyPaidPlan           bool        `json:"fully_paid_plan"`
    //VatTaxId                vat_tax_id
    Id                      uint64      `json:"id"`
    //Metadata              RetrieveSaleMetadata   `json:"meta"`
    //Product               RetrieveSaleProduct   `json:"product"`
    Coupon                  RetrieveSaleCoupon   `json:"coupon"`
    //Transactions          RetrieveSaleTransactions   `json:"transactions"`
    User                    ListUsersUser   `json:"user"`
    //Enrollments             []ListEnrollmentsEnrollment   `json:"enrollments"`
}

type _RetrieveSale RetrieveSale

type RetrieveSaleCoupon struct {
    CreatedAt             string      `json:"created_at"`
    ScopeName             string      `json:"scope_name"`
    PricingPlanSpecific   bool        `json:"pricing_plan_specific"`
    NewPurchasePrice      uint64      `json:"new_purchase_price"`
    FormattedDiscount     string      `json:"formatted_discount"`
    FormattedPrice        string      `json:"formatted_price"`
    CalculatedDiscount    uint64      `json:"calculated_discount"`
    Currency              string      `json:"currency"`
    DiscountPercent       float32     `json:"discount_percent"`
    //DiscountAmount        uint64      `json:"discount_amount"`
    NumberAvailable       uint64      `json:"number_available"`
    Code                  string      `json:"code"`
    Name                  string      `json:"name"`
    ExpirationDate        string      `json:"expiration_date"`
    ProductId             uint64      `json:"product_id"`
    IsPublished           bool        `json:"is_published"`
    DurationKind          string      `json:"duration_kind"`
    NumberOfUses          uint64      `json:"number_of_uses"`
    //ClassPeriodId         uint64      `json:"class_period_id"`
    SiteWide              bool        `json:"site_wide"`
    Id                    uint64      `json:"id"`
    //Metadata              RetrieveSaleCouponMetadata  `json:"meta"`
    //Product               RetrieveSaleProduct     `json:"product"`
    //CreatorProduct        ???     `json:"creator_product"`
    //ProductCollection     ???     `json:"product_collection"`
}


//type RetrieveSaleMetadata ListUsersMetadata

func (s *RetrieveSale) UnmarshalJSON(jsonStr []byte) error {
    s2 := _RetrieveSale{}

    err := json.Unmarshal(jsonStr, &s2)
    if err != nil {
        return err
    }

    *s = RetrieveSale(s2)

    return nil
}

