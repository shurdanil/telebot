package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	g "github.com/novalagung/gubrak/v2"
)

type userModel struct {
	LoginMode    bool
	PasswordMode bool
	Login        string
	Password     string
	Token        string
}

var (
	// Menu texts
	firstMenu = "<b>Меню</b>\n"

	portalButton = "Portal"

	userDB map[int64]userModel

	bot *tgbotapi.BotAPI

	firstMenuMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(portalButton, "https://assist.riichimahjong.org/"),
		),
	)
)

func main() {
	var err error
	bot, err = tgbotapi.NewBotAPI("6376065784:AAGKmlSoMezH60wk8HWaMBvi8-_E_rSpO3s")
	if err != nil {
		// Abort if something is wrong
		log.Panic(err)
	}

	// Set this to true to log all interactions with telegram servers
	bot.Debug = false

	userDB = make(map[int64]userModel)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	// Create a new cancellable background context. Calling `cancel()` leads to the cancellation of the context
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// `updates` is a golang channel which receives telegram updates
	updates := bot.GetUpdatesChan(u)

	// Pass cancellable context to goroutine
	go receiveUpdates(ctx, updates)

	// Tell the user the bot is online
	log.Println("Start listening for updates. Press enter to stop")

	// Wait for a newline symbol, then cancel handling updates
	bufio.NewReader(os.Stdin).ReadBytes('\n')
	cancel()

}

func receiveUpdates(ctx context.Context, updates tgbotapi.UpdatesChannel) {
	// `for {` means the loop is infinite until we manually stop it
	for {
		select {
		// stop looping if ctx is cancelled
		case <-ctx.Done():
			return
		// receive update from channel and then handle it
		case update := <-updates:
			handleUpdate(update)
		}
	}
}

func handleUpdate(update tgbotapi.Update) {
	switch {
	// Handle messages
	case update.Message != nil:
		handleMessage(update.Message)
		break

	// Handle button clicks
	case update.CallbackQuery != nil:
		handleButton(update.CallbackQuery)
		break
	}
}

func handleMessage(message *tgbotapi.Message) {
	user := message.From
	text := message.Text

	if user == nil {
		return
	}

	// Print to console
	log.Printf("%s wrote %s", user.FirstName, text)

	var err error
	if strings.HasPrefix(text, "/") {
		err = handleCommand(message.Chat.ID, text)
	} else if userDB[message.Chat.ID].LoginMode && len(text) > 0 {
		userObject, ok := userDB[message.Chat.ID]
		if ok {
			userObject.LoginMode = false
			userObject.Login = text
			userObject.PasswordMode = true
			userDB[message.Chat.ID] = userObject
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, "Введите свой пароль")
		_, err = bot.Send(msg)
	} else if userDB[message.Chat.ID].PasswordMode && len(text) > 0 {
		userObject, ok := userDB[message.Chat.ID]
		if ok {
			userObject.Password = text
			userObject.PasswordMode = false
			userDB[message.Chat.ID] = userObject
		}
		msg := tgbotapi.NewMessage(message.Chat.ID, "Спасибо)")
		_, err = bot.Send(msg)

		posturl := "https://userapi.riichimahjong.org/v2/common.Frey/Authorize"

		body := []byte("\n\u0011" + userObject.Login + "\u0012\f" + userObject.Password)

		r, err := http.NewRequest("POST", posturl, bytes.NewBuffer(body))
		if err != nil {
			panic(err)
		}

		r.Header.Add("authority", "userapi.riichimahjong.org")
		r.Header.Add("content-type", "application/protobuf")

		client := &http.Client{}
		res, err := client.Do(r)
		if err != nil {
			panic(err)
		}

		fmt.Println(144, res.StatusCode)
		if err != nil {
			fmt.Println(err.Error())
			msg := tgbotapi.NewMessage(message.Chat.ID, "Запрос вернул ошибку :(")
			_, err = bot.Send(msg)
		} else if res.StatusCode != 200 {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Что-то не так с логином или паролем :(\nВыполните повторно команду /login")
			_, err = bot.Send(msg)
		} else {
			b, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalln(err)
			}

			token := g.From(strings.Split(string(b), "")).Drop(5).Result()
			fmt.Println(168, token)
			if token != nil {
				fmt.Println(147, strings.Join(token.([]string), ""))
			}
		}
		fmt.Println(130, userDB[message.Chat.ID])
	} else {
		// This is equivalent to forwarding, without the sender's name
		copyMsg := tgbotapi.NewCopyMessage(message.Chat.ID, message.Chat.ID, message.MessageID)
		_, err = bot.CopyMessage(copyMsg)
	}

	if err != nil {
		log.Printf("An error occured: %s", err.Error())
	}
}

// When we get a command, we react accordingly
func handleCommand(chatId int64, command string) error {
	var err error

	switch command {
	case "/login":
		err = login(chatId)
		break
	case "/menu":
		err = sendMenu(chatId)
		break
	}

	return err
}

func handleButton(query *tgbotapi.CallbackQuery) {
	var text string

	markup := tgbotapi.NewInlineKeyboardMarkup()
	message := query.Message
	//
	//if query.Data == nextButton {
	//	text = secondMenu
	//	markup = secondMenuMarkup
	//} else if query.Data == backButton {
	//	text = firstMenu
	//	markup = firstMenuMarkup
	//}

	callbackCfg := tgbotapi.NewCallback(query.ID, "")
	bot.Send(callbackCfg)

	// Replace menu text and keyboard
	msg := tgbotapi.NewEditMessageTextAndMarkup(message.Chat.ID, message.MessageID, text, markup)
	msg.ParseMode = tgbotapi.ModeHTML
	bot.Send(msg)
}

func sendMenu(chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, firstMenu)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = firstMenuMarkup
	_, err := bot.Send(msg)
	return err
}

func login(chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, "Введите свой логин")
	_, ok := userDB[chatId]
	if ok {
		userDB[chatId] = userModel{LoginMode: true}
	} else {
		userDB[chatId] = userModel{
			LoginMode: true,
		}
	}
	_, err := bot.Send(msg)
	return err
}
