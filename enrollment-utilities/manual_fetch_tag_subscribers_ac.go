package main

import (
	"bytes"
	"log"
	"fmt"
    "time"
    "os"
	flag "github.com/spf13/pflag"
	ac "bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] [TAG] \n", os.Args[0])
     fmt.Printf("Search for ActiveCampaign contacts with the given TAG (must be an exact match).\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun, exactMatch bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "To Be Implemented")
	flag.BoolVarP(&exactMatch, "exact-match", "e", false, "To Be Implemented") //"Tag name must match exactly, otherwise it can be a substring")

	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		log.Fatal("No tag name provided")
		return
	}

	tagName := string(args[0])

    ac.SecretsFilePath = "ac_secrets.yml"
    start := time.Now()
    allContacts, err := ac.GetContactsByTag(tagName)
    if err != nil {
        log.Printf("Error retrieving contact. %v\n", err)
        return
    }
    duration := time.Since(start)
    log.Printf("Got %d contacts in: %v", len(allContacts), duration)

    printListOfSubscribersEmails(allContacts)
}

func printListOfSubscribersEmails(list []ac.ListContactsContact) {
	var subscriberListBuffer bytes.Buffer

	for i, c := range list {
		if i > 0 {
			subscriberListBuffer.WriteString(", ")
		}
		subscriberListBuffer.WriteString(c.Email)
	}
	fmt.Println(subscriberListBuffer.String())
}
