package models

import "time"

type StatusPoint string

const (
	StatusOpen   StatusPoint = "open"
	StatusClosed StatusPoint = "close"
)

type Point struct {
	ID        int32       `json:"id"`
	User_ID   string      `json:"user_id"`
	Clock_In  *time.Time  `json:"clock_in"`
	Clock_Out *time.Time  `json:"clock_out"`
	Status    StatusPoint `json:"status"`
	Location  StatusPoint `json:"location"`
	Photo     StatusPoint `json:"foto"`
	CreatedAt *time.Time  `json:"createdAt"`
	UpdatedAt *time.Time  `json:"updatedAt"`
}
