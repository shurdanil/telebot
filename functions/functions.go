package functions

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	m "main/models"
	"strconv"
	"strings"
)

func RoundMap(roundIndex int32) string {
	if roundIndex < 5 {
		return "Восток"
	}
	if roundIndex < 9 {
		return "Юг"
	}
	return "Запад"
}

func Scores(gameOverview m.GameType, players []string, chatId int64) tgbotapi.MessageConfig {
	return tgbotapi.NewMessage(chatId, strings.Join(
		[]string{
			RoundMap(gameOverview.SessionState.RoundIndex) + " " + strconv.FormatInt(int64(gameOverview.SessionState.RoundIndex), 10),
			"Хонб - " + strconv.FormatInt(int64(gameOverview.SessionState.HonbaCount), 10),
			"Риичи палок - " + strconv.FormatInt(int64(gameOverview.SessionState.RiichiCount), 10),
			players[0],
			players[1],
			players[2],
			players[3],
		}, "\n"))
}

func Players(gameOverview m.GameType) (players []string) {

	var myScores int32

	for i, player := range gameOverview.Players {
		if i == 0 {
			myScores = player.Score
			players = append(players, "Мои - "+strconv.FormatInt(int64(player.Score), 10))
		} else {
			players = append(players, player.Title+" - "+strconv.FormatInt(int64(player.Score), 10)+" ("+strconv.FormatInt(int64(player.Score-myScores), 10)+")")
		}
	}
	return
}

func Watch(sessionHash string, chatId int64) tgbotapi.MessageConfig {
	monitoring := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отслеживать?", strings.Join([]string{
				"monitoring",
				sessionHash,
			}, "|"))),
	)

	msg := tgbotapi.NewMessage(chatId, "Есть игра!")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = monitoring
	return msg
}

func EventSelect(events m.EventType, chatId int64) tgbotapi.MessageConfig {
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

	msg := tgbotapi.NewMessage(chatId, "Выберите событие")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.ReplyMarkup = eventsMenu
	return msg
}
