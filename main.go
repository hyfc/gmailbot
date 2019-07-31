package main

import (
	"gmailbot/gmail"
	"io/ioutil"
	"log"
	"os"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	gmailapi "google.golang.org/api/gmail/v1"
)

func main() {
	botToken, err := ioutil.ReadFile("BotToken")
	check(err)
	bot, err := tgbotapi.NewBotAPI(string(botToken))
	check(err)

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 100

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "start":
				addPeriodicTask(bot, update.Message.Chat.ID, checkNewMsg, 5)
				msg.Text = "Start forwarding mails."
			case "status":
				msg.Text = "I'm ok."
			default:
				msg.Text = "I don't know that command"
			}
			bot.Send(msg)
		}

	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func addPeriodicTask(bot *tgbotapi.BotAPI, ChatID int64, task func(bot *tgbotapi.BotAPI, ChatID int64) *gmailapi.Message, interval int64) {
	ticker := time.NewTicker(time.Second * time.Duration(interval))
	go func() {
		<-ticker.C
		task(bot, ChatID)
	}()
}

func checkNewMsg(bot *tgbotapi.BotAPI, ChatID int64) *gmailapi.Message {
	f, err := os.OpenFile("lastMsgID", os.O_CREATE|os.O_RDWR, 0666)
	defer f.Close()
	check(err)
	lastMsgID, err := ioutil.ReadFile("lastMsgID")
	check(err)
	ID := gmail.GetNewestMessageID()
	if ID != string(lastMsgID) {
		msg := gmail.GetMessage(ID)
		headers := make(map[string]string)
		for _, header := range msg.Payload.Headers {
			name := header.Name
			value := header.Value
			headers[name] = value
		}
		chatMsg := tgbotapi.NewMessage(ChatID, "")
		chatMsg.Text += ("*" + headers["From"] + "*\n")
		chatMsg.Text += (headers["Subject"] + "\n\n")
		chatMsg.Text += (headers["Date"] + "\n")
		chatMsg.Text += msg.Snippet
		chatMsg.ParseMode = "Markdown"
		bot.Send(chatMsg)
		_, err := f.Write([]byte(ID))
		check(err)
		return msg
	}

	return nil
}