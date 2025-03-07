package main

import (
	"encoding/json"
	"io"
	"math/rand"
	"os"
	"strconv"
)

const versesListsFileName = "versesLists.json"

var bible Bible
var versesLists []VersesList

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

type LongVerse struct {
	Book    int   `json:"book"`
	Chapter int   `json:"chapter"`
	Verses  []int `json:"verses"`
}

type VersesList struct {
	Id      int
	Title   string      `json:"title"`
	List    []LongVerse `json:"list"`
	OwnerId int64
	Writers []int64
	Readers []int64
	Public  bool
}

type VersesListFile struct {
	Lists []VersesList `json:"lists"`
}

func getBibleFromFile() {
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

	err = json.Unmarshal(b, &bible)
	if err != nil {
		panic(err)
	}
}

func getVersesListsFromFile() {
	fi, err := os.Open(versesListsFileName)
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

	var versesListsFile VersesListFile
	err = json.Unmarshal(b, &versesListsFile)
	if err != nil {
		panic(err)
	}

	versesLists = versesListsFile.Lists
}

func saveVersesListsToFile() error {
	fo, err := os.Create(versesListsFileName)
	if err != nil {
		return err
	}

	data := VersesListFile{versesLists}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = fo.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (bible *Bible) getVerse(book int, chapter int, verse int) string {
	return string(bible.Books[book-1].Chapters[chapter-1][verse-1])
}

func (bible *Bible) getRandomVerse() string {
	book := bible.Books[rand.Intn(len(bible.Books))]
	chapterNum := rand.Intn(len(book.Chapters))
	chapter := book.Chapters[chapterNum]
	verseNum := rand.Intn(len(chapter))
	return formatResult(string(chapter[verseNum]), book.ShortTitle, chapterNum+1, []int{verseNum + 1})
}

func (list *VersesList) getRandomVerse() string {
	longVerse := list.List[rand.Intn(len(list.List))]
	result := bible.getVerse(longVerse.Book, longVerse.Chapter, longVerse.Verses[0])
	prev := longVerse.Verses[0]
	for i := 1; i < len(longVerse.Verses); i++ {
		if longVerse.Verses[i]-1 != prev {
			result += " …"
		}
		prev = longVerse.Verses[i]
		result += " " + bible.getVerse(longVerse.Book, longVerse.Chapter, prev)
	}
	return formatResult(result, bible.Books[longVerse.Book-1].ShortTitle, longVerse.Chapter, longVerse.Verses)
}

func getRandomVerseFromList(listId int) string {
	for _, verseList := range versesLists {
		if verseList.Id == listId {
			return verseList.getRandomVerse()
		}
	}
	return ""
}

func formatResult(text string, book string, chapter int, verses []int) string {
	result := "\"" + text + "\" (" + book + ". " + strconv.Itoa(chapter) + ":" + strconv.Itoa(verses[0])
	prev := verses[0]
	prevBegin := verses[0]
	for i := 1; i < len(verses); i++ {
		if verses[i]-1 != prev {
			if prev != prevBegin {
				result += "-" + strconv.Itoa(prev)
			}
			result += "," + strconv.Itoa(verses[i])
			prevBegin = verses[i]
		}
		prev = verses[i]
	}
	if prev != prevBegin {
		result += "-" + strconv.Itoa(prev)
	}
	result += ")"
	return result
}
