package main

import (
	//"bytes"
	"log"
	"fmt"
    "time"
    "os"
    "github.com/xiam/to"
	flag "github.com/spf13/pflag"
	teachable "bitbucket.org/dagoodma/nancyhillis-go/teachable"
    "regexp"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] [EMAIL_OR_ID]\n", os.Args[0])
     fmt.Printf("Search for a Teachable student with the given email or" +
        " ID. Returns all students if no ID or email is given.\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun, exactMatch bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "To Be Implemented")
	flag.BoolVarP(&exactMatch, "exact-match", "e", false, "To Be Implemented") //"Tag name must match exactly, otherwise it can be a substring")

    flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

    userEmailOrId := ""
    var regexEmail = regexp.MustCompile(`^\S+@\S+$`)
    var regexId = regexp.MustCompile(`^\d+$`)

	if len(args) > 0 {
        userEmailOrId = string(args[0])
	}

    teachable.SecretsFilePath = "teachable_secrets.yml"
    start := time.Now()
    var users []teachable.ListUsersUser

    if userEmailOrId == "" {
        var err error
        users, err = teachable.GetAllUsers()
        if err != nil {
            log.Printf("Error retrieving all users: %s\n", err)
            return
        }
    } else if regexEmail.FindString(userEmailOrId) != "" {
        u, err := teachable.GetUserByEmail(userEmailOrId)
        if err != nil {
            log.Printf("Error retrieving user with email '%s': %s\n", userEmailOrId, err)
            return
        }
        log.Printf("Got user:")
        log.Println(u)
        users = append(users, *u)
    } else if regexId.FindString(userEmailOrId) != "" {
        u, err := teachable.GetUserById(to.Uint64(userEmailOrId))
        if err != nil {
            log.Printf("Error retrieving user with ID %s: %s\n", userEmailOrId, err)
            return
        }
        log.Printf("Got user:")
        log.Println(u)
        users = append(users, *u)
    } else {
        log.Fatalf("Expected a valid email address or an ID, but got: %s", userEmailOrId)
        return
    }
    duration := time.Since(start)
    log.Printf("Got %d users in: %v", len(users), duration)

    fmt.Println(teachable.UserSlice(users))
}
