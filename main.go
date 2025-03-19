package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron/v2"
)

var scheduler gocron.Scheduler

func main() {
	getBibleFromFile()
	getVersesListsFromFile()
	println(getRandomVerseFromList(1))
	createWebhook()
	getAdminId()

	statsTimezone := defaultTimezone
	loc, err := time.LoadLocation(statsTimezone)
	if err != nil {
		panic(err)
	}
	statsLocation = loc

	defaultLocation, err = time.LoadLocation(defaultTimezone)
	if err != nil {
		panic(err)
	}
	schedulerTimezone := defaultTimezone
	location, err := time.LoadLocation(schedulerTimezone)
	if err != nil {
		panic(err)
	}
	scheduler, err = gocron.NewScheduler(gocron.WithLocation(location))
	if err != nil {
		panic(err)
	}

	scheduler.Start()
	defer func() { scheduler.Shutdown() }()

	err = connectToDb()
	if err != nil { panic(err) }
	readTimezonesDiffsFile()
	err = setCronJobs()
	if err != nil { panic(err) }
	err = dbClearOldSends()
	if err != nil { panic(err) }
	err = createRandomTimeJobsAfterRestart()
	if err != nil { panic(err) }
	err = setDailyRandomTimeTasks()
	if err != nil { panic(err) }
	scheduler.NewJob(gocron.CronJob("0 1 * * *", false), gocron.NewTask(func() {
		setDailyRandomTimeTasks()
		dbClearOldSends()
	}))

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, "Error reading body", 400)
			println(err.Error())
			return
		}
		var update Update
		err = json.Unmarshal(body, &update)
		if err != nil {
			http.Error(writer, "Error parsing body", 400)
			println(err.Error())
			return
		}
		if update.CallbackQuery != nil {
			chatId := update.CallbackQuery.Message.Chat.Id
			writer.WriteHeader(200)
			if update.CallbackQuery.Data == "addcron cron" {
				dbUpdateMessageStatus(chatId, MessageStatusAddCronCron)
				message := SendMessage{
					ChatId: chatId,
					Text: "Введите строку в формате cron\\. Можно разделить несколько расписаний с помощью точки с запятой\\. " +
						"Например: `0 9 * * 6,7; 0 6-22/2 * * 1-5`",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 1" {
				dbUpdateMessageStatus(chatId, MessageStatusAddCron1)
				message := SendMessage{
					ChatId:    chatId,
					Text:      "Введите время в формате `чч:мм`\\. Например: `18:03`, или `07:40`",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 2" {
				dbUpdateMessageStatus(chatId, MessageStatusAddCron2)
				message := SendMessage{
					ChatId: chatId,
					Text: "Введите время в формате `чч:мм`\\. Можно разделить несколько расписаний с помощью запятой\\. " +
						"Например: `18:03, 07:40`, или `01:00, 10:20, 23:59`",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 3" {
				dbUpdateMessageStatus(chatId, MessageStatusAddCron3)
				message := SendMessage{
					ChatId: chatId,
					Text: "Введите номер дня недели и время в формате `д чч:мм`\\. Например: `1 18:03`, или `7 07:40`\\. " +
						"\\(1 \\- понедельник, 7 \\- воскресенье\\)",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 4" {
				dbUpdateMessageStatus(chatId, MessageStatusAddCron4)
				message := SendMessage{
					ChatId: chatId,
					Text: "Введите номер дня недели и время в формате `д чч:мм`\\. Можно разделить несколько расписаний с помощью запятой\\. " +
						"Например: `1 18:03, 7 07:40`\\. \\(1 \\- понедельник, 7 \\- воскресенье\\)",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 5" {
				dbUpdateMessageStatus(chatId, MessageStatusAddCron5)
				message := SendMessage{
					ChatId: chatId,
					Text: "Введите время начала и конца промежутка для отправки в случайное время " +
						"в формате `чч:мм, чч:мм`\\. Например: `07:40, 18:03`\\.",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if len(update.CallbackQuery.Data) > 11 && update.CallbackQuery.Data[:11] == "removecron:" {
				err := removeCronForChat(chatId, update.CallbackQuery.Data[11:])
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				message := SendMessage{
					ChatId:      chatId,
					Text:        "Расписание `" + strings.Trim(update.CallbackQuery.Data[11:], " ") + "` удалено",
					ParseMode:   "MarkdownV2",
					ReplyMarkup: ReplyKeyboardRemove,
				}
				go sendMessage(message)
			} else if len(update.CallbackQuery.Data) > 17 && update.CallbackQuery.Data[:17] == "removerandomtime:" {
				id, err := strconv.Atoi(update.CallbackQuery.Data[17:])
				if err != nil {
					println(err.Error())
					return
				}
				err = removeRandomTimeRegular(update.CallbackQuery.Message.Chat.Id, id)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				message := SendMessage{
					ChatId:      chatId,
					Text:        "Расписание случайного времени отправки удалено",
					ReplyMarkup: ReplyKeyboardRemove,
				}
				go sendMessage(message)
			}
			return
		} else if update.Message != nil {
			chatId := update.Message.Chat.Id
			err := dbAddChat(chatId, update.Message.Chat.ChatType)
			if err != nil {
				sendErrorMessage(chatId)
				return
			}
			writer.WriteHeader(200)

			statsDay := time.Now().In(statsLocation).Format(time.DateOnly)
			dbStatPlusOne(statsDay, "msg_received")
			dbStatUpdateChatsList(statsDay, "chats_received", chatId)

			if update.Message.Text == "/cancel" || update.Message.Text == "/cancel@"+BotName {
				messageStatus, _ := dbGetMessageStatus(chatId)
				if messageStatus != MessageStatusDefault {
					dbUpdateMessageStatus(chatId, MessageStatusDefault)
					message := SendMessage{
						ChatId:      chatId,
						Text:        "Операция отменена",
						ReplyMarkup: ReplyKeyboardRemove,
					}
					go sendMessage(message)
				}
				return
			}
			if update.Message.Text == "/addregular" || update.Message.Text == "/addregular@"+BotName {
				dbStatPlusOne(statsDay, "cmd_addregular")
				message := SendMessage{
					ChatId: chatId,
					Text:   "Выберите периодичность",
					ReplyMarkup: InlineKeyboardMarkup{[][]InlineKeyboardButton{
						{{"Раз в день", "addcron 1"}, {"Несколько раз в день", "addcron 2"}},
						{{"Раз в неделю", "addcron 3"}, {"Несколько раз в неделю", "addcron 4"}},
						{{"Случайно в промежутке, каждый день", "addcron 5"}},
						{{"Задать строку cron", "addcron cron"}},
					}},
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/getregular" || update.Message.Text == "/getregular@"+BotName {
				dbStatPlusOne(statsDay, "cmd_getregular")
				crons, err := dbGetAllCrons(chatId)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				randomTimes, err := dbGetAllRandomTimes(chatId)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				text := "Текущие расписания:"
				for _, cron := range crons {
					text += "\n" + cronToString(cron)
				}
				for _, rt := range randomTimes {
					text += "\n" + randomTimeToString(rt)
				}
				if len(crons)+len(randomTimes) == 0 {
					text = "Нет регулярных расписаний"
				}
				message := SendMessage{
					ChatId: chatId,
					Text:   text,
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/getregularcron" || update.Message.Text == "/getregularcron@"+BotName {
				dbStatPlusOne(statsDay, "cmd_getregularcron")
				crons, err := dbGetAllCrons(chatId)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				randomTimes, err := dbGetAllRandomTimes(chatId)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				text := "Текущие расписания:\n`"
				for i, cron := range crons {
					if i > 0 {
						text += "; "
					}
					text += cron
				}
				text += "`"
				for _, rt := range randomTimes {
					text += "\n" + randomTimeToString(rt)
				}
				if len(crons)+len(randomTimes) == 0 {
					text = "Нет регулярных расписаний"
				}
				message := SendMessage{
					ChatId:    chatId,
					Text:      text,
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/removeregular" || update.Message.Text == "/removeregular@"+BotName {
				dbStatPlusOne(statsDay, "cmd_removeregular")
				crons, err := dbGetAllCrons(chatId)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				randomTimes, err := dbGetAllRandomTimes(chatId)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				if len(crons)+len(randomTimes) == 0 {
					message := SendMessage{
						ChatId: chatId,
						Text:   "Нет регулярных расписаний",
					}
					go sendMessage(message)
					return
				}
				replyMarkup := InlineKeyboardMarkup{[][]InlineKeyboardButton{}}
				for _, cron := range crons {
					replyMarkup.InlineKeyboard = append(replyMarkup.InlineKeyboard,
						[]InlineKeyboardButton{{cronToString(cron), "removecron:" + cron}})
				}
				for _, randomTime := range randomTimes {
					replyMarkup.InlineKeyboard = append(replyMarkup.InlineKeyboard,
						[]InlineKeyboardButton{{randomTimeToShortString(randomTime), "removerandomtime:" + strconv.Itoa(randomTime.Id)}})
				}
				message := SendMessage{
					ChatId:      chatId,
					Text:        "Выберите расписание для удаления",
					ReplyMarkup: replyMarkup,
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/clearregular" || update.Message.Text == "/clearregular@"+BotName {
				dbStatPlusOne(statsDay, "cmd_clearregular")
				clearCronsForChat(chatId, false)
				clearRandomTimesForChat(chatId)
				message := SendMessage{
					ChatId:    chatId,
					Text:      "Расписания очищены",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/random" || update.Message.Text == "/random@"+BotName ||
				update.Message.Text == "/verse" || update.Message.Text == "/verse@"+BotName ||
				update.Message.Text == randomVerseTextMessage {
				dbStatPlusOne(statsDay, "cmd_random")
				message := SendMessage{
					ChatId: chatId,
					Text:   bible.getRandomVerse(),
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/settimezone" || update.Message.Text == "/settimezone@"+BotName {
				dbStatPlusOne(statsDay, "cmd_settimezone")
				dbUpdateMessageStatus(chatId, MessageStatusSetTimezone)
				if update.Message.Chat.ChatType == ChatTypePrivate {
					message := SendMessage{
						ChatId: chatId,
						Text: "Отправьте геопозицию, введите [название](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) часового пояса " +
							"\\(Например: `Europe/Moscow`\\), или выберите разницу с UTC \\(Например: `UTC+1`\\)",
						ParseMode:          "MarkdownV2",
						ReplyMarkup:        chooseTimezoneKeyboard,
						LinkPreviewOptions: LinkPreviewOptions{true},
					}
					go sendMessage(message)
					return
				} else {
					message := SendMessage{
						ChatId: chatId,
						Text: "Введите [название](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) часового пояса " +
							"\\(Например: `Europe/Moscow`\\), или выберите разницу с UTC \\(Например: `UTC+1`\\)",
						ParseMode:          "MarkdownV2",
						ReplyMarkup:        chooseTimezoneKeyboardNoLocation,
						LinkPreviewOptions: LinkPreviewOptions{true},
					}
					go sendMessage(message)
					return
				}
			}
			if update.Message.Text == "/gettimezone" || update.Message.Text == "/gettimezone@"+BotName {
				dbStatPlusOne(statsDay, "cmd_gettimezone")
				timezone, err := dbGetTimezone(chatId)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				message := SendMessage{
					ChatId:    chatId,
					Text:      fmt.Sprintf("Текущий часовой пояс: `%s`", displayTimezone(timezone)),
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/broadcast" || update.Message.Text == "/broadcast@"+BotName {
				if update.Message.From.Id == adminId {
					dbUpdateMessageStatus(chatId, MessageStatusBroadcast)
					message := SendMessage{
						ChatId:      chatId,
						Text:        "Отправьте сообщение для общей рассылки",
						ReplyMarkup: ReplyKeyboardRemove,
					}
					go sendMessage(message)
					return
				}
			}
			if (len(update.Message.Text) > 6 && update.Message.Text[:7] == "/stats ") || (update.Message.Text == "/stats") ||
				(len(update.Message.Text) > 6+len(BotName) && update.Message.Text[:7+len(BotName)] == "/stats@"+BotName) {
				if update.Message.From.Id == adminId || update.Message.From.Id == developerId {
					args := strings.Split(update.Message.Text, " ")
					startDate := time.Now().In(statsLocation).Add(-7 * 24 * time.Hour).Format(time.DateOnly)
					endDate := time.Now().In(statsLocation).Format(time.DateOnly)
					if len(args) > 2 {
						endDate = args[2]
					}
					if len(args) > 1 {
						startDate = args[1]
					}
					text, err := getStatsMessageText(startDate, endDate, "none")
					if err != nil {
						sendErrorMessage(chatId)
						return
					}
					go sendMessage(SendMessage{
						ChatId:      chatId,
						Text:        text,
						ParseMode:   "MarkdownV2",
						ReplyMarkup: ReplyKeyboardRemove,
					})
					return
				}
			}
			if (len(update.Message.Text) > 6 && update.Message.Text[:7] == "/statsw") ||
				(len(update.Message.Text) > 7+len(BotName) && update.Message.Text[:8+len(BotName)] == "/statsw@"+BotName) {
				if update.Message.From.Id == adminId || update.Message.From.Id == developerId {
					args := strings.Split(update.Message.Text, " ")
					startDate := time.Now().In(statsLocation).Add(-26 * 7 * 24 * time.Hour).Format(time.DateOnly)
					endDate := time.Now().In(statsLocation).Format(time.DateOnly)
					if len(args) > 2 {
						endDate = args[2]
					}
					if len(args) > 1 {
						startDate = args[1]
					}
					text, err := getStatsMessageText(startDate, endDate, "week")
					if err != nil {
						sendErrorMessage(chatId)
						return
					}
					go sendMessage(SendMessage{
						ChatId:      chatId,
						Text:        text,
						ParseMode:   "MarkdownV2",
						ReplyMarkup: ReplyKeyboardRemove,
					})
					return
				}
			}
			if (len(update.Message.Text) > 6 && update.Message.Text[:7] == "/statsm") ||
				(len(update.Message.Text) > 7+len(BotName) && update.Message.Text[:8+len(BotName)] == "/statsm@"+BotName) {
				if update.Message.From.Id == adminId || update.Message.From.Id == developerId {
					args := strings.Split(update.Message.Text, " ")
					startDate := "2024-11-17"
					endDate := time.Now().In(statsLocation).Format(time.DateOnly)
					if len(args) > 2 {
						endDate = args[2]
					}
					if len(args) > 1 {
						startDate = args[1]
					}
					text, err := getStatsMessageText(startDate, endDate, "month")
					if err != nil {
						sendErrorMessage(chatId)
						return
					}
					go sendMessage(SendMessage{
						ChatId:      chatId,
						Text:        text,
						ParseMode:   "MarkdownV2",
						ReplyMarkup: ReplyKeyboardRemove,
					})
					return
				}
			}
			if update.Message.Text == "/start" || update.Message.Text == "/start@"+BotName {
				dbStatPlusOne(statsDay, "cmd_start")
				message := getStartMessage(chatId)
				go sendMessage(message)
				dbUpdateMessageStatus(chatId, MessageStatusSetTimezone)
				return
			}
			messageStatus, err := dbGetMessageStatus(chatId)
			if err != nil {
				sendErrorMessage(chatId)
			}
			if messageStatus >= 1 && messageStatus <= 5 {
				if update.Message.Text != "" {
					var crons []string
					var err error = nil
					if messageStatus == MessageStatusAddCronCron {
						for _, cron := range strings.Split(update.Message.Text, ";") {
							trimmed := strings.Trim(cron, " ")
							if checkValidCron(trimmed) {
								crons = append(crons, trimmed)
							} else {
								message := SendMessage{
									ChatId: chatId,
									Text:   "Некорректный формат. Попробуйте ещё раз",
								}
								go sendMessage(message)
								return
							}
						}
					} else if messageStatus == MessageStatusAddCron1 {
						var cron string
						cron, err = parseTimeToCron(update.Message.Text)
						crons = []string{cron}
					} else if messageStatus == MessageStatusAddCron2 {
						crons, err = parseListTimesToCron(update.Message.Text)
					} else if messageStatus == MessageStatusAddCron3 {
						var cron string
						cron, err = parseWeekDayTimeToCron(update.Message.Text)
						crons = []string{cron}
					} else if messageStatus == MessageStatusAddCron4 {
						crons, err = parseListWeekDayTimesToCron(update.Message.Text)
					}
					if err != nil {
						message := SendMessage{
							ChatId: chatId,
							Text:   "Некорректный формат. Попробуйте ещё раз",
						}
						go sendMessage(message)
						return
					}
					err = addCronsForChat(crons, chatId, false)
					if err != nil {
						if errors.Is(err, errExistingCron) {
							go sendMessage(SendMessage{
								ChatId:      chatId,
								Text:        "Такое расписание уже установлено",
								ReplyMarkup: ReplyKeyboardRemove,
							})
							return
						} else {
							sendErrorMessage(chatId)
							return
						}
					}
					dbUpdateMessageStatus(chatId, MessageStatusDefault)
					message := SendMessage{
						ChatId:      chatId,
						Text:        "Расписание успешно добавлено",
						ReplyMarkup: ReplyKeyboardRemove,
					}
					go sendMessage(message)
					return
				}
			}
			if messageStatus == MessageStatusAddCron5 {
				if update.Message.Text != "" {
					times, err := parseListTimes(update.Message.Text)
					if err != nil || len(times) != 2 {
						message := SendMessage{
							ChatId: chatId,
							Text:   "Некорректный формат. Попробуйте ещё раз",
						}
						go sendMessage(message)
						return
					}
					addRandomTimeRegular(chatId, times[0], times[1])
					dbUpdateMessageStatus(chatId, MessageStatusDefault)
					message := SendMessage{
						ChatId: chatId,
						Text:   "Расписание успешно добавлено",
					}
					go sendMessage(message)
					return
				}
			}
			if messageStatus == MessageStatusSetTimezone {
				var timezone string
				if update.Message.Location != nil {
					var err1 error
					timezone, err1 = getTimezoneByLocation(*update.Message.Location)
					_, err2 := time.LoadLocation(timezone)
					if err1 != nil || err2 != nil {
						message := SendMessage{
							ChatId: chatId,
							Text: "Не удалось определить часовой пояс по местоположению\\. Можете попробовать ещё раз, или отправить " +
								"[название](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) часового пояса " +
								"\\(Например: `Europe/Moscow`\\)\\.",
							ParseMode:          "MarkdownV2",
							LinkPreviewOptions: LinkPreviewOptions{true},
						}
						go sendMessage(message)
						return
					}
				} else {
					timezone = getTimezoneByDiff(update.Message.Text)
					_, err := time.LoadLocation(timezone)
					if err != nil {
						message := SendMessage{
							ChatId: chatId,
							Text: "Не удалось определить часовой пояс\\. Можете попробовать ещё раз\\. Названия часовых поясов можно посмотреть " +
								"[здесь](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)\\. " +
								"Примеры: `Europe/Moscow`, `America/Los_Angeles`\\.",
							ParseMode: "MarkdownV2",
						}
						go sendMessage(message)
						return
					}
				}
				dbUpdateChatData(chatId, MessageStatusDefault, timezone)
				go recreateJobsForChat(chatId)
				text := "Часовой пояс `" + displayTimezone(timezone) + "` успешно установлен\\. "
				crons, err := dbGetAllCrons(chatId)
				if err != nil {
					sendErrorMessage(chatId)
					return
				}
				if len(crons) == 0 {
					message := SendMessage{
						ChatId:      chatId,
						Text:        text,
						ParseMode:   "MarkdownV2",
						ReplyMarkup: ReplyKeyboardRemove,
					}
					go sendMessage(message)
					return
				}
				if len(crons) == 1 {
					text += "Текущее расписание будет считаться по новому поясу\\."
				} else {
					text += "Текущие расписания будут считаться по новому поясу\\."
				}
				for _, cron := range crons {
					text += "\n" + escapingSymbols(cronToString(cron))
				}
				message := SendMessage{
					ChatId:      chatId,
					Text:        text,
					ParseMode:   "MarkdownV2",
					ReplyMarkup: ReplyKeyboardRemove,
				}
				go sendMessage(message)
				return
			} else if messageStatus == MessageStatusBroadcast {
				if update.Message.From.Id == adminId {
					if update.Message.Text != "" {
						dbUpdateMessageStatus(chatId, MessageStatusDefault)
						broadcastMessageToAll(update.Message.Text, update.Message.Entities)
						message := SendMessage{
							ChatId:      adminId,
							Text:        "Сообщение разослано",
							ReplyMarkup: ReplyKeyboardRemove,
						}
						go sendMessage(message)
						return
					}
				}
			}
			return
		}
		writer.WriteHeader(200)
	})
	port := os.Getenv("LOCAL_PORT")
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
