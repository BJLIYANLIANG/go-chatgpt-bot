package core

import (
	"encoding/json"
	"github.com/samber/lo"
	"os"
	"strings"
	"sync"
)

func (i *ChatGptConf) MatchGroupName(groupName string) bool {
	i.groupNameWhiteList.Do(func() {
		i.groupNameWhiteListMapping = make(map[string]bool)
		for _, name := range i.GroupNameWhiteList {
			i.groupNameWhiteListMapping[name] = true
		}
	})
	return i.groupNameWhiteListMapping[groupName]
}

func (i *ChatGptConf) MatchGroupChatPrefix(context string) bool {
	for _, prefix := range i.GroupChatPrefix {
		if strings.HasPrefix(context, prefix) {
			return true
		}
	}
	return false
}

func (i *ChatGptConf) GetDefaultPrompt() openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: i.CharacterDesc,
	}
}

func (i *ChatGptConf) AddGroupNameWhiteList(name string) {
	i.GroupNameWhiteList = append(i.GroupNameWhiteList, name)
	i.GroupNameWhiteList = lo.Uniq(i.GroupNameWhiteList)
	i.groupNameWhiteList = sync.Once{}
}

func (i *ChatGptConf) RemoveGroupNameWhiteList(name string) {
	idx := lo.IndexOf(i.GroupNameWhiteList, name)
	if idx != -1 {
		i.GroupNameWhiteList = append(i.GroupNameWhiteList[:idx], i.GroupNameWhiteList[idx+1:]...)
		i.groupNameWhiteList = sync.Once{}
	}
}

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

func (i *ConfHelper) ConversationMaxTokens() int {
	if i.conf.ConversationMaxTokens == 0 {
		return 1000
	}
	return i.conf.ConversationMaxTokens
}

func (i *ConfHelper) ConversationTimeout() int {
	if i.conf.ConversationTimeout == 0 {
		return 3600
	}
	return i.conf.ConversationTimeout
}

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
