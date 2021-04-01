package main

import (
    "os"
	//"bytes"
	"log"
	"fmt"
    "time"
	flag "github.com/spf13/pflag"
	"bitbucket.org/dagoodma/nancyhillis-go/teachable"
	ac "bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
    "regexp"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] <EMAIL> \n", os.Args[0])
     fmt.Printf("Search for a student in Teachable and AC and compare course enrollment between each given the user email\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "To Be Implemented")
	//flag.BoolVarP(&exactMatch, "exact-match", "e", false, "To Be Implemented")

    flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

    var regexEmail = regexp.MustCompile(`^\S+@\S+$`)
	if len(args) < 1 {
        log.Fatal("No email provided")
        return
    }
    email := string(args[0])
    if regexEmail.FindString(email) == "" {
        log.Fatal("Expected valid email address but got: %s", email)
        return
    }

    teachable.SecretsFilePath = "teachable_secrets.yml"
    ac.SecretsFilePath = "ac_secrets.yml"

    start := time.Now()

    // -------------- Active Campaign
    log.Println("Fetching contact in ActiveCampaign...")
    contact, err := ac.GetContactByEmail(email)
    if err != nil {
        log.Printf("Failed fetching contact '%s' in ActiveCampaign: %s\n", email, err)
    } else {
        // Contact
        log.Println(contact)

        // Tags
        tags, err := ac.GetContactTags(contact.Id)
        if err != nil {
            log.Printf("Failed fetching enrollments for user '%s' in Teachable: %s\n", email, err)
            return
        }
        log.Printf("Tags: %s\n\n", ac.GetTagsString(tags))

        // Automations
        /* TODO
        automations, err := ac.GetContactAutomations(contact.Id)
        if err != nil {
            log.Printf("Failed fetching enrollments for user '%s' in Teachable: %s\n", email, err)
            return
        }
        log.Printf("Tags: %s\n", ac.GetTagsString(tags))
        */
    }

    // -------------- Teachable
    log.Println("Fetching user in Teachable...")
    user, err := teachable.GetUserByEmail(email)
    if err != nil {
        log.Printf("Failed fetching user '%s' in Teachable: %s\n", email, err)
    } else {
        log.Println(user)
        enrollments, err := teachable.GetUserEnrollments(user.Id)
        if err != nil {
            log.Printf("Failed fetching enrollments for user '%s' in Teachable: %s\n", email, err)
            return
        }
        log.Printf("Enrollments for Teachable user '%s':", email)
        for i := range(enrollments) {
            fmt.Println(&enrollments[i])
        }
    }

    duration := time.Since(start)
    log.Printf("Found user in: %v", duration)

    log.Println("Active Campaign profile: ", ac.GetContactProfileUrlById(contact.Id))
    log.Println("Teachable profile: ", teachable.GetUserProfileUrlById(user.Id))
}
