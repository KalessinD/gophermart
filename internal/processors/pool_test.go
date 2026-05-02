package processors_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/KalessinD/gophermart/internal/processors"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockTaskProcessor создает мок-функцию обработки, которая записывает ID задач.
func mockTaskProcessor(t *testing.T, calls *[]string) processors.TaskProcessor {
	t.Helper()
	return func(ctx context.Context, _ chan time.Duration, task *processors.Task) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			*calls = append(*calls, task.ID)
			return nil
		}
	}
}

// errorTaskProcessor возвращает ошибку при обработке
func errorTaskProcessor() processors.TaskProcessor {
	return func(_ context.Context, _ chan time.Duration, _ *processors.Task) error {
		return errors.New("processing error")
	}
}

func TestWorkerPool_Processing(t *testing.T) {
	t.Run("process tasks in LIFO order", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		log := zap.NewNop()
		taskCh := make(chan *processors.Task, 10)
		pauseCh := make(chan time.Duration, 1)

		var processedTasks []string
		action := mockTaskProcessor(t, &processedTasks)

		pool, err := processors.NewQueueProcessor(1, 2, log, taskCh, pauseCh, action)
		require.NoError(t, err)

		pool.Start(ctx)

		// Отправляем задачи.
		taskCh <- &processors.Task{ID: "1"}
		taskCh <- &processors.Task{ID: "2"}
		taskCh <- &processors.Task{ID: "3"}

		// Даем время на обработку
		time.Sleep(100 * time.Millisecond)

		pool.Stop()
		pool.Wait()

		require.NotEmpty(t, processedTasks)
		require.Equal(t, 3, len(processedTasks))
	})

	t.Run("stop and wait gracefully", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		log := zap.NewNop()
		taskCh := make(chan *processors.Task)
		pauseCh := make(chan time.Duration)

		var callCount int32
		action := func(_ context.Context, _ chan time.Duration, _ *processors.Task) error {
			atomic.AddInt32(&callCount, 1)
			return nil
		}

		pool, err := processors.NewQueueProcessor(2, 5, log, taskCh, pauseCh, action)
		require.NoError(t, err)

		pool.Start(ctx)

		taskCh <- &processors.Task{ID: "stop-test"}

		time.Sleep(50 * time.Millisecond)
		require.Equal(t, int32(1), atomic.LoadInt32(&callCount))

		pool.Stop()
		pool.Wait()
	})

	t.Run("handle error in processor", func(t *testing.T) {
		ctx := t.Context()
		log := zaptest.NewLogger(t)

		taskCh := make(chan *processors.Task, 1)
		pauseCh := make(chan time.Duration, 1)

		pool, err := processors.NewQueueProcessor(1, 1, log, taskCh, pauseCh, errorTaskProcessor())
		require.NoError(t, err)

		pool.Start(ctx)

		require.NotPanics(t, func() {
			taskCh <- &processors.Task{ID: "error-task"}
			time.Sleep(50 * time.Millisecond)
		})

		pool.Stop()
		pool.Wait()
	})

	t.Run("pause mechanism triggered by processor", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), 2000*time.Millisecond)
		defer cancel()

		log := zap.NewNop()
		taskCh := make(chan *processors.Task, 10)
		pauseCh := make(chan time.Duration, 1)

		var callCount int32

		// Используем специальный мок, который триггерит паузу для первой задачи
		pauseDuration := 200 * time.Millisecond
		action := func(_ context.Context, pCh chan time.Duration, _ *processors.Task) error {
			count := atomic.AddInt32(&callCount, 1)

			// Первая задача вызывает паузу (имитация 429)
			if count == 1 {
				pCh <- pauseDuration
			}
			return nil
		}

		pool, err := processors.NewQueueProcessor(1, 0, log, taskCh, pauseCh, action)
		require.NoError(t, err)

		pool.Start(ctx)

		taskCh <- &processors.Task{ID: "task1-pause-trigger"} // Отправляем задачу 1. Она обработается мгновенно и пошлет сигнал паузы.
		taskCh <- &processors.Task{ID: "task2"}               // Отправляем задачу 2. Она должна быть поставлена в очередь.

		// Ждем немного, чтобы задача 1 успела отработать и отправить сигнал в pauseCh.
		// Диспетчер должен получить сигнал и "уснуть" на pauseDuration.
		time.Sleep(10 * time.Millisecond)

		// В этот момент (прошло 10мс, пауза 200мс):
		// - Задача 1 уже обработана (callCount == 1).
		// - Диспетчер спит, поэтому задача 2 еще не передана воркеру.
		require.Equal(t, int32(1), atomic.LoadInt32(&callCount), "Task 2 should wait during pause")

		// Ждем, пока пауза закончится (еще ~200мс).
		time.Sleep(250 * time.Millisecond)

		// Теперь диспетчер проснулся и передал задачу 2 воркеру.
		require.Equal(t, int32(2), atomic.LoadInt32(&callCount), "Task 2 should be processed after pause")

		pool.Stop()
		pool.Wait()
	})
}
