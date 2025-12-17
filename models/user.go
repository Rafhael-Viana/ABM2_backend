package models

import "time"

type StatusUser string

const (
	StatusActive    StatusUser = "active"
	StatusInactive  StatusUser = "inactive"
	StatusVacations StatusUser = "vacations"
)

type User struct {
	ID         int32      `json:"id"`
	User_ID    string     `json:"user_id"`
	Name       string     `json:"name"`
	Senha      string     `json:"senha"`
	Setor      string     `json:"setor"`
	Cargo      string     `json:"cargo"`
	Nascimento *time.Time `json:"birth"`
	Username   string     `json:"username"`
	Email      string     `json:"email"`
	Status     StatusUser `json:"status"`
	Role       string     `json:"role"`
}

func (u *User) IsValidStatus(status StatusUser) bool {
	return status == StatusActive || status == StatusInactive
}
