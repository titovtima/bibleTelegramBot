package main

import (
	"os"
	"strconv"
)

var adminId int64

func getAdminId() {
	var err error
	adminId, err = strconv.ParseInt(os.Getenv("ADMIN_ID"), 10, 64)
	if err != nil {
		panic(err)
	}
}

func broadcastMessageToAll(text string, entities []MessageEntity) {
	for _, chatData := range chatsData {
		if chatData.ChatId == adminId { continue }
		message := SendMessage{
			ChatId: chatData.ChatId,
			Text: text,
			Entities: entities,
		}
		go sendMessage(message)
	}
}
