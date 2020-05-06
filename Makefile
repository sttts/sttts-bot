build:
	go build -o sttts-bot .
.PHONY: build

update-deps:
	GO111MODULE=on go mod vendor
.PHONY: update-deps

deploy:
	oc apply -f manifests
.PHONY: deploy
