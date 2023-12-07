package endpoints

const (
	baseUrl            = "https://userapi.riichimahjong.org/v2/common."
	frey               = baseUrl + "Frey/"
	mimir              = baseUrl + "Mimir/"
	Authorize          = frey + "Authorize"
	GetMyEvents        = mimir + "GetMyEvents"
	GetCurrentSessions = mimir + "GetCurrentSessions"
	GetSessionOverview = mimir + "GetSessionOverview"
)
