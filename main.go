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
	config struct {
		Token    string `json:"token"`
		Login    string `json:"login"`
		Password string `json:"password"`
	}
)

func main() {

	config = f.CreateConfig()

	var err error
	bot, err = tgbotapi.NewBotAPI(config.Token)
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
			b, err := io.ReadAll(response.Body)
			if err != nil {
				return
			}
			err = json.Unmarshal(b, &data)
			if err != nil {
				log.Fatalln(err)
			}

			userObject, ok := userDB[message.Chat.ID]
			if ok {
				userObject.Token = data.AuthToken
				userObject.PersonId = data.PersonId
				userDB[message.Chat.ID] = userObject
			}

			selectEventsMenu(message.Chat.ID)

		}
	}

	if err != nil {
		log.Printf("An error occured: %s", err.Error())
	}
}

func selectEventsMenu(chatId int64) {

	var err error
	userObject, ok := userDB[chatId]
	if !ok {
		userObject.Login = config.Login
		userObject.Password = config.Password
		userDB[chatId] = userObject

		response, err := r.Authorize(userObject)

		if err != nil {
			fmt.Println("Error", err.Error())
			msg := tgbotapi.NewMessage(chatId, "Запрос вернул ошибку :(")
			_, err = bot.Send(msg)
		} else if response.StatusCode != 200 {
			b, err := io.ReadAll(response.Body)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println("Error", string(b))
			msg := tgbotapi.NewMessage(chatId, "Что-то не так с логином или паролем :(\nВыполните повторно команду /login")
			_, err = bot.Send(msg)
		} else {

			var data m.Target
			b, err := io.ReadAll(response.Body)
			if err != nil {
				return
			}
			err = json.Unmarshal(b, &data)
			if err != nil {
				log.Fatalln(err)
			}

			userObject, ok := userDB[chatId]
			if ok {
				userObject.Token = data.AuthToken
				userObject.PersonId = data.PersonId
				userDB[chatId] = userObject
			}
		}
	}

	var events m.EventType
	err = r.Post(e.GetMyEvents, []byte{}, userDB[chatId], &events)
	if err != nil {
		msg := tgbotapi.NewMessage(chatId, "Что-то пошло не так")
		bot.Send(msg)
		return
	}
	if len(events.Events) == 0 {
		msg := tgbotapi.NewMessage(chatId, "Вы не добавлены ни в одно событие")
		bot.Send(msg)
		return
	}

	msg := f.EventSelect(events, chatId)
	_, err = bot.Send(msg)
	if err != nil {
		log.Fatalln(err)
	}

}

// When we get a command, we react accordingly
func handleCommand(chatId int64, command string) error {
	var err error

	switch command {
	case "/login":
		err = login(chatId)
	case "/select":
		selectEventsMenu(chatId)
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
		err := r.Post(e.GetCurrentSessions, body, userDB[message.Chat.ID], &sessions)
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
			msg = f.Watch(sessions.Sessions[0].SessionHash, message.Chat.ID)
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
	var honbaCount int32

	body := []byte(`{"session_hash":"` + sessionHash + `"}`)
	for {

		var gameOverview m.GameType
		err := r.Post(e.GetSessionOverview, body, user, &gameOverview)
		if err != nil {
			msg := tgbotapi.NewMessage(chatId, "Что-то пошло не так")
			bot.Send(msg)
			return
		}
		if gameOverview.SessionState.Finished {
			msg := tgbotapi.NewMessage(chatId, "Игра закончена")
			bot.Send(msg)
			selectEventsMenu(chatId)
			return
		}

		if roundIndex != gameOverview.SessionState.RoundIndex || honbaCount != gameOverview.SessionState.HonbaCount {

			roundIndex = gameOverview.SessionState.RoundIndex
			honbaCount = gameOverview.SessionState.HonbaCount
			players := f.Players(gameOverview, user)
			msg := f.Scores(gameOverview, players, chatId)
			_, _ = bot.Send(msg)
		}
		time.Sleep(time.Second * 10)
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
