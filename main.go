package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"
	e "main/endpoints"
	f "main/functions"
	m "main/models"
	r "main/request"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	firstMenu    = "<b>Меню</b>\n"
	portalButton = "Portal"
	url          = "https://assist.riichimahjong.org/"
	token        = "6376065784:AAGKmlSoMezH60wk8HWaMBvi8-_E_rSpO3s"
)

var (
	// Menu texts
	userDB map[int64]m.UserModel

	bot *tgbotapi.BotAPI

	firstMenuMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL(portalButton, url),
		),
	)
)

func main() {
	var err error
	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		// Abort if something is wrong
		log.Panic(err)
	}

	// Set this to true to log all interactions with telegram servers
	bot.Debug = false

	userDB = make(map[int64]m.UserModel)

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

		response, err := r.Authorize(userDB[message.Chat.ID])

		if err != nil {
			fmt.Println("Error", err.Error())
			msg := tgbotapi.NewMessage(message.Chat.ID, "Запрос вернул ошибку :(")
			_, err = bot.Send(msg)
		} else if response.StatusCode != 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println("Error", string(b))
			msg := tgbotapi.NewMessage(message.Chat.ID, "Что-то не так с логином или паролем :(\nВыполните повторно команду /login")
			_, err = bot.Send(msg)
		} else {

			var data m.Target
			err = json.NewDecoder(response.Body).Decode(&data)
			if err != nil {
				log.Fatalln(err)
			}

			userObject, ok := userDB[message.Chat.ID]
			if ok {
				userObject.Token = data.AuthToken
				userObject.PersonId = data.PersonId
				userDB[message.Chat.ID] = userObject
			}

			var events m.EventType
			err := r.Post(e.GetMyEvents, []byte{}, userDB[message.Chat.ID], events)
			if err != nil {
				msg := tgbotapi.NewMessage(message.Chat.ID, "Что-то пошло не так")
				bot.Send(msg)
				return
			}

			msg = f.EventSelect(events, message.Chat.ID)
			_, err = bot.Send(msg)
			if err != nil {
				log.Fatalln(err)
			}

		}
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

	message := query.Message

	if strings.Contains(query.Data, "selectEvent") {
		textList := strings.Split(query.Data, "|")
		text = "Выбрано событие: " + textList[2]

		callbackCfg := tgbotapi.NewCallback(query.ID, "")
		bot.Send(callbackCfg)

		msg := tgbotapi.NewMessage(message.Chat.ID, text)
		bot.Send(msg)

		userObject, _ := userDB[message.Chat.ID]

		body := []byte(`{"player_id":` + strconv.Itoa(userObject.PersonId) + `,"event_id":` + textList[1] + `}`)

		var sessions m.SessionsType
		err := r.Post(e.GetCurrentSessions, body, userDB[message.Chat.ID], sessions)
		if err != nil {
			msg := tgbotapi.NewMessage(message.Chat.ID, "Что-то пошло не так")
			bot.Send(msg)
			return
		}

		if len(sessions.Sessions) == 0 {
			msg = tgbotapi.NewMessage(message.Chat.ID, "Запущенных игр нет")
			bot.Send(msg)
			return
		}

		if len(sessions.Sessions) == 1 {
			f.Watch(sessions.Sessions[0].SessionHash, message.Chat.ID)
			bot.Send(msg)
		}
	} else if strings.Contains(query.Data, "monitoring") {
		sessionHash := strings.Split(query.Data, "|")[1]
		userObject, _ := userDB[message.Chat.ID]

		go monitor(sessionHash, userObject, message.Chat.ID)

	}

}

func monitor(sessionHash string, user m.UserModel, chatId int64) {

	var roundIndex int32

	body := []byte(`{"session_hash":"` + sessionHash + `"}`)
	for {

		var gameOverview m.GameType
		err := r.Post(e.GetSessionOverview, body, user, gameOverview)
		if err != nil {
			msg := tgbotapi.NewMessage(chatId, "Что-то пошло не так")
			bot.Send(msg)
			return
		}

		if roundIndex != gameOverview.SessionState.RoundIndex {

			roundIndex = gameOverview.SessionState.RoundIndex
			players := f.Players(gameOverview)
			msg := f.Scores(gameOverview, players, chatId)
			bot.Send(msg)
		}
		time.Sleep(time.Second * 15)
	}
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
		userDB[chatId] = m.UserModel{LoginMode: true}
	} else {
		userDB[chatId] = m.UserModel{
			LoginMode: true,
		}
	}
	_, err := bot.Send(msg)
	return err
}
