package main

import (
	"database/sql"
	"os"
	"sync"
	"time"
	_ "github.com/lib/pq"
)

var connStr = os.Getenv("DB_CONNECT_STRING")
var database *sql.DB

var dbMutex sync.Mutex

func connectToDb() error {
	var err error
	println(connStr)
	database, err = sql.Open("postgres", connStr)
	return err
}

func handleDbError(err error) {
	sendErrorReport(err, "Ошибка при работе с базой данных")
}

func dbGetMessageStatus(chatId int64) (MessageStatus, error) {
	result := database.QueryRow("select message_status from chat where id = $1;", chatId)
	var status MessageStatus
	err := result.Scan(&status)
	if err != nil {
		handleDbError(err)
	}
	return status, err
}

func dbUpdateMessageStatus(chatId int64, messageStatus MessageStatus) error {
	_, err := database.Exec("update chat set message_status = $1 where id = $2;", messageStatus, chatId)
	if err != nil {
		handleDbError(err)
	}
	return err
}

func dbUpdateChatData(chatId int64, messageStatus MessageStatus, timezone string) error {
	_, err := database.Exec("update chat set message_status = $1 and timezone = $2 where id = $3;", messageStatus, timezone, chatId)
	if err != nil {
		handleDbError(err)
	}
	return err
}

func dbGetTimezone(chatId int64) (string, error) {
	row := database.QueryRow("select timezone from chat where id = $1;", chatId)
	var timezone string
	err := row.Scan(&timezone)
	if err != nil {
		handleDbError(err)
	}
	return timezone, err
}

func dbAddCron(chatId int64, cron string) error {
	_, err := database.Exec("insert into verses_cron(chat_id, cron) values ($1, $2);", chatId, cron)
	if err != nil {
		handleDbError(err)
	}
	return err
}

func dbGetAllCrons(chatId int64) ([]string, error) {
	rows, err := database.Query("select cron from verses_cron where chat_id = $1;", chatId)
	if err != nil {
		handleDbError(err)
		return []string{}, nil
	}
	arr := []string{}
	for rows.Next() {
		var cron string
		err = rows.Scan(&cron)
		if err != nil {
			handleDbError(err)
			return arr, err
		}
		arr = append(arr, cron)
	}
	return arr, nil
}

func dbRemoveCron(chatId int64, cron string) error {
	_, err := database.Exec("delete from verses_cron where chat_id = $1 and cron = $2;", chatId, cron)
	if err != nil {
		handleDbError(err)
	}
	return err
}

func dbRemoveAllCronsForChat(chatId int64) error {
	_, err := database.Exec("delete from verses_cron where chat_id = $1;", chatId)
	if err != nil {
		handleDbError(err)
	}
	return err
}

func dbStatPlusOne(date string, stat string) error {
	_, err := database.Exec("insert into stats(date, name, count) values ($1, $2, 1) on conflict (date, name) do update set count = stats.count + 1;", date, stat)
	if err != nil {
		handleDbError(err)
	}
	return err
}

func dbStatUpdateChatsList(date string, stat string, chatId int64) error {
	_, err := database.Exec("insert into stats_list_chats(date, name, chat_id) values ($1, $2, $3) on conflict do nothing;", date, stat, chatId)
	if err != nil {
		handleDbError(err)
	}
	return err
}

func dbAddChat(chatId int64, chatType TelegramChatType) error {
	_, err := database.Exec(
		"insert into chat(id, type) values ($1, $2) on conflict (id) do update set type = excluded.type;",
		chatId, chatTypeToInt(chatType))
	if err != nil {
		handleDbError(err)
	}
	return err
}

func dbGetAllChats() ([]int64, error) {
	rows, err := database.Query("select id from chat;")
	if err != nil {
		handleDbError(err)
		return []int64{}, err
	}
	result := []int64{}
	for rows.Next() {
		var id int64
		err = rows.Scan(&id)
		if err != nil {
			handleDbError(err)
			return result, err
		}
		result = append(result, id)
	}
	return result, nil
}

func dbGetAllRandomTimes(chatId int64) ([]RandomTimeVerse, error) {
	rows, err := database.Query("select id, weekday, start_time, duration from random_time_verses where chat_id = $1;", chatId)
	if err != nil {
		handleDbError(err)
		return []RandomTimeVerse{}, err
	}
	result := []RandomTimeVerse{}
	for rows.Next() {
		var id int
		var weekday int
		var start_time int
		var duration int
		err = rows.Scan(&id, &weekday, &start_time, &duration)
		if err != nil {
			handleDbError(err)
			return []RandomTimeVerse{}, err
		}
		rows2, err := database.Query("select timestamp from next_sends where random_time_id = $1 order by timestamp;", id)
		if err != nil {
			handleDbError(err)
			return []RandomTimeVerse{}, err
		}
		nextSends := []time.Time{}
		for rows2.Next() {
			var t time.Time
			rows2.Scan(&t)
			nextSends = append(nextSends, t)
		}
		result = append(result, RandomTimeVerse{id, weekday, start_time, duration, nextSends})
	}
	return result, nil
}

func dbAddRandomTime(chatId int64, randomTime RandomTimeVerse) (int, error) {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	_, err := database.Exec("begin;")
	defer database.Exec("rollback;")
	if err != nil {
		handleDbError(err)
		return 0, err
	}
	row := database.QueryRow("insert into random_time_verses (id, chat_id, weekday, start_time, duration) values "+
		"((select min_key from keys where name = 'random_time_verses'), $1, $2, $3, $4) returning id;",
		chatId, randomTime.WeekDay, randomTime.StartTime, randomTime.Duration)
	var id int
	err = row.Scan(&id)
	if err != nil {
		handleDbError(err)
		return 0, err
	}
	_, err = database.Exec(
		"update keys set min_key = min_key + 1 where name = 'random_time_verses';" +
			"commit;")
	if err != nil {
		handleDbError(err)
		return 0, err
	}
	return id, nil
}

func dbGetRandomTimeById(randomTimeId int) (RandomTimeVerse, error) {
	row := database.QueryRow("select weekday, start_time, duration from random_time_verses where id = $1;", randomTimeId)
	var weekday, start_time, duration int
	err := row.Scan(&weekday, &start_time, &duration)
	if err != nil {
		handleDbError(err)
		return RandomTimeVerse{}, err
	}
	rows2, err := database.Query("select timestamp from next_sends where random_time_id = $1 order by timestamp;", randomTimeId)
	if err != nil {
		handleDbError(err)
		return RandomTimeVerse{}, err
	}
	nextSends := []time.Time{}
	for rows2.Next() {
		var t time.Time
		rows2.Scan(&t)
		nextSends = append(nextSends, t)
	}
	return RandomTimeVerse{randomTimeId, weekday, start_time, duration, nextSends}, nil
}

func dbRemoveRandomTime(chatId int64, randomTimeId int) error {
	_, err := database.Exec(
		"delete from random_time_verses where chat_id = $1 and id = $2;", chatId, randomTimeId)
	if err != nil {
		handleDbError(err)
		return err
	}
	return nil
}

func dbRemoveAllRandomTimesForChat(chatId int64) error {
	_, err := database.Exec(
		"delete from random_time_verses where chat_id = $1;", chatId)
	if err != nil {
		handleDbError(err)
		return err
	}
	return nil
}

func dbAddNextSend(randomTimeId int, send time.Time) error {
	_, err := database.Exec(
		"insert into next_sends (random_time_id, timestamp) values ($1, $2) on conflict do nothing;", randomTimeId, send)
	if err != nil {
		handleDbError(err)
		return err
	}
	return nil
}

func dbClearOldSends() error {
	_, err := database.Exec("delete from next_sends where timestamp < now();")
	if err != nil {
		handleDbError(err)
		return err
	}
	return nil
}

func dbGetStatsInRange(startDate string, endDate string) ([]Stats, error) {
	rows, err := database.Query("select count, date, name from stats where date >= $1 and date <= $2 order by date;",
		startDate, endDate)
	if err != nil {
		handleDbError(err)
		return nil, err
	}
	result := []Stats{}
	oldDate := ""
	for rows.Next() {
		var count int
		var date time.Time
		var name string
		err := rows.Scan(&count, &date, &name)
		if err != nil {
			handleDbError(err)
			return nil, err
		}
		dateStr := date.Format(time.DateOnly)
		if dateStr > oldDate {
			result = append(result, Stats{dateStr, make(map[string]int)})
		}
		result[len(result)-1].Count[name] = count
		oldDate = dateStr
	}
	return result, nil
}

func dbGetStatsListChatsInRange(startDate string, endDate string) ([]StatsListChats, error) {
	rows, err := database.Query("select chat_id, date, name from stats_list_chats where date >= $1 and date <= $2 order by date, name;",
		startDate, endDate)
	if err != nil {
		handleDbError(err)
		return nil, err
	}
	result := []StatsListChats{}
	oldDate := ""
	for rows.Next() {
		var chatId int64
		var date time.Time
		var name string
		err := rows.Scan(&chatId, &date, &name)
		if err != nil {
			handleDbError(err)
			return nil, err
		}
		dateStr := date.Format(time.DateOnly)
		if dateStr > oldDate {
			result = append(result, StatsListChats{dateStr, make(map[string][]int64)})
		}
		result[len(result)-1].Chats[name] = append(result[len(result)-1].Chats[name], chatId)
		oldDate = dateStr
	}
	return result, nil
}
