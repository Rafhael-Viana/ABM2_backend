package models

import "time"

type Setor struct {
	ID         int32      `json:"id"`
	Setor_ID   string     `json:"user_id"`
	Nome       string     `json:"name"`
	Quantidade int32      `json:"qtd_users"`
	Lider      string     `json:"lider"`
	CreatedBy  string     `json:"createdBy"`
	CreatedAt  *time.Time `json:"createdAt"`
	UpdatedAt  *time.Time `json:"updatedAt"`
}
