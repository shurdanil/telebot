package functions

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	m "main/models"
	"math"
	"strconv"
	"strings"
)

func WindMap(roundIndex int32) string {
	if roundIndex < 5 {
		return "В"
	}
	if roundIndex < 9 {
		return "Ю"
	}
	return "З"
}

func RoundMap(roundIndex int32) int32 {
	if roundIndex < 5 {
		return roundIndex
	}
	if roundIndex < 9 {
		return roundIndex - 4
	}
	return roundIndex - 8
}

type playerType struct {
	Seat   int
	Dealer string
	Info   string
}

func SeatMap(playerIndex int) string {
	if playerIndex == 1 {
		return "Ш"
	}
	if playerIndex == 2 {
		return "Т"
	}
	return "К"
}

func Scores(gameOverview m.GameType, players []playerType, chatId int64) tgbotapi.MessageConfig {
	return tgbotapi.NewMessage(chatId,
		strings.Join(
			[]string{
				fmt.Sprintf("%s - %d, Х - %d, Р - %d",
					WindMap(gameOverview.SessionState.RoundIndex),
					RoundMap(gameOverview.SessionState.RoundIndex),
					gameOverview.SessionState.HonbaCount,
					gameOverview.SessionState.RiichiCount,
				),
				fmt.Sprintf("%s%s", players[0].Dealer, players[0].Info),
				fmt.Sprintf("%s%s%s", players[1].Dealer, SeatMap(1), players[1].Info),
				fmt.Sprintf("%s%s%s", players[2].Dealer, SeatMap(2), players[2].Info),
				fmt.Sprintf("%s%s%s", players[3].Dealer, SeatMap(3), players[3].Info),
			}, "\n"))
}

func Players(gameOverview m.GameType, me m.UserModel) []playerType {

	players := make([]playerType, 4)
	var myScores int32
	var meDealer bool
	var myIndex int
	for i, player := range gameOverview.Players {
		if int(player.Id) == me.PersonId {
			myScores = player.Score
			meDealer = player.Id == gameOverview.SessionState.Dealer
			myIndex = i
		}
	}

	for i, player := range gameOverview.Players {
		dealer := player.Id == gameOverview.SessionState.Dealer

		if int(player.Id) == me.PersonId {
			str := fmt.Sprintf("Мои - %.1f", float32(myScores)/1000)
			players[i] = playerType{
				Seat:   0,
				Dealer: "",
				Info:   str,
			}
			if dealer {
				players[i].Dealer = "!"
			}
		} else {
			delta := player.Score - myScores
			players[i] = playerType{
				Seat:   0,
				Dealer: "",
				Info:   fmt.Sprintf(" - %.1f (%.1f, %s)", float32(player.Score)/1000, float32(delta)/1000.0, Hans(delta, meDealer, dealer, gameOverview.SessionState)),
			}

			if dealer {
				players[i].Dealer = "!"
			}
		}
	}

	var result []playerType
	result = append(result, players[myIndex:]...)
	result = append(result, players[:myIndex]...)
	return result
}

func Hans(delta int32, meDealer bool, notMeDealer bool, sessionState m.SessionStateType) string {
	var response string

	deltaAbs := (int32(math.Abs(float64(delta))) - sessionState.HonbaCount*300 - sessionState.RiichiCount*1000) / 2
	fmt.Println(62, meDealer, notMeDealer, delta, deltaAbs)
	if (meDealer && delta > 0) || (notMeDealer && delta < 0) {
		switch {
		case deltaAbs <= 1500:
			response = "1/30"
		case deltaAbs <= 2000:
			response = "1/40"
		case deltaAbs <= 2400:
			response = "1/50, 2/25"
		case deltaAbs <= 2900:
			response = "2/30"
		case deltaAbs <= 3900:
			response = "2/40"
		case deltaAbs <= 4800:
			response = "2/50(3/25)"
		case deltaAbs <= 5800:
			response = "3/30"
		case deltaAbs <= 9600:
			response = "3/50(4/25)"
		case deltaAbs <= 11600:
			response = "4/30"
		case deltaAbs <= 12000:
			response = "4-5"
		case deltaAbs <= 18000:
			response = "6-7"
		case deltaAbs <= 24000:
			response = "8-10"
		case deltaAbs <= 36000:
			response = "11-12"
		default:
			response = "∞"
		}
	} else {
		switch {
		case deltaAbs <= 1000:
			response = "1/30"
		case deltaAbs <= 1300:
			response = "1/40"
		case deltaAbs <= 1600:
			response = "1/50(2/25)"
		case deltaAbs <= 2000:
			response = "2/30"
		case deltaAbs <= 2600:
			response = "2/40"
		case deltaAbs <= 3200:
			response = "2/50(3/25)"
		case deltaAbs <= 3900:
			response = "3/30"
		case deltaAbs <= 5200:
			response = "3/40"
		case deltaAbs <= 6400:
			response = "3/50(4/25)"
		case deltaAbs <= 7700:
			response = "4/30"
		case deltaAbs <= 8000:
			response = "4-5"
		case deltaAbs <= 12000:
			response = "6-7"
		case deltaAbs <= 16000:
			response = "8-10"
		case deltaAbs <= 24000:
			response = "11-12"
		default:
			response = "∞"
		}
	}
	if delta < 0 {
		return "-" + response
	}
	return response

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
