package main

import (
    "os"
	"bytes"
	"log"
	"fmt"
    "time"
	flag "github.com/spf13/pflag"
	ac "bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
    "regexp"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] [EMAIL_OR_ID] \n", os.Args[0])
     fmt.Printf("Search for an ActiveCampaign subscriber with the given email or ID\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun, exactMatch bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "To Be Implemented")
	flag.BoolVarP(&exactMatch, "exact-match", "e", false, "To Be Implemented")

    flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
        log.Fatal("No email or ID provided")
        return
    }
    emailOrId := string(args[0])
    var regexEmail = regexp.MustCompile(`^\S+@\S+$`)
    var regexId = regexp.MustCompile(`^\d+$`)

    ac.SecretsFilePath = "ac_secrets.yml"
    start := time.Now()

    //var contact = new(ac.RetrieveContact{})
    if regexEmail.FindString(emailOrId) != "" {
        contact, err := ac.GetContactByEmail(emailOrId)
        if err != nil {
            log.Printf("Error retrieving contact with email '%s': %s\n", emailOrId, err)
            return
        }
        log.Printf("Got contact: %#v", contact)
    } else if regexId.FindString(emailOrId) != "" {
        contact, err := ac.GetContactById(emailOrId)
        if err != nil {
            log.Printf("Error retrieving contact with ID %s: %s\n", emailOrId, err)
            return
        }
        log.Printf("Got contact: %#v", contact)
    } else {
        log.Fatal("Expected an email address or an ID, but got: %s", emailOrId)
        return
    }
    duration := time.Since(start)
    log.Printf("Found contact in: %v", duration)
}

func printListOfAutomations(list []ac.ListAutomationsAutomation) {
	var automationListBuffer bytes.Buffer

	for i, a := range list {
		if i > 0 {
			automationListBuffer.WriteString(", ")
		}
		automationListBuffer.WriteString(a.Name)
	}
	fmt.Println(automationListBuffer.String())
}


