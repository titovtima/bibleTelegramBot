package main

import (
	"encoding/json"
	"io"
	"math/rand"
	"os"
	"strconv"
)

type Verse string
type Chapter []Verse
type Book struct {
	Title      string    `json:"title"`
	ShortTitle string    `json:"shortTitle"`
	Chapters   []Chapter `json:"chapters"`
}
type Bible struct {
	Books []Book `json:"books"`
}

func getBibleFromFile() *Bible {
	fi, err := os.Open("bible.json")
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

	var bible Bible
	err = json.Unmarshal(b, &bible)
	if err != nil {
		panic(err)
	}

	return &bible
}

func (bible *Bible) getVerse(book int, chapter int, verse int) string {
	return string(bible.Books[book-1].Chapters[chapter-1][verse-1])
}

func (bible *Bible) getRandomVerse() string {
	book := bible.Books[rand.Intn(len(bible.Books))]
	chapterNum := rand.Intn(len(book.Chapters))
	chapter := book.Chapters[chapterNum]
	verseNum := rand.Intn(len(chapter))
	return `"` + string(chapter[verseNum]) + `" ` +
		book.ShortTitle + " " + strconv.Itoa(chapterNum+1) + ":" + strconv.Itoa(verseNum+1)
}
