package main

import (
	"encoding/json"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const chatsDataFileName = "chatsData.json"
const defaultTimezone = "Europe/Moscow"
const statsFileName = "stats.json"
var defaultLocation *time.Location

var chatsData []ChatData
var chatsCronJobsIds = make(map[int64]map[string]uuid.UUID)
var chatsRandomTimeJobsIds = make(map[int64]map[int]map[string]uuid.UUID)

type MessageStatus int

const (
	MessageStatusDefault     MessageStatus = 0
	MessageStatusAddCron1    MessageStatus = 1
	MessageStatusAddCron2    MessageStatus = 2
	MessageStatusAddCron3    MessageStatus = 3
	MessageStatusAddCron4    MessageStatus = 4
	MessageStatusAddCronCron MessageStatus = 5
	MessageStatusAddCron5    MessageStatus = 6
	MessageStatusSetTimezone MessageStatus = 20
	MessageStatusBroadcast   MessageStatus = 10000
)

type RandomTimeVerse struct{
	Id        int
	WeekDay   int
	StartTime Time
	Duration  int
	NextSends []time.Time
}

type ChatData struct {
	ChatId        int64
	MessageStatus MessageStatus
	VersesCrons   []string
	RandomTime    []RandomTimeVerse
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
		chatsRandomTimeJobsIds[chatData.ChatId] = make(map[int]map[string]uuid.UUID)
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
		data := ChatData{chatId, MessageStatusDefault, []string{}, []RandomTimeVerse{}, defaultTimezone}
		chatsData = append(chatsData, data)
		saveChatsDataToFile()
		chatsCronJobsIds[chatId] = make(map[string]uuid.UUID)
		chatsRandomTimeJobsIds[chatId] = make(map[int]map[string]uuid.UUID)
		return &data
	}
	return &chatsData[ind]
}

const randomVerseTextMessage = "Следующий случайный стих"

type Stats struct {
	MessagesSent     int64
	MessagesReceived int64
	ChatsSent        []int64
	ChatsReceived    []int64
	ScheduledSent    int64
	Commands         CommandsStats
}

type CommandsStats struct {
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

type StatsFile map[string]*Stats

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

func getCurrentDayStats() *Stats {
	now := time.Now().In(statsLocation)
	dayString := formatDate(now)
	if statsFile[dayString] == nil {
		var dayStats Stats
		statsFile[dayString] = &dayStats
	}
	return statsFile[dayString]
}

func formatDate(t time.Time) string {
	return strconv.Itoa(t.Local().Year()) + "-" + strconv.Itoa(int(t.Local().Month())) + "-" + strconv.Itoa(t.Local().Day())
}

type DayStats struct {
	Stats *Stats
	Date  string
}

type DayStatsArray []DayStats

func (dsArr DayStatsArray) Len() int { return len(dsArr) }
func (dsArr DayStatsArray) Less(i, j int) bool { return dsArr[i].Date < dsArr[j].Date }
func (dsArr DayStatsArray) Swap(i, j int) { dsArr[i], dsArr[j] = dsArr[j], dsArr[i] }

type PeriodStats struct {
	Stats     *Stats
	StartDate string
	EndDate   string
}

func (s1 *CommandsStats) plus(s2 *CommandsStats) *CommandsStats {
	var result CommandsStats
	result.Random = s1.Random + s2.Random
	result.GetRegular = s1.GetRegular + s2.GetRegular
	result.AddRegular = s1.AddRegular + s2.AddRegular
	result.RemoveRegular = s1.RemoveRegular + s2.RemoveRegular
	result.ClearRegular = s1.ClearRegular + s2.ClearRegular
	result.GetTimezone = s1.GetTimezone + s2.GetTimezone
	result.SetTimezone = s1.SetTimezone + s2.SetTimezone
	result.GetRegularCron = s1.GetRegularCron + s2.GetRegularCron
	result.Start = s1.Start + s2.Start
	return &result
}

func (s1 *Stats) plus(s2 *Stats) *Stats {
	var result Stats
	result.MessagesReceived = s1.MessagesReceived + s2.MessagesReceived
	result.MessagesSent = s1.MessagesSent + s2.MessagesSent
	result.ScheduledSent = s1.ScheduledSent + s2.ScheduledSent
	result.ChatsReceived = append([]int64{}, s1.ChatsReceived...)
	for _, chat := range s2.ChatsReceived {
		if !slices.Contains(result.ChatsReceived, chat) {
			result.ChatsReceived = append(result.ChatsReceived, chat)
		}
	}
	result.ChatsSent = append([]int64{}, s1.ChatsSent...)
	for _, chat := range s2.ChatsSent {
		if !slices.Contains(result.ChatsSent, chat) {
			result.ChatsSent = append(result.ChatsSent, chat)
		}
	}
	result.Commands = *s1.Commands.plus(&s2.Commands)
	return &result
}

func normalizeDateString(s string) string {
	if len(s) == 10 {
		return s
	}
	result := s[:5]
	if s[6] == '-' {
		result += "0" + s[5:7]
		if len(s) == 8 {
			result += "0" + s[7:8]
		} else {
			result += s[7:9]
		}
	} else {
		result += s[5:8]
		if len(s) == 9 {
			result += "0" + s[8:9]
		} else {
			result += s[8:10]
		}
	}
	return result
}

func getStatsMessage(chatId int64, startDate string, endDate string, groupBy string) SendMessage {
	var dailyStats DayStatsArray
	for day, stats := range statsFile {
		if day >= startDate && day <= endDate {
			dailyStats = append(dailyStats, DayStats{stats, normalizeDateString(day)})
		}
	}
	sort.Sort(dailyStats)
	text := "*Общее количество пользователей: " + strconv.Itoa(len(chatsData)) + "*\n\n"
	periodStats := []PeriodStats{}
	if groupBy == "week" || groupBy == "month" {
		periodStats = append(periodStats, PeriodStats{dailyStats[0].Stats, dailyStats[0].Date, dailyStats[0].Date})
		for i := 1; i < len(dailyStats); i++ {
			dayStats := dailyStats[i]
			t, _ := time.Parse(time.DateOnly, dayStats.Date)
			prevDayStats := dailyStats[i-1]
			t2, _ := time.Parse(time.DateOnly, prevDayStats.Date)
			if (groupBy == "week" && (t.Weekday() <= t2.Weekday() || t.Sub(t2) > time.Hour * 24 * 7)) || 
				(groupBy == "month" && (t.Month() > t2.Month() || t.Year() > t2.Year())) {
				periodStats = append(periodStats, PeriodStats{dayStats.Stats, dayStats.Date, dayStats.Date})
			} else {
				periodStats[len(periodStats) - 1].EndDate = dayStats.Date
				periodStats[len(periodStats) - 1].Stats = periodStats[len(periodStats) - 1].Stats.plus(dayStats.Stats)
			}
		}
	}
	if groupBy == "week" || groupBy == "month" {
		text += "*Отправленные сообщения*\n"
		for _, stats := range periodStats {
			text += escapingSymbols(stats.StartDate + " - " + stats.EndDate) + ": " + strconv.FormatInt(stats.Stats.MessagesSent, 10) + "\n"
		}
		text += "\n*Активных чатов*\n"
		for _, stats := range periodStats {
			text += escapingSymbols(stats.StartDate + " - " + stats.EndDate) + ": " + strconv.Itoa(len(stats.Stats.ChatsSent)) + "\n"
		}
		text += "\n*Отправлено случайных стихов по расписанию*\n"
		for _, stats := range periodStats {
			text += escapingSymbols(stats.StartDate + " - " + stats.EndDate) + ": " + strconv.FormatInt(stats.Stats.ScheduledSent, 10) + "\n"
		}
		text += "\n*Отправлено случайных стихов по запросу*\n"
		for _, stats := range periodStats {
			text += escapingSymbols(stats.StartDate + " - " + stats.EndDate) + ": " + strconv.FormatInt(stats.Stats.Commands.Random, 10) + "\n"
		}
	} else {
		text += "*Отправленные сообщения*\n"
		for _, stats := range dailyStats {
			text += escapingSymbols(stats.Date) + ": " + strconv.FormatInt(stats.Stats.MessagesSent, 10) + "\n"
		}
		text += "\n*Активных чатов*\n"
		for _, stats := range dailyStats {
			text += escapingSymbols(stats.Date) + ": " + strconv.Itoa(len(stats.Stats.ChatsSent)) + "\n"
		}
		text += "\n*Отправлено случайных стихов по расписанию*\n"
		for _, stats := range dailyStats {
			text += escapingSymbols(stats.Date) + ": " + strconv.FormatInt(stats.Stats.ScheduledSent, 10) + "\n"
		}
		text += "\n*Отправлено случайных стихов по запросу*\n"
		for _, stats := range dailyStats {
			text += escapingSymbols(stats.Date) + ": " + strconv.FormatInt(stats.Stats.Commands.Random, 10) + "\n"
		}
	}

	return SendMessage{
		ChatId: chatId,
		Text: text,
		ParseMode: "MarkdownV2",
		ReplyMarkup: ReplyKeyboardRemove,
	}
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
