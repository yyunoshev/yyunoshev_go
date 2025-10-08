package models

import "time"

// Weather представляет информацию о погоде для конкретного города
type Weather struct {
	City        string    `json:"city"`
	Temperature float64   `json:"temperature"`
	UpdatedAt   time.Time `json:"updated_at"`
}
