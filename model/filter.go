package model

import "time"

type IncidentFilter struct {
	EventID     *uint
	Status      *string
	StartedFrom *time.Time
	StartedTo   *time.Time
	Search      *string
}

type EventFilter struct {
	StartedFrom *time.Time
	StartedTo   *time.Time
	Search      *string
}

type UserFilter struct {
	Role   *string
	Search *string
}
