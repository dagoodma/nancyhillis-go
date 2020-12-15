package main

import (
	//"fmt"
	"flag"
	"log"
	//"os"
	//"time"
	"strconv"
	//"strings"

	"github.com/davecgh/go-spew/spew"
	"gopkg.in/cheggaaa/pb.v1"

	mm "bitbucket.org/dagoodma/nancyhillis-go/membermouse"
	//"bitbucket.org/dagoodma/nancyhillis-go/util"
)

var Debug = false // supress extra messages if false

// Note that we will be using our own customer error handler: HandleError()
func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		log.Fatal("No membermouse IDs given")
		return
	}

	bar := pb.StartNew(len(args))
	var allMembers, activeMembers, adminMembers, compedMembers, canceledMembers, unknownMembers []*mm.Member
	for _, idStr := range args {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			log.Fatalf("Expected integer for member id \"%s\". %q",
				idStr, err)
		}
		m, err := mm.GetMemberById(uint32(id))
		if err != nil {
			log.Println("Failed to fetch member with id \"%s\". %q",
				idStr, err)
			continue
		}
		s, err := m.GetStatus()
		if err != nil {
			log.Println("Failed to determine status of member with id \"%s\". %q",
				idStr, err)
			continue
		}
		/*
			log.Println("Here with: ")
			spew.Dump(s)
		*/
		allMembers = append(allMembers, m)
		isAdmin := m.MembershipLevel == ""

		if isAdmin {
			adminMembers = append(adminMembers, m)
		} else if s.IsActive && !s.IsComped {
			activeMembers = append(activeMembers, m)
		} else if s.IsComped {
			compedMembers = append(compedMembers, m)
		} else if s.IsCanceled || s.IsPendingCancel {
			canceledMembers = append(canceledMembers, m)
		} else {
			unknownMembers = append(unknownMembers, m)
		}

		bar.Increment()
	}

	bar.FinishPrint("Finished!")

	log.Printf("Found %d total, %d active, %d comped, %d canceled, %d admin, and %d unknown members.\n",
		len(allMembers), len(activeMembers), len(compedMembers), len(canceledMembers),
		len(adminMembers), len(unknownMembers))
	log.Println("Active members:")
	spew.Dump(activeMembers)
	//spew.Dump(allMembers)

	return
}
