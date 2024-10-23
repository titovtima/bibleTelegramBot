package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httputil"
)

const TelegramApiToken = "my_telegram_token"
const TelegramApiUrl = "https://api.telegram.org/bot" + TelegramApiToken
const UrlForWebhook = "https://test.titovtima.ru/test_bot"
const BotName = "bot_nickname"

type TelegramChatType string

const (
	ChatTypePrivate    TelegramChatType = "private"
	ChatTypeGroup      TelegramChatType = "group"
	ChatTypeSupergroup TelegramChatType = "supergroup"
	ChatTypeChannel    TelegramChatType = "channel"
)

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

type Message struct {
	Id   int64
	From TelegramUser
	Date int64
	Chat TelegramChat
	Text string
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
	UpdateId      int            `json:"update_id"`
	Message       *Message
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type SendMessage struct {
	ChatId      int64       `json:"chat_id"`
	Text        string      `json:"text"`
	ReplyMarkup ReplyMarkup `json:"reply_markup,omitempty"`
	ParseMode   string      `json:"parse_mode,omitempty"`
}

type ReplyMarkup interface { ImplementsReplyMarkup() }

type KeyboardButton struct {
	Text string `json:"text"`
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

type ReplyKeyboardRemove struct {
	RemoveKeyboard bool `json:"remove_keyboard"`
}
func (r ReplyKeyboardRemove) ImplementsReplyMarkup() {}

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
		println(err)
		return
	}
	req, err := http.NewRequest("POST", TelegramApiUrl+"/sendMessage", bytes.NewReader(b))
	if err != nil {
		println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	_, err = client.Do(req)
	if err != nil {
		println(err)
		return
	}
}
