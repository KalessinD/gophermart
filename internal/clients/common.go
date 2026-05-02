package clients

import "time"

type (
	ErrServiceIsBusy2 struct {
		message string
		delay   time.Duration
	}
)

// Error делает структуру реализацией интерфейса error
func (e *ErrServiceIsBusy2) Error() string {
	return e.message
}

func (e *ErrServiceIsBusy2) GetDelay() time.Duration {
	return e.delay
}

func NewErrServiceIsBusy(delay time.Duration) *ErrServiceIsBusy2 {
	return &ErrServiceIsBusy2{
		message: "service is to bysu to respond",
		delay:   delay,
	}
}
