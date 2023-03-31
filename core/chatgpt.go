package core

import (
	"bytes"
	"context"
	"fmt"
	"github.com/eatmoreapple/openwechat"
	"github.com/pkg/errors"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/cobra"

	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

var ChatGPTCommand = &cobra.Command{
	Use:   "start",
	Short: "启动chatgpt",
	Long:  fmt.Sprintf(""),
	Run:   processMessage,
}

var (
	handler    MessageHandler
	confHelper *ConfHelper
	configFile string
	logLevel   string
)

func init() {
	ChatGPTCommand.PersistentFlags().StringVarP(&configFile, "configFile", "c", "chatgpt.json", "-c chatgpt.json")
}

func processMessage(cmd *cobra.Command, args []string) {
	InitLogger(logLevel)

	initConfHelper()

	buildOpenAIService()
	bot := buildWechatBotService()

	serve(bot)
}

func messageHandler(msg *openwechat.Message) {
	var replyErr error

	switch msg.MsgType {
	case openwechat.MsgTypeText:
		match, message, err := confHelper.MatchGroupFilter(msg)
		if err != nil {
			Logger.Debug(fmt.Sprintf("匹配群聊过滤规则失败: %s, err:%s", message, err.Error()))
			return
		}
		if !match {
			Logger.Debug(fmt.Sprintf("匹配群聊过滤规则失败: %s", message))
			return
		}

		_, replyErr = handler.replyText(msg)
	case openwechat.MsgTypeSys:
		_, replyErr = handler.replySys(msg)
	case 51:
		replyErr = nil
	default:
		replyErr = errors.New("暂不支持该类型消息, " + msg.MsgType.String())
	}
	if replyErr != nil {
		Logger.Warn("处理消息失败: " + replyErr.Error())
	}
}

func (h MessageHandler) replyText(msg *openwechat.Message) (*openwechat.SentMessage, error) {
	msgContent := msg.Content
	isGroupMessage := msg.IsComeFromGroup()

	senderName := h.GetSenderName(msg)
	msgContent = h.extractMsgContent(isGroupMessage, msgContent)

	Logger.Info(fmt.Sprintf("Receive: %s, %s", senderName, msgContent))

	if msgContent == "ping" {
		return msg.ReplyText("pong")
	} else if msgContent == "context" {
		messages := h.chatContext.GetString(senderName)
		return msg.ReplyText(messages)
	} else if msgContent == "reload" {
		if _, err := confHelper.LoadJsonConf(); err != nil {
			Logger.Error(err.Error())
		}
		return msg.ReplyText("reload success")
	} else if strings.HasPrefix(msgContent, "admin") {
		_, adminErr := h.handleAdminCommand(msg, msgContent, senderName)
		return nil, errors.WithMessage(adminErr, "admin command error")
	}

	newMessage := h.buildChatGPTRequestMessage(msgContent)
	h.chatContext.SetDefaultMessage(senderName)
	h.chatContext.AppendMessage(senderName, newMessage)

	messages := h.chatContext.GetMessages(senderName)
	completionReq := h.buildCompletionRequest(messages)

	resp, err := h.openapiClient.CreateChatCompletion(context.Background(), completionReq)
	if err != nil {
		return nil, errors.WithMessage(err, "openai api error")
	}

	responseBody := h.extractChatGPTResponseBody(resp)
	responseText := h.formatChatGPTResponse(msg, responseBody, false)

	assistanceMessage := h.buildChatGPTAssistantContextMessage(responseBody)
	h.chatContext.AppendMessage(senderName, &assistanceMessage)

	logInOutMessage(senderName, msgContent, responseBody, h.chatContext.GetTimestampMessages(senderName))

	return msg.ReplyText(responseText)
}

func (h MessageHandler) buildChatGPTRequestMessage(msgContent string) *ChatCompletionMessage {
	return &ChatCompletionMessage{
		ChatCompletionMessage: openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: msgContent,
		},
	}
}

func (h MessageHandler) buildChatGPTAssistantContextMessage(responseBody string) ChatCompletionMessage {
	return ChatCompletionMessage{
		ChatCompletionMessage: openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: responseBody,
		},
	}
}

func (h MessageHandler) formatChatGPTResponse(msg *openwechat.Message, responseBody string, isPyp bool) string {
	content := strings.TrimSpace(responseBody)
	if isPyp {
		content = h.fillPypMessageMentionUser(msg, content)
	} else if msg.IsSendByGroup() {
		content = h.fillGroupMessageMentionUser(msg, content)
	}
	return content
}

func (h MessageHandler) extractChatGPTResponseBody(resp openai.ChatCompletionResponse) string {
	rspContent := resp.Choices[0].Message.Content
	return rspContent
}

func (h MessageHandler) buildCompletionRequest(messages []openai.ChatCompletionMessage) openai.ChatCompletionRequest {
	completionReq := openai.ChatCompletionRequest{
		Model:            openai.GPT3Dot5Turbo,
		Messages:         messages,
		MaxTokens:        confHelper.GetConf().ConversationMaxTokens,
		Temperature:      0.9,
		FrequencyPenalty: 1,
		TopP:             1,
		PresencePenalty:  1,
	}
	return completionReq
}

func (h MessageHandler) extractMsgContent(isGroupMessage bool, msgContent string) string {
	if isGroupMessage {
		for _, prefix := range confHelper.GetConf().GroupChatPrefix {
			if strings.HasPrefix(msgContent, prefix) {
				// remove prefix
				msgContent = strings.TrimPrefix(msgContent, prefix)
				msgContent = strings.TrimSpace(msgContent)
				return msgContent
			}
		}
	}
	return msgContent
}

func logInOutMessage(senderName string, req string, rsp string, contexts ChatCompletionMessages) {
	sb := bytes.Buffer{}
	sb.WriteString("\nsenderName:")
	sb.WriteString(senderName)
	sb.WriteString("\n")
	sb.WriteString("req:")
	sb.WriteString(req)
	sb.WriteString("\n")
	sb.WriteString("rsp:")
	sb.WriteString(rsp)
	sb.WriteString("\n")
	sb.WriteString("contexts:")
	sb.WriteString("\n")
	for _, c := range contexts {
		sb.WriteString("\t-time:")
		sb.WriteString("\t" + time.Unix(int64(c.Timestamp), 0).Format("2006-01-02 15:04:05"))
		sb.WriteString("\t-role:")
		sb.WriteString("\t" + c.Role)
		sb.WriteString("\t" + "-content:")
		sb.WriteString("\t" + c.Content)
		sb.WriteString("\n")
	}
	Logger.Info(sb.String())
}

func buildWechatBotService() *openwechat.Bot {
	bot := openwechat.DefaultBot(openwechat.Desktop)                   // 桌面模式
	bot.SyncCheckCallback = func(resp openwechat.SyncCheckResponse) {} // 忽略回调输出

	// 注册消息处理函数
	bot.MessageHandler = messageHandler // 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	if err := bot.Login(); err != nil {
		Logger.Panic(err.Error())
		return nil
	}
	return bot
}

func buildOpenAIService() {
	client := openai.NewClient(confHelper.GetConf().Token)
	handler.openapiClient = client
	handler.chatContext = buildDefaultChatContext()
}

func buildDefaultChatContext() *ChatContext {
	return &ChatContext{
		items: make(map[string]ChatCompletionMessages),
	}
}

func serve(bot *openwechat.Bot) {
	// 获取登陆的用户
	user, err := bot.GetCurrentUser()
	if err != nil {
		Logger.Error(err.Error())
		return
	}

	Logger.Info("登陆成功, 当前用户: " + user.NickName)

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}

func initConfHelper() {
	confHelper = NewConfHelper(configFile)
	if _, err := confHelper.LoadJsonConf(); err != nil {
		Logger.Panic(err.Error())
	}
}

type MessageHandler struct {
	openapiClient *openai.Client
	chatContext   *ChatContext
}

func (h MessageHandler) saveAndLoadConf(msg *openwechat.Message, response string) (*openwechat.SentMessage, error) {
	confHelper.SaveJsonConf(confHelper.GetConf())
	confHelper.LoadJsonConf()
	return msg.ReplyText(response)
}

func (h MessageHandler) fillGroupMessageMentionUser(msg *openwechat.Message, content string) string {
	user, err := msg.SenderInGroup()
	if err != nil {
		Logger.Error("获取群成员信息失败: " + err.Error())
		return content
	}
	return fmt.Sprintf("@%s %s", user.NickName, content)
}

// GetSenderName 获取发送者名称
func (h MessageHandler) GetSenderName(msg *openwechat.Message) string {
	if msg.IsComeFromGroup() {
		sender, err := msg.SenderInGroup()
		if err != nil {
			Logger.Error("获取群成员信息失败: " + err.Error())
			return msg.FromUserName
		}
		return fmt.Sprintf("Group:%s(%d)", sender.NickName, sender.Uin)
	}
	sender, err := msg.Sender()
	if err != nil {
		Logger.Error("获取用户信息失败: " + err.Error())
		return msg.FromUserName
	}
	return fmt.Sprintf("Person:%s(%d)", sender.NickName, sender.Uin)
}

// replySys 处理系统消息
func (h MessageHandler) replySys(msg *openwechat.Message) (*openwechat.SentMessage, error) {
	const paiyipaiSuffix = "拍了拍我"
	isPyp := strings.HasSuffix(msg.Content, paiyipaiSuffix)
	Logger.Info("收到系统消息: " + msg.Content)
	if isPyp {
		replyText := h.formatChatGPTResponse(msg, "别拍了，我是机器人，我只会回答你的问题，不会回答你的拍砖", isPyp)
		return msg.ReplyText(replyText)
	}
	return nil, nil
}

// fillPypMessageMentionUser 填充拍一拍消息中的@用户
func (h MessageHandler) fillPypMessageMentionUser(msg *openwechat.Message, content string) string {
	tokens := strings.Split(msg.Content, `"`)
	if len(tokens) >= 1 {
		return fmt.Sprintf("@%s %s", tokens[1], content)
	}
	return content
}

type ChatContext struct {
	items map[string]ChatCompletionMessages
	sync.RWMutex
}

// SetDefaultMessage 设置默认消息
func (u *ChatContext) SetDefaultMessage(key string) {
	u.Lock()
	defer u.Unlock()
	_, ok := u.items[key]
	if ok {
		return
	}
	u.items[key] = ChatCompletionMessages{
		{ChatCompletionMessage: confHelper.GetConf().GetDefaultPrompt(), Timestamp: 0xffffffff},
	}
	return
}

func (u *ChatContext) GetString(key string) string {
	u.RLock()
	defer u.RUnlock()
	values, ok := u.items[key]
	if !ok {
		return "null"
	}
	sb := bytes.Buffer{}
	for _, value := range values {
		sb.WriteString(fmt.Sprintf("%s: %s\n", value.Role, value.Content))
	}
	return sb.String()
}

type ChatCompletionMessage struct {
	openai.ChatCompletionMessage
	Timestamp uint64
}

type ChatCompletionMessages []*ChatCompletionMessage

func (c *ChatCompletionMessages) RemoveSecondItem() {
	if len(*c) <= 2 {
		return
	}
	*c = append((*c)[:1], (*c)[2:]...)
}

// FillTimestamp 填充时间戳
func (c *ChatCompletionMessage) FillTimestamp() {
	if c.Timestamp == 0 {
		c.Timestamp = uint64(time.Now().Unix())
	}
}

// GetValidChatCompletionMessages 获取有效时间范围内的聊天消息
func (c *ChatCompletionMessages) GetValidChatCompletionMessages() []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, 0)
	for _, v := range *c {
		if v.IsExpired() {
			continue
		}
		result = append(result, v.ChatCompletionMessage)
	}
	return result
}

// GetValidMessages 获取有效时间范围内的聊天消息
func (c *ChatCompletionMessages) GetValidMessages() ChatCompletionMessages {
	result := make([]*ChatCompletionMessage, 0)
	for _, v := range *c {
		if v.IsExpired() {
			continue
		}
		result = append(result, v)
	}
	return result
}

// IsExpired 判断消息是否过期
func (i *ChatCompletionMessage) IsExpired() bool {
	messageExpireTimestamp := i.Timestamp + uint64(confHelper.ConversationTimeout())
	currentTimestamp := uint64(time.Now().Unix())
	return messageExpireTimestamp < currentTimestamp
}

// AppendMessage 追加消息
func (u *ChatContext) AppendMessage(key string, value *ChatCompletionMessage) {
	u.Lock()
	defer u.Unlock()

	ms := u.items[key]

	value.FillTimestamp()
	validMs := ms.GetValidMessages()
	validMs = append(validMs, value)

	totalToken := 0
	maxToken := confHelper.ConversationMaxTokens()

	for _, c := range validMs {
		totalToken += utf8.RuneCountInString(c.Content)
	}
	if totalToken > maxToken {
		validMs.RemoveSecondItem()
	}
	u.items[key] = validMs
}

// Clear 清除消息
func (u *ChatContext) Clear(key string) {
	u.Lock()
	defer u.Unlock()
	u.items[key] = make(ChatCompletionMessages, 0)
}

// ClearAll 清除所有消息
func (u *ChatContext) ClearAll() {
	u.Lock()
	defer u.Unlock()
	u.items = make(map[string]ChatCompletionMessages, 0)
}

// GetMessages 获取消息
func (u *ChatContext) GetMessages(senderName string) []openai.ChatCompletionMessage {
	u.RLock()
	defer u.RUnlock()
	val := u.items[senderName]
	return val.GetValidChatCompletionMessages()
}

// GetTimestampMessages 获取消息
func (u *ChatContext) GetTimestampMessages(senderName string) ChatCompletionMessages {
	u.RLock()
	defer u.RUnlock()
	return u.items[senderName]
}

// handleAdminCommand 处理管理员命令
func (h MessageHandler) handleAdminCommand(msg *openwechat.Message, msgContent string, senderName string,
) (*openwechat.SentMessage, error) {
	tokens := strings.Split(msgContent, " ")
	if len(tokens) < 4 {
		return msg.ReplyText("admin command format error")
	}
	command := tokens[1]
	subCommand := tokens[2]
	value := tokens[3]

	if command == "group" {
		if subCommand == "add" {
			confHelper.GetConf().AddGroupNameWhiteList(value)
			return h.saveAndLoadConf(msg, "add group chat prefix success")
		}
		if subCommand == "remove" {
			confHelper.GetConf().RemoveGroupNameWhiteList(value)
			return h.saveAndLoadConf(msg, "remove group chat prefix success")
		}
		if subCommand == "list" {
			return h.saveAndLoadConf(msg, strings.Join(confHelper.GetConf().GroupNameWhiteList, "\n"))
		}
	}
	if command == "prompt" {
		if subCommand == "set" {
			confHelper.GetConf().SetDefaultPrompt(value)
			return h.saveAndLoadConf(msg, "set default prompt success")
		}
		if subCommand == "get" {
			return h.saveAndLoadConf(msg, confHelper.GetConf().GetDefaultPrompt().Content)
		}
	}
	if command == "context" {
		if subCommand == "clear" {
			h.chatContext.Clear(senderName)
			return h.saveAndLoadConf(msg, "clear context success")
		}
		if strings.ToLower(subCommand) == "clearall" {
			h.chatContext.ClearAll()
			return h.saveAndLoadConf(msg, "clear all context success")
		}
	}
	return nil, nil
}
