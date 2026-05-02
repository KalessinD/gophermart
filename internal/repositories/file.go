//go:generate mockgen -source=file.go -destination=mocks/mock_persist.go -package=mocks
package repositories

import (
	"encoding/json"
	"io"
	"os"
	"sync"

	"github.com/KalessinD/gophermart/internal/models"
)

type FileStorage struct {
	filePath string
	mu       sync.Mutex
}

type PersistStorageInterface interface {
	Save(orders models.OrdersList) error
	Restore() (models.OrdersList, error)
	Erase() error
}

func (m *FileStorage) Restore() (models.OrdersList, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fh, err := os.OpenFile(m.filePath, os.O_RDONLY, 0o600)
	if err != nil {
		// if not exists, it's not an error
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer fh.Close()

	// reading the small data
	data, err := io.ReadAll(fh)
	if err != nil {
		return nil, err
	}

	// делать нечего, пойдём курить бамбук
	if len(data) == 0 {
		return nil, nil
	}

	var orders models.OrdersList
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, err
	}

	return orders, nil
}

func (m *FileStorage) Save(orders models.OrdersList) error {
	data, err := json.Marshal(orders)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// os.WriteFile открывает файл с флагом O_TRUNC
	return os.WriteFile(m.filePath, data, 0o600)
}

// Erase удаляет файл с сохраненными данными.
func (m *FileStorage) Erase() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	err := os.Remove(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}

func NewFileStorage(filePath string) PersistStorageInterface {
	return &FileStorage{
		filePath: filePath,
	}
}
