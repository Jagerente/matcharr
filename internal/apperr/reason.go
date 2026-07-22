package apperr

// Reason is the stable error taxonomy the frontend keys its copy on.
// Renaming a value breaks client-side error texts.
type Reason string

const (
	ReasonInternal          Reason = "internal"
	ReasonBadRequest        Reason = "bad_request"
	ReasonSessionNotFound   Reason = "session_not_found"
	ReasonInvalidToken      Reason = "invalid_token"
	ReasonHostOnly          Reason = "host_only"
	ReasonBadState          Reason = "bad_state"
	ReasonSessionFinished   Reason = "session_finished"
	ReasonSessionStarted    Reason = "session_started"
	ReasonSessionFull       Reason = "session_full"
	ReasonUnknownMovie      Reason = "unknown_movie"
	ReasonNotEnoughPlayers  Reason = "not_enough_participants"
	ReasonEmptySource       Reason = "empty_source"
	ReasonDeckFinished      Reason = "deck_finished"
	ReasonOutOfOrder        Reason = "swipe_out_of_order"
	ReasonNothingToUndo     Reason = "nothing_to_undo"
	ReasonAlreadyMatched    Reason = "already_matched"
	ReasonBadDeckLimit      Reason = "bad_deck_limit"
	ReasonSelfKick          Reason = "self_kick"
	ReasonNoParticipant     Reason = "participant_not_found"
	ReasonHostCannotLeave   Reason = "host_cannot_leave"
	ReasonUnknownPreset     Reason = "unknown_preset"
	ReasonBadSettings       Reason = "bad_settings"
	ReasonNameRequired      Reason = "name_required"
	ReasonNameTooLong       Reason = "name_too_long"
	ReasonSourceUnavailable Reason = "source_unavailable"
	ReasonMovieUnavailable  Reason = "movie_unavailable"
	ReasonRequestsDisabled  Reason = "requests_disabled"
	ReasonProfilesFailed    Reason = "profiles_unavailable"
	ReasonRequestFailed     Reason = "request_failed"
	ReasonRateLimited       Reason = "rate_limited"
	ReasonServerBusy        Reason = "server_busy"
)
