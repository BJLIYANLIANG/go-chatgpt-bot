export GOPRIVATE=""
export GOPROXY=
export GOSUMDB=

.PHONY: all deps bin cross clean

all:
	@$(MAKE) bin

deps:
	go mod tidy
	go mod download

bin:
	go build -o bin/chatgpt-bot main.go

cross:
	GOOS=linux GOARCH=amd64 \
	go build -o bin/chatgpt-bot_linux main.go

clean:
	rm -rf bin
