package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type userModel struct {
	LoginMode    bool
	PasswordMode bool
	Login        string
	Password     string
	Token        string
	PersonId     int
}

var (
	// Menu texts
	firstMenu = "<b>Меню</b>\n"

	portalButton = "Portal"
	userDB       map[int64]userModel

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

		authUrl := "https://userapi.riichimahjong.org/v2/common.Frey/Authorize"

		var body = []byte(`{"email":"` + userDB[message.Chat.ID].Login + `", "password": "` + userDB[message.Chat.ID].Password + `"}`)

		r, err := http.NewRequest("POST", authUrl, bytes.NewBuffer(body))
		if err != nil {
			panic(err)
		}

		r.Header.Add("authority", "userapi.riichimahjong.org")
		r.Header.Add("accept", "application/json")
		r.Header.Add("Content-Type", "application/json")

		client := &http.Client{}
		res, err := client.Do(r)
		if err != nil {
			panic(err)
		}

		if err != nil {
			fmt.Println("Error", err.Error())
			msg := tgbotapi.NewMessage(message.Chat.ID, "Запрос вернул ошибку :(")
			_, err = bot.Send(msg)
		} else if res.StatusCode != 200 {
			b, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println("Error", string(b))
			msg := tgbotapi.NewMessage(message.Chat.ID, "Что-то не так с логином или паролем :(\nВыполните повторно команду /login")
			_, err = bot.Send(msg)
		} else {

			type target struct {
				PersonId  int    `json:"personId"`
				AuthToken string `json:"authToken"`
			}

			var data target
			err = json.NewDecoder(res.Body).Decode(&data)
			if err != nil {
				log.Fatalln(err)
			}
			err = r.Body.Close()
			if err != nil {
				return
			}

			fmt.Println(168, data.AuthToken)
			userObject, ok := userDB[message.Chat.ID]
			if ok {
				userObject.Token = data.AuthToken
				userObject.PersonId = data.PersonId
				userDB[message.Chat.ID] = userObject
			}

			eventsUrl := "https://gameapi.riichimahjong.org/v2/common.Mimir/GetMyEvents"

			body = []byte{}

			r, err := http.NewRequest("POST", eventsUrl, bytes.NewBuffer(body))
			if err != nil {
				panic(err)
			}

			r.Header.Add("authority", "userapi.riichimahjong.org")
			r.Header.Add("accept", "application/json; charset=UTF-8';")
			r.Header.Add("Content-Type", "application/json; charset=UTF-8';")
			r.Header.Add("x-auth-token", userDB[message.Chat.ID].Token)
			r.Header.Add("x-current-person-id", strconv.Itoa(userDB[message.Chat.ID].PersonId))

			client := &http.Client{}
			res, err = client.Do(r)
			if err != nil {
				panic(err)
			}
			b, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println(217, string(b))

			type event struct {
				Id          int    `json:"id"`
				Title       string `json:"title"`
				Description string `json:"description"`
			}

			type eventType struct {
				Events []event `json:"events"`
			}

			var events eventType
			err = json.Unmarshal(b, &events)
			if err != nil {
				log.Fatalln(err)
			}
			fmt.Println(228, events)

			var eventButtons [][]tgbotapi.InlineKeyboardButton

			for _, e := range events.Events {
				eventButtons = append(eventButtons, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData(e.Title, strings.Join([]string{
						"selectEvent",
						strconv.Itoa(e.Id),
						e.Title,
					}, "|")),
				))
			}
			eventsMenu := tgbotapi.NewInlineKeyboardMarkup(
				eventButtons...,
			)

			msg := tgbotapi.NewMessage(message.Chat.ID, "Выберите событие")
			msg.ParseMode = tgbotapi.ModeHTML
			msg.ReplyMarkup = eventsMenu
			_, err = bot.Send(msg)
			if err != nil {
				log.Fatalln(err)
			}

			//fetch("https://userapi.riichimahjong.org/v2/common.Frey/QuickAuthorize", {
			//	"headers": {
			//		"accept": "application/protobuf",
			//			"accept-language": "en-US,en;q=0.9,ru;q=0.8",
			//			"content-type": "application/protobuf",
			//			"sec-ch-ua": "\"Google Chrome\";v=\"119\", \"Chromium\";v=\"119\", \"Not?A_Brand\";v=\"24\"",
			//			"sec-ch-ua-mobile": "?0",
			//			"sec-ch-ua-platform": "\"Linux\"",
			//			"sec-fetch-dest": "empty",
			//			"sec-fetch-mode": "cors",
			//			"sec-fetch-site": "same-site",
			//			"x-auth-token": "00e66e9481dc0dfe0d8da4b03b953d769098f22a4fdbe04ea836fb93ed1afb263f1513516f8b6881a3db43c708543fc8",
			//			"x-current-person-id": "956",
			//			"x-twirp": "true",
			//			"Referer": "https://assist.riichimahjong.org/",
			//			"Referrer-Policy": "strict-origin-when-cross-origin"
			//	},
			//	"body": "\b¼\u0007\u0012`00e66e9481dc0dfe0d8da4b03b953d769098f22a4fdbe04ea836fb93ed1afb263f1513516f8b6881a3db43c708543fc8",
			//		"method": "POST"
			//});

			//quickAuthUrl := "https://userapi.riichimahjong.org/v2/common.Frey/AuthAuthorize"
			//
			//r, err = http.NewRequest("POST", quickAuthUrl, bytes.NewBuffer(jsonStr))
			//if err != nil {
			//	panic(err)
			//}
			//
			//r.Header.Add("accept", "application/json")
			//r.Header.Add("Content-Type", "application/json")
			//r.Header.Add("x-auth-token", userDB[message.Chat.ID].Token)
			//
			//client := &http.Client{}
			//res, err := client.Do(r)
			//if err != nil {
			//	panic(err)
			//}
			//
			//fmt.Println(225, res)
			//fmt.Println(r, res)
			//for _, cookie := range res.Cookies() {
			//	fmt.Println("Found a cookie named:", cookie.Name)
			//}
			//
			//b, err = io.ReadAll(res.Body)
			//if err != nil {
			//	log.Fatalln(err)
			//}
			//fmt.Println(262, string(b))
			//fmt.Println(262, binary.BigEndian.Uint16(b))

		}
		fmt.Println(130, userDB[message.Chat.ID])

		//msg := tgbotapi.NewMessage(message.Chat.ID, firstMenu)
		//msg.ParseMode = tgbotapi.ModeHTML
		//msg.ReplyMarkup = firstMenuMarkup
		//_, err := bot.Send(msg)
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

	message := query.Message

	fmt.Println(352, query)
	if strings.Contains(query.Data, "selectEvent") {
		textList := strings.Split(query.Data, "|")
		text = "Выбрано событие: " + textList[2]
	}

	callbackCfg := tgbotapi.NewCallback(query.ID, "")
	bot.Send(callbackCfg)

	msg := tgbotapi.NewMessage(message.Chat.ID, text)
	bot.Send(msg)

}

func sendMenu(chatId int64) error {
	msg := tgbotapi.NewMessage(chatId, firstMenu)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = firstMenuMarkup
	_, err := bot.Send(msg)
	return err
}

func selectEvent(chatId int64) error {
	fmt.Println(367, chatId)
	msg := tgbotapi.NewMessage(chatId, "Событие выбрано")
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
