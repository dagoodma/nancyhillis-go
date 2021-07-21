package main

import (
    "os"
	"bytes"
    "strings"
	"log"
	"fmt"
    "time"
    "regexp"
	flag "github.com/spf13/pflag"
    "github.com/xiam/to"
	"bitbucket.org/dagoodma/nancyhillis-go/teachable"
	ac "bitbucket.org/dagoodma/nancyhillis-go/activecampaign"
	"bitbucket.org/dagoodma/dagoodma-go/gsheetwrap"
	//"bitbucket.org/dagoodma/dagoodma-go/util"
	"gopkg.in/Iwark/spreadsheet.v2"
	"google.golang.org/api/sheets/v4"
)


var Debug = false // supress extra messages if false
var GoogleSheetSleepTime, _ = time.ParseDuration("0.5s")

var AddHyperlinksToReport = true

// Automations they must be in (one of) if they're enrolled
var AutomationNames = []string{"SJ_Overdue_Billing", "SJC_Renewal_Invitation",
    "SJC_Enrolled", "SJC_YearlyMember", "SJC_YearlyMember_Legacy",
    "SJC_Renewal_Invitation", "SJC_PaymentFailed", "SJC_1_Month_Trial",
    "SJC_CollectingTestimonials", "SJC_Import_Rainmaker"}

// Actually, 1_Month_Trial is not a valid automation for Enrolled tag,
// but we'll simplify and not check for that here.

var TagEnrolledSuffix = "Enrolled"
var TagRainmakerSuffix = "Enrolled_Rainmaker"

// 1_Month_Trial  for ATC: Students_Rainmaker_Importing
var ExtraTags = []string{"SJC_4ExtraMonths", "SJC_Cancelled", "SJC_FreeAccess",
    "SJC_FoundingMembers", "SJC_NonFoundingMembers", "SJC_PaymentFailed",
    "SJC_PaypalPayment", "SJC_Renewal_2019", "SJC_Renewal_April",
    "SJC_Renewal_G1", "SJC_Revoked", "SJC_TechGlitch", "SJC_YearlyMember",
    "SJ_Founder", "SJ_Founder_FreeYear", "SJ_Founder_NeedMigrate",
    "SJ_Overdue", "SJC_Overdue", "SJ_Renewed", "SJ_Revoked",
    "SJ_Completed", "SJ_Cohort1_March2018", "SJ_Cohort2_April2018",
    "SJ_Cohort3_June2018", "SJ_Cohort4_July2018", "SJ_Cohort5_August2018",
    "SJ_Cohort6_December2018"}

var ReportFolderId = "1Sw8QyhMuGtHPOrCqun6tBDxY8QT5zjAf"

func myUsage() {
     fmt.Printf("Usage: %s [OPTIONS] <COURSE_CSV_FILE> \n", os.Args[0])
     fmt.Printf("Compare course enrollment between Teachable and ActiveCampaign for" +
                " SJ/SJC course using a Teachable student record CSV file. SJC has" +
                " many tags and automations that are different from the other courses," +
                " and the course prefix can be either SJ or SJC.\n" +
                "\n")
     flag.PrintDefaults()
}

func main() {
	var verbose int
	var dryRun, quiet, skipAutomations, skipExtraTags, excludeValid, includeRainmaker bool

	flag.CountVarP(&verbose, "verbose", "v", "Print output with increasing verbosity")
	flag.BoolVarP(&dryRun, "dry-run", "d", false, "Print results without creating report spreadsheet")
	flag.BoolVarP(&quiet, "quiet", "q", false, "Don't print any output if possible")
	flag.BoolVarP(&skipAutomations, "skip-automations", "s", false, "Don't include AC automation info in the list")
	flag.BoolVarP(&skipExtraTags, "skip-extra-tags", "k", false, "Don't include extra AC tags info in the list")
	flag.BoolVarP(&excludeValid, "exclude-valid", "x", false, "Don't include users who are enrolled in Teachable and AC")
	flag.BoolVarP(&includeRainmaker, "include-rainmaker", "r", false, "Include Rainmaker students in the report")
	//flag.BoolVarP(&exactMatch, "exact-match", "e", false, "To Be Implemented")

    flag.Usage = myUsage
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
        log.Fatal("No Teachable course student CSV file given")
        return
    }

    csvFile := string(args[0])

    // TODO remove these overrides
    teachable.SecretsFilePath = "teachable_secrets.yml"
    ac.SecretsFilePath = "ac_secrets.yml"
    gsheetwrap.SecretsFilePath = "gsheet_client_secrets.json"

    teachable.DEBUG = false
    ac.DEBUG = false

    courseAcronym := "SJC" // for Teachable

    /******************************************************************
     * Fetching Data
     ******************************************************************/
    // -------------- Active Campaign
    start := time.Now()
    firstStart := time.Now()
    // By SJ_Enrolled tag
    oldTagName := "SJ" + "_" + TagEnrolledSuffix
    if verbose > 0 {
        log.Println("Fetching contacts in ActiveCampaign with tag: ", oldTagName)
    }
    contactsWithOldTag, err := ac.GetContactsByTag(oldTagName)
    if err != nil {
        log.Printf("Failed fetching contacts with tag '%s' in ActiveCampaign: %s\n", oldTagName, err)
        return
    }
    // By SJC_Enrolled tag
    tagName := "SJC" + "_" + TagEnrolledSuffix
    if verbose > 0 {
        log.Println("Fetching contacts in ActiveCampaign with tag: ", tagName)
    }
    contactsWithTag, err := ac.GetContactsByTag(tagName)
    if err != nil {
        log.Printf("Failed fetching contacts with tag '%s' in ActiveCampaign: %s\n", tagName, err)
        return
    }
    // By SJC_Enrolled_Rainmaker tag
    var contactsWithRainmakerTag []ac.ListContactsContact
    rainmakerTagName := "SJC" + "_" + TagRainmakerSuffix
    rainmakerTagStr := ""
    if includeRainmaker {
        rainmakerTagStr = fmt.Sprintf(" and tag '%s'", rainmakerTagName)
        if verbose > 0 {
            log.Println("Fetching contacts in ActiveCampaign with tag: ", rainmakerTagName)
        }
        contactsWithRainmakerTag, err = ac.GetContactsByTag(rainmakerTagName)
    }

    // Combine lists and look for duplicates
    contactIdByEmail := make(map[string]string)
    var allTagContacts, duplicateContacts []ac.ListContactsContact

    allTagContacts = append(allTagContacts, contactsWithTag...)
    for _, c := range(allTagContacts) {
        if _, ok := contactIdByEmail[c.Email]; !ok {
            contactIdByEmail[c.Email] = c.Id
        } else {
            duplicateContacts = append(duplicateContacts, c)
        }
    }
    for _, c := range(contactsWithOldTag) {
        if _, ok := contactIdByEmail[c.Email]; !ok {
            contactIdByEmail[c.Email] = c.Id
            allTagContacts = append(allTagContacts, c)
        }
    }
    for _, c := range(contactsWithRainmakerTag) {
        if _, ok := contactIdByEmail[c.Email]; !ok {
            contactIdByEmail[c.Email] = c.Id
            allTagContacts = append(allTagContacts, c)
        }
    }
    if len(duplicateContacts) > 0 {
        fmt.Printf("Got duplicate contacts in AC:")
        for _, v := range(duplicateContacts) {
            fmt.Printf("%s, ", v.Email)
        }
        return
    }

    duration := time.Since(start)
    if !quiet {
        log.Printf("Fetched %d contacts from AC with tag '%s' and '%s'%s in: %v",
            len(allTagContacts), tagName, oldTagName, rainmakerTagStr, duration)
    }

    // Do some follow-up queries of AC...

    // Get automations
    contactAutomationsByEmail := make(map[string][]string)
    if !skipAutomations {
        if !quiet {
            log.Printf("Fetching all contacts from %d automations in AC...",
                len(AutomationNames))
        }
        start = time.Now()
        for _, v := range AutomationNames {
            // Get automation from AC
            automationName := v
            if verbose > 1 {
                log.Printf("Fetching automation '%s' from AC...", automationName)
            }
            automations, err := ac.GetAutomationsByName(automationName, true)
            if err != nil {
                log.Printf("Error retrieving automations for '%s': %v\n", automationName, err)
                continue
            }
            if len(automations) > 1 {
                log.Printf("Found multiple automations when searching for: %s", automationName)
                return
            }
            // Get contacts in automation
            a := automations[0]
            if verbose > 1 {
                log.Printf("Fetching contacts for automation '%s' (id=%s) from AC...", a.Name, a.Id)
            }
            automationContacts, err := ac.GetAutomationContacts(&a)
            if err != nil {
                log.Printf("Error retrieving automation contacts for '%s' (id=%s): %v\n",
                    a.Name, a.Id, err)
                return
            }
            if len(automationContacts) == 0 {
                if verbose > 0 {
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
            if verbose > 1 {
                log.Printf("Filtered automation contact list from %d to %d to include" +
                           " only those who have not completed the automation.",
                           len(automationContacts), len(contactsScheduled))
            }
            automationContacts = contactsScheduled

            // Get info on all contacts in automation (for email)
            if verbose > 1 {
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
                    contactIdByEmail[email] = c.Id
                }
                contactAutomationsByEmail[email] = append(
                    contactAutomationsByEmail[email], automationName)
            }
        }

        duration = time.Since(start)
        if !quiet {
            log.Printf("Got %d automations for a total %d contacts in AC in: %v",
                len(AutomationNames), len(contactAutomationsByEmail), duration)
        }
    }

    // Get extra tags
    contactExtraTagsByEmail := make(map[string][]string)
    totalExtrasCount := 0
    if !skipExtraTags {
        if !quiet {
            log.Printf("Fetching all contacts with %d extra tags in AC...",
                len(ExtraTags))
        }
        start = time.Now()

        for _, v := range ExtraTags  {
            // Get tag from AC
            extraTagName := v
            if verbose > 1 {
                log.Printf("Fetching tag '%s' from AC...", extraTagName)
            }
            e, err := ac.GetContactsByTag(extraTagName)
            if err != nil {
                if verbose > 0 {
                    log.Printf("Error retrieving contacts for tag '%s': %v\n", extraTagName, err)
                }
                continue
            }
            if len(e) == 0 {
                if verbose > 0 {
                    log.Printf("Skipping empty tag '%s'...", extraTagName)
                }
                continue
            }
            // Filter the list to those who are enrolled or rainmaker
            extraTagContacts := ac.GetContactList(e)
            extrasCount := 0
            for _, c := range allTagContacts {
                if extraTagContacts.Contains(c.Email) {
                    email := strings.ToLower(c.Email)
                    if _, ok := contactExtraTagsByEmail[email]; !ok {
                        contactExtraTagsByEmail[email] = []string{}
                        contactIdByEmail[email] = c.Id
                    }
                    contactExtraTagsByEmail[email] = append(
                        contactExtraTagsByEmail[email], extraTagName)
                    totalExtrasCount++
                    extrasCount++
                }
            }
            if verbose > 1 {
                log.Printf("Filtered to %d contacts for tag '%s' from tag list with" +
                " %d contacts.", extrasCount, extraTagName, len(extraTagContacts))
            }
        } // for _, v := range ExtraTags 

        duration = time.Since(start)
        if !quiet {
            log.Printf("Found %d contacts with %d extra tags in AC in: %v",
                totalExtrasCount, len(ExtraTags), duration)
        }
    } // if !skipExtraTags

    // -------------- Teachable
    start = time.Now()
    if !quiet {
        log.Printf("Finding '%s' course in Teachable...", courseAcronym)
    }
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
    if !quiet {
        log.Printf("Got course '%s' (id=%d, friendly_url=%s) in Teachable with acronym '%s' in: %v",
            course.Name, course.Id, course.FriendlyUrl, courseAcronym, duration)
    }

    // Now get course enrollment
    if !quiet {
        log.Printf("Fetching enrollments for '%s' course in Teachable...", course.Name)
    }
    start = time.Now()
    teachableStudents, err := teachable.GetCourseStudentsCsv(csvFile)
    if err != nil {
        log.Printf("Failed reading students from CSV file '%s': %s", csvFile, err)
        return
    }
    // De-dupe the list
    var teachableStudentsDeduped []teachable.ListUsersUser
    //var duplicateStudents []teachable.ListUsersUser
    duplicateStudents := 0
    studentIdByEmail := make(map[string]string)
    for _, s := range(teachableStudents) {
        if _, ok := studentIdByEmail[s.Email]; !ok {
            studentIdByEmail[s.Email] = to.String(s.Id)
            teachableStudentsDeduped = append(teachableStudentsDeduped, s)
        } else {
            //duplicateStudents = append(duplicateStudents, s)
            duplicateStudents += 1
        }
    }
    /*
    if len(duplicateStudents) > 0 {
        fmt.Printf("Got duplicate students in Teachable:")
        for _, v := range(duplicateStudents) {
            fmt.Printf("%s, ", v.Email)
        }
        return
    }
    */
    if !quiet && duplicateStudents > 0 {
        log.Printf("Removed %d duplicate students from Teachable list; was %d and" +
                   " is now %d students.", duplicateStudents, len(teachableStudents),
                   len(teachableStudentsDeduped))
    }
    teachableStudents = teachableStudentsDeduped

    duration = time.Since(start)
    if !quiet {
        log.Printf("Got %d Teachable students in course with acronym '%s' in: %v",
            len(teachableStudents), courseAcronym, duration)
    }
    //fmt.Println(teachable.UserSlice(teachableStudents))

    /******************************************************************
     * Analyzing
     ******************************************************************/
    // --------------- Build student list
    // Now go through AC and Teachable and compare
    start = time.Now()
    studentsByEmail := make(CourseStudents)
    // AC tag SJC_Enrolled
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
    // AC tag SJ_Enrolled
    for i := range(contactsWithOldTag) {
        c := contactsWithOldTag[i]
        email := strings.ToLower(c.Email)
        if _, ok := studentsByEmail[email]; !ok {
            studentsByEmail[email] = &CourseStudent{Email: email}
        }
        studentsByEmail[email].AcContact = &c
        studentsByEmail[email].IsInAc = true
        studentsByEmail[email].HasOldTag = true
        studentsByEmail[email].AcProfileUrl = ac.GetContactProfileUrlById(c.Id)
    }
    if includeRainmaker {
        // AC tag SJC_Enrolled_Rainmaker
        for i := range(contactsWithRainmakerTag) {
            c := contactsWithRainmakerTag[i]
            email := strings.ToLower(c.Email)
            if _, ok := studentsByEmail[email]; !ok {
                studentsByEmail[email] = &CourseStudent{Email: email}
            }
            studentsByEmail[email].AcContact = &c
            studentsByEmail[email].IsFromRainmaker = true
            studentsByEmail[email].AcProfileUrl = ac.GetContactProfileUrlById(c.Id)
        }
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
        }
        // Report automation students missing in both Teachable and AC
        if !quiet {
            log.Printf("%d students in %s automations in AC but are not enrolled in Teachable or AC:\n",
                len(automationContactsMissingInAc), courseAcronym)
        }
        if verbose > 0 {
            for k, v := range(automationContactsMissingInAc) {
                //fmt.Println("\t", v.Email)
                fmt.Printf("\t%s -- %+q\n", k, v)
            }
        }
    }

    // AC extra tags
    extraTagContactsMissingInAc := make(map[string][]string)
    if !skipExtraTags {
        for k, v := range(contactExtraTagsByEmail) {
            if _, ok := studentsByEmail[k]; !ok {
                extraTagContactsMissingInAc[k] = v
                continue // exclude them from spreadsheet, but make separate list
            }
            studentsByEmail[k].AcExtraTags = v
            for _, tag := range(v) {
                matched, _ := regexp.MatchString(`(?i)_(founding|founder)`, tag)
                if matched {
                    studentsByEmail[k].IsFoundingMember = true
                }
            }
        }
        // Report extra tag students that were missing
        if !quiet {
            log.Printf("%d contacts had extra tags for SJC/SJ, but were not enrolled in AC:\n",
                len(extraTagContactsMissingInAc))
        }
        if verbose > 0 {
            for k, v := range(extraTagContactsMissingInAc) {
                fmt.Printf("\t%s -- %+q\n", k, v)
            }
        }
    }

    duration = time.Since(start)
    if !quiet {
        log.Printf("Built student list for '%s' from Teachable and AC in: %v",
            courseAcronym, duration)
    }

    // Build lists to report on
    var rainmakerStudentsMissingInAc StudentList
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
        if includeRainmaker && !v.IsInAc {
            rainmakerStudentsMissingInAc = append(rainmakerStudentsMissingInAc,
                studentsByEmail[k])
        }
    }

    /******************************************************************
     * Reporting
     ******************************************************************/
    printStudentLists := verbose > 1
    printStudentListColon := ":"
    if !printStudentLists {
        printStudentListColon = "."
    }
    // Report results 
    if !quiet {
        // Report total students missing in AC (but are in Teachable)
        log.Printf("%d students enrolled in SJC in Teachable that are not in AC%s\n",
            len(studentsMissingInAc), printStudentListColon)
        if verbose > 1 {
            for _, v := range(studentsMissingInAc) {
                fmt.Println("\t", v)
            }
        }
        // Report Enrolled students in AC
        log.Printf("%d students with SJ/SJC_Enrolled tag in AC but are not enrolled in Teachable%s\n",
            len(studentsMissingInTeachable), printStudentListColon)
        if verbose > 1 {
            for _, v := range(studentsMissingInTeachable) {
                fmt.Println("\t", v)
            }
        }
        // Report Rainmaker students 
        if includeRainmaker {
            log.Printf("%d students with SJC_Enrolled_Rainmaker tag in AC but is not enrolled in AC%s\n",
                len(rainmakerStudentsMissingInAc), printStudentListColon)
            if verbose > 1 {
                for _, v := range(rainmakerStudentsMissingInAc) {
                    fmt.Println("\t", v)
                }
            }
        }

        duration = time.Since(start)
        if verbose > 0 {
            log.Printf("Finished comparing students between Teachable and AC in: %v", duration)
        }

        if Debug {
            log.Println("All students:")
            log.Println(&studentsByEmail)
            log.Println("Teachable students:")
            log.Println(&studentIdByEmail)
        }
    } // if !quiet

    // Create spreadsheet report
    start = time.Now()

    dryRunStr := " (dry-run)"
    if !dryRun {
        dryRunStr = ""
    }

    allStr := "_all"
    if excludeValid {
        allStr = ""
    }
    rainmakerStr := ""
    if includeRainmaker {
        rainmakerStr = "_rainmaker"
    }
    t := time.Now()
    reportSpreadsheetName := fmt.Sprintf("SJC_Enrolled%s%s_AC-Teachable_Comparison_" +
        "%d-%02d-%02dT%02d:%02d:%02d", allStr, rainmakerStr, t.Year(),
        t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
	if !quiet {
		log.Printf("Creating%s report spreadsheet \"%s\"...\n",
            dryRunStr, reportSpreadsheetName)
	}

    // Get the service and spreadsheet
    var ss *spreadsheet.Spreadsheet
    validUsers := 0
    row := 0
    if !dryRun {
        // Create spreadsheet
        var err error
        ss, err = gsheetwrap.CreateSpreadsheet(reportSpreadsheetName)
        if err != nil {
            log.Printf("Failed creating spreadsheet '%s': %s",
                reportSpreadsheetName, err)
            return
        }

        if verbose > 1 {
            log.Printf("Created spreadsheet '%s' with ID: %s", reportSpreadsheetName, ss.ID)
            log.Printf("Moving the spreadsheet '%s' to folder: %s",
                reportSpreadsheetName, ReportFolderId)
        }


		err = gsheetwrap.MoveSpreadsheetToFolder(ss.ID, ReportFolderId)
		if err != nil {
            log.Printf("Failed moving to folder: %s", err)
            return
		}
        if verbose > 0 {
            log.Printf("Moved spreadsheet '%s' to folder: %s'", ss.ID, ReportFolderId)
        }

        // Write the data
        if verbose > 1 {
            log.Printf("Writing%s data to spreadsheet: %s", reportSpreadsheetName)
        }

        sheet, err := ss.SheetByIndex(0)
        if err != nil {
            log.Printf("Failed getting first sheet in spreadsheet: %s", err)
            return
        }

        headerRow := []string{"Email (from both lists):", "In Teachable?", "In AC (SJ/SJC tag)?",
            "Old Tag (SJ tag)?", "Is Founder?"}
        if includeRainmaker {
            headerRow = append(headerRow, fmt.Sprintf(
                "From Rainmaker (has %s tag)?", rainmakerTagName))
        }
        if !skipAutomations {
            headerRow = append(headerRow, "AC Automations")
        }
        if !skipExtraTags {
            headerRow = append(headerRow, "AC Extra Tags")
        }
        headerRow = append(headerRow, "Notes")
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
            hasOldTag := "No"
            if v.HasOldTag {
                hasOldTag = "Yes"
            }
            isFounder := "No"
            if v.IsFoundingMember {
                isFounder = "Yes"
            }
            isInTeachable := "Yes"
            if !v.IsInTeachable {
                isInTeachable = "No"
                // Verify everything
                if _, ok := studentIdByEmail[v.Email]; ok {
                    log.Printf("Integrity error: %s is in Teachable but was marked the opposite",
                        v.Email)
                    return
                }
            }
            isFromRainmaker := "No"
            if includeRainmaker && v.IsFromRainmaker {
                isFromRainmaker = "Yes"
            }
            if AddHyperlinksToReport {
                if v.IsInAc {
                    isInAc = fmt.Sprintf("=HYPERLINK(\"%s\", \"%s\")",
                        v.AcProfileUrl, isInAc)
                }
                if v.IsInTeachable {
                    isInTeachable = fmt.Sprintf("=HYPERLINK(\"%s\", \"%s\")",
                        v.TeachableProfileUrl, isInTeachable)
                }
                if includeRainmaker && v.IsFromRainmaker {
                    isFromRainmaker  = fmt.Sprintf("=HYPERLINK(\"%s\", \"%s\")",
                        v.AcProfileUrl, isFromRainmaker)
                }
            }
            newRow := []string{v.Email, isInTeachable, isInAc, hasOldTag, isFounder}
            if includeRainmaker {
                newRow = append(newRow, isFromRainmaker)
            }
            if !skipAutomations {
                acAutomations := strings.Join(v.AcAutomations, ", ")
                newRow = append(newRow, acAutomations)
            }
            if !skipExtraTags {
                acExtraTags := strings.Join(v.AcExtraTags, ", ")
                newRow = append(newRow, acExtraTags)
            }
            for i, v := range newRow {
                sheet.Update(row, i, v)
            }
            row += 1
        } // for _, v := range(studentsByEmail)
        err = sheet.Synchronize()
        if err != nil {
            log.Printf("Failed writing %d rows to file '%s': %v", row,
                reportSpreadsheetName, err)
            return
        }

        // Write the style/conditional formatting data
        formatEndColumnIndex := int64(5)
        if includeRainmaker {
            formatEndColumnIndex = int64(6)
        }
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
                EndColumnIndex: formatEndColumnIndex,
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
                EndColumnIndex: formatEndColumnIndex,
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
    } // if !dryRun

    validStr := "with"
    if excludeValid {
        validStr = "excluding"
    }
    duration = time.Since(start)
    if !quiet {
        log.Printf("Finished creating%s spreadsheet report '%s' with %d rows" +
        " and %s %d valid students in: %v", dryRunStr,
        reportSpreadsheetName, row, validStr, validUsers, duration)
    }

    totalDuration := time.Since(firstStart)
    if !quiet {
        log.Printf("Total exeuction time: %v\n", totalDuration)
    }
}

type CourseStudent struct {
    Email               string
    TeachableUser       *teachable.ListUsersUser
    AcContact           *ac.ListContactsContact
    IsInTeachable       bool
    IsInAc              bool
    IsFromRainmaker     bool
    HasOldTag           bool
    IsFoundingMember    bool
    AcProfileUrl        string
    TeachableProfileUrl string
    AcAutomations       []string
    AcExtraTags         []string
}

type StudentList []*CourseStudent

func (s *CourseStudent) String() string {
    return fmt.Sprintf("%s\tac=%t, teachable=%t, rainmaker=%t, old_tag=%t, automations=%+v, tags=%+v",
        s.Email, s.IsInAc, s.IsInTeachable, s.IsFromRainmaker, s.HasOldTag, s.AcAutomations, s.AcExtraTags)
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

