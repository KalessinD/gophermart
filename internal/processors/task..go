package processors

import (
	"context"

	"github.com/KalessinD/gophermart/internal/models"
)

type (
	Task     models.Order
	TaskList []*Task

	TaskProcessor func(context.Context, *Task) error
)
