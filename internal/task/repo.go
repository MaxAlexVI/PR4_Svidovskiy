package task

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

// ошибки для хранилища
var (
	ErrNotFound     = errors.New("task not found")
	ErrInvalidTitle = errors.New("title must be between 3 and 100 characters")
)

// Repo для хранения тасков 
type Repo struct {
	mu       sync.RWMutex
	seq      int64
	items    map[int64]*Task
	dataFile string
}

// Создание нового объекта Repo
func NewRepo() *Repo {
	return &Repo{
		items:    make(map[int64]*Task),
		dataFile: "tasks.json",
	}
}

// Валидация длины заголовка задач
func validateTitle(title string) error {
	if len(title) < 3 || len(title) > 100 {
		return ErrInvalidTitle
	}
	return nil
}

// Загрузка данных из JSON файла
func (r *Repo) LoadFromFile(fileName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dataFile = fileName

	data, err := os.ReadFile(fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return err
	}

	for _, task := range tasks {
		r.items[task.ID] = task
		if task.ID > r.seq {
			r.seq = task.ID
		}
	}

	return nil
}

// Сохранение данных в файл (БЕЗ РЕКУРСИИ)
func (r *Repo) saveToFile() error {
	// Получаем задачи напрямую, без вызова методов, чтобы избежать рекурсии
	r.mu.RLock()
	tasks := make([]*Task, 0, len(r.items))
	for _, t := range r.items {
		tasks = append(tasks, t)
	}
	r.mu.RUnlock()

	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.dataFile, data, 0644)
}

// Вывод всех тасков (БЕЗ СОХРАНЕНИЯ В ФАЙЛ)
func (r *Repo) List() []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Task, 0, len(r.items))
	for _, t := range r.items {
		out = append(out, t)
	}
	return out
}

// ListWithPagination возвращает задачи с пагинацией и фильтрацией
func (r *Repo) ListWithPagination(page, limit int, doneFilter *bool) []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Вычисляем смещение для пагинации
	offset := (page - 1) * limit
	
	// Фильтруем задачи
	var filtered []*Task
	for _, task := range r.items {
		// Применяем фильтр по статусу done, если указан
		if doneFilter != nil && task.Done != *doneFilter {
			continue
		}
		filtered = append(filtered, task)
	}

	total := len(filtered)
	
	// Применяем пагинацию
	start := offset
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	// Возвращаем пустой массив, если нет задач для текущей страницы
	if start >= total {
		return []*Task{}
	}

	// Возвращаем только задачи для текущей страницы
	return filtered[start:end]
}

// Вывод таска по id
func (r *Repo) Get(id int64) (*Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

// Создание таска
func (r *Repo) Create(title string) (*Task, error) {
	if err := validateTitle(title); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	now := time.Now()
	t := &Task{
		ID:        r.seq,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
		Done:      false,
	}
	r.items[t.ID] = t

	// Сохраняем изменения на диск (В ОТДЕЛЬНОЙ ФУНКЦИИ БЕЗ БЛОКИРОВКИ)
	go r.asyncSaveToFile()

	return t, nil
}

// Обновление таска
func (r *Repo) Update(id int64, title string, done bool) (*Task, error) {
	if err := validateTitle(title); err != nil {
		return nil, err
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	t.Title = title
	t.Done = done
	t.UpdatedAt = time.Now()

	// Сохраняем изменения на диск (В ОТДЕЛЬНОЙ ФУНКЦИИ БЕЗ БЛОКИРОВКИ)
	go r.asyncSaveToFile()

	return t, nil
}

// Удаление таска
func (r *Repo) Delete(id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.items[id]; !ok {
		return ErrNotFound
	}
	delete(r.items, id)

	// Сохраняем изменения на диск (В ОТДЕЛЬНОЙ ФУНКЦИИ БЕЗ БЛОКИРОВКИ)
	go r.asyncSaveToFile()

	return nil
}

// Асинхронное сохранение в файл (чтобы не блокировать основные операции)
func (r *Repo) asyncSaveToFile() {
	if err := r.saveToFile(); err != nil {

	}
}