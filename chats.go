package main

import (
	"encoding/json"
	"io"
	"os"
	"slices"

	"github.com/google/uuid"
)

const chatsDataFileName = "chatsData.json"
const defaultTimezone = "Europe/Moscow"

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
	println("saving chats data")
	fo, err := os.Create(chatsDataFileName)
	if err != nil {
		return err
	}

	data := ChatsDataFile{chatsData}
	b, err := json.Marshal(data)

	println(string(b))
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
