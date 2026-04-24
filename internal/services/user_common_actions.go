package servives

import "database/sql"

type (
	CommonAction struct {
		database *sql.DB
	}

	UserCommonActions interface {
		Login() error
		Register() error
	}
)

func NewCommonAction() UserCommonActions {
	return &CommonAction{
		database: nil,
	}
}

func (s CommonAction) Login() error {
	return nil
}

func (s CommonAction) Register() error {
	return nil
}
