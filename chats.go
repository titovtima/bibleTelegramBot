package main

import (
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const defaultTimezone = "Europe/Moscow"

var defaultLocation *time.Location

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

type RandomTimeVerse struct {
	Id        int
	WeekDay   int
	StartTime int
	Duration  int
	NextSends []time.Time
}

const randomVerseTextMessage = "Следующий случайный стих"

type Stats struct {
	Date  string
	Count map[string]int
}

type StatsListChats struct {
	Date  string
	Chats map[string][]int64
}

var statsLocation *time.Location

type PeriodStats struct {
	StartDate string
	EndDate   string
	Count     map[string]int
}

type PeriodStatsListChats struct {
	StartDate string
	EndDate   string
	Chats     map[string][]int64
}

func (ps *PeriodStats) plus(s Stats) {
	ps.StartDate = min(ps.StartDate, s.Date)
	ps.EndDate = max(ps.EndDate, s.Date)
	for name, count := range s.Count {
		ps.Count[name] += count
	}
}

func (ps *PeriodStatsListChats) plus(s StatsListChats) {
	ps.StartDate = min(ps.StartDate, s.Date)
	ps.EndDate = max(ps.EndDate, s.Date)
	for name, chats := range s.Chats {
		for _, chatId := range chats {
			if !slices.Contains(ps.Chats[name], chatId) {
				ps.Chats[name] = append(ps.Chats[name], chatId)
			}
		}
	}
}

func getStatsMessageText(startDate string, endDate string, groupBy string) (string, error) {
	// startDateT, err := time.Parse(time.DateOnly, startDate)
	// if err != nil {
	// 	return "", err
	// }
	// endDateT, err := time.Parse(time.DateOnly, endDate)
	// if err != nil {
	// 	return "", err
	// }
	stats, err := dbGetStatsInRange(startDate, endDate)
	if err != nil {
		return "", err
	}
	statsListChats, err := dbGetStatsListChatsInRange(startDate, endDate)
	if err != nil {
		return "", err
	}
	chats, err := dbGetAllChats()
	if err != nil {
		return "", err
	}
	text := "*Общее количество чатов: " + strconv.Itoa(len(chats)) + "*\n\n"
	if groupBy == "week" || groupBy == "month" || groupBy == "year" {
		psArr := []PeriodStats{{stats[0].Date, stats[0].Date, stats[0].Count}}
		for i := 1; i < len(stats); i++ {
			dayStats := stats[i]
			t, _ := time.Parse(time.DateOnly, dayStats.Date)
			prevDayStats := stats[i-1]
			t2, _ := time.Parse(time.DateOnly, prevDayStats.Date)
			if (groupBy == "week" && (t.Weekday() <= t2.Weekday() || t.Sub(t2) > time.Hour*24*7)) ||
				(groupBy == "month" && (t.Month() > t2.Month() || t.Year() > t2.Year())) ||
				(groupBy == "year" && t.Year() > t2.Year()) {
				psArr = append(psArr, PeriodStats{dayStats.Date, dayStats.Date, dayStats.Count})
			} else {
				psArr[len(psArr)-1].plus(dayStats)
			}
		}
		psListChatsArr := []PeriodStatsListChats{{statsListChats[0].Date, statsListChats[0].Date, statsListChats[0].Chats}}
		for i := 1; i < len(statsListChats); i++ {
			dayStats := statsListChats[i]
			t, _ := time.Parse(time.DateOnly, dayStats.Date)
			prevDayStats := statsListChats[i-1]
			t2, _ := time.Parse(time.DateOnly, prevDayStats.Date)
			if (groupBy == "week" && (t.Weekday() <= t2.Weekday() || t.Sub(t2) > time.Hour*24*7)) ||
				(groupBy == "month" && (t.Month() > t2.Month() || t.Year() > t2.Year())) ||
				(groupBy == "year" && t.Year() > t2.Year()) {
				psListChatsArr = append(psListChatsArr, PeriodStatsListChats{dayStats.Date, dayStats.Date, dayStats.Chats})
			} else {
				psListChatsArr[len(psListChatsArr)-1].plus(dayStats)
			}
		}
		text += "*Отправленные сообщения*\n"
		for _, stat := range psArr {
			text += escapingSymbols(stat.StartDate+" - "+stat.EndDate) + ": " + strconv.Itoa(stat.Count["msg_sent"]) + "\n"
		}
		text += "\n*Активных чатов*\n"
		for _, stat := range psListChatsArr {
			text += escapingSymbols(stat.StartDate+" - "+stat.EndDate) + ": " + strconv.Itoa(len(stat.Chats["chats_sent"])) + "\n"
		}
		text += "\n*Отправлено случайных стихов по расписанию*\n"
		for _, stat := range psArr {
			text += escapingSymbols(stat.StartDate+" - "+stat.EndDate) + ": " + strconv.Itoa(stat.Count["scheduled_sent"]) + "\n"
		}
		text += "\n*Отправлено случайных стихов по запросу*\n"
		for _, stat := range psArr {
			text += escapingSymbols(stat.StartDate+" - "+stat.EndDate) + ": " + strconv.Itoa(stat.Count["cmd_random"]) + "\n"
		}
	} else {
		text += "*Отправленные сообщения*\n"
		for _, stat := range stats {
			text += escapingSymbols(stat.Date) + ": " + strconv.Itoa(stat.Count["msg_sent"]) + "\n"
		}
		text += "\n*Активных чатов*\n"
		for _, stat := range statsListChats {
			text += escapingSymbols(stat.Date) + ": " + strconv.Itoa(len(stat.Chats["chats_sent"])) + "\n"
		}
		text += "\n*Отправлено случайных стихов по расписанию*\n"
		for _, stat := range stats {
			text += escapingSymbols(stat.Date) + ": " + strconv.Itoa(stat.Count["scheduled_sent"]) + "\n"
		}
		text += "\n*Отправлено случайных стихов по запросу*\n"
		for _, stat := range stats {
			text += escapingSymbols(stat.Date) + ": " + strconv.Itoa(stat.Count["cmd_random"]) + "\n"
		}
	}
	return text, nil
}

func sendErrorMessage(chatId int64) {
	go sendMessage(SendMessage{
		ChatId: chatId,
		Text:   "Внутренняя ошибка бота. Уже работаем над исправлением.",
	})
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
