package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func loadDictionary() []string {
	file, err := os.Open("sh.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var words []string
	for scanner.Scan() {
		words = append(words, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return words
}

var titleMatcher = regexp.MustCompile(`^([^=]+)(=.*)$`)
var undesiredSections = regexp.MustCompile(`(?s)====?(?:Conjugation|Declension|Derived terms|Pronunciation)====?[^=]*`)
var mainDefinitionSearcher = regexp.MustCompile(`(?s)===([^=]+)===[^#]*# ([^\n]*)`)
var removeTransitiveness = regexp.MustCompile(`{{indtr\|[^}|]*\|([^}])}}\s*`)
var removeCurlyLink = regexp.MustCompile(`{{[^}]*[|=]([^|}=]+)}}`)
var removeSquareLink = regexp.MustCompile(`\[\[(?:[^|]*\|)?([^|\]]*)\]\]`)

type Word struct {
	title            string
	grammaticalClass string
	mainDefinition   string
	err              error
}

func Parse(s string) Word {
	titleSize := strings.Index(s, "=")
	if titleSize == 0 {
		return Word{
			err: errors.New("Invalid title"),
		}
	}

	title := s[:titleSize]

	// replace escaped line brakes with newline
	s = strings.Replace(s[titleSize:], "\\n", "\n", -1)

	// remove undesired sections
	s = undesiredSections.ReplaceAllString(s, "")

	section := mainDefinitionSearcher.FindStringSubmatch(s)
	if len(section) < 3 {
		return Word{
			err: errors.New("No mainDefinition found"),
		}
	}

	mainDefinition := removeTransitiveness.ReplaceAllString(section[2], "")
	mainDefinition = removeCurlyLink.ReplaceAllString(mainDefinition, "$1")
	mainDefinition = removeSquareLink.ReplaceAllString(mainDefinition, "$1")
	return Word{
		title:            title,
		grammaticalClass: section[1],
		mainDefinition:   mainDefinition,
	}
}

func getPoll(dictionary []string, correctLineNumber int) (int, tgbotapi.SendPollConfig) {
	options := [3]Word{}
	grammaticalClass := ""
	for i := 0; i < 3; {
		lineNumber := rand.Intn(len(dictionary))
		if i == 0 {
			if correctLineNumber != -1 {
				lineNumber = correctLineNumber
			} else {
				correctLineNumber = lineNumber
			}
		}
		options[i] = Parse(dictionary[lineNumber])
		if options[i].err != nil {
			log.Printf("Error while parsing line %d: %s", lineNumber, options[i].err)
			continue
		}
		if grammaticalClass == "" {
			grammaticalClass = options[i].grammaticalClass
		} else if options[i].grammaticalClass != grammaticalClass {
			continue
		}

		i++
	}

	correctAnswerIndex := rand.Intn(3)
	if correctAnswerIndex != 0 {
		aux := options[0]
		options[0] = options[correctAnswerIndex]
		options[correctAnswerIndex] = aux
	}

	correctAnswer := options[correctAnswerIndex]
	return correctLineNumber, tgbotapi.SendPollConfig{
		Type:     "quiz",
		Question: fmt.Sprintf("%s (%s)", correctAnswer.mainDefinition, correctAnswer.grammaticalClass),
		Options: []string{
			options[0].title,
			options[1].title,
			options[2].title,
		},
		CorrectOptionID: int64(correctAnswerIndex),
		IsAnonymous:     true,
	}
}

type poll struct {
	chatID         int64
	wordLineNumber int
}

func sendPoll(
	dictionary []string,
	bot *tgbotapi.BotAPI,
	chatID int64,
	polls map[string]poll,
	chatIdWords map[int64][]int,
) {
	words := chatIdWords[chatID]
	correctLineNumber := -1
	if len(words) > 0 {
		correctLineNumber = words[0]
		chatIdWords[chatID] = words[1:]
	}
	correctLineNumber, sendPollConfig := getPoll(dictionary, correctLineNumber)
	sendPollConfig.BaseChat = tgbotapi.BaseChat{
		ChatID: chatID,
	}

	message, err := bot.Send(sendPollConfig)
	if err != nil {
		log.Printf("Error while sending poll: %s", err)
		return
	}

	polls[message.Poll.ID] = poll{
		chatID:         chatID,
		wordLineNumber: correctLineNumber,
	}
}

func main() {
	telegramBotToken := os.Args[1]
	bot, err := tgbotapi.NewBotAPI(telegramBotToken)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)
	dictionary := loadDictionary()
	subscribers := make(map[int64]bool)
	polls := make(map[string]poll)
	chatIdWords := make(map[int64][]int)

	// start a separate thread to send a message every hour
	go func() {
		for {
			time.Sleep(time.Hour)
			for subscriber := range subscribers {
				sendPoll(dictionary, bot, subscriber, polls, chatIdWords)
			}
		}
	}()

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			sendPoll(dictionary, bot, update.Message.Chat.ID, polls, chatIdWords)

			// add to the subscribers list
			subscribers[update.Message.Chat.ID] = true
		} else if update.Poll != nil {
			log.Printf("poll %#v", update.Poll)
			poll := polls[update.Poll.ID]
			sendPoll(dictionary, bot, poll.chatID, polls, chatIdWords)

			// if the answer was incorrect
			if update.Poll.Options[update.Poll.CorrectOptionID].VoterCount == 0 {
				chatIdWords[poll.chatID] = append(chatIdWords[poll.chatID], poll.wordLineNumber)
			}
		}
	}
}
