package main

import (
    "os"
	//"bytes"
	"log"
	"fmt"
    "time"
    "github.com/xiam/to"
	flag "github.com/spf13/pflag"
	"bitbucket.org/dagoodma/nancyhillis-go/teachable"
    "regexp"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] <EMAIL_OR_ID> \n", os.Args[0])
     fmt.Printf("Search for a Teachable student's enrollments using the given email or student ID\n\n")
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

    teachable.SecretsFilePath = "teachable_secrets.yml"
    start := time.Now()

    var err error
    var user = new(teachable.ListUsersUser)
    if regexEmail.FindString(emailOrId) != "" {
        user, err = teachable.GetUserByEmail(emailOrId)
        if err != nil {
            log.Printf("Error retrieving user with email '%s': %s\n", emailOrId, err)
            return
        }
        log.Printf("Got user: %#v", user)
    } else if regexId.FindString(emailOrId) != "" {
        user, err = teachable.GetUserById(to.Uint64(emailOrId))
        if err != nil {
            log.Printf("Error retrieving user with ID %s: %s\n", emailOrId, err)
            return
        }
        //log.Printf("Got user: %#v", user)
    } else {
        log.Fatal("Expected an email address or an ID, but got: %s", emailOrId)
        return
    }
    duration := time.Since(start)
    log.Printf("Found user in: %v", duration)
    fmt.Println(user)

    // Now get the user's enrollments
    enrollments, err := teachable.GetUserEnrollments(user.Id)
    if err != nil {
        log.Printf("Failed fetching enrollments for user '%s': %s\n", emailOrId, err)
        return
    }
    duration = time.Since(start)
    log.Printf("Found user enrollments in: %v", duration)

    log.Printf("Enrollments for user '%s':", emailOrId)
    for i := range(enrollments) {
        fmt.Println(&enrollments[i])
    }
}
