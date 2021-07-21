package main

import (
  "os"
	//"bytes"
	"log"
	"fmt"
  "time"
  "regexp"
	flag "github.com/spf13/pflag"
	teachable "bitbucket.org/dagoodma/nancyhillis-go/teachable"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] [SALE_ID] \n", os.Args[0])
     fmt.Printf("Fetch a sale by ID in Teachable\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	//var dryRun bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	//flag.BoolVarP(&dryRun, "dry-run", "d", false, "To Be Implemented")

  flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
    log.Fatal("No sale ID provided")
    return
  }
  saleId := string(args[0])
  var regexId = regexp.MustCompile(`^\d+$`)

  teachable.SecretsFilePath = "teachable_secrets.yml"
  start := time.Now()

  var sale *teachable.RetrieveSale
  var err error
  if regexId.FindString(saleId) != "" {
    sale, err = teachable.GetSaleById(saleId)
    if err != nil {
      log.Printf("Error retrieving sale with ID %s: %s\n", saleId, err)
      return
    }
  } else {
    log.Fatal("Expected a sale ID, but got: %s", saleId)
    return
  }
  log.Printf("Got sale: %s\n", sale)

  log.Printf("User email from sale: %s\n", sale.User.Email)
  duration := time.Since(start)
  log.Printf("Found sale in: %v", duration)
}
