package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/klog"
)

const Version = "0.0.1"

type options struct {
	GithubEndpoint string
	Slack          SlackOptions
	Bugzilla       BugzillaOptions
}

func Validate(opt *options) error {
	return ValidateSlack(&opt.Slack)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	klog.SetOutput(os.Stderr)

	opt := &options{
		GithubEndpoint: "https://api.github.com",
	}

	pflag.StringVar(&opt.GithubEndpoint, "github-endpoint", opt.GithubEndpoint, "An optional proxy for connecting to github.")
	AddSlackFlags(&opt.Slack)
	AddBugzillaFlags(&opt.Bugzilla)
	klog.InitFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlag(flag.Lookup("v"))

	pflag.Parse()

	if err := Validate(opt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(2)
	}

	bz, err := NewBugzilla(opt.Bugzilla)
	if err != nil {
		return err
	}
	defer bz.Close()

	bot := NewSlackBot(opt.Slack, bz)
	for {
		if err := bot.Start(); err != nil && !isRetriable(err) {
			return err
		}
		time.Sleep(5 * time.Second)
	}
}
