package repositories

import (
	"github.com/KalessinD/gophermart/internal/models"
)

type (
	PersistStorageInterface interface {
		Save(orders models.OrdersList) error
		Restore() (models.OrdersList, error)
		Erase() error
	}
)
