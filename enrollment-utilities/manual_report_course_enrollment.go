package main

import (
    "os"
	"bytes"
    "strings"
	"log"
	"fmt"
    "time"
	flag "github.com/spf13/pflag"
    "regexp"
    "github.com/xiam/to"
	"bitbucket.org/dagoodma/nancyhillis-go/teachable"
	ac "bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
	"bitbucket.org/dagoodma/dagoodma-go/gsheetwrap"
	//"bitbucket.org/dagoodma/dagoodma-go/util"
	"gopkg.in/Iwark/spreadsheet.v2"
)


var Debug = true // supress extra messages if false
var GoogleSheetSleepTime, _ = time.ParseDuration("0.5s")

var ReportFolderId = "1Sw8QyhMuGtHPOrCqun6tBDxY8QT5zjAf"

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] <COURSE_ACRONYM> \n", os.Args[0])
     fmt.Printf("Compare course enrollment between Teachable and ActiveCampaign for a given course.\n" +
                "Possible course acronyms to report on are:\n" +
                "TAJC: The Artist's Journey Course\n\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without creating report spreadsheet")
	//flag.BoolVarP(&exactMatch, "exact-match", "e", false, "To Be Implemented")

    flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
        log.Fatal("No course acronym")
        return
    }
    var courseAcronymRegex = regexp.MustCompile(`^(tajc|tajm|ewc|sjc|sjm|atc|lys|bundle_tajc-ewc|tap_challenge|tapcip)$`)
    courseAcronym := strings.ToLower(string(args[0]))

    if courseAcronymRegex.FindString(courseAcronym) == "" {
        log.Fatal("Expected valid course acronym but got: ", courseAcronym)
        return
    }

    teachable.SecretsFilePath = "teachable_secrets.yml"
    ac.SecretsFilePath = "ac_secrets.yml"
    teachable.DEBUG = false
    ac.DEBUG = false

    // -------------- Active Campaign
    start := time.Now()
    firstStart := time.Now()
    tagName := strings.ToUpper(courseAcronym) + "_Enrolled"
    log.Println("Fetching contacts in ActiveCampaign with tag: ", tagName)
    contactsWithTag, err := ac.GetContactsByTag(tagName)
    if err != nil {
        log.Printf("Failed fetching contacts with tag '%s' in ActiveCampaign: %s\n", tagName, err)
        return
    }
    duration := time.Since(start)
    log.Printf("Got %d contacts with tag '%s' in: %v", len(contactsWithTag), tagName, duration)

    // TODO get contacts in automation as well
    // (if _Enrolled, should be in _Enrolled, _Renewal_Invitation, _YearlyMember, or Comp trial invite...)

    // -------------- Teachable
    start = time.Now()
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
    courseAcronym = strings.ToUpper(courseAcronym) // for printing before
    if course == nil {
        log.Printf("Failed to find course with acronym '%s' in Teachable", courseAcronym)
        return
    }
    duration = time.Since(start)
    log.Printf("Got course '%s' (id=%d, friendly_url=%s) in Teachable with acronym '%s' in: %v",
        course.Name, course.Id, course.FriendlyUrl, courseAcronym, duration)

    // Now get course enrollment
    start = time.Now()
    teachableStudents, err := teachable.GetCourseStudents(course.Id)
    if err != nil {
        log.Printf("Failed fetching students for course '%s' (id=%d): %s",
            course.Name, course.Id, err)
        return
    }
    duration = time.Since(start)
    log.Printf("Got %d Teachable students in course with acronym '%s' in: %v",
        len(teachableStudents), courseAcronym, duration)
    //fmt.Println(teachable.UserSlice(teachableStudents))

    // --------------- Reporting
    // Now go through AC and Teachable and compare
    start = time.Now()
    studentsByEmail := make(CourseStudents)
    for i := range(contactsWithTag) {
        c := contactsWithTag[i]
        if _, ok := studentsByEmail[c.Email]; !ok {
            studentsByEmail[c.Email] = &CourseStudent{Email: c.Email}
        }
        studentsByEmail[c.Email].AcContact = &c
        studentsByEmail[c.Email].IsInAc = true
    }

    for i := range(teachableStudents) {
        s := teachableStudents[i]
        if _, ok := studentsByEmail[s.Email]; !ok {
            studentsByEmail[s.Email] = &CourseStudent{Email: s.Email}
        }
        studentsByEmail[s.Email].TeachableUser = &s
        studentsByEmail[s.Email].IsInTeachable = true
    }

    duration = time.Since(start)
    log.Printf("Compared list of students in '%s' between Teachable and AC in: %v",
        courseAcronym, duration)
    //fmt.Println(&studentsByEmail)

    var studentsMissingInAc StudentList
    var studentsMissingInTeachable StudentList
    for k, v := range(studentsByEmail)  {
        if !v.IsInAc {
            studentsMissingInAc = append(studentsMissingInAc,
                studentsByEmail[k])
        }
        if !v.IsInTeachable {
            studentsMissingInTeachable = append(studentsMissingInTeachable,
                studentsByEmail[k])
        }
    }

    log.Printf("%d students enrolled in %s in Teachable that are not in AC:\n",
        len(studentsMissingInAc), courseAcronym)
    for _, v := range(studentsMissingInAc) {
        //fmt.Println("\t", v.Email)
        fmt.Println("\t", v)
    }
    log.Printf("%d students with %s_Enrolled tag in AC but are not enrolled in Teachable:\n",
        len(studentsMissingInAc), courseAcronym)
    for _, v := range(studentsMissingInTeachable) {
        //fmt.Println("\t", v.Email)
        fmt.Println("\t", v)
    }
    duration = time.Since(start)
    log.Printf("Finished comparing students between Teachable and AC in: %v", duration)

    // ...
    // Create spreadsheet report
    start = time.Now()
    gsheetwrap.SecretsFilePath = "gsheet_client_secrets.json"
    dryRunStr := " (dry-run)"
    if !dryRun {
        dryRunStr = ""
    }
    t := time.Now()
    reportSpreadsheetName := fmt.Sprintf("%s_Enrolled_AC-Teachable_Comparison_" +
        "%d-%02d-%02dT%02d:%02d:%02d", courseAcronym, t.Year(), t.Month(),
        t.Day(), t.Hour(), t.Minute(), t.Second())
	if verbose > 0 {
		log.Printf("Creating%s report spreadsheet \"%s\"...\n",
            dryRunStr, reportSpreadsheetName)
	}

    // Get the service and spreadsheet
    var ss *spreadsheet.Spreadsheet
    if !dryRun {
        var err error
        ss, err = gsheetwrap.CreateSpreadsheet(reportSpreadsheetName)
        if err != nil {
            log.Printf("Failed creating spreadsheet '%s': %s",
                reportSpreadsheetName, err)
            return
        }
    }

    // Move the spreadsheet
    if Debug {
        log.Printf("Moving%s the spreadsheet '%s' to folder: %s",
            dryRunStr, reportSpreadsheetName, ReportFolderId)
    }
    if !dryRun {
        log.Printf("Spreadsheet '%s': %s", reportSpreadsheetName, ss.ID)

		err := gsheetwrap.MoveSpreadsheetToFolder(ss.ID, ReportFolderId)
		if err != nil {
            log.Printf("Failed moving to folder: %s", err)
            return
		}
        if Debug {
            log.Printf("Moved spreadsheet '%s' to folder: %s'", ss.ID, ReportFolderId)
        }
    }

    // Write the data
    writeStart := time.Now()
    if Debug {
        log.Printf("Writing%s data to spreadsheet: %s", dryRunStr, reportSpreadsheetName)
    }
    if !dryRun {
        sheet, err := ss.SheetByIndex(0)
        if err != nil {
            log.Printf("Failed getting first sheet in spreadsheet: %s", err)
            return
        }

        row := 0
        headerRow := []string{"Email (from both lists):", "In Teachable?", "In ActiveCampaign?"}
        for i, v := range headerRow {
            sheet.Update(row, i, v)
        }
        row += 1

        for _, v := range(studentsByEmail)  {
            isInAc := "Yes"
            if !v.IsInAc {
                isInAc = "No"
            }
            isInTeachable := "Yes"
            if !v.IsInTeachable {
                isInTeachable = "No"
            }
            newRow := []string{v.Email, isInTeachable, isInAc}
            for i, v := range newRow {
                sheet.Update(row, i, v)
            }
            row += 1
        }
        err = sheet.Synchronize()
        if err != nil {
            log.Printf("Failed writing %d rows to file '%s': %v",
                len(studentsByEmail) + 1, reportSpreadsheetName, err)
            return
        }
    }
    if Debug {
        duration = time.Since(writeStart)
        log.Printf("Wrote%s %d rows to spreadsheet '%s' in: %v", dryRunStr,
            len(studentsByEmail) + 1, reportSpreadsheetName, duration)
    }

    duration = time.Since(start)
    log.Printf("Finished creating%s spreadsheet report in: %v", dryRunStr, duration)

    totalDuration := time.Since(firstStart)
    log.Printf("Total exeuction time: %v\n", totalDuration)
}

type CourseStudent struct {
    Email           string
    TeachableUser   *teachable.ListUsersUser
    AcContact       *ac.ListContactsContact
    IsInTeachable   bool
    IsInAc          bool
    //acAutomation
}

type StudentList []*CourseStudent

func (s *CourseStudent) String() string {
    return fmt.Sprintf("%s\t\tteachable=%t\tac=%t", s.Email, s.IsInTeachable, s.IsInAc)
}

type CourseStudents map[string]*CourseStudent

func (l *CourseStudents) String() string {
	var studentListBuffer bytes.Buffer
    studentListBuffer.WriteString(fmt.Sprintf("List contains %d students:\n", len(*l)))
    for _, v := range(*l) {
        studentListBuffer.WriteString(fmt.Sprintf("\t%s\n", v))
    }
    return studentListBuffer.String()
}

