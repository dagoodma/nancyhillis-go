package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"

	"github.com/Songmu/prompter"
	"github.com/davecgh/go-spew/spew"
	flag "github.com/spf13/pflag"
	"github.com/stripe/stripe-go"
	"gopkg.in/cheggaaa/pb.v1"

	mm "bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	"bitbucket.org/dagoodma/nancyhillis-go/stripewrap"
	//sj "bitbucket.org/dagoodma/nancyhillis-go/studiojourney"
	_ "bitbucket.org/dagoodma/nancyhillis-go/util"
)

var Debug = false // supress extra messages if false

// Country code lookup stuff
var CountryCodeDataFilePath = "./data/country_codes.json"

type Country struct {
	data map[string]string
}

func main() {
	var verbose int
	var dryRun bool
	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without updating Stripe customers")

	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		log.Fatal("No membermouse IDs given")
		return
	}

	var allMembers, adminMembers, alreadyImportedMembers, notImportedMembers []*mm.Member
	//var newCustomers, updatedCustomers []*stripe.Customer
	stripeCustomerByMemberId := map[uint32]*(stripe.Customer){}

	// Lookup and import each customer by email
	haveError := false
	count := len(args)
	log.Printf("Looking up %d members in Membermouse & Stripe...\n", count)
	bar := pb.StartNew(count)
	bar.Increment()
	for _, idStr := range args {
		id, err := strconv.ParseUint(idStr, 10, 32) // convert MM id to uint64
		if err != nil {
			log.Fatalf("Expected integer for member id \"%s\". %q",
				idStr, err)
		}
		m, err := mm.GetMemberById(uint32(id))
		if err != nil {
			log.Println("Failed to fetch member with id \"%s\". %q\n",
				idStr, err)
			continue
		}
		s, err := m.GetStatus()
		if err != nil {
			log.Println("Failed to determine status of member with id \"%s\". %q\n",
				idStr, err)
			continue
		}
		email := s.Email

		allMembers = append(allMembers, m)
		isAdmin := m.MembershipLevel == ""

		if isAdmin {
			adminMembers = append(adminMembers, m)
			continue // don't look up admins in Stripe
		}
		// Look for their email in Stripe
		c, err := stripewrap.GetCustomerByEmail(email)

		// Did we not find them in Stripe?
		if err != nil || c == nil {
			if verbose > 1 {
				log.Printf("Failed finding member %d in stripe. %v\n",
					id, err)
			}
			if s.IsMigrated {
				log.Printf("Error: Member %d with email \"%s\" was not in Stripe but already migrated.\n",
					id, email)
				haveError = true
				continue // skip this and keep moving. report the issues later
			}
			notImportedMembers = append(notImportedMembers, m)
		} else if c != nil {
			// We found them in Stripe
			if verbose > 1 {
				log.Printf("Found member %d with email \"%s\" in stripe with customer id: %s\n",
					id, email, c.ID)
			}
			stripeCustomerByMemberId[uint32(id)] = c
			alreadyImportedMembers = append(alreadyImportedMembers, m)
		}

		bar.Increment()
	}
	bar.FinishPrint("Finished looking up members.")

	// Report
	log.Printf("Found %d total, %d imported, %d not imported, %d admin members.\n",
		len(allMembers), len(alreadyImportedMembers), len(notImportedMembers), len(adminMembers))

	//spew.Dump(alreadyImportedMembers)
	//spew.Dump(stripeCustomerByMemberId)

	if verbose > 1 {
		log.Println("Not imported members:")
		printListOfMembersEmails(notImportedMembers)
	}

	if haveError {
		log.Fatalln("Encountered errors looking up members. Exiting...")
		return
	}

	// Import members
	country := &Country{}
	country = country.readCountriesDataFile()
	log.Printf("Importing %d new customers into Stripe...\n", len(notImportedMembers))
	//bar = pb.StartNew(len(notImportedMembers))
	for _, m := range notImportedMembers {
		if verbose > 0 {
			spew.Dump(m)
		}
		fullName := fmt.Sprintf("%s %s", m.FirstName, m.LastName)
		countryCode := ""
		var err error
		if len(m.BillingCountry) > 0 {
			countryCode, err = country.GetCode(m.BillingCountry)
			if err != nil {
				log.Fatalf("Failed looking up billing country \"%s\" for member \"%s\". %v",
					m.BillingCountry, m.Email, err)
			}
		}
		msg := fmt.Sprintf("Import \"%s\" (%s, country=%s) into Stripe?",
			m.Email, fullName, countryCode)
		if prompter.YN(msg, true) {
			// do something
			if !dryRun {
				c, err := stripewrap.CreateCustomer(m.Email, fullName)
				if err != nil {
					log.Fatalf("Failed creating stripe customer \"%s\". %v",
						fullName, err)
				}
				c.Metadata["contact_name"] = fullName
				c.Metadata["first_name"] = m.FirstName
				c.Metadata["email"] = m.Email
				if len(countryCode) > 0 {
					c.Metadata["country"] = countryCode
				}
				c.Metadata["sj_billing_complete"] = "true"
				c.Metadata["sj_founder"] = "true"
				c2, err := stripewrap.UpdateCustomerMetadata(c.ID, c.Metadata)
				if err != nil {
					log.Fatalf("Failed updating stripe customer metadata \"%s\". %v",
						fullName, err)
				}
				if verbose > 1 {
					log.Printf("Finished adding stripe customer for member %d:\n",
						m.MemberId)
					spew.Dump(c2)
				}
			} else {
				log.Printf("Dry run skipping creating customer.\n")
			}
		}
		//bar.Increment()
	}
	//bar.FinishPrint("Finished importing members into Stripe.")

	return
}

func printListOfMembersEmails(list []*mm.Member) {
	var memberListBuffer bytes.Buffer

	for i, m := range list {
		if i > 0 {
			memberListBuffer.WriteString(", ")
		}
		memberListBuffer.WriteString(m.Email)
	}
	fmt.Println(memberListBuffer.String())
}

// NewCountry will create a new instance of country struct
func NewCountry() *Country {
	instance := &Country{}

	instance = instance.readCountriesDataFile()

	return instance
}

// Read countries data file
func (country *Country) readCountriesDataFile() *Country {
	if len(country.data) > 0 {
		return country
	}

	dataPath, _ := filepath.Abs(CountryCodeDataFilePath)

	file, _ := ioutil.ReadFile(dataPath)

	json.Unmarshal([]byte(file), &country.data)

	return country
}

// Get a single country by the country ISO 3166-1 Alpha-2 code
func (country *Country) GetCode(countryName string) (string, error) {
	code := country.readCountriesDataFile().data[countryName]

	if len(code) < 1 {
		return "", NewValidationError(countryName)
	}

	return code, nil
}

// ValidationError struct
type ValidationError struct {
	message string
}

// implementing error method
func (v ValidationError) Error() string {
	return v.message
}

// NewValidationError will create a new validation error class
func NewValidationError(name string) error {
	ve := &ValidationError{}
	ve.message = "The country [ " + name + " ] does not exist. No ISO 3166-1 Alpha-2 code found."

	return ve
}
