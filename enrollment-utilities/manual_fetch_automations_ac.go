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
     fmt.Printf("Search for ActiveCampaign automations with the given NAME, or list all automations.\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun, exactMatch bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "To Be Implemented")
	flag.BoolVarP(&exactMatch, "exact-match", "e", false, "Automation name must match exactly, otherwise it can be a substring")

    flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

    automationName := ""
	if len(args) < 1 {
		log.Printf("No automation name specified, fetching all automations...")
	} else {
        automationName = string(args[0])
    }

    ac.SecretsFilePath = "ac_secrets.yml"
    start := time.Now()
    allAutomations, err := ac.GetAutomationsByName(automationName, exactMatch)
    if err != nil {
        log.Printf("Error retrieving automations. %v\n", err)
        return
    }
    duration := time.Since(start)
    log.Printf("Got %d automations in: %v", len(allAutomations), duration)

    if len(automationName) == 0 {
        l := ac.GetAutomationList(allAutomations)
        log.Println(&l)
    } else {
        log.Println(&allAutomations[0])
    }
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

