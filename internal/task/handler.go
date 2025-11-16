package task

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// Handler управляет HTTP запросами для задач
type Handler struct {
	repo *Repo  // Ссылка на хранилище задач
}

// NewHandler создает новый обработчик HTTP запросов
func NewHandler(repo *Repo) *Handler {
	return &Handler{repo: repo}
}

// Routes настраивает маршруты для API v1
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	
	// Настраиваем все маршруты для задач
	r.Get("/", h.list)           // GET /api/v1/tasks - список задач с пагинацией и фильтрацией
	r.Post("/", h.create)        // POST /api/v1/tasks - создать новую задачу
	r.Get("/{id}", h.get)        // GET /api/v1/tasks/{id} - получить задачу по ID
	r.Put("/{id}", h.update)     // PUT /api/v1/tasks/{id} - обновить задачу
	r.Delete("/{id}", h.delete)  // DELETE /api/v1/tasks/{id} - удалить задачу
	
	return r
}

// list обрабатывает GET запрос для получения списка задач с пагинацией и фильтрацией
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	// Парсим параметры пагинации (значения по умолчанию: page=1, limit=10)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1  // Страница по умолчанию
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 10  // Лимит по умолчанию
	} else if limit > 100 {
		limit = 100  // Ограничиваем максимальный лимит для защиты от перегрузки
	}

	// Парсим фильтр по статусу выполнения
	var doneFilter *bool
	if doneStr := r.URL.Query().Get("done"); doneStr != "" {
		done, err := strconv.ParseBool(doneStr)
		if err == nil {
			doneFilter = &done
		}
	}

	// Получаем задачи с пагинацией и фильтрацией - ТЕПЕРЬ ПРОСТО МАССИВ ЗАДАЧ
	tasks := h.repo.ListWithPagination(page, limit, doneFilter)
	
	// Если нет задач для указанной страницы, возвращаем пустой массив
	writeJSON(w, http.StatusOK, tasks)
}

// get обрабатывает GET запрос для получения задачи по ID
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, bad := parseID(w, r)
	if bad {
		return
	}

	t, err := h.repo.Get(id)
	if err != nil {
		httpError(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// createReq структура для запроса создания задачи
type createReq struct {
	Title string `json:"title"`
}

// create обрабатывает POST запрос для создания новой задачи
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req createReq
	
	// Декодируем JSON тело запроса
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Создаем задачу в хранилище
	t, err := h.repo.Create(req.Title)
	if err != nil {
		// Обрабатываем ошибки валидации
		if err == ErrInvalidTitle {
			httpError(w, http.StatusBadRequest, err.Error())
		} else {
			httpError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

// updateReq структура для запроса обновления задачи
type updateReq struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// update обрабатывает PUT запрос для обновления задачи
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, bad := parseID(w, r)
	if bad {
		return
	}

	var req updateReq
	
	// Декодируем JSON тело запроса
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "invalid json")
		return
	}

	// Обновляем задачу в хранилище
	t, err := h.repo.Update(id, req.Title, req.Done)
	if err != nil {
		// Обрабатываем различные типы ошибок
		if err == ErrNotFound {
			httpError(w, http.StatusNotFound, err.Error())
		} else if err == ErrInvalidTitle {
			httpError(w, http.StatusBadRequest, err.Error())
		} else {
			httpError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, t)
}

// delete обрабатывает DELETE запрос для удаления задачи
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, bad := parseID(w, r)
	if bad {
		return
	}

	if err := h.repo.Delete(id); err != nil {
		httpError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)  // 204 No Content - успешное удаление
}

// Вспомогательные функции

// parseID извлекает и валидирует ID задачи из URL параметров
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		httpError(w, http.StatusBadRequest, "invalid id: must be positive integer")
		return 0, true
	}
	return id, false
}

// writeJSON отправляет JSON ответ с указанным HTTP статусом
func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)  // Игнорируем ошибку кодирования
}

// httpError отправляет JSON ответ с ошибкой
func httpError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}