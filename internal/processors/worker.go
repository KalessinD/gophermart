package processors

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type (

	// Сотрудник-одиночка для обработки заказов
	Worker struct {
		tasksCh     <-chan *Task
		postProcess TaskProcessor
		id          string
		log         *zap.Logger
	}

	// Интерфейс сотрудника
	WorkerInterface interface {
		Run(ctx context.Context)
		ID() string
	}
)

// Сотворение сотрудника
func NewWorker(ch <-chan *Task, log *zap.Logger, action TaskProcessor) (WorkerInterface, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	worker := &Worker{
		tasksCh:     ch,
		postProcess: action,
		id:          id.String(),
		log:         log,
	}

	return worker, nil
}

func (w *Worker) ID() string {
	return w.id
}

func (w *Worker) Run(ctx context.Context) {
	slog := w.log.Sugar()
	slog.Infof("worker %d has been started", w.id)

	for {
		select {
		case <-ctx.Done():
			return
		case task, opened := <-w.tasksCh:
			if !opened {
				return
			}
			err := w.postProcess(ctx, task)
			if err != nil {
				slog.Errorf("Failed to process task (orderID: %s, userID: %s): %s", task.ID, task.UserID, err.Error())
			}
		}
	}
}
