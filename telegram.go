package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"time"
)

var TelegramApiToken string = os.Getenv("TELEGRAM_API_TOKEN")
var TelegramApiUrl string = "https://api.telegram.org/bot" + TelegramApiToken
var UrlForWebhook = os.Getenv("URL_FOR_WEBHOOK")
var BotName = os.Getenv("BOT_USERNAME")

type TelegramChatType string

const (
	ChatTypePrivate    TelegramChatType = "private"
	ChatTypeGroup      TelegramChatType = "group"
	ChatTypeSupergroup TelegramChatType = "supergroup"
	ChatTypeChannel    TelegramChatType = "channel"
)

func chatTypeToInt(chatType TelegramChatType) int {
	switch chatType {
	case ChatTypePrivate:
		return 0
	case ChatTypeGroup:
		return 1
	case ChatTypeSupergroup:
		return 2
	case ChatTypeChannel:
		return 3
	default:
		return -1
	}
}

type TelegramChat struct {
	Id       int64
	ChatType TelegramChatType `json:"type"`
}

type TelegramUser struct {
	Id           int64
	FirstName    string
	LastName     string
	Username     string
	LanguageCode string
}

type Location struct {
	Latitude  float64
	Longitude float64
}

type Message struct {
	Id       int64
	From     TelegramUser
	Date     int64
	Chat     TelegramChat
	Text     string
	Location *Location
	Entities []MessageEntity
}

type MaybeInaccessibleMessage struct {
	Chat      TelegramChat
	MessageId int
	Date      int
}

type CallbackQuery struct {
	Id      string
	From    TelegramUser
	Data    string
	Message *MaybeInaccessibleMessage
}

type Webhook struct {
	Url            string   `json:"url"`
	AllowedUpdates []string `json:"allowed_updates"`
}

type WebhookResponse struct {
	Method string `json:"method"`
	ChatId int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type Update struct {
	UpdateId      int `json:"update_id"`
	Message       *Message
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type MessageEntity struct {
	Type          string       `json:"type"`
	Offset        int          `json:"offset"`
	Length        int          `json:"length"`
	Url           string       `json:"url,omitempty"`
	User          TelegramUser `json:"user,omitempty"`
	Language      string       `json:"language,omitempty"`
	CustomEmojiId string       `json:"custom_emoji_id,omitempty"`
}

type LinkPreviewOptions struct {
	IsDisabled bool `json:"is_disabled"`
}

type SendMessage struct {
	ChatId             int64              `json:"chat_id"`
	Text               string             `json:"text"`
	ReplyMarkup        ReplyMarkup        `json:"reply_markup,omitempty"`
	ParseMode          string             `json:"parse_mode,omitempty"`
	LinkPreviewOptions LinkPreviewOptions `json:"link_preview_options,omitempty"`
	Entities           []MessageEntity    `json:"entities,omitempty"`
}

type ReplyMarkup interface{ ImplementsReplyMarkup() }

type KeyboardButton struct {
	Text            string `json:"text"`
	RequestLocation bool   `json:"request_location,omitempty"`
}

type ReplyKeyboardMarkup struct {
	Keyboard [][]KeyboardButton `json:"keyboard"`
}

func (r ReplyKeyboardMarkup) ImplementsReplyMarkup() {}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

func (i InlineKeyboardMarkup) ImplementsReplyMarkup() {}

type ReplyKeyboardRemoveType struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}

func (r ReplyKeyboardRemoveType) ImplementsReplyMarkup() {}

var ReplyKeyboardRemove = ReplyKeyboardRemoveType{true}

func createWebhook() {
	client := http.Client{}
	webhook := Webhook{UrlForWebhook, []string{}}
	b, err := json.Marshal(webhook)
	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest("POST", TelegramApiUrl+"/setWebhook", bytes.NewReader(b))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	d, _ := httputil.DumpResponse(response, true)
	println(string(d))
	println()
}

func sendMessage(m SendMessage) {
	client := http.Client{}
	b, err := json.Marshal(m)
	if err != nil {
		println(err.Error())
		return
	}
	req, err := http.NewRequest("POST", TelegramApiUrl+"/sendMessage", bytes.NewReader(b))
	if err != nil {
		println(err.Error())
		return
	}
	req.Header.Add("Content-Type", "application/json")
	_, err = client.Do(req)
	if err != nil {
		println(err.Error())
		return
	}
	statsDay := time.Now().In(statsLocation).Format(time.DateOnly)
	dbStatPlusOne(statsDay, "msg_sent")
	dbStatUpdateChatsList(statsDay, "chats_sent", m.ChatId)
}

func escapingSymbols(str string) string {
	symbols := "_*[]()~`>#+-=|{}.!"
	res := ""
	for _, s := range str {
		if strings.Contains(symbols, string(s)) {
			res += "\\"
		}
		res += string(s)
	}
	return res
}
