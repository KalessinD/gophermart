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
}
