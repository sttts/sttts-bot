package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/handlers"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/spf13/pflag"
	"k8s.io/klog"
)

type SlackOptions struct {
	Token             string
	ListenAddress     string
	VerificationToken string
}

func AddSlackFlags(opt *SlackOptions) {
	pflag.StringVar(&opt.ListenAddress, "slack-listen", "0.0.0.0:3000", "Address and port to listen on.")

	opt.Token = os.Getenv("SLACK_BOT_TOKEN")
	opt.VerificationToken = os.Getenv("SLACK_VERIFICATION_TOKEN")
}

func ValidateSlack(opt *SlackOptions) error {
	if len(opt.Token) == 0 {
		return fmt.Errorf("the environment variable SLACK_BOT_TOKEN must be set")
	}
	if len(opt.VerificationToken) == 0 {
		return fmt.Errorf("the environment variable SLACK_VERIFICATION_TOKEN must be set")
	}

	return nil
}

type SlackBot struct {
	token             string
	listenAddress     string
	verificationToken string
	bz                *Bugzilla
}

func NewSlackBot(opt SlackOptions, bz *Bugzilla) *SlackBot {
	return &SlackBot{
		token:             opt.Token,
		listenAddress:     opt.ListenAddress,
		verificationToken: opt.VerificationToken,
		bz:                bz,
	}
}

func (b *SlackBot) Start() error {
	client := slack.New(b.token, slack.OptionDebug(true))

	mux := http.NewServeMux()
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		body := buf.String()
		eventsAPIEvent, e := slackevents.ParseEvent(json.RawMessage(body), slackevents.OptionVerifyToken(&slackevents.TokenComparator{VerificationToken: b.verificationToken}))
		if e != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		switch eventsAPIEvent.Type {
		case slackevents.URLVerification:
			var r *slackevents.ChallengeResponse
			err := json.Unmarshal([]byte(body), &r)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
			}
			w.Header().Set("Content-Type", "text")
			w.Write([]byte(r.Challenge))

		case slackevents.CallbackEvent:
			innerEvent := eventsAPIEvent.InnerEvent
			klog.Infof("CallbackEvent: %s", innerEvent.Type)
			switch ev := innerEvent.Data.(type) {
			case *slackevents.AppMentionEvent:
				client.PostMessage(ev.Channel, slack.MsgOptionText("Yes, hello.", false))
			case *slackevents.MessageEvent:
				// ignore my own messages
				if len(ev.BotID) > 0 {
					break
				}

				klog.Infof("MessageEvent: %#v", ev)
				switch {
				case strings.HasPrefix(ev.Text, "say "):
					client.PostMessage(ev.Channel, slack.MsgOptionText(ev.Text[4:], false))
				case strings.HasPrefix(ev.Text, "help"):
					client.PostMessage(ev.Channel, slack.MsgOptionText("TODO", false))
				case strings.HasPrefix(ev.Text, "version"):
					client.PostMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("Thanks for asking! I'm running `%s` ( https://github.com/sttts/sttts-bot )", Version), false))
				case ev.Text == "debug" && ev.ChannelType == "im":
					client.PostMessage(ev.Channel, slack.MsgOptionText(fmt.Sprintf("%#v", ev), false))
				default:
					client.PostMessage(ev.Channel, slack.MsgOptionText("unrecognized command, msg me `help` for a list of all commands", false))
				}
			}
		}
	})

	klog.Infof("sttts-bot up and listening to slack on %s", b.listenAddress)
	return http.ListenAndServe(b.listenAddress, handlers.LoggingHandler(os.Stdout, mux))
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
