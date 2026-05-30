package entity

import (
	"time"

	"github.com/google/uuid"
)

// UserProfile is the aggregate root for the user bounded context.
// Auth concerns (password, provider) live in the auth module.
type UserProfile struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Bio         string
	Preferences Preferences
	Plan        SubscriptionPlan
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Preferences struct {
	Theme        string `json:"theme"`
	Language     string `json:"language"`
	Timezone     string `json:"timezone"`
	Notifications bool  `json:"notifications"`
}

type SubscriptionPlan string

const (
	PlanFree    SubscriptionPlan = "FREE"
	PlanPro     SubscriptionPlan = "PRO"
	PlanEnterprise SubscriptionPlan = "ENTERPRISE"
)
