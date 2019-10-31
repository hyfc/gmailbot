package bot

import (
	"gmailbot/gmail"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

//Loop runs forever getting command from user and retrieving mails.
func Loop() {
	tokenBytes, err := ioutil.ReadFile("BotToken") //TODO: move to config.json
	check(err)
	botToken := strings.Replace(string(tokenBytes), "\n", "", 1)

	nameBytes, err := ioutil.ReadFile("UserName")
	check(err)
	userName := string(nameBytes)

	bot, err := tgbotapi.NewBotAPI(botToken)
	check(err)

	bot.Debug = false

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 100

	updates, err := bot.GetUpdatesChan(u)
	started := false
	for update := range updates {
		if update.Message == nil {
			continue
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			if update.Message.Chat.UserName != userName {
				msg.Text = "User Unauthorized."
				bot.Send(msg)
				continue
			}

			switch update.Message.Command() {
			case "start":
				if !started {
					go enterPeriodicTask(bot, update.Message.Chat.ID, checkNewMsg, 120) //TODO: make interval configurable.
					msg.Text = "Start forwarding mails."
					started = true
				} else {
					msg.Text = "Already started."
				}
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

func enterPeriodicTask(bot *tgbotapi.BotAPI, ChatID int64, task func(bot *tgbotapi.BotAPI, ChatID int64), interval int64) {
	for {
		ticker := time.NewTicker(time.Second * time.Duration(interval))
		<-ticker.C
		task(bot, ChatID)
	}
}

func checkNewMsg(bot *tgbotapi.BotAPI, ChatID int64) {
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
		chatMsg.Text += (headers["Date"] + "\n") //TODO: Convert UTC to local TZ specified by config.
		chatMsg.Text += msg.Snippet
		chatMsg.ParseMode = "Markdown"
		bot.Send(chatMsg)
		_, err := f.Write([]byte(ID))
		check(err)
	}
}
