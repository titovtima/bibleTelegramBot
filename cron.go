package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

var validCronMinNumbers = []int{0, 0, 1, 1, 0}
var validCronMaxNumbers = []int{59, 23, 31, 12, 6}

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

func timeToString(time int) string {
	result := ""
	hours := time / 60
	minutes := time % 60
	if hours < 10 {
		result += "0"
	}
	result += strconv.Itoa(hours) + ":"
	if minutes < 10 {
		result += "0"
	}
	result += strconv.Itoa(minutes)
	return result
}

func parseTime(timeInput string) (int, error) {
	spl := strings.Split(strings.Trim(timeInput, " "), ":")
	if len(spl) != 2 {
		return 0, io.ErrShortWrite
	}
	i1, err1 := strconv.Atoi(spl[0])
	i2, err2 := strconv.Atoi(spl[1])
	if err1 != nil || err2 != nil {
		return 0, io.ErrShortWrite
	}
	if i1 < 0 || i2 < 0 || i1 > 23 || i2 > 59 {
		return 0, io.ErrShortWrite
	}
	return i1 * 60 + i2, nil
}

func timeToCron(t int) string {
	return fmt.Sprintf("%d %d * * *", t % 60, t / 60)
}

func parseTimeToCron(timeInput string) (string, error) {
	t, err := parseTime(timeInput)
	if err != nil {
		return "", err
	}
	return timeToCron(t), nil
}

func parseListTimes(timeInput string) ([]int, error) {
	spl := strings.Split(strings.Trim(timeInput, " "), ",")
	list := []int{}
	for _, part := range spl {
		t, err := parseTime(part)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
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

func parseWeekDayTime(timeInput string) (int, error) {
	spl := strings.Split(strings.Trim(timeInput, " "), " ")
	if len(spl) != 2 {
		return 0, io.ErrShortWrite
	}
	i1, err1 := strconv.Atoi(spl[0])
	t, err2 := parseTime(spl[1])
	if err1 != nil || err2 != nil {
		return 0, io.ErrShortWrite
	}
	if i1 < 0 || i1 > 7 {
		return 0, io.ErrShortWrite
	}
	i1 %= 7
	return i1 * 24 * 60 + t, nil
}

func weekDayTimeToCron(t int) string {
	return fmt.Sprintf("%d %d * * %d", t % 60, t / 60 % 24, t / (24 * 60))
}

func parseWeekDayTimeToCron(timeInput string) (string, error) {
	t, err := parseWeekDayTime(timeInput)
	if err != nil {
		return "", err
	}
	return weekDayTimeToCron(t), nil
}

func parseListWeekDayTimes(timeInput string) ([]int, error) {
	spl := strings.Split(strings.Trim(timeInput, " "), ",")
	list := []int{}
	for _, part := range spl {
		t, err := parseWeekDayTime(part)
		if err != nil {
			return nil, err
		}
		list = append(list, t)
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
	if parts[0] == "*" && parts[1] == "*" && parts[2] == "*" && parts[3] == "*" && parts[4] == "*" {
		return "Каждую минуту"
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
	endTime := randomTime.StartTime + randomTime.Duration
	return "Каждый день в случайное время с " + timeToString(randomTime.StartTime) + " до " + timeToString(endTime)
}

func randomTimeToShortString(randomTime RandomTimeVerse) string {
	endTime := randomTime.StartTime + randomTime.Duration
	return "Случайно с " + timeToString(randomTime.StartTime) + " до " + timeToString(endTime)
}

var errExistingCron = errors.New("cron already exists")

func addCronsForChat(crons []string, chatId int64, onlyJob bool) error {
	if chatsCronJobsIds[chatId] == nil {
		chatsCronJobsIds[chatId] = make(map[string]uuid.UUID)
	}
	exCrons, err := dbGetAllCrons(chatId)
	if err != nil { return err }
	if !onlyJob {
		for _, cron := range crons {
			if slices.Contains(exCrons, cron) {
				if len(crons) > 1 {
					continue;
				} else {
					return errExistingCron
				}
			}
			err := dbAddCron(chatId, cron)
			if err != nil {
				return err
			}
		}
	}
	timezone, err := dbGetTimezone(chatId)
	if err != nil { return err }
	for _, cron := range crons {
		job, err := scheduler.NewJob(gocron.CronJob(fmt.Sprintf("TZ=%s %s", timezone, cron), false),
			gocron.NewTask(randomVerseTask, chatId))
		if err != nil {
			println(err.Error())
			continue
		}
		chatsCronJobsIds[chatId][cron] = job.ID()
	}
	return nil
}

func randomVerseTask(chatId int64) {
	message := SendMessage{
		ChatId: chatId,
		Text:   bible.getRandomVerse(),
	}
	dbStatPlusOne(time.Now().In(statsLocation).Format(time.DateOnly), "scheduled_sent")
	go sendMessage(message)
}

func randomTimeTask(chatId int64, randomTime RandomTimeVerse, date string) {
	randomVerseTask(chatId)
	delete(chatsRandomTimeJobsIds[chatId][randomTime.Id], date)
}

func addRandomTimeForDay(day time.Time, randomTime RandomTimeVerse, chatId int64) error {
	if chatsRandomTimeJobsIds[chatId] == nil {
		chatsRandomTimeJobsIds[chatId] = make(map[int]map[string]uuid.UUID)
	}
	if chatsRandomTimeJobsIds[chatId][randomTime.Id] == nil {
		chatsRandomTimeJobsIds[chatId][randomTime.Id] = make(map[string]uuid.UUID)
	}
	timezone, err := dbGetTimezone(chatId)
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = defaultLocation
	}
	dayStartTime := time.Date(day.Year(), day.Month(), day.Day(), randomTime.StartTime / 60, randomTime.StartTime % 60, 0, 0, loc)
	if dayStartTime.Before(time.Now()) {
		return nil
	}
	if len(randomTime.NextSends) > 0 && randomTime.NextSends[len(randomTime.NextSends)-1].After(dayStartTime) {
		return nil
	}
	duration := rand.Intn(randomTime.Duration) + 1
	newTime := dayStartTime.Add(time.Duration(duration) * time.Minute)
	dbAddNextSend(randomTime.Id, newTime)
	job, err := scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(newTime)),
		gocron.NewTask(func () {
			randomTimeTask(chatId, randomTime, dayStartTime.Format(time.DateOnly))
		}))
	if err != nil {
		sendErrorReport(err, "error adding random time for day job")
		println(err.Error())
	} else {
		chatsRandomTimeJobsIds[chatId][randomTime.Id][dayStartTime.Format(time.DateOnly)] = job.ID()
	}
	return nil
}

func setDailyRandomTimeTasks() error {
	now := time.Now().In(defaultLocation)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day() + 1, 0, 0, 0, 0, defaultLocation)
	chats, err := dbGetAllChats()
	if err != nil {
		return err
	}
	for _, chatId := range chats {
		randomTimes, err := dbGetAllRandomTimes(chatId)
		if err != nil {
			return err
		}
		for _, rt := range randomTimes {
			addRandomTimeForDay(now, rt, chatId)
			addRandomTimeForDay(tomorrow, rt, chatId)
		}
	}
	return err
}

func createRandomTimeJobsAfterRestart() error {
	chats, err := dbGetAllChats()
	if err != nil {
		return err
	}
	for _, chatId := range chats {
		if chatsRandomTimeJobsIds[chatId] == nil {
			chatsRandomTimeJobsIds[chatId] = make(map[int]map[string]uuid.UUID)
		}
		randomTimes, err := dbGetAllRandomTimes(chatId)
		if err != nil {
			return err
		}
		for _, rt := range randomTimes {
			if chatsRandomTimeJobsIds[chatId][rt.Id] == nil {
				chatsRandomTimeJobsIds[chatId][rt.Id] = make(map[string]uuid.UUID)
			}
			for _, send := range rt.NextSends {
				job, err := scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(send)),
					gocron.NewTask(func () {
						randomTimeTask(chatId, rt, send.Format(time.DateOnly))
					}))
				if err != nil {
					sendErrorReport(err, "error creating random time job")
					println("error creating random time job", err.Error())
					return err
				} else {
					chatsRandomTimeJobsIds[chatId][rt.Id][send.Format(time.DateOnly)] = job.ID()
				}
			}
		}
	}
	return nil
}

func setCronJobs() error {
	chats, err := dbGetAllChats()
	if err != nil { return err }
	for _, chatId := range chats {
		crons, err := dbGetAllCrons(chatId)
		if err != nil { return err }
		err = addCronsForChat(crons, chatId, true)
		if err != nil { return err }
	}
	return nil
}

func addRandomTimeRegular(chatId int64, startTime int, endTime int) error {
	randomTime := RandomTimeVerse{-1, -1, startTime, endTime - startTime, []time.Time{}}
	id, err := dbAddRandomTime(chatId, randomTime)
	if err != nil { return err }
	randomTime.Id = id
	now := time.Now()
	timezone, err := dbGetTimezone(chatId)
	if err != nil { return err }
	loc, err := time.LoadLocation(timezone)
	if err != nil { loc = defaultLocation }
	now = now.In(loc)
	err = addRandomTimeForDay(now, randomTime, chatId) 
	if err != nil { return err }
	err = addRandomTimeForDay(time.Date(now.Year(), now.Month(), now.Day() + 1, 0, 0, 0, 0, loc), randomTime, chatId)
	if err != nil { return err }
	return nil
}

func removeRandomTimeRegular(chatId int64, randomId int) error {
	randomTime, err := dbGetRandomTimeById(randomId)
	if err != nil { return err }
	for _, send := range randomTime.NextSends {
		scheduler.RemoveJob(chatsRandomTimeJobsIds[chatId][randomId][send.String()[:10]])
	}
	err = dbRemoveRandomTime(chatId, randomId)
	return err
}

func clearCronsForChat(chatId int64, onlyJobs bool) error {
	idsList := chatsCronJobsIds[chatId]
	for _, jobId := range idsList {
		scheduler.RemoveJob(jobId)
	}
	chatsCronJobsIds[chatId] = make(map[string]uuid.UUID)
	if !onlyJobs {
		err := dbRemoveAllCronsForChat(chatId)
		return err
	}
	return nil
}

func clearRandomTimesForChat(chatId int64) error {
	for _, chatMap := range chatsRandomTimeJobsIds {
		for _, rtMap := range chatMap {
			for _, jobId := range rtMap {
				scheduler.RemoveJob(jobId)
			}
		}
	}
	return dbRemoveAllRandomTimesForChat(chatId)
}

func recreateJobsForChat(chatId int64) error {
	randomTimes, err := dbGetAllRandomTimes(chatId)
	if err != nil { return err }
	crons, err := dbGetAllCrons(chatId)
	if err != nil { return err }
	err = clearCronsForChat(chatId, true)
	if err != nil { return err }
	err = addCronsForChat(crons, chatId, true)
	if err != nil { return err }
	err = clearRandomTimesForChat(chatId)
	if err != nil { return err }
	for _, rt := range randomTimes {
		err = addRandomTimeRegular(chatId, rt.StartTime, rt.StartTime + rt.Duration)
		if err != nil { return err }
	}
	return nil
}

func removeCronForChat(chatId int64, cron string) error {
	cron = strings.Trim(cron, " ")
	err := dbRemoveCron(chatId, cron)
	if err != nil { return err }
	scheduler.RemoveJob(chatsCronJobsIds[chatId][cron])
	return nil
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
