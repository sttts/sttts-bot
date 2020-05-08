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

	slackgo "github.com/slack-go/slack"
	"github.com/spf13/pflag"
	"k8s.io/klog"

	"github.com/sttts/sttts-bot/bugzilla"
	"github.com/sttts/sttts-bot/slacker"
)

const Version = "0.0.1"

type options struct {
	GithubEndpoint string
	Slack          slacker.Options
	Bugzilla       bugzilla.Options
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
	bugzilla.AddBugzillaFlags(&opt.Bugzilla)
	klog.InitFlags(flag.CommandLine)
	pflag.CommandLine.AddGoFlag(flag.Lookup("v"))

	pflag.Parse()

	if err := Validate(opt); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v", err)
		os.Exit(2)
	}

	bz, err := bugzilla.NewBugzilla(opt.Bugzilla)
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
		Handler: func(req slacker.Request, w slacker.ResponseWriter) {
			msg := req.StringParam("message", "")
			w.Reply(msg)
		},
	})
	slack.Command("bz-stats", &slacker.CommandDefinition{
		Description: "Show group B Bugzilla statistics.",
		Handler: func(req slacker.Request, w slacker.ResponseWriter) {
			urls := map[string]string{
				"blockers": "cmdtype=dorem&remaction=run&namedcmd=openshift-group-b-blockers&sharer_id=290313",
				"customer": "cmdtype=dorem&list_id=11029281&namedcmd=openshift-group-b-customer&remaction=run&sharer_id=290313",
				"priority": "cmdtype=dorem&list_id=11029283&namedcmd=openshift-group-b-prio&remaction=run&sharer_id=290313",
				"triage":   "cmdtype=dorem&remaction=run&namedcmd=openshift-group-b-triage&sharer_id=290313",
				"junk":     "cmdtype=dorem&remaction=run&namedcmd=openshift-group-b-junk&sharer_id=290313",
			}
			stats := map[string]int{}
			for k, url := range urls {
				_, _, _, err := w.Client().SendMessage(req.Event().Channel,
					slackgo.MsgOptionPostEphemeral(req.Event().User),
					slackgo.MsgOptionText(fmt.Sprintf("Querying %q...", url), false))
				if err != nil {
					klog.Error(err)
				}

				bugs, err := bz.BugList(&bugzilla.BugListQuery{CustomQuery: url})
				if err != nil {
					_, _, _, err := w.Client().SendMessage(req.Event().Channel,
						slackgo.MsgOptionPostEphemeral(req.Event().User),
						slackgo.MsgOptionText(fmt.Sprintf("failed to query bug list %q: %v", url, err), false))
					if err != nil {
						klog.Error(err)
					}
					return
				}

				stats[k] = len(bugs)
			}

			//msg := request.StringParam("message", "")
			if err := w.Reply(fmt.Sprintf(`Blockers Bugs Total (https://red.ht/2KJlqiO)
%d
Bugs With Customer Case (https://red.ht/2VNOuvQ)
%d
Priority Bugs (https://red.ht/2Ym0CWG)
%d
Bugs To Triage (https://red.ht/3d0yOLj)
%d
Junk Bugs (https://red.ht/2VQ9TEz)
%d`, stats["blockers"], stats["customer"], stats["priority"], stats["triage"], stats["junk"])); err != nil {
				klog.Error(err)
			}
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
