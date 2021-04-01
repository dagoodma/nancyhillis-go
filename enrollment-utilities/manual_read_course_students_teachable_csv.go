package main

import (
	//"bytes"
	"log"
	"fmt"
    "time"
    "os"
    //"github.com/xiam/to"
    //"strings"
	flag "github.com/spf13/pflag"
	teachable "bitbucket.org/dagoodma/nancyhillis-go/teachable"
    //"regexp"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] <COURSE_CSV_FILE>\n", os.Args[0])
     fmt.Printf("Lists all students enrolled in a given course in Teachable via a CSV file.\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "To Be Implemented")

    flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
        log.Fatal("No course record CSV file given")
        return
    }
    csvFile := string(args[0])

    //
    start := time.Now()
    // Now get course enrollment
    start = time.Now()
    students, err := teachable.GetCourseStudentsCsv(csvFile)
    if err != nil {
        log.Printf("Failed reading students from CSV file '%s': %s",
            csvFile, err)
        return
    }

    // Check for duplicates in list
    var duplicateStudents []teachable.ListUsersUser
    studentIdByEmail := make(map[string]string)
    for _, s := range(students) {
        if _, ok := studentIdByEmail[s.Email]; !ok {
            studentIdByEmail[s.Email] = string(s.Id)
        } else {
            duplicateStudents = append(duplicateStudents, s)
        }
    }
    if len(duplicateStudents) > 0 {
        fmt.Printf("WARNING: Got %d duplicate students in Teachable from list of %d students:",
            len(duplicateStudents), len(students))
        for _, v := range(duplicateStudents) {
            fmt.Printf("%s (ID=%d), ", v.Email, v.Id)
        }
    }
    duration := time.Since(start)
    log.Printf("Got %d Teachable students in course with CSV file '%s' in: %v",
        len(students), csvFile, duration)

    duration = time.Since(start)
    log.Printf("Got %d users in: %v", len(students), duration)

    fmt.Println(teachable.UserSlice(students))
}

