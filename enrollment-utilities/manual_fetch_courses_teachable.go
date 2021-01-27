package main

import (
    "os"
	//"bytes"
	"log"
	"fmt"
    "time"
	flag "github.com/spf13/pflag"
	"bitbucket.org/dagoodma/nancyhillis-go/teachable"
    "regexp"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] [ID] \n", os.Args[0])
     fmt.Printf("Fetches a list of all courses in Teachable, or a course with the given ID\n\n")
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

    id := ""
	if len(args) > 0 {
        id = string(args[0])
    }
    var regexId = regexp.MustCompile(`^\d+$`)

    teachable.SecretsFilePath = "teachable_secrets.yml"
    start := time.Now()

    if regexId.FindString(id) != "" {
        course, err := teachable.GetCourse(id)
        if err != nil {
            log.Printf("Error retrieving course with ID %s: %s\n", id, err)
            return
        }
        log.Println("Got course:\n", course)
    } else {
        allCourses, err := teachable.GetAllCourses()
        if err != nil {
            log.Printf("Error retrieving course with ID %s: %s\n", id, err)
            return
        }
        log.Println("Got all courses:")
        for _, v := range allCourses {
            fmt.Println(&v)
        }
    }

    duration := time.Since(start)
    log.Printf("Finished in: %v", duration)
}

