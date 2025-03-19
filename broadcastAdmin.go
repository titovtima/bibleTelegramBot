package main

import (
	"os"
	"strconv"
	"time"

	"github.com/go-co-op/gocron/v2"
)

var adminId int64
var developerId int64

func getAdminId() {
	var err error
	adminId, err = strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	if err != nil {
		panic(err)
	}
	developerId, err = strconv.ParseInt(os.Getenv("DEVELOPER_ID"), 10, 64)
	if err != nil {
		panic(err)
	}
}

func broadcastMessageToAll(text string, entities []MessageEntity) error {
	chats, err := dbGetAllChats()
	if err != nil { return err }
	for _, chatId := range chats {
		if chatId == adminId { continue }
		message := SendMessage{
			ChatId: chatId,
			Text: text,
			Entities: entities,
		}
		go sendMessage(message)
	}
	return nil
}

const errorReportTimeout = 60 * time.Second
var errorReportTimeoutOn = false

func sendErrorReport(err error, msg string) {
	if errorReportTimeoutOn {
		return
	}
	errorReportTimeoutOn = true
	scheduler.NewJob(gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(time.Now().Add(errorReportTimeout))), gocron.NewTask(func () {
		errorReportTimeoutOn = false
	}))
	sendMessage(SendMessage{
		ChatId: developerId,
		Text: msg + "\n" + err.Error(),
	})
}
