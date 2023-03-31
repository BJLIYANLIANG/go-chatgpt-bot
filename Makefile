export GOPRIVATE=""
export GOPROXY=
export GOSUMDB=

OK_COLOR=\033[32;01m
NO_COLOR=\033[0m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
M=\033[34;01m[MAKE]\033[0m

.PHONY: all deps bin cross clean install

all:
	@$(MAKE) bin

deps:
	go mod tidy
	go mod download

bin:
	@printf "$(OK_COLOR)==> $(NO_COLOR)编译服务\n"
	go build -o bin/chatgpt-bot main.go

cross:
	GOOS=linux GOARCH=amd64 \
	go build -o bin/chatgpt-bot_linux main.go

clean:
	rm -rf bin

install: bin
	@printf "$(OK_COLOR)==> $(NO_COLOR)安装服务到chatgpt-bot目录\n"
	@printf "$(OK_COLOR)==> $(NO_COLOR)创建目录\n"
	mkdir -p chatgpt-bot/bin chatgpt-bot/scripts chatgpt-bot/etc chatgpt-bot/log /data/coredump
	@printf "$(OK_COLOR)==> $(NO_COLOR)拷贝可执行文件\n"
	cp bin/chatgpt-bot chatgpt-bot/bin
	@printf "$(OK_COLOR)==> $(NO_COLOR)拷贝脚本文件\n"
	cp scripts/* chatgpt-bot/scripts
	@printf "$(OK_COLOR)==> $(NO_COLOR)拷贝配置文件\n"
	cp etc/chatgpt.json.example chatgpt-bot/etc/chatgpt.json

	@printf "$(OK_COLOR)==> $(NO_COLOR)安装完成\n"
	@printf "$(OK_COLOR)==> $(WARN_COLOR)请修改配置文件: chatgpt-bot/etc/chatgpt.json\n"
	@printf "$(OK_COLOR)==> $(ERROR_COLOR)请执行命令启动服务: bash chatgpt-bot/scripts/start.sh\n"
	@printf "$(OK_COLOR)==> $(ERROR_COLOR)请执行命令停止服务: bash chatgpt-bot/scripts/shutdown.sh\n"