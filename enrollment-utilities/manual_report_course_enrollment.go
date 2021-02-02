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
	"google.golang.org/api/sheets/v4"
)


var Debug = true // supress extra messages if false
var GoogleSheetSleepTime, _ = time.ParseDuration("0.5s")

var AddHyperlinksToReport = true

var AutomationSuffixes = []string{"Enrolled", "YearlyMember", "Renewal_Invitation",
    "PaymentFailed", "Complimentary_Trial_Invite", "CollectingTestimonials"}

/*
var AutomationsSuffixes = []string{"CollectingTestimonials", "Complimentary_Trial_Invite",
    "ManualAccess", "Renewal_Invitation", "YearlyMember", "PaymentFailed", "Revoke",
    "Invitation", "Cancelled"}
*/

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
	var dryRun, skipAutomations, excludeValid bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without creating report spreadsheet")
	flag.BoolVarP(&skipAutomations, "skip-automations", "s", false, "Don't include AC automation info in the list")
	flag.BoolVarP(&excludeValid, "exclude-valid", "x", false, "Don't include users who are enrolled in Teachable and AC")
	//flag.BoolVarP(&exactMatch, "exact-match", "e", false, "To Be Implemented")

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
    ac.SecretsFilePath = "ac_secrets.yml"
    teachable.DEBUG = false
    ac.DEBUG = false

    // -------------- Active Campaign
    start := time.Now()
    firstStart := time.Now()
    tagName := courseAcronym + "_Enrolled"
    log.Println("Fetching contacts in ActiveCampaign with tag: ", tagName)
    contactsWithTag, err := ac.GetContactsByTag(tagName)
    if err != nil {
        log.Printf("Failed fetching contacts with tag '%s' in ActiveCampaign: %s\n", tagName, err)
        return
    }
    duration := time.Since(start)
    log.Printf("Got %d contacts with tag '%s' in: %v", len(contactsWithTag), tagName, duration)

    // Get automations
    //var automationContacs
    contactAutomationsByEmail := make(map[string][]string)
    contactAutomationsIdByEmail := make(map[string]string)
    if !skipAutomations {
        start = time.Now()
        for _, v := range AutomationSuffixes {
            // Get automation from AC
            automationName := fmt.Sprintf("%s_%s", courseAcronym, v)
            if Debug {
                log.Printf("Fetching automation '%s' from AC...", automationName)
            }
            automations, err := ac.GetAutomationsByName(automationName, true)
            if err != nil {
                log.Printf("Error retrieving automations for '%s': %v\n", automationName, err)
                //return
                continue
            }
            if len(automations) > 1 {
                log.Printf("Found multiple automations when searching for: %s", automationName)
                return
            }
            // Get contacts in automation
            a := automations[0]
            if Debug {
                log.Printf("Fetching contacts for automation '%s' (id=%s) from AC...", a.Name, a.Id)
            }
            automationContacts, err := ac.GetAutomationContacts(&a)
            if err != nil {
                log.Printf("Error retrieving automation contacts for '%s' (id=%s): %v\n",
                    a.Name, a.Id, err)
                return
            }
            if len(automationContacts) == 0 {
                if Debug {
                    log.Printf("Skipping empty automation '%s' (id=%s)...", a.Name, a.Id)
                }
                continue
            }
            // Filter those who are completed
            var contactsScheduled []ac.ListContactAutomationsContact
            for _, c := range automationContacts {
                if !c.IsCompleted {
                    contactsScheduled  = append(contactsScheduled, c)
                }
            }
            if Debug {
                log.Printf("Filtered automation contact list from %d to %d to include" +
                           " only those who have not completed the automation.",
                           len(automationContacts), len(contactsScheduled))
            }
            automationContacts = contactsScheduled

            // Get info on all contacts in automation (for email)
            if Debug {
                log.Printf("Fetching info on %d contacts for automation '%s' (id=%s) from AC...",
                    len(automationContacts), a.Name, a.Id)
            }
            contacts, err := ac.GetAutomationContactsInfo(automationContacts)
            if err != nil {
                log.Printf("Error retrieving more info on each contact in automation '%s'" +
                           " (id=%s) contact list: %s", a.Name, a.Id, err)
                return
            }
            // Add them to the map with automation name
            for _, c := range contacts {
                email := strings.ToLower(c.Email)
                if _, ok := contactAutomationsByEmail[email]; !ok {
                    contactAutomationsByEmail[email] = []string{}
                    contactAutomationsIdByEmail[email] = c.Id
                }
                contactAutomationsByEmail[email] = append(
                    contactAutomationsByEmail[email], automationName)
            }
        }
        //log.Printf("HEre: %#v", contactAutomationsById)

        duration = time.Since(start)
        log.Printf("Got %d automations for a total %d contacts in AC in: %v",
            len(AutomationSuffixes), len(contactAutomationsByEmail), duration)
    }

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
    // AC tag <COURSE>_Enrolled
    for i := range(contactsWithTag) {
        c := contactsWithTag[i]
        email := strings.ToLower(c.Email)
        if _, ok := studentsByEmail[email]; !ok {
            studentsByEmail[email] = &CourseStudent{Email: email}
        }
        studentsByEmail[email].AcContact = &c
        studentsByEmail[email].IsInAc = true
        studentsByEmail[email].AcProfileUrl = ac.GetContactProfileUrlById(c.Id)
    }

    // Teachable students in course
    for i := range(teachableStudents) {
        s := teachableStudents[i]
        if _, ok := studentsByEmail[s.Email]; !ok {
            studentsByEmail[s.Email] = &CourseStudent{Email: s.Email}
        }
        studentsByEmail[s.Email].TeachableUser = &s
        studentsByEmail[s.Email].IsInTeachable = true
        studentsByEmail[s.Email].TeachableProfileUrl =
            teachable.GetUserProfileUrlById(s.Id)
    }

    // AC automations
    automationContactsMissingInAc := make(map[string][]string)
    if !skipAutomations {
        for k, v := range(contactAutomationsByEmail) {
            if _, ok := studentsByEmail[k]; !ok {
                automationContactsMissingInAc[k] = v
                continue // exclude them from spreadsheet, but make separate list
            }
            studentsByEmail[k].AcAutomations = v
            // TODO do we need this?
            id := contactAutomationsIdByEmail[k]
            studentsByEmail[k].AcProfileUrl = ac.GetContactProfileUrlById(id)
        }
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
        len(studentsMissingInTeachable), courseAcronym)
    for _, v := range(studentsMissingInTeachable) {
        //fmt.Println("\t", v.Email)
        fmt.Println("\t", v)
    }
    log.Printf("%d students in %s automations in AC but are not enrolled in Teachable or AC:\n",
        len(automationContactsMissingInAc), courseAcronym)
    for k, v := range(automationContactsMissingInAc) {
        //fmt.Println("\t", v.Email)
        fmt.Printf("\t%s -- %+q\n", k, v)
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
    allStr := "_all"
    if excludeValid {
        allStr = ""
    }
    reportSpreadsheetName := fmt.Sprintf("%s_Enrolled%s_AC-Teachable_Comparison_" +
        "%d-%02d-%02dT%02d:%02d:%02d", courseAcronym, allStr, t.Year(), t.Month(),
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
    validUsers := 0
    row := 0
    if Debug {
        log.Printf("Writing%s data to spreadsheet: %s", dryRunStr, reportSpreadsheetName)
    }
    if !dryRun {
        sheet, err := ss.SheetByIndex(0)
        if err != nil {
            log.Printf("Failed getting first sheet in spreadsheet: %s", err)
            return
        }

        headerRow := []string{"Email (from both lists):", "In Teachable?",
                              "In ActiveCampaign?"}
        if !skipAutomations {
            headerRow = append(headerRow, "AC Automations")
        }
        for i, v := range headerRow {
            sheet.Update(row, i, v)
        }
        row += 1

        for _, v := range(studentsByEmail)  {
            if v.IsInAc && v.IsInTeachable {
                validUsers += 1
                if excludeValid  {
                    continue
                }
            }
            isInAc := "Yes"
            if !v.IsInAc {
                isInAc = "No"
            }
            isInTeachable := "Yes"
            if !v.IsInTeachable {
                isInTeachable = "No"
            }
            if AddHyperlinksToReport {
                isInAc = fmt.Sprintf("=HYPERLINK(\"%s\", \"%s\")",
                    v.AcProfileUrl, isInAc)
                isInTeachable = fmt.Sprintf("=HYPERLINK(\"%s\", \"%s\")",
                    v.TeachableProfileUrl, isInTeachable)
            }
            newRow := []string{v.Email, isInTeachable, isInAc}
            if !skipAutomations {
                acAutomations := strings.Join(v.AcAutomations, ", ")
                newRow = append(newRow, acAutomations)
            }
            for i, v := range newRow {
                sheet.Update(row, i, v)
            }
            row += 1
        }
        err = sheet.Synchronize()
        if err != nil {
            log.Printf("Failed writing %d rows to file '%s': %v", row,
                reportSpreadsheetName, err)
            return
        }

        // Write the style/conditional formatting data
        boolRuleYes := sheets.ConditionalFormatRule{
            BooleanRule: &sheets.BooleanRule{
                Condition: &sheets.BooleanCondition{
                    Type: "TEXT_CONTAINS",
                    Values: []*sheets.ConditionValue{
                        &sheets.ConditionValue{UserEnteredValue: "Yes"},
                    },
                },
                Format: &sheets.CellFormat{
                    BackgroundColor: &sheets.Color{Red: 0.643, Green: 0.910, Blue: 0.804},
                },
            },
            Ranges: []*sheets.GridRange{&sheets.GridRange{
                StartColumnIndex: 1,
                EndColumnIndex: 3,
                StartRowIndex: 1,
                EndRowIndex: to.Int64(row),
            }},
        }
        boolRuleNo := sheets.ConditionalFormatRule{
            BooleanRule: &sheets.BooleanRule{
                Condition: &sheets.BooleanCondition{
                    Type: "TEXT_CONTAINS",
                    Values: []*sheets.ConditionValue{
                        &sheets.ConditionValue{UserEnteredValue: "No"},
                    },
                },
                Format: &sheets.CellFormat{
                    BackgroundColor: &sheets.Color{Red: 0.957, Green: 0.780, Blue: 0.765},
                },
            },
            Ranges: []*sheets.GridRange{&sheets.GridRange{
                StartColumnIndex: 1,
                EndColumnIndex: 3,
                StartRowIndex: 1,
                EndRowIndex: to.Int64(row),
            }},
        }
        err = gsheetwrap.AddConditionalFormatRuleToSpreadsheet(ss.ID, &boolRuleYes, &boolRuleNo)
        if err != nil {
            log.Printf("Failed to write conditional formatting rule to spreadsheet (id=%d): %s",
                ss.ID, err)
            return
        }
    }
    if Debug {
        duration = time.Since(writeStart)
        log.Printf("Wrote%s %d rows to spreadsheet '%s' in: %v", dryRunStr,
            row, reportSpreadsheetName, duration)
    }
    validStr := "with"
    if excludeValid {
        validStr = "excluding"
    }
    duration = time.Since(start)
    log.Printf("Finished creating%s spreadsheet report with %d rows and %s %d valid" +
               " students in: %v", dryRunStr, row, validStr, validUsers, duration)

    totalDuration := time.Since(firstStart)
    log.Printf("Total exeuction time: %v\n", totalDuration)
}

type CourseStudent struct {
    Email               string
    TeachableUser       *teachable.ListUsersUser
    AcContact           *ac.ListContactsContact
    IsInTeachable       bool
    IsInAc              bool
    AcProfileUrl        string
    TeachableProfileUrl string
    AcAutomations       []string
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

