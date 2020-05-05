package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"k8s.io/klog"

	"github.com/sttts/sttts-bot/slacker"
)

const Version = "0.0.1"

type options struct {
	GithubEndpoint string
	Slack          slacker.Options
	Bugzilla       BugzillaOptions
}

func Validate(opt *options) error {
	return slacker.ValidateOptions(&opt.Slack)
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
	slacker.AddFlags(&opt.Slack)
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

	slack := slacker.NewSlacker(opt.Slack)
	slack.Command("version", &slacker.CommandDefinition{
		Description: "Report the version of the bot",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			response.Reply(fmt.Sprintf("Thanks for asking! I'm running `%s` ( https://github.com/sttts/sttts-bot", Version))
		},
	})
	slack.Command("say <message>", &slacker.CommandDefinition{
		Description: "Say something.",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			msg := request.StringParam("message", "")
			response.Reply(msg)
		},
	})
	slack.DefaultCommand(func(req slacker.Request, w slacker.ResponseWriter) {
		w.Reply("Unknown command")
	})

	for {
		if err := slack.Listen(context.Background()); err != nil && !isRetriable(err) {
			return err
		}
		time.Sleep(5 * time.Second)
	}
}

func isRetriable(err error) bool {
	// there are several conditions that result from closing the connection on our side
	switch {
	case err == nil,
		err == io.EOF,
		strings.Contains(err.Error(), "use of closed network connection"):
		return true
	case strings.Contains(err.Error(), "cannot unmarshal object into Go struct field"):
		// this could be a legitimate error, so log it to ensure we can debug
		klog.Infof("warning: Ignoring serialization error and continuing: %v", err)
		return true
	default:
		return false
	}
}
