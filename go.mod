module github.com/sttts/sttts-bot

go 1.13

require (
	github.com/fatih/color v1.9.0
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/howeyc/gopass v0.0.0-20190910152052-7cb4b85ec19c
	github.com/mangelajo/track v0.0.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/shomali11/commander v0.0.0-20191122162317-51bc574c29ba
	github.com/shomali11/proper v0.0.0-20190608032528-6e70a05688e7
	github.com/slack-go/slack v0.6.4
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.3
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v0.18.1
	k8s.io/klog v1.0.0
	sigs.k8s.io/yaml v1.2.0
)

replace github.com/mangelajo/track v0.0.0 => github.com/sttts/track v0.0.0-20200430132636-a27fbe883173
