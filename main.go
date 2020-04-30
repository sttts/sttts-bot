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
	Bugzilla       BugzillaOptions
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
	AddBugzillaFlags(&opt.Bugzilla)
	klog.InitFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlag(flag.Lookup("v"))

	pflag.Parse()

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	if len(botToken) == 0 {
		return fmt.Errorf("the environment variable SLACK_BOT_TOKEN must be set")
	}

	bz, err := NewBugzilla(opt.Bugzilla)
	if err != nil {
		return err
	}
	defer bz.Close()

	bot := NewBot(botToken, bz)
	for {
		if err := bot.Start(); err != nil && !isRetriable(err) {
			return err
		}
		time.Sleep(5 * time.Second)
	}
}
