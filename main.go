package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

func main() {
	var bible *Bible

	go func() {
		bible = getBibleFromFile()
	}()
	createWebhook()

	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(writer, "Error reading body", 400)
			println(err)
			return
		}
		var update Update
		err = json.Unmarshal(body, &update)
		if err != nil {
			http.Error(writer, "Error parsing body", 400)
			println(err)
			return
		}
		writer.WriteHeader(200)
		if update.Message.Text != "" && update.Message.Chat.Id != 0 {
			message := SendMessage{
				ChatId: update.Message.Chat.Id,
				Text:   bible.getRandomVerse(),
				ReplyMarkup: ReplyKeyboardMarkup{[][]KeyboardButton{{{
					"Следующий случайный стих"}}}},
			}
			go sendMessage(message)
		}
	})
	log.Fatal(http.ListenAndServe(":2403", nil))
}
