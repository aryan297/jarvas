package event

import (
	"time"

	"github.com/google/uuid"
)

const (
	EvtUserRegistered = "auth.user_registered"
	EvtUserLoggedIn   = "auth.user_logged_in"
	EvtTokenRefreshed = "auth.token_refreshed"
	EvtUserLoggedOut  = "auth.user_logged_out"
)

type UserRegistered struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Provider  string    `json:"provider"`
	OccuredAt time.Time `json:"occured_at"`
}

func (e UserRegistered) EventName() string { return EvtUserRegistered }

type UserLoggedIn struct {
	UserID    uuid.UUID `json:"user_id"`
	IPAddress string    `json:"ip_address"`
	OccuredAt time.Time `json:"occured_at"`
}

func (e UserLoggedIn) EventName() string { return EvtUserLoggedIn }

type UserLoggedOut struct {
	UserID    uuid.UUID `json:"user_id"`
	OccuredAt time.Time `json:"occured_at"`
}

func (e UserLoggedOut) EventName() string { return EvtUserLoggedOut }
