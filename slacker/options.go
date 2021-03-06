package slacker

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

type Options struct {
	Token             string
	ListenAddress     string
	VerificationToken string
}

func AddFlags(opt *Options) {
	pflag.StringVar(&opt.ListenAddress, "slack-listen", "0.0.0.0:3000", "Address and port to listen on.")

	opt.Token = os.Getenv("SLACK_BOT_TOKEN")
	opt.VerificationToken = os.Getenv("SLACK_VERIFICATION_TOKEN")
}

func ValidateOptions(opt *Options) error {
	if len(opt.Token) == 0 {
		return fmt.Errorf("the environment variable SLACK_BOT_TOKEN must be set")
	}
	if len(opt.VerificationToken) == 0 {
		return fmt.Errorf("the environment variable SLACK_VERIFICATION_TOKEN must be set")
	}

	return nil
}