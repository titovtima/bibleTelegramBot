package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
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

	readChatsDataFromFile()
	readTimezonesDiffsFile()
	readStatsFile()
	createRandomTimeJobsAfterRestart()
	scheduler.NewJob(gocron.CronJob("0 1 * * *", false), gocron.NewTask(func ()  {
		setDailyRandomTimeTasks()
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
			chatData := getChatData(update.CallbackQuery.Message.Chat.Id)
			if chatData == nil {
				http.Error(writer, "Error getting user data", 400)
				return
			}
			writer.WriteHeader(200)
			if update.CallbackQuery.Data == "addcron cron" {
				chatData.MessageStatus = MessageStatusAddCronCron
				saveChatsDataToFile()
				message := SendMessage{
					ChatId: update.CallbackQuery.Message.Chat.Id,
					Text: "Введите строку в формате cron\\. Можно разделить несколько расписаний с помощью точки с запятой\\. " +
						"Например: `0 9 * * 6,7; 0 6-22/2 * * 1-5`",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 1" {
				chatData.MessageStatus = MessageStatusAddCron1
				saveChatsDataToFile()
				message := SendMessage{
					ChatId:    update.CallbackQuery.Message.Chat.Id,
					Text:      "Введите время в формате `чч:мм`\\. Например: `18:03`, или `07:40`",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 2" {
				chatData.MessageStatus = MessageStatusAddCron2
				saveChatsDataToFile()
				message := SendMessage{
					ChatId: update.CallbackQuery.Message.Chat.Id,
					Text: "Введите время в формате `чч:мм`\\. Можно разделить несколько расписаний с помощью запятой\\. " +
						"Например: `18:03, 07:40`, или `01:00, 10:20, 23:59`",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 3" {
				chatData.MessageStatus = MessageStatusAddCron3
				saveChatsDataToFile()
				message := SendMessage{
					ChatId: update.CallbackQuery.Message.Chat.Id,
					Text: "Введите номер дня недели и время в формате `д чч:мм`\\. Например: `1 18:03`, или `7 07:40`\\. " +
						"\\(1 \\- понедельник, 7 \\- воскресенье\\)",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 4" {
				chatData.MessageStatus = MessageStatusAddCron4
				saveChatsDataToFile()
				message := SendMessage{
					ChatId: update.CallbackQuery.Message.Chat.Id,
					Text: "Введите номер дня недели и время в формате `д чч:мм`\\. Можно разделить несколько расписаний с помощью запятой\\. " +
						"Например: `1 18:03, 7 07:40`\\. \\(1 \\- понедельник, 7 \\- воскресенье\\)",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if update.CallbackQuery.Data == "addcron 5" {
				chatData.MessageStatus = MessageStatusAddCron5
				saveChatsDataToFile()
				message := SendMessage{
					ChatId: update.CallbackQuery.Message.Chat.Id,
					Text: "Введите время начала и конца промежутка для отправки в случайное время " +
						"в формате `чч:мм, чч:мм`\\. Например: `07:40, 18:03`\\.",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
			} else if len(update.CallbackQuery.Data) > 11 && update.CallbackQuery.Data[:11] == "removecron:" {
				removeCronForChat(update.CallbackQuery.Message.Chat.Id, update.CallbackQuery.Data[11:])
				message := SendMessage{
					ChatId:      update.CallbackQuery.Message.Chat.Id,
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
				removeRandomTimeRegular(update.CallbackQuery.Message.Chat.Id, id)
				message := SendMessage{
					ChatId:      update.CallbackQuery.Message.Chat.Id,
					Text:        "Расписание случайного времени отправки удалено",
					ReplyMarkup: ReplyKeyboardRemove,
				}
				go sendMessage(message)
			}
			return
		} else if update.Message != nil {
			chatData := getChatData(update.Message.Chat.Id)
			if chatData == nil {
				http.Error(writer, "Error getting user data", 400)
				return
			}
			writer.WriteHeader(200)

			dayStats := getCurrentDayStats()
			dayStats.MessagesReceived++
			if slices.Index(dayStats.ChatsReceived, chatData.ChatId) == -1 {
				dayStats.ChatsReceived = append(dayStats.ChatsReceived, chatData.ChatId)
			}

			if update.Message.Text == "/cancel" || update.Message.Text == "/cancel@"+BotName {
				if chatData.MessageStatus != MessageStatusDefault {
					chatData.MessageStatus = MessageStatusDefault
					saveChatsDataToFile()
					message := SendMessage{
						ChatId:      update.Message.Chat.Id,
						Text:        "Операция отменена",
						ReplyMarkup: ReplyKeyboardRemove,
					}
					go sendMessage(message)
				}
				return
			}
			if update.Message.Text == "/addregular" || update.Message.Text == "/addregular@"+BotName {
				dayStats.Commands.AddRegular++
				message := SendMessage{
					ChatId: update.Message.Chat.Id,
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
				dayStats.Commands.GetRegular++
				crons := chatData.VersesCrons
				text := "Текущие расписания:"
				for _, cron := range crons {
					text += "\n" + cronToString(cron)
				}
				for _, rt := range chatData.RandomTime {
					text += "\n" + randomTimeToString(rt)
				}
				if len(crons) + len(chatData.RandomTime) == 0 {
					text = "Нет регулярных расписаний"
				}
				message := SendMessage{
					ChatId: update.Message.Chat.Id,
					Text:   text,
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/getregularcron" || update.Message.Text == "/getregularcron@"+BotName {
				dayStats.Commands.GetRegularCron++
				crons := chatData.VersesCrons
				text := "Текущие расписания:\n`"
				for i, cron := range crons {
					if i > 0 {
						text += "; "
					}
					text += cron
				}
				text += "`"
				for _, rt := range chatData.RandomTime {
					text += "\n" + randomTimeToString(rt)
				}
				if len(crons) + len(chatData.RandomTime) == 0 {
					text = "Нет регулярных расписаний"
				}
				message := SendMessage{
					ChatId:    update.Message.Chat.Id,
					Text:      text,
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/removeregular" || update.Message.Text == "/removeregular@"+BotName {
				dayStats.Commands.RemoveRegular++
				crons := chatData.VersesCrons
				randomTimes := chatData.RandomTime
				if len(crons) + len(randomTimes) == 0 {
					message := SendMessage{
						ChatId: update.Message.Chat.Id,
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
				for _, randomTime := range chatData.RandomTime {
					replyMarkup.InlineKeyboard = append(replyMarkup.InlineKeyboard,
						[]InlineKeyboardButton{{randomTimeToShortString(randomTime), "removerandomtime:" + strconv.Itoa(randomTime.Id)}})
				}
				message := SendMessage{
					ChatId:      update.Message.Chat.Id,
					Text:        "Выберите расписание для удаления",
					ReplyMarkup: replyMarkup,
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/clearregular" || update.Message.Text == "/clearregular@"+BotName {
				dayStats.Commands.ClearRegular++
				clearCronsForChat(update.Message.Chat.Id, false)
				message := SendMessage{
					ChatId:    update.Message.Chat.Id,
					Text:      "Расписания очищены",
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/random" || update.Message.Text == "/random@"+BotName ||
					update.Message.Text == "/verse" || update.Message.Text == "/verse@"+BotName ||
					update.Message.Text == randomVerseTextMessage {
				dayStats.Commands.Random++
				message := SendMessage{
					ChatId: update.Message.Chat.Id,
					Text:   bible.getRandomVerse(),
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/settimezone" || update.Message.Text == "/settimezone@"+BotName {
				dayStats.Commands.SetTimezone++
				chatData.MessageStatus = MessageStatusSetTimezone
				saveChatsDataToFile()
				if update.Message.Chat.ChatType == ChatTypePrivate {
					message := SendMessage{
						ChatId: update.Message.Chat.Id,
						Text: "Отправьте геопозицию, введите [название](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) часового пояса " +
							"\\(Например: `Europe/Moscow`\\), или выберите разницу с UTC \\(Например: `UTC+1`\\)",
						ParseMode:   "MarkdownV2",
						ReplyMarkup: chooseTimezoneKeyboard,
						LinkPreviewOptions: LinkPreviewOptions{true},
					}
					go sendMessage(message)
					return
				} else {
					message := SendMessage{
						ChatId: update.Message.Chat.Id,
						Text: "Введите [название](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) часового пояса " +
							"\\(Например: `Europe/Moscow`\\), или выберите разницу с UTC \\(Например: `UTC+1`\\)",
						ParseMode: "MarkdownV2",
						ReplyMarkup: chooseTimezoneKeyboardNoLocation,
						LinkPreviewOptions: LinkPreviewOptions{true},
					}
					go sendMessage(message)
					return
				}
			}
			if update.Message.Text == "/gettimezone" || update.Message.Text == "/gettimezone@"+BotName {
				dayStats.Commands.GetTimezone++
				message := SendMessage{
					ChatId:    update.Message.Chat.Id,
					Text:      fmt.Sprintf("Текущий часовой пояс: `%s`", displayTimezone(chatData.Timezone)),
					ParseMode: "MarkdownV2",
				}
				go sendMessage(message)
				return
			}
			if update.Message.Text == "/broadcast" || update.Message.Text == "/broadcast@"+BotName {
				if update.Message.From.Id == adminId {
					chatData.MessageStatus = MessageStatusBroadcast
					saveChatsDataToFile()
					message := SendMessage{
						ChatId:    update.Message.Chat.Id,
						Text:      "Отправьте сообщение для общей рассылки",
						ReplyMarkup: ReplyKeyboardRemove,
					}
					go sendMessage(message)
					return
				}
			}
			if (len(update.Message.Text) > 6 && update.Message.Text[:7] == "/stats ") || (update.Message.Text == "/stats") ||
					(len(update.Message.Text) > 6 + len(BotName) && update.Message.Text[:7+len(BotName)] == "/stats@"+BotName) {
				if update.Message.From.Id == adminId {
					args := strings.Split(update.Message.Text, " ")
					startDate := "2024-11-17"
					endDate := formatDate(time.Now())
					if len(args) > 2 {
						endDate = args[2]
					}
					if len(args) > 1 {
						startDate = args[1]
					}
					message := getStatsMessage(update.Message.Chat.Id, startDate, endDate, "none")
					go sendMessage(message)
					return
				}
			}
			if (len(update.Message.Text) > 6 && update.Message.Text[:7] == "/statsw") ||
					(len(update.Message.Text) > 7 + len(BotName) && update.Message.Text[:8+len(BotName)] == "/statsw@"+BotName) {
				if update.Message.From.Id == adminId {
					args := strings.Split(update.Message.Text, " ")
					startDate := "2024-11-17"
					endDate := formatDate(time.Now())
					if len(args) > 2 {
						endDate = args[2]
					}
					if len(args) > 1 {
						startDate = args[1]
					}
					message := getStatsMessage(update.Message.Chat.Id, startDate, endDate, "week")
					go sendMessage(message)
					return
				}
			}
			if (len(update.Message.Text) > 6 && update.Message.Text[:7] == "/statsm") ||
					(len(update.Message.Text) > 7 + len(BotName) && update.Message.Text[:8+len(BotName)] == "/statsm@"+BotName) {
				if update.Message.From.Id == adminId {
					args := strings.Split(update.Message.Text, " ")
					startDate := "2024-11-17"
					endDate := formatDate(time.Now())
					if len(args) > 2 {
						endDate = args[2]
					}
					if len(args) > 1 {
						startDate = args[1]
					}
					message := getStatsMessage(update.Message.Chat.Id, startDate, endDate, "month")
					go sendMessage(message)
					return
				}
			}
			if update.Message.Text == "/start" || update.Message.Text == "/start@"+BotName {
				dayStats.Commands.Start++
				message := getStartMessage(update.Message.Chat.Id)
				go sendMessage(message)
				chatData.MessageStatus = MessageStatusSetTimezone
				saveChatsDataToFile()
				return
			}
			if chatData.MessageStatus >= 1 && chatData.MessageStatus <= 5 {
				if update.Message.Text != "" {
					var crons []string
					var err error = nil
					if chatData.MessageStatus == MessageStatusAddCronCron {
						for _, cron := range strings.Split(update.Message.Text, ";") {
							trimmed := strings.Trim(cron, " ")
							if checkValidCron(trimmed) {
								crons = append(crons, trimmed)
							} else {
								message := SendMessage{
									ChatId: update.Message.Chat.Id,
									Text:   "Некорректный формат. Попробуйте ещё раз",
								}
								go sendMessage(message)
								return
							}
						}
					} else if chatData.MessageStatus == MessageStatusAddCron1 {
						var cron string
						cron, err = parseTimeToCron(update.Message.Text)
						crons = []string{cron}
					} else if chatData.MessageStatus == MessageStatusAddCron2 {
						crons, err = parseListTimesToCron(update.Message.Text)
					} else if chatData.MessageStatus == MessageStatusAddCron3 {
						var cron string
						cron, err = parseWeekDayTimeToCron(update.Message.Text)
						crons = []string{cron}
					} else if chatData.MessageStatus == MessageStatusAddCron4 {
						crons, err = parseListWeekDayTimesToCron(update.Message.Text)
					}
					if err != nil {
						message := SendMessage{
							ChatId: update.Message.Chat.Id,
							Text:   "Некорректный формат. Попробуйте ещё раз",
						}
						go sendMessage(message)
						return
					}
					addCronsForChat(crons, update.Message.Chat.Id, false)
					chatData.MessageStatus = MessageStatusDefault
					saveChatsDataToFile()
					message := SendMessage{
						ChatId: update.Message.Chat.Id,
						Text:   "Расписание успешно добавлено",
						ReplyMarkup: ReplyKeyboardRemove,
					}
					go sendMessage(message)
					return
				}
			}
			if chatData.MessageStatus == MessageStatusAddCron5 {
				if update.Message.Text != "" {
					times, err := parseListTimes(update.Message.Text)
					if err != nil || len(times) != 2 {
						message := SendMessage{
							ChatId: update.Message.Chat.Id,
							Text:   "Некорректный формат. Попробуйте ещё раз",
						}
						go sendMessage(message)
						return
					}
					addRandomTimeRegular(chatData.ChatId, times[0], times[1])
					chatData.MessageStatus = MessageStatusDefault
					saveChatsDataToFile()
					message := SendMessage{
						ChatId: update.Message.Chat.Id,
						Text:   "Расписание успешно добавлено",
					}
					go sendMessage(message)
					return
				}
			}
			if chatData.MessageStatus == MessageStatusSetTimezone {
				var timezone string
				if update.Message.Location != nil {
					var err1 error
					timezone, err1 = getTimezoneByLocation(*update.Message.Location)
					_, err2 := time.LoadLocation(timezone)
					if err1 != nil || err2 != nil {
						message := SendMessage{
							ChatId: update.Message.Chat.Id,
							Text: "Не удалось определить часовой пояс по местоположению\\. Можете попробовать ещё раз, или отправить " +
								"[название](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) часового пояса " +
								"\\(Например: `Europe/Moscow`\\)\\.",
							ParseMode: "MarkdownV2",
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
							ChatId: update.Message.Chat.Id,
							Text: "Не удалось определить часовой пояс\\. Можете попробовать ещё раз\\. Названия часовых поясов можно посмотреть " +
								"[здесь](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)\\. " +
								"Примеры: `Europe/Moscow`, `America/Los_Angeles`\\.",
							ParseMode: "MarkdownV2",
						}
						go sendMessage(message)
						return
					}
				}
				chatData.Timezone = timezone
				chatData.MessageStatus = MessageStatusDefault
				saveChatsDataToFile()
				go recreateJobsForChat(chatData.ChatId)
				text := "Часовой пояс `" + displayTimezone(timezone) + "` успешно установлен\\. "
				if len(chatData.VersesCrons) == 0 {
					message := SendMessage{
						ChatId:    update.Message.Chat.Id,
						Text:      text,
						ParseMode: "MarkdownV2",
						ReplyMarkup: ReplyKeyboardRemove,
					}
					go sendMessage(message)
					return
				}
				if len(chatData.VersesCrons) == 1 {
					text += "Текущее расписание будет считаться по новому поясу\\."
				} else {
					text += "Текущие расписания будут считаться по новому поясу\\."
				}
				for _, cron := range chatData.VersesCrons {
					text += "\n" + escapingSymbols(cronToString(cron))
				}
				message := SendMessage{
					ChatId:      update.Message.Chat.Id,
					Text:        text,
					ParseMode:   "MarkdownV2",
					ReplyMarkup: ReplyKeyboardRemove,
				}
				go sendMessage(message)
				return
			} else if chatData.MessageStatus == MessageStatusBroadcast {
				if update.Message.From.Id == adminId {
					if update.Message.Text != "" {
						chatData.MessageStatus = MessageStatusDefault
						saveChatsDataToFile()
						broadcastMessageToAll(update.Message.Text, update.Message.Entities)
						message := SendMessage{
							ChatId: adminId,
							Text: "Сообщение разослано",
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
