package main

import (
	//"bytes"
	"log"
	"fmt"
    "time"
    "os"
    "github.com/xiam/to"
    "strings"
	flag "github.com/spf13/pflag"
	teachable "bitbucket.org/dagoodma/nancyhillis-go/teachable"
    "regexp"
)


var Debug = false // supress extra messages if false

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] <COURSE_ACRONYM>\n", os.Args[0])
     fmt.Printf("Lists all students enrolled in a given course in Teachable.\n\n")
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

	if len(args) < 1 {
        log.Fatal("No course acronym")
        return
    }
    var courseAcronymRegex = regexp.MustCompile(`^(TAJC|TAJM|EWC|SJC|SJM|ATC|LYS|BUNDLE_TAJC-EWC|TAP_CHALLENGE|TAPCIP)$`)
    courseAcronym := strings.ToUpper(string(args[0]))

    if courseAcronymRegex.FindString(courseAcronym) == "" {
        log.Fatal("Expected valid course acronym but got: ", courseAcronym)
        return
    }

    teachable.SecretsFilePath = "teachable_secrets.yml"

    //
    start := time.Now()
    log.Println("Fetching courses in Teachable...")
    allCourses, err := teachable.GetAllCourses()
    if err != nil {
        log.Printf("Failed fetching all courses in Teachable: %s\n", err)
        return
    }

    var course *teachable.RetrieveCourse
    for _, c := range allCourses {
        c2, err := teachable.GetCourse(to.String(c.Id))
        if err != nil {
            log.Printf("Failed fetching course '%s' (id=%d) in Teachable: %s\n", course.Name, course.Id, err)
            return
        }
        if c2.IsAcronym(courseAcronym) {
            course = c2
            break
        }
    }
    if course == nil {
        log.Printf("Failed to find course with acronym '%s' in Teachable", courseAcronym)
        return
    }
    duration := time.Since(start)
    log.Printf("Got course '%s' (id=%d, friendly_url=%s) in Teachable with acronym '%s' in: %v",
        course.Name, course.Id, course.FriendlyUrl, courseAcronym, duration)

    // Now get course enrollment
    start = time.Now()
    students, err := teachable.GetCourseStudents(course.Id)
    if err != nil {
        log.Printf("Failed fetching students for course '%s' (id=%d): %s",
            course.Name, course.Id, err)
        return
    }
    duration = time.Since(start)
    log.Printf("Got %d Teachable students in course with acronym '%s' in: %v",
        len(students), courseAcronym, duration)

    duration = time.Since(start)
    log.Printf("Got %d users in: %v", len(students), duration)

    fmt.Println(teachable.UserSlice(students))
}

