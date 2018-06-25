package teachable

import (
	//"drip"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"strings"
	//"time"
	//"util"

	"github.com/gosexy/to"
	"github.com/gosexy/yaml"
	//"github.com/ugorji/go/codec"
)

/*
 * Settings
 */
var SecretsFilePath = "/var/webhook/secrets/teachable_secrets.yml"

/*
 * Action handlers
 */
func EnsureValidRelicId(actualRelicId string) error {
	secrets := yaml.New()
	secrets, err := yaml.Open(SecretsFilePath)
	if err != nil {
		log.Fatalf("Could not open YAML secrets file: %s", err.Error())
	}

	expectedRelicId := to.String(secrets.Get("RELIC_ID"))

	if actualRelicId != expectedRelicId {
		//log.Printf("WARNING: Invalid relic ID \"%s\" when expecting: %s\n",
		message := fmt.Sprintf("Received invalid relic ID \"%s\" in webhook", actualRelicId)
		log.Printf("Error: %s", message)
		return errors.New(message)
	}
	return nil
}

func EnsureValidWebhook(h *WebhookHeader, d []byte) error {
	// Validate the header
	relicId := h.XNewrelicId
	err := EnsureValidRelicId(relicId)
	if err != nil {
		return err
	}

	if len(h.XNewrelicTransaction) < 1 {
		return errors.New("Missing relic transaction field")
	}

	if !strings.Contains(h.UserAgent, "rest-client") {
		message := fmt.Sprintf("Incorrect user agent field \"%s\"", h.UserAgent)
		return errors.New(message)
	}

	// Validate the data
	ss := to.String(d)
	if ss == "null" {
		message := fmt.Sprintf("Invalid or malformed data")
		return errors.New(message)
	}

	return nil
}

/*
 * Messages and unmarshallers
 */
// Json Header of incomming Teachable webhook messages
type WebhookHeader struct {
	Accept               string                 `json:"Accept"`
	AcceptEncoding       string                 `json:"Accept-Encoding"`
	ContentLength        string                 `json:"Content-Length"`
	ContentType          string                 `json:"Content-Type"`
	UserAgent            string                 `json:"User-Agent"`
	XNewrelicId          string                 `json:"X-Newrelic-Id"`
	XNewrelicTransaction string                 `json:"X-Newrelic-Transaction"`
	Extra                map[string]interface{} `json:"-"`
}

type _WebhookHeader WebhookHeader

func (h *WebhookHeader) UnmarshalJSON(jsonStr []byte) error {
	h2 := _WebhookHeader{}

	err := json.Unmarshal(jsonStr, &h2)
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonStr, &(h2.Extra))
	if err != nil {
		return err
	}

	typ := reflect.TypeOf(h2)
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		jsonTag := strings.Split(field.Tag.Get("json"), ",")[0]
		if jsonTag != "" && jsonTag != "-" {
			delete(h2.Extra, jsonTag)
		}
	}

	*h = WebhookHeader(h2)

	return nil
}

// Student joined school
type NewStudent struct {
	Type        string           `json:"type"`
	Id          float64          `json:"id"`
	Created     string           `json:"created"`
	HookEventId float64          `json:"hook_event_id"`
	Object      NewStudentObject `json:"object,string"`
	//Extra map[string]interface{}
}

type NewStudentObject struct {
	Email                          string  `json:"email"`
	Name                           string  `json:"name"`
	IdRaw                          float64 `json:"id"`
	Id                             string
	SchoolIdRaw                    float64 `json:"school_id"`
	SchoolId                       string
	Role                           string `json:"role"`
	UnsubscribeFromMarketingEmails bool   `json:"unsubscribe_from_marketing_emails"`
	//Extra map[string]interface{}
}

type _NewStudent NewStudent

func (s *NewStudent) UnmarshalJSON(jsonStr []byte) error {
	s2 := _NewStudent{}

	err := json.Unmarshal(jsonStr, &s2)
	if err != nil {
		return err
	}

	s2.Object.Id = to.String(to.Uint64(s2.Object.IdRaw))
	s2.Object.SchoolId = to.String(to.Uint64(s2.Object.SchoolIdRaw))

	*s = NewStudent(s2)

	return nil
}

// Student enrolled in course
type StudentEnrolled struct {
	Type        string                `json:"type"`
	Id          float64               `json:"id"`
	Created     string                `json:"created"`
	HookEventId float64               `json:"hook_event_id"`
	Object      StudentEnrolledObject `json:"object,string"`
	//Extra map[string]interface{}
}

type StudentEnrolledObject struct {
	Coupon             string  `json:"coupon"`
	CourseIdRaw        float64 `json:"course_id"`
	CourseId           string
	CreatedAt          string  `json:"created_at"`
	EnrolledAt         string  `json:"enrolled_at"`
	HasFullAccess      bool    `json:"has_full_access"`
	IdRaw              float64 `json:"id"`
	Id                 string
	IsActive           bool    `json:"is_active"`
	PercentComplete    float64 `json:"percent_complete"`
	PrimaryCourseIdRaw float64 `json:"primary_course_id"`
	PrimaryCourseId    string
	SaleIdRaw          float64 `json:"sale_id"`
	SaleId             string
	User               StudentEnrolledUser `json:"user,string"`
	UserIdRaw          float64             `json:"user_id"`
	UserId             string
	//Course
	//Meta
	//Extra map[string]interface{}
}

type StudentEnrolledUser struct {
	Email                          string  `json:"email"`
	Name                           string  `json:"name"`
	IdRaw                          float64 `json:"id"`
	Id                             string
	SchoolIdRaw                    float64 `json:"school_id"`
	SchoolId                       string
	Role                           string  `json:"role"`
	UnsubscribeFromMarketingEmails bool    `json:"unsubscribe_from_marketing_emails"`
	GravatarUrl                    string  `json:"gravatar_url"`
	IsAffiliate                    bool    `json:"is_affiliate"`
	IsAuthor                       bool    `json:"is_author"`
	IsOwner                        bool    `json:"is_owner"`
	IsStaffUser                    bool    `json:"is_staff_user"`
	IsStudent                      bool    `json:"is_student"`
	IsTeachableAccount             bool    `json:"is_teachable_account"`
	JoinedAt                       string  `json:"joined_at"`
	CurrentSignInIp                string  `json:"current_sign_in_ip"`
	CurrentSignInAt                string  `json:"current_sign_in_at"`
	LastSignInIp                   string  `json:"last_sign_in_ip"`
	PaypalEmail                    string  `json:"paypal_email"`
	SignInCountRaw                 float64 `json:"sign_in_count"`
	SignInCount                    uint64
	//LastFour string
	//Meta
	//Notes
	//SignedUpAffiliateCode string
	//Source
}

type _StudentEnrolled StudentEnrolled

func (s *StudentEnrolled) UnmarshalJSON(jsonStr []byte) error {
	s2 := _StudentEnrolled{}

	err := json.Unmarshal(jsonStr, &s2)
	if err != nil {
		return err
	}

	// Convert from raw types in StudentEnrolledObject struct
	s2.Object.Id = to.String(to.Uint64(s2.Object.IdRaw))
	s2.Object.CourseId = to.String(to.Uint64(s2.Object.CourseIdRaw))
	s2.Object.PrimaryCourseId = to.String(to.Uint64(s2.Object.PrimaryCourseIdRaw))
	s2.Object.SaleId = to.String(to.Uint64(s2.Object.SaleIdRaw))
	s2.Object.UserId = to.String(to.Uint64(s2.Object.UserIdRaw))

	// Convert from raw types in StudentEnrolledUser struct
	s2.Object.User.Id = to.String(to.Uint64(s2.Object.User.IdRaw))
	s2.Object.User.SchoolId = to.String(to.Uint64(s2.Object.User.SchoolIdRaw))
	s2.Object.User.SignInCount = to.Uint64(s2.Object.User.SignInCountRaw)

	*s = StudentEnrolled(s2)

	return nil
}

// Student cancelled course
type StudentCancelled struct {
	Type        string                 `json:"type"`
	Id          float64                `json:"id"`
	Created     string                 `json:"created"`
	HookEventId float64                `json:"hook_event_id"`
	Object      StudentCancelledObject `json:"object,string"`
	//Extra map[string]interface{}
}

type StudentCancelledObject struct {
	SchoolIdRaw float64 `json:"school_id"`
	SchoolId    string
	IdRaw       float64 `json:"id"`
	Id          string
	IsActive    bool                 `json:"is_active"`
	User        StudentCancelledUser `json:"user,string"`
	UserIdRaw   float64              `json:"user_id"`
	UserId      string
	//Coupon             string  `json:"coupon"`
	//Course
	//Meta
	//Extra map[string]interface{}
}

type StudentCancelledUser struct {
	Email                          string  `json:"email"`
	Name                           string  `json:"name"`
	IdRaw                          float64 `json:"id"`
	Id                             string
	SchoolIdRaw                    float64 `json:"school_id"`
	SchoolId                       string
	Role                           string  `json:"role"`
	UnsubscribeFromMarketingEmails bool    `json:"unsubscribe_from_marketing_emails"`
	LastSignInAt                   string  `json:"last_sign_in_at"`
	SignInCountRaw                 float64 `json:"sign_in_count"`
	SignInCount                    uint64
	//Source
}

type _StudentCancelled StudentCancelled

func (s *StudentCancelled) UnmarshalJSON(jsonStr []byte) error {
	s2 := _StudentCancelled{}

	err := json.Unmarshal(jsonStr, &s2)
	if err != nil {
		return err
	}

	// Convert from raw types in StudentEnrolledObject struct
	s2.Object.Id = to.String(to.Uint64(s2.Object.IdRaw))
	s2.Object.SchoolId = to.String(to.Uint64(s2.Object.SchoolIdRaw))
	s2.Object.UserId = to.String(to.Uint64(s2.Object.UserIdRaw))

	// Convert from raw types in StudentEnrolledUser struct
	s2.Object.User.Id = to.String(to.Uint64(s2.Object.User.IdRaw))
	s2.Object.User.SchoolId = to.String(to.Uint64(s2.Object.User.SchoolIdRaw))
	s2.Object.User.SignInCount = to.Uint64(s2.Object.User.SignInCountRaw)

	*s = StudentCancelled(s2)

	return nil
}

// Student updated profile message
type StudentUpdated struct {
	Type        string               `json:"type"`
	Id          float64              `json:"id"`
	Created     string               `json:"created"`
	HookEventId float64              `json:"hook_event_id"`
	Object      StudentUpdatedObject `json:"object,string"`
	//Extra map[string]interface{}
}

type StudentUpdatedObject struct {
	Email                          string  `json:"email"`
	Name                           string  `json:"name"`
	IdRaw                          float64 `json:"id"`
	Id                             string
	SchoolIdRaw                    float64 `json:"school_id"`
	SchoolId                       string
	Role                           string `json:"role"`
	UnsubscribeFromMarketingEmails bool   `json:"unsubscribe_from_marketing_emails"`
	OldName                        string `json:"old_name"`
	NewName                        string `json:"new_name"`
	OldEmail                       string `json:"old_email"`
	NewEmail                       string `json:"new_email"`
	NameUpdated                    bool
	EmailUpdated                   bool
	//Extra map[string]interface{}
}

type _StudentUpdated StudentUpdated

func (s *StudentUpdated) UnmarshalJSON(jsonStr []byte) error {
	s2 := _StudentUpdated{}

	err := json.Unmarshal(jsonStr, &s2)
	if err != nil {
		return err
	}

	s2.Object.Id = to.String(to.Uint64(s2.Object.IdRaw))
	s2.Object.SchoolId = to.String(to.Uint64(s2.Object.SchoolIdRaw))

	s2.Object.NameUpdated = len(s2.Object.OldName) != 0 &&
		len(s2.Object.NewName) != 0 && s2.Object.OldName != s2.Object.NewName
	s2.Object.EmailUpdated = len(s2.Object.OldEmail) != 0 &&
		len(s2.Object.NewEmail) != 0 && s2.Object.OldEmail != s2.Object.NewEmail

	*s = StudentUpdated(s2)

	return nil
}
