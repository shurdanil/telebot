package models

type UserModel struct {
	LoginMode    bool
	PasswordMode bool
	Login        string
	Password     string
	Token        string
	PersonId     int
}

type PlayerInSession struct {
	Id         int32  `json:"id"`
	Title      string `json:"title"`
	Score      int32  `json:"score"`
	HasAvatar  bool   `json:"hasAvatar"`
	LastUpdate string `json:"lastUpdate"`
}

type SessionType struct {
	SessionHash string            `json:"sessionHash"`
	Status      string            `json:"status"`
	Players     []PlayerInSession `json:"players"`
}

type SessionsType struct {
	Sessions []SessionType `json:"sessions"`
}

type Penalty struct {
	Who    int32  `json:"who"`
	Amount int32  `json:"amount"`
	Reason string `json:"reason"`
}

type IntermediateResultOfSession struct {
	PlayerId     int32 `json:"playerId"`
	Score        int32 `json:"score"`
	PenaltyScore int32 `json:"penaltyScore"`
}

type SessionStateType struct {
	Dealer         int32                         `json:"dealer"`
	RoundIndex     int32                         `json:"roundIndex"`
	RiichiCount    int32                         `json:"riichiCount"`
	HonbaCount     int32                         `json:"honbaCount"`
	Finished       bool                          `json:"finished"`
	LastHandStated bool                          `json:"lastHandStated"`
	Scores         []IntermediateResultOfSession `json:"scores"`
	Penalties      []Penalty                     `json:"penalties"`
}

type GameType struct {
	Id           int32             `json:"id"`
	EventId      int32             `json:"eventId"`
	Players      []PlayerInSession `json:"players"`
	SessionState SessionStateType  `json:"state"`
}

type Target struct {
	PersonId  int    `json:"personId"`
	AuthToken string `json:"authToken"`
}

type Event struct {
	Id          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type EventType struct {
	Events []Event `json:"events"`
}
