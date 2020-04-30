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
	ProwConfigPath string
	JobConfigPath  string
	GithubEndpoint string
	ForcePROwner   string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	opt := &options{
		GithubEndpoint: "https://api.github.com",
	}
	pflag.StringVar(&opt.GithubEndpoint, "github-endpoint", opt.GithubEndpoint, "An optional proxy for connecting to github.")
	pflag.CommandLine.AddGoFlag(flag.Lookup("v"))
	pflag.Parse()
	klog.SetOutput(os.Stderr)

	botToken := os.Getenv("BOT_TOKEN")
	if len(botToken) == 0 {
		return fmt.Errorf("the environment variable BOT_TOKEN must be set")
	}

	bot := NewBot(botToken)
	for {
		if err := bot.Start(); err != nil && !isRetriable(err) {
			return err
		}
		time.Sleep(5 * time.Second)
	}
}
