package processors

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type (
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

// Метод создания рабочего коллектива
func NewQueueProcessor(poolSize, bufSize int, log *zap.Logger, action TaskProcessor) (WorkerPoolInterface, error) {
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
		worker, err := NewWorker(pool.workerCh, log, action)
		if err != nil {
			return nil, err
		}
		pool.workers[i] = worker
	}

	return pool, nil
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
		defer wp.wg.Done()
		wp.runDispatcher(ctx)
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
			// нас попросили поработать
		}

		for {
			wp.mx.Lock()

			if len(wp.pendingTasks) == 0 {
				wp.mx.Unlock()
				break
			}

			// LIFO
			lastIdx := len(wp.pendingTasks) - 1
			task := wp.pendingTasks[lastIdx]

			// зануляем ссылку, чтобы GC мог удалить объект
			wp.pendingTasks[lastIdx] = nil

			// Урезаем слайс. Теперь последний элемент выпал, но capacity осталась той же.
			wp.pendingTasks = wp.pendingTasks[:lastIdx]

			wp.mx.Unlock()

			select {
			case wp.workerCh <- task:

			case <-ctx.Done():
				return
			}
		}
	}
}
