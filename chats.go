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

func getCurrentStatsDay() string {
	now := time.Now().In(statsLocation)
	return strconv.Itoa(now.Local().Year()) + "-" + strconv.Itoa(int(now.Local().Month())) + "-" + strconv.Itoa(now.Local().Day())
}

func getStartMessage(chatId int64) SendMessage {
	return SendMessage{
		ChatId: chatId,
		Text: escapingSymbols("Добро пожаловать! Я - бот для отправки случайных стихов из Библии. Например:\n\n"+getRandomVerseFromList(2)+
			"\n\nЧтобы получить случайный стих, используйте команду /random.\n\n"+
			"Можете настроить расписания получения случайных стихов с помощью команд /getregular, /addregular, /removeregular, /clearregular.\n\n") +
			"По умолчанию установлен часовой пояс `Europe/Moscow` \\(UTC\\+3\\)\\. " +
			"Можете отправить геопозицию для определения вашего часового пояса, ввести название вручную, выбрать разницу с UTC, " +
			"или использовать /cancel для сохранения `Europe/Moscow`\\.\n\n" +
			"Вы можете использовать команды /gettimezone и /settimezone для просмотра и смены часового пояса\\.",
		ReplyMarkup: chooseTimezoneKeyboard,
		ParseMode:   "MarkdownV2",
	}
}
