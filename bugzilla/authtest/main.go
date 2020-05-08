package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/klog"

	"github.com/sttts/sttts-bot/bugzilla"
)

func main() {
	klog.SetOutput(os.Stderr)

	opt := bugzilla.Options{}
	bugzilla.AddBugzillaFlags(&opt)
	klog.InitFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlag(flag.Lookup("v"))

	pflag.Parse()

	bz, err := bugzilla.NewBugzilla(opt)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer bz.Close()

	bugs, err := bz.BugList(&bugzilla.BugListQuery{
		CustomQuery: "cmdtype=dorem&remaction=run&namedcmd=openshift-group-b-blockers&sharer_id=290313",
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Found %d bugs\n", len(bugs))
}
