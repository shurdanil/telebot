package endpoints

const (
	userApi            = "https://userapi.riichimahjong.org/v2/common."
	gameApi            = "https://gameapi.riichimahjong.org/v2/common."
	frey               = "Frey/"
	mimir              = "Mimir/"
	Authorize          = userApi + frey + "Authorize"
	GetMyEvents        = gameApi + mimir + "GetMyEvents"
	GetCurrentSessions = gameApi + mimir + "GetCurrentSessions"
	GetSessionOverview = gameApi + mimir + "GetSessionOverview"
)
