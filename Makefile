build:
	go build -o sttts-bot .
.PHONY: build

update-deps:
	GO111MODULE=on go mod vendor
.PHONY: update-deps

deploy:
	oc new-app --strategy=docker https://github.com/sttts/sttts-bot.git
.PHONY: deploy
