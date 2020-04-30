package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/shomali11/slacker"
	"k8s.io/klog"
)

type Bot struct {
	token string
	bz    *Bugzilla
}

func NewBot(token string, bz *Bugzilla) *Bot {
	return &Bot{
		token: token,
		bz:    bz,
	}
}

func (b *Bot) Start() error {
	slack := slacker.NewClient(b.token)

	slack.DefaultCommand(func(request slacker.Request, response slacker.ResponseWriter) {
		response.Reply("unrecognized command, msg me `help` for a list of all commands")
	})

	slack.Command("say <message>", &slacker.CommandDefinition{
		Description: "Say something",
		Example:     "say \"Hello world!\"",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			channel := request.Event().Channel
			if !isDirectMessage(channel) {
				response.Reply("this command is only accepted via direct message")
				return
			}

			msg := request.StringParam("message", "")
			response.Reply(msg)
		},
	})

	slack.Command("version", &slacker.CommandDefinition{
		Description: "Report the version of the bot",
		Handler: func(request slacker.Request, response slacker.ResponseWriter) {
			response.Reply(fmt.Sprintf("Thanks for asking! I'm running `%s` ( https://github.com/sttts/sttts-bot )", Version))
		},
	})

	klog.Infof("sttts-bot up and listening to slack")
	return slack.Listen(context.Background())
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

func isDirectMessage(channel string) bool {
	return strings.HasPrefix(channel, "D")
}
