package processors

import (
	"context"
	"sync"

	"github.com/KalessinD/gophermart/internal/models"
	"go.uber.org/zap"
	// repository "github.com/KalessinD/gophermart/internal/repositories/postgresql"
)

type (
	// Тип задания в очереди
	Task     models.Order
	TaskList []*Task

	TaskProcessor func(context.Context, *Task) error

	// Сотрудник-одиночка для обработки заказов
	Worker struct {
		tasksCh     <-chan *Task
		postProcess TaskProcessor
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
		workers      []WorkerInterface
		wg           sync.WaitGroup
		mx           sync.Mutex
		workerCh     chan *Task
		cancel       context.CancelFunc
		log          *zap.Logger
		pendingTasks TaskList
		hasTasks     chan struct{}
	}

	// Интерфейс для управления коллективом сотрудников
	WorkerPoolInterface interface {
		Process(ctx context.Context, task *Task)
		Start(ctx context.Context)
		Stop()
		Wait()
	}
)

// Сотворение сотрудника
func NewWorker(id int, ch <-chan *Task, log *zap.Logger, action TaskProcessor) WorkerInterface {
	return &Worker{
		tasksCh:     ch,
		postProcess: action,
		id:          id,
		log:         log,
	}
}

// Метод создания рабочего коллектива
func NewQueueProcessor(poolSize, bufSize int, log *zap.Logger, action TaskProcessor) WorkerPoolInterface {
	pool := &WorkerPool{
		workers:      make([]WorkerInterface, poolSize),
		wg:           sync.WaitGroup{},
		mx:           sync.Mutex{},
		workerCh:     make(chan *Task, bufSize),
		pendingTasks: make(TaskList, 0, bufSize),
		hasTasks:     make(chan struct{}, 1),
		log:          log,
	}

	for i := range poolSize {
		pool.workers[i] = NewWorker(i, pool.workerCh, log, action)
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

// Обработка заказа
func (wp *WorkerPool) Process(ctx context.Context, task *Task) {
	wp.mx.Lock()
	wp.pendingTasks = append(wp.pendingTasks, task)
	wp.mx.Unlock()

	select {
	// case <-ctx.Done():
	// При Graceful Shutdown мы не обработаем задание, если диспетчер спал
	// return
	case wp.hasTasks <- struct{}{}:
		// толкнули диспетчера, чтобы не спал
	default:
		// если диспетчер занят, то просто идём по своим делам и никого не держим - задание в очереди
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

	wp.wg.Add(1)

	go func() {
		wp.runDispatcher(ctx)
		wp.wg.Done()
	}()

	go func() {
		wp.wg.Wait()
		close(wp.workerCh)
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

func (wp *WorkerPool) runDispatcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-wp.hasTasks:
			// нас разбудили и попросили поработать
		}

		for {
			wp.mx.Lock()

			// Если нет задач в очереди, возвращаемся на цикла выше и ждём задач
			if len(wp.pendingTasks) == 0 {
				wp.mx.Unlock()
				break
			}

			task := wp.pendingTasks[0]

			wp.mx.Unlock()

			select {
			case wp.workerCh <- task:
				wp.mx.Lock()
				if len(wp.pendingTasks) > 0 {
					wp.pendingTasks = wp.pendingTasks[1:]
				}
				wp.mx.Unlock()

			case <-ctx.Done():
				return
			}
		}
	}
}
