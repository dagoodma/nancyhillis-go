package main

import (
    "os"
	"bytes"
	"log"
	"fmt"
    "time"
	flag "github.com/spf13/pflag"
	ac "bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] [NAME] \n", os.Args[0])
     fmt.Printf("Search for ActiveCampaign contacts in an automation with the given NAME.\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun, printEmails, showCompleted, showAll bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "To Be Implemented")
	flag.BoolVarP(&printEmails, "emails", "e", false, "Print the emails of each contact")
	flag.BoolVarP(&showCompleted, "completed", "c", false, "Print the emails of each contact")
	flag.BoolVarP(&showAll, "all", "a", false, "Show both completed and scheduled contacts")

    flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

    if showCompleted && showAll {
        log.Fatal("Cannot use both 'completed' and 'all' flags together")
        return
    }

	if len(args) < 1 {
        log.Fatal("No automation name provided")
        return
	}
    automationName := string(args[0])

    ac.SecretsFilePath = "ac_secrets.yml"
    start := time.Now()
    automations, err := ac.GetAutomationsByName(automationName, true)
    if err != nil {
        log.Printf("Error retrieving automations. %v\n", err)
        return
    }
	for _, a := range automations {
        log.Printf("Got automation with name: %#v", a.Name)
    }
    if len(automations) > 1 {
        log.Printf("Found multiple automations when searching for: %s", automationName)
        return
    }
    a := automations[0]
    contacts, err := ac.GetAutomationContacts(&a)
    if err != nil {
        log.Printf("Error retrieving automation contacts. %v\n", err)
        return
    }
    if showCompleted {
        var contactsCompleted []ac.ListContactAutomationsContact
        for _, c := range contacts {
            if c.IsCompleted {
                contactsCompleted = append(contactsCompleted, c)
            }
        }
        log.Printf("Filtered contact list from %d to %d to include only those" +
            " who have completed the automation.", len(contacts), len(contactsCompleted))
        contacts = contactsCompleted
    } else if !showAll {
        var contactsScheduled []ac.ListContactAutomationsContact
        for _, c := range contacts {
            if !c.IsCompleted {
                contactsScheduled = append(contactsScheduled, c)
            }
        }
        log.Printf("Filtered contact list from %d to %d to include only those" +
            " currently in the automation.", len(contacts), len(contactsScheduled))
        contacts = contactsScheduled
    }
    duration := time.Since(start)
    log.Printf("Got %d automation contacts in: %v", len(contacts), duration)

    if !printEmails {
        l := ac.GetAutomationContactList(contacts)
        fmt.Println(&l)
    } else {
        start = time.Now()
        contacts2, err := ac.GetAutomationContactsInfo(contacts)
        if err != nil {
            log.Printf("Error retrieving more info on each contact in automation list: %s", err)
            return
        }
        duration = time.Since(start)
        log.Printf("Got more info on %d automation contacts in: %v", len(contacts), duration)
        printListOfSubscribersEmails(contacts2)
    }
}

func printListOfAutomationContacts(list []ac.ListContactAutomationsContact) {
	var automationContactsListBuffer bytes.Buffer

	for i, a := range list {
		if i > 0 {
			automationContactsListBuffer.WriteString(", ")
		}
        automationContactsListBuffer.WriteString("contact_" + a.Contact)
	}
	fmt.Println(automationContactsListBuffer.String())
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
