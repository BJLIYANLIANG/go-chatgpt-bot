package core

import (
	"encoding/json"
	"github.com/eatmoreapple/openwechat"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"github.com/sashabaranov/go-openai"
	"os"
	"strings"
	"sync"
)

// MatchGroupName 判断是否群聊名称是否符合
func (i *ChatGptConf) MatchGroupName(groupName string) bool {
	i.groupNameWhiteList.Do(func() {
		i.groupNameWhiteListMapping = make(map[string]bool)
		for _, name := range i.GroupNameWhiteList {
			if len(name) == 0 {
				continue
			}
			i.groupNameWhiteListMapping[name] = true
		}
	})
	if len(i.groupNameWhiteListMapping) == 0 {
		return true
	}
	return i.groupNameWhiteListMapping[groupName]
}

// MatchGroupChatMentionPrefix 判断是否是群聊@机器人的消息
func (i *ChatGptConf) MatchGroupChatMentionPrefix(context string) bool {
	for _, prefix := range i.GroupChatPrefix {
		if len(prefix) == 0 {
			continue
		}
		if strings.HasPrefix(context, prefix) {
			return true
		}
	}
	return false
}

// GetDefaultPrompt 获取默认的提示
func (i *ChatGptConf) GetDefaultPrompt() openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: i.CharacterDesc,
	}
}

// AddGroupNameWhiteList 添加群聊白名单
func (i *ChatGptConf) AddGroupNameWhiteList(name string) {
	i.GroupNameWhiteList = append(i.GroupNameWhiteList, name)
	i.GroupNameWhiteList = lo.Uniq(i.GroupNameWhiteList)
	i.groupNameWhiteList = sync.Once{}
}

// RemoveGroupNameWhiteList 移除群聊白名单
func (i *ChatGptConf) RemoveGroupNameWhiteList(name string) {
	idx := lo.IndexOf(i.GroupNameWhiteList, name)
	if idx != -1 {
		i.GroupNameWhiteList = append(i.GroupNameWhiteList[:idx], i.GroupNameWhiteList[idx+1:]...)
		i.groupNameWhiteList = sync.Once{}
	}
}

// SetDefaultPrompt 设置默认提示
func (i *ChatGptConf) SetDefaultPrompt(value string) {
	i.CharacterDesc = value
}

type ConfHelper struct {
	conf *ChatGptConf
	file string
}

func NewTestConfHelper() *ConfHelper {
	return &ConfHelper{
		conf: &ChatGptConf{
			Token:                     "",
			GroupChatPrefix:           nil,
			GroupNameWhiteList:        nil,
			ConversationMaxTokens:     100,
			CharacterDesc:             "test",
			ConversationTimeout:       0,
			groupNameWhiteList:        sync.Once{},
			groupNameWhiteListMapping: nil,
		},
		file: "test.json",
	}
}

func NewConfHelper(file string) *ConfHelper {
	return &ConfHelper{file: file}
}

func (i *ConfHelper) GetConf() *ChatGptConf {
	return i.conf
}

func (i *ConfHelper) MatchGroupFilter(msg *openwechat.Message) (bool, string, error) {
	if !msg.IsComeFromGroup() {
		return true, "不是来自群组的信息", nil
	}
	senderFrom, err := msg.Sender()
	if err != nil {
		return false, "失败", errors.Wrap(err, "获取群消息群组失败")
	}
	matchPrefix := i.conf.MatchGroupChatMentionPrefix(msg.Content)
	matchGroupName := i.conf.MatchGroupName(senderFrom.NickName)

	errMsg := ""
	if !matchPrefix {
		errMsg += "不是群聊@机器人的消息;"
	}
	if !matchGroupName {
		errMsg += "群聊名称不符合;"
	}
	return matchPrefix && matchGroupName, errMsg, nil
}

// ConversationMaxTokens 获取对话最大长度
func (i *ConfHelper) ConversationMaxTokens() int {
	if i.conf.ConversationMaxTokens == 0 {
		return 1000
	}
	return i.conf.ConversationMaxTokens
}

// ConversationTimeout 获取对话超时时间
func (i *ConfHelper) ConversationTimeout() int {
	if i.conf.ConversationTimeout == 0 {
		return 3600
	}
	return i.conf.ConversationTimeout
}

// LoadJsonConf 从文件中加载配置
func (i *ConfHelper) LoadJsonConf() (conf *ChatGptConf, err error) {
	conf = &ChatGptConf{}

	// read content from file and unmarshal into json
	data, err := os.ReadFile(i.file)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	i.conf = conf
	return conf, nil
}

// SaveJsonConf 保存配置到文件
func (i *ConfHelper) SaveJsonConf(conf *ChatGptConf) error {
	data, err := json.MarshalIndent(conf, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(i.file, data, 0644)
}

type ChatGptConf struct {
	Token                 string   `json:"token"`
	GroupChatPrefix       []string `json:"group_chat_prefix"`
	GroupNameWhiteList    []string `json:"group_name_white_list"`
	ConversationMaxTokens int      `json:"conversation_max_tokens"`
	CharacterDesc         string   `json:"character_desc"`
	ConversationTimeout   int      `json:"conversation_timeout"`

	groupNameWhiteList        sync.Once
	groupNameWhiteListMapping map[string]bool
}
