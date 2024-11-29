package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

var validCronMinNumbers = []int{0, 0, 1, 1, 0}
var validCronMaxNumbers = []int{59, 23, 31, 12, 7}

func checkValidCron(cron string) bool {
	parts := strings.Split(cron, " ")
	for ind, part := range parts {
		spl1 := strings.Split(part, "/")
		if len(spl1) > 2 {
			return false
		}
		if len(spl1) == 2 {
			i, err := strconv.Atoi(spl1[1])
			if err != nil || i <= 0 || i > validCronMaxNumbers[ind] {
				return false
			}
		}
		spl2 := strings.Split(spl1[0], ",")
		for _, ran := range spl2 {
			if ran == "*" {
				continue
			}
			spl3 := strings.Split(ran, "-")
			if len(spl3) == 1 {
				i, err := strconv.Atoi(spl3[0])
				if err != nil || i < validCronMinNumbers[ind] || i > validCronMaxNumbers[ind] {
					return false
				}
			} else {
				i1, err1 := strconv.Atoi(spl3[0])
				i2, err2 := strconv.Atoi(spl3[0])
				if err1 != nil || err2 != nil || i1 < validCronMinNumbers[ind] || i1 > i2 || i2 > validCronMaxNumbers[ind] {
					return false
				}
			}
		}
	}
	return true
}

type Time struct {
	Hours   int
	Minutes int
}

func substractTimes (a Time, b Time) int {
	res := a.Minutes - b.Minutes + 60 * (a.Hours - b.Hours)
	if res < 0 { res += 24 * 60 }
	return res
}

func (t *Time) addDuration(duration int) Time {
	return Time{(t.Hours + (t.Minutes + duration) / 60) % 24, (t.Minutes + duration) % 60}
}

func parseTime(timeInput string) (*Time, error) {
	spl := strings.Split(strings.Trim(timeInput, " "), ":")
	if len(spl) != 2 {
		return nil, io.ErrShortWrite
	}
	i1, err1 := strconv.Atoi(spl[0])
	i2, err2 := strconv.Atoi(spl[1])
	if err1 != nil || err2 != nil {
		return nil, io.ErrShortWrite
	}
	if i1 < 0 || i2 < 0 || i1 > 23 || i2 > 59 {
		return nil, io.ErrShortWrite
	}
	return &Time{i1, i2}, nil
}

func timeToCron(t Time) string {
	return fmt.Sprintf("%d %d * * *", t.Minutes, t.Hours)
}

func parseTimeToCron(timeInput string) (string, error) {
	t, err := parseTime(timeInput)
	if err != nil {
		return "", err
	}
	return timeToCron(*t), nil
}

func parseListTimes(timeInput string) ([]Time, error) {
	spl := strings.Split(strings.Trim(timeInput, " "), ",")
	list := []Time{}
	for _, part := range spl {
		t, err := parseTime(part)
		if err != nil {
			return nil, err
		}
		list = append(list, *t)
	}
	return list, nil
}

func parseListTimesToCron(timeInput string) ([]string, error) {
	l, err := parseListTimes(timeInput)
	if err != nil {
		return nil, err
	}
	list := []string{}
	for _, t := range l {
		list = append(list, timeToCron(t))
	}
	return list, nil
}

type WeekDayTime struct {
	WeekDay int
	Hours   int
	Minutes int
}

func parseWeekDayTime(timeInput string) (*WeekDayTime, error) {
	spl := strings.Split(strings.Trim(timeInput, " "), " ")
	if len(spl) != 2 {
		return nil, io.ErrShortWrite
	}
	i1, err1 := strconv.Atoi(spl[0])
	t, err2 := parseTime(spl[1])
	if err1 != nil || err2 != nil {
		return nil, io.ErrShortWrite
	}
	if i1 < 0 || i1 > 7 {
		return nil, io.ErrShortWrite
	}
	return &WeekDayTime{i1, t.Hours, t.Minutes}, nil
}

func weekDayTimeToCron(t WeekDayTime) string {
	return fmt.Sprintf("%d %d * * %d", t.Minutes, t.Hours, t.WeekDay)
}

func parseWeekDayTimeToCron(timeInput string) (string, error) {
	t, err := parseWeekDayTime(timeInput)
	if err != nil {
		return "", err
	}
	return weekDayTimeToCron(*t), nil
}

func parseListWeekDayTimes(timeInput string) ([]WeekDayTime, error) {
	spl := strings.Split(strings.Trim(timeInput, " "), ",")
	list := []WeekDayTime{}
	for _, part := range spl {
		t, err := parseWeekDayTime(part)
		if err != nil {
			return nil, err
		}
		list = append(list, *t)
	}
	return list, nil
}

func parseListWeekDayTimesToCron(timeInput string) ([]string, error) {
	l, err := parseListWeekDayTimes(timeInput)
	if err != nil {
		return nil, err
	}
	list := []string{}
	for _, t := range l {
		list = append(list, weekDayTimeToCron(t))
	}
	return list, nil
}

var weekDaysNames = []string{
	"Каждое воскресенье",
	"Каждый понедельник",
	"Каждый вторник",
	"Каждую среду",
	"Каждый четверг",
	"Каждую пятницу",
	"Каждую субботу",
	"Каждое воскресенье"}

func cronToString(cron string) string {
	parts := strings.Split(cron, " ")
	if strings.Contains(parts[0], ",") || strings.Contains(parts[0], "-") || strings.Contains(parts[0], "/") || strings.Contains(parts[0], "*") {
		return cron
	}
	if strings.Contains(parts[1], ",") || strings.Contains(parts[1], "-") || strings.Contains(parts[1], "/") || strings.Contains(parts[1], "*") {
		return cron
	}
	if len(parts[0]) == 1 {
		parts[0] = "0" + parts[0]
	}
	if parts[1] == "*" && parts[2] == "*" && parts[3] == "*" && parts[4] == "*" {
		return fmt.Sprintf("Каждый час в %s минут", parts[0])
	}
	if len(parts[1]) == 1 {
		parts[1] = "0" + parts[1]
	}
	if parts[2] == "*" && parts[3] == "*" && parts[4] == "*" {
		return fmt.Sprintf("Каждый день в %s:%s", parts[1], parts[0])
	}
	weekDay, err := strconv.Atoi(parts[4])
	if err != nil { return cron }
	if parts[2] == "*" && parts[3] == "*" {
		return fmt.Sprintf("%s в %s:%s", weekDaysNames[weekDay], parts[1], parts[0])
	}
	return cron
}

func randomTimeToString(randomTime RandomTimeVerse) string {
	endTime := randomTime.StartTime.addDuration(randomTime.Duration)
	return "Каждый день в случайное время с " +
		strconv.Itoa(randomTime.StartTime.Hours) + ":" + strconv.Itoa(randomTime.StartTime.Minutes) + " до " +
		strconv.Itoa(endTime.Hours) + ":" + strconv.Itoa(endTime.Minutes)
}

func randomTimeToShortString(randomTime RandomTimeVerse) string {
	endTime := randomTime.StartTime.addDuration(randomTime.Duration)
	return "Случайно с " +
		strconv.Itoa(randomTime.StartTime.Hours) + ":" + strconv.Itoa(randomTime.StartTime.Minutes) + " до " +
		strconv.Itoa(endTime.Hours) + ":" + strconv.Itoa(endTime.Minutes)
}

func addCronsForChat(crons []string, chatId int64, onlyJob bool) {
	chatData := getChatData(chatId)
	if !onlyJob {
		chatData.VersesCrons = append(chatData.VersesCrons, crons...)
		saveChatsDataToFile()
	}
	for _, cron := range crons {
		job, err := scheduler.NewJob(gocron.CronJob(fmt.Sprintf("TZ=%s %s", chatData.Timezone, cron), false),
			gocron.NewTask(randomVerseTask, chatId))
		if err != nil {
			println(err.Error())
			continue
		}
		chatsCronJobsIds[chatId][cron] = job.ID()
	}
}

func randomVerseTask(chatId int64) {
	message := SendMessage{
		ChatId: chatId,
		Text:   bible.getRandomVerse(),
	}
	dayStats := getCurrentDayStats()
	dayStats.ScheduledSent++
	go sendMessage(message)
}

func addRandomTimeForDay(day time.Time, randomTime RandomTimeVerse, chatData *ChatData) {
	duration := rand.Intn(randomTime.Duration) + 1
	loc, err := time.LoadLocation(chatData.Timezone)
	if err != nil {
		loc = defaultLocation
	}
	dayStartTime := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc).
		Add(time.Duration(randomTime.StartTime.Hours) * time.Hour).Add(time.Duration(randomTime.StartTime.Minutes) * time.Minute)
	if len(randomTime.NextSends) > 0 && randomTime.NextSends[len(randomTime.NextSends)-1].After(dayStartTime) {
		return;
	}
	newTime := dayStartTime.Add(time.Duration(duration) * time.Minute)
	if newTime.Before(time.Now()) {
		return;
	}
	randomTime.NextSends = append(randomTime.NextSends, newTime)
	job, err := scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(newTime)),
		gocron.NewTask(func () {
			randomVerseTask(chatData.ChatId)
			randomTime.NextSends = filter(randomTime.NextSends, func(t time.Time) bool { return t.Sub(newTime) == 0; })
			delete(chatsRandomTimeJobsIds[chatData.ChatId][randomTime.Id], dayStartTime.String()[:10])
		}))
	if err != nil {
		println(err.Error())
	} else {
		chatsRandomTimeJobsIds[chatData.ChatId][randomTime.Id][dayStartTime.String()[:10]] = job.ID()
	}
}

func setDailyRandomTimeTasks() {
	now := time.Now()
	for _, chatData := range chatsData {
		for _, randomTime := range chatData.RandomTime {
			addRandomTimeForDay(now, randomTime, &chatData)
		}
	}
	saveChatsDataToFile()
}

func createRandomTimeJobsAfterRestart() {
	for _, chatData := range chatsData {
		for _, randomTime := range chatData.RandomTime {
			for _, send := range randomTime.NextSends {
				job, err := scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(send)),
					gocron.NewTask(func () {
						randomVerseTask(chatData.ChatId)
						randomTime.NextSends = filter(randomTime.NextSends, func(t time.Time) bool { return t.Sub(send) == 0; })
						delete(chatsRandomTimeJobsIds[chatData.ChatId][randomTime.Id], send.String()[:10])
					}))
				if err != nil {
					println("error creating random time job", err.Error())
				}
				chatsRandomTimeJobsIds[chatData.ChatId][randomTime.Id][send.String()[:10]] = job.ID()
			}
		}
	}
}

func addRandomTimeRegular(chatId int64, startTime Time, endTime Time) {
	chatData := getChatData(chatId)
	maxId := 0
	for _, rt := range chatData.RandomTime {
		maxId = max(maxId, rt.Id)
	}
	randomTime := RandomTimeVerse{maxId + 1, -1, startTime, substractTimes(endTime, startTime), []time.Time{}}
	chatData.RandomTime = append(chatData.RandomTime, randomTime)
	now := time.Now()
	loc, err := time.LoadLocation(chatData.Timezone)
	if err != nil { loc = defaultLocation }
	now = now.In(loc)
	chatsRandomTimeJobsIds[chatData.ChatId][randomTime.Id] = make(map[string]uuid.UUID)
	addRandomTimeForDay(now, randomTime, chatData)
	addRandomTimeForDay(time.Date(now.Year(), now.Month(), now.Day() + 1, 0, 0, 0, 0, loc), randomTime, chatData)
	saveChatsDataToFile()
}

func removeRandomTimeRegular(chatId int64, randomId int) {
	chatData := getChatData(chatId)
	filtered := filter(chatData.RandomTime, func(rt RandomTimeVerse) bool { return rt.Id == randomId })
	if len(filtered) < 1 { return }
	for _, randomTime := range filtered {
		for _, send := range randomTime.NextSends {
			scheduler.RemoveJob(chatsRandomTimeJobsIds[chatId][randomId][send.String()[:10]])
		}
		delete(chatsRandomTimeJobsIds[chatId], randomId)
	}
	chatData.RandomTime = filter(chatData.RandomTime, func(rt RandomTimeVerse) bool { return rt.Id != randomId })
}

func clearCronsForChat(chatId int64, onlyJobs bool) {
	chatData := getChatData(chatId)
	idsList := chatsCronJobsIds[chatId]
	for _, jobId := range idsList {
		scheduler.RemoveJob(jobId)
	}
	for _, rt := range chatData.RandomTime {
		removeRandomTimeRegular(chatId, rt.Id)
	}
	chatsCronJobsIds[chatId] = make(map[string]uuid.UUID)
	if !onlyJobs {
		chatData := getChatData(chatId)
		chatData.VersesCrons = []string{}
	}
	saveChatsDataToFile()
}

func recreateJobsForChat(chatId int64) {
	chatData := getChatData(chatId)
	randomTimes := []RandomTimeVerse{}
	randomTimes = append(randomTimes, chatData.RandomTime...)
	copy(chatData.RandomTime, randomTimes)
	clearCronsForChat(chatId, true)
	addCronsForChat(chatData.VersesCrons, chatId, true)
	for _, rt := range randomTimes {
		addRandomTimeRegular(chatId, rt.StartTime, rt.StartTime.addDuration(rt.Duration))
	}
}

func removeCronForChat(chatId int64, cron string) {
	cron = strings.Trim(cron, " ")
	chatData := getChatData(chatId)
	chatData.VersesCrons = filter(chatData.VersesCrons, func (c string) bool { return c != cron })
	saveChatsDataToFile()
	scheduler.RemoveJob(chatsCronJobsIds[chatId][cron])
}

func filter[T any](ss []T, test func(T) bool) (ret []T) {
    for _, s := range ss {
        if test(s) {
            ret = append(ret, s)
        }
    }
    return
}

type CurrentTimeResponse struct {
	Timezone string `json:"timeZone"`
}

func getTimezoneByLocation(location Location) (string, error) {
	resp, err := http.Get(fmt.Sprintf(
		"https://www.timeapi.io/api/time/current/coordinate?latitude=%f&longitude=%f", location.Latitude, location.Longitude))
	if err != nil { return "", err }
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil { return "", err }
	var respData CurrentTimeResponse
	err = json.Unmarshal(body, &respData)
	if err != nil { return "", err }
	return respData.Timezone, nil
}

var chooseTimezoneKeyboard = ReplyKeyboardMarkup{[][]KeyboardButton{{{ "Отправить геопозицию", true }}, 
	{{"UTC+0", false}, {"UTC+1", false}, {"UTC+2", false}, {"UTC+3", false}},
	{{"UTC+4", false}, {"UTC+5", false}, {"UTC+6", false}, {"UTC+7", false}},
	{{"UTC+8", false}, {"UTC+9", false}, {"UTC+10", false}, {"UTC+11", false}},
	{{"UTC+12", false}, {"UTC+13", false}, {"UTC+14", false}},
	{{"UTC-1", false}, {"UTC-2", false}, {"UTC-3", false}, {"UTC-4", false}},
	{{"UTC-5", false}, {"UTC-6", false}, {"UTC-7", false}, {"UTC-8", false}},
	{{"UTC-9", false}, {"UTC-10", false}, {"UTC-11", false}, {"UTC-12", false}},
}}

var chooseTimezoneKeyboardNoLocation = ReplyKeyboardMarkup{[][]KeyboardButton{ 
	{{"UTC+0", false}, {"UTC+1", false}, {"UTC+2", false}, {"UTC+3", false}},
	{{"UTC+4", false}, {"UTC+5", false}, {"UTC+6", false}, {"UTC+7", false}},
	{{"UTC+8", false}, {"UTC+9", false}, {"UTC+10", false}, {"UTC+11", false}},
	{{"UTC+12", false}, {"UTC+13", false}, {"UTC+14", false}},
	{{"UTC-1", false}, {"UTC-2", false}, {"UTC-3", false}, {"UTC-4", false}},
	{{"UTC-5", false}, {"UTC-6", false}, {"UTC-7", false}, {"UTC-8", false}},
	{{"UTC-9", false}, {"UTC-10", false}, {"UTC-11", false}, {"UTC-12", false}},
}}

type TimezoneDiff struct{
	Diff     string
	Timezone string
}

var timezonesDiffs []TimezoneDiff

func readTimezonesDiffsFile() {
	fi, err := os.Open("timeZonesFromDiff.json")
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

	err = json.Unmarshal(b, &timezonesDiffs)
	if err != nil {
		panic(err)
	}
}

func getTimezoneByDiff(diff string) string {
	for _, diffTz := range timezonesDiffs {
		if diffTz.Diff == diff {
			return diffTz.Timezone
		}
	}
	return diff
}

func displayTimezone(timezone string) string {
	for _, diffTz := range timezonesDiffs {
		if diffTz.Timezone == timezone {
			return diffTz.Diff
		}
	}
	return timezone
}
