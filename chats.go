package main

import (
	"encoding/json"
	"io"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const chatsDataFileName = "chatsData.json"
const defaultTimezone = "Europe/Moscow"
const statsFileName = "stats.json"

var chatsData []ChatData
var chatsCronJobsIds = make(map[int64]map[string]uuid.UUID)

type MessageStatus int

const (
	MessageStatusDefault     MessageStatus = 0
	MessageStatusAddCron1    MessageStatus = 1
	MessageStatusAddCron2    MessageStatus = 2
	MessageStatusAddCron3    MessageStatus = 3
	MessageStatusAddCron4    MessageStatus = 4
	MessageStatusAddCronCron MessageStatus = 5
	MessageStatusSetTimezone MessageStatus = 20
	MessageStatusBroadcast   MessageStatus = 10000
	MessageStatusGetStats    MessageStatus = 10001
)

type ChatData struct {
	ChatId        int64
	MessageStatus MessageStatus
	VersesCrons   []string
	Timezone      string
}

type ChatsDataFile struct {
	ChatsData []ChatData
}

func readChatsDataFromFile() {
	fi, err := os.Open(chatsDataFileName)
	if err != nil {
		return
	}
	defer func() {
		if err := fi.Close(); err != nil {
			panic(err)
		}
	}()

	b, err := io.ReadAll(fi)
	if err != nil {
		panic(err)
	}

	var data ChatsDataFile
	err = json.Unmarshal(b, &data)
	if err != nil {
		panic(err)
	}

	chatsData = data.ChatsData

	for _, chatData := range chatsData {
		chatsCronJobsIds[chatData.ChatId] = make(map[string]uuid.UUID)
		addCronsForChat(chatData.VersesCrons, chatData.ChatId, true)
	}
}

func saveChatsDataToFile() error {
	fo, err := os.Create(chatsDataFileName)
	if err != nil {
		return err
	}

	data := ChatsDataFile{chatsData}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fo.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func getChatData(chatId int64) *ChatData {
	ind := slices.IndexFunc(chatsData, func(cd ChatData) bool { return cd.ChatId == chatId })
	if ind == -1 {
		data := ChatData{chatId, MessageStatusDefault, []string{}, defaultTimezone}
		chatsData = append(chatsData, data)
		saveChatsDataToFile()
		chatsCronJobsIds[chatId] = make(map[string]uuid.UUID)
		return &data
	}
	return &chatsData[ind]
}

const randomVerseTextMessage = "Следующий случайный стих"
var nextRandomReplyMarkup = ReplyKeyboardMarkup{[][]KeyboardButton{{{randomVerseTextMessage, false}}}}

type DayStats struct {
	MessagesSent     int64
	MessagesReceived int64
	ChatsSent        []int64
	ChatsReceived    []int64
	ScheduledSent    int64
	Commands         CommandsDayStats
}

type CommandsDayStats struct {
	Random         int64
	GetRegular     int64
	AddRegular     int64
	RemoveRegular  int64
	ClearRegular   int64
	GetTimezone    int64
	SetTimezone    int64
	GetRegularCron int64
	Start          int64
}

type StatsFile map[string]*DayStats

var statsFile StatsFile

func readStatsFile() {
	fi, err := os.Open(statsFileName)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := fi.Close(); err != nil {
			panic(err)
		}
	}()

	b, err := io.ReadAll(fi)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &statsFile)
	if err != nil {
		panic(err)
	}
}

func saveStatsFile() error {
	fo, err := os.Create(statsFileName)
	if err != nil {
		return err
	}

	data := statsFile
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fo.Write(b)
	if err != nil {
		return err
	}

	return nil
}

var statsLocation *time.Location

func getCurrentDayStats() *DayStats {
	now := time.Now().In(statsLocation)
	dayString := formatDate(now)
	if statsFile[dayString] == nil {
		var dayStats DayStats
		statsFile[dayString] = &dayStats
	}
	return statsFile[dayString]
}

func formatDate(t time.Time) string {
	return strconv.Itoa(t.Local().Year()) + "-" + strconv.Itoa(int(t.Local().Month())) + "-" + strconv.Itoa(t.Local().Day())
}

func getStatsMessage(chatId int64, startDate string, endDate string, groupBy string) SendMessage {
	m := statsFile
	filtered := make(map[string]*DayStats)
	for day, stats := range m {
		if day >= startDate && day <= endDate {
			filtered[day] = stats
		}
	}
	m = filtered
	if groupBy == "week" {

	}
	text := "*Общее количество пользователей: " + strconv.Itoa(len(chatsData)) + "*\n\n"
	text += "*Отправленные сообщения*\n"
	for period, stats := range m {
		text += escapingSymbols(period) + ": " + strconv.FormatInt(stats.MessagesSent, 10) + "\n"
	}
	text += "\n*Активных чатов*\n"
	for period, stats := range m {
		text += escapingSymbols(period) + ": " + strconv.Itoa(len(stats.ChatsSent)) + "\n"
	}
	text += "\n*Отправлено случайных стихов по расписанию*\n"
	for period, stats := range m {
		text += escapingSymbols(period) + ": " + strconv.FormatInt(stats.ScheduledSent, 10) + "\n"
	}
	text += "\n*Отправлено случайных стихов по запросу*\n"
	for period, stats := range m {
		text += escapingSymbols(period) + ": " + strconv.FormatInt(stats.Commands.Random, 10) + "\n"
	}

	return SendMessage{
		ChatId: chatId,
		Text: text,
		ParseMode: "MarkdownV2",
		ReplyMarkup: nextRandomReplyMarkup,
	}
}

func getStartMessage(chatId int64) SendMessage {
	return SendMessage{
		ChatId: chatId,
		Text: escapingSymbols(getRandomVerseFromList(2) + "\n———————\n" +
			"Добро пожаловать в бот отправки случайных стихов из Библии.\n\n" +
			"Настройки дневного расписания автоматического получения случайных стихов в Меню (левый нижний угол)." + 
			"Сейчас вы получаете один случайный индивидуальный стих в случайное время дня с 9 утра до 8 вечера.\n\n" +
			"Выберете свой часовой пояс из кнопок ниже и нажмите большую кнопку, которая появится."),
		ReplyMarkup: chooseTimezoneKeyboard,
		ParseMode:   "MarkdownV2",
	}
}
