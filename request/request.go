package request

import (
	"bytes"
	"encoding/json"
	"io"
	e "main/endpoints"
	m "main/models"
	"net/http"
	"strconv"
)

func Post[T interface{}](url string, body []byte, user m.UserModel, response T) (err error) {

	r, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return
	}

	r.Header.Add("authority", "userapi.riichimahjong.org")
	r.Header.Add("accept", "application/json; charset=UTF-8';")
	r.Header.Add("Content-Type", "application/json; charset=UTF-8';")
	r.Header.Add("x-auth-token", user.Token)
	r.Header.Add("x-current-person-id", strconv.Itoa(user.PersonId))

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, &response)
	return

}

func Authorize(user m.UserModel) (response *http.Response, err error) {
	var body = []byte(`{"email":"` + user.Login + `", "password": "` + user.Password + `"}`)

	r, err := http.NewRequest("POST", e.Authorize, bytes.NewBuffer(body))
	if err != nil {
		return
	}

	r.Header.Add("authority", "userapi.riichimahjong.org")
	r.Header.Add("accept", "application/json")
	r.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	response, err = client.Do(r)
	return
}
