package main

import (
	"bufio"
	"log"
	"math/rand"
	"regexp"
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

// extract the definition from a string using regular expressions
func extractDefinition(s string) string {
	re := regexp.MustCompile(`\[.*\]`)
	return re.FindString(s)
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

	// start a separate thread to send a message every hour
	go func() {
		for {
			time.Sleep(time.Hour)
			randomIndex := rand.Intn(len(dictionary))
			for subscriber, _ := range subscribers {
				msg := tgbotapi.NewMessage(subscriber, dictionary[randomIndex])
				bot.Send(msg)
			}
		}
	}()

	for update := range updates {
		if update.Message != nil { // If we got a message
			log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)
			randomIndex := rand.Intn(len(dictionary))

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text+" "+dictionary[randomIndex])
			msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)

			// add to the subscribers list
			subscribers[update.Message.Chat.ID] = true
		}
	}
}
