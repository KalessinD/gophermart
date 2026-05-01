package processor

import (
	"context"
	"sync"

	"go.uber.org/zap"
	// repository "github.com/KalessinD/gophermart/internal/repositories/postgresql"
)

type (
	// Тип задания в очереди
	Task struct {
		OrderID string
		UserID  string
		Status  string
		Accrual uint32
	}

	// Сотрудник-одиночка для обработки заказов
	Worker struct {
		ch          <-chan *Task
		postProcess func(*Task) error
		id          int
		log         *zap.Logger
		// repository repository.SQLStorageInterface
	}

	// Интерфейс сотрудника
	WorkerInterface interface {
		Run(ctx context.Context)
		ID() int
	}

	// Объединение сотрудников в коллектив для обработки заказов сообща
	WorkerPool struct {
		workers []WorkerInterface
		wg      sync.WaitGroup
		mx      sync.Mutex
		ch      chan *Task
		cancel  context.CancelFunc
		log     *zap.Logger
	}

	// Интерфейс для управления коллективом сотрудников
	WorkerPoolInterface interface {
		Process(ctx context.Context, task *Task)
		Start(ctx context.Context)
		Stop()
		Wait()
	}
)

func NewTask(orderID, userID string) *Task {
	return &Task{
		OrderID: orderID,
		UserID:  userID,
	}
}

// Сотворение сотрудника
func NewWorker(id int, ch <-chan *Task, log *zap.Logger, action func(*Task) error) WorkerInterface {
	return &Worker{
		ch:          ch,
		postProcess: action,
		id:          id,
		log:         log,
	}
}

// Метод создания рабочего коллектива
func NewWorkerPool(poolSize, bufSize int, log *zap.Logger, action func(*Task) error) WorkerPoolInterface {
	pool := &WorkerPool{
		workers: make([]WorkerInterface, poolSize),
		wg:      sync.WaitGroup{},
		mx:      sync.Mutex{},
		ch:      make(chan *Task, bufSize),
		log:     log,
	}

	for i := range poolSize {
		pool.workers[i] = NewWorker(i, pool.ch, log, action)
	}

	return pool
}

func (w *Worker) ID() int {
	return w.id
}

func (w *Worker) Run(ctx context.Context) {
	slog := w.log.Sugar()
	slog.Infof("worker %d has been started", w.id)

	for {
		select {
		case <-ctx.Done():
			return
		case task, opened := <-w.ch:
			if !opened {
				return
			}
			err := w.postProcess(task)
			if err != nil {
				slog.Errorf("Failed to process task (orderId: %s, userID: %s): %s", task.OrderID, task.UserID, err.Error())
			}
			// default:
			// log.Debug("blocked")
			// time.Sleep(time.Millisecond * 100)
		}
	}
}

// Обработка заказа
func (wp *WorkerPool) Process(ctx context.Context, task *Task) {
	select {
	case <-ctx.Done():
		return
	case wp.ch <- task:
		return
		// default:
		// wp.queue = append(wp.queue, task)
		// return
	}
}

// Просим рабочий коллектив приступить к работе
func (wp *WorkerPool) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	wp.cancel = cancel

	wp.wg.Add(len(wp.workers))

	for _, worker := range wp.workers {
		go func(ctx context.Context, worker WorkerInterface, log *zap.Logger) {
			defer wp.wg.Done()
			worker.Run(ctx)
			log.Sugar().Infof("worker %d is done", worker.ID())
		}(ctx, worker, wp.log)
	}

	go func() {
		wp.wg.Wait()
		close(wp.ch)
	}()
}

// Отправляем коллектив сотрудников по домам
func (wp *WorkerPool) Stop() {
	wp.mx.Lock()
	defer wp.mx.Unlock()
	if wp.cancel != nil {
		wp.cancel()
		wp.cancel = nil
	}
}

// Ожидаем когда сотрудники доделают работу
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
