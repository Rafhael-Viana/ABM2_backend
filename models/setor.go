package models

import "time"

type Setor struct {
	ID         int32      `json:"id"`
	Setor_ID   string     `json:"setor_id"`
	Lider_ID   *string    `json:"lider_id"`
	Nome       string     `json:"name"`
	Quantidade int32      `json:"qtd_users"`
	Lider      string     `json:"lider"`
	CreatedBy  string     `json:"createdBy"`
	CreatedAt  *time.Time `json:"createdAt"`
	UpdatedAt  *time.Time `json:"updatedAt"`
}
