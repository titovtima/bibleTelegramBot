package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

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
			println(err)
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
	go sendMessage(message)
}

func clearCronsForChat(chatId int64, onlyJobs bool) {
	idsList := chatsCronJobsIds[chatId]
	for _, jobId := range idsList {
		scheduler.RemoveJob(jobId)
	}
	chatsCronJobsIds[chatId] = make(map[string]uuid.UUID)
	if !onlyJobs {
		chatData := getChatData(chatId)
		chatData.VersesCrons = []string{}
		saveChatsDataToFile()
	}
}

func recreateJobsForChat(chatId int64) {
	chatData := getChatData(chatId)
	clearCronsForChat(chatId, true)
	addCronsForChat(chatData.VersesCrons, chatId, true)
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

var chooseTimezoneKeyboard = ReplyKeyboardMarkup{[][]KeyboardButton{{{ "Определить время по геопозиции", true }}, 
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
