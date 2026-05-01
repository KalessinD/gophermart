package models

type (
	AccrualResponse struct {
		Order   string `json:"order"`
		Status  string `json:"status"`
		Accrual string `json:"accrual"` // будем потом переводить в целые копейки
	}
)
