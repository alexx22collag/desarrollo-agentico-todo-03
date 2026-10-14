package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// Niveles de prioridad admitidos para una tarea.
const (
	PriorityHigh   = "alta"
	PriorityMedium = "media"
	PriorityLow    = "baja"
)

// TaskListOrder ordena por prioridad (alta, media, baja) y, dentro de cada nivel, por antigüedad.
const TaskListOrder = "CASE priority WHEN 'alta' THEN 0 WHEN 'media' THEN 1 ELSE 2 END, created_at asc, id asc"

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Priority  string    `json:"priority" db:"priority" form:"priority"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// PriorityLabel devuelve el nombre visible de la prioridad.
func (t Task) PriorityLabel() string {
	switch t.Priority {
	case PriorityHigh:
		return "Alta"
	case PriorityMedium:
		return "Media"
	case PriorityLow:
		return "Baja"
	}
	return t.Priority
}

// applyDefaultPriority asigna la prioridad media cuando no se indica ninguna.
func (t *Task) applyDefaultPriority() {
	if strings.TrimSpace(t.Priority) == "" {
		t.Priority = PriorityMedium
		return
	}
	t.Priority = strings.TrimSpace(t.Priority)
}

// BeforeValidate se ejecuta antes de los métodos "pop.Validate*".
func (t *Task) BeforeValidate(tx *pop.Connection) error {
	t.applyDefaultPriority()
	return nil
}

// BeforeSave se ejecuta antes de cada Create o Update, aunque no se valide.
func (t *Task) BeforeSave(tx *pop.Connection) error {
	t.applyDefaultPriority()
	return nil
}

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	switch t.Priority {
	case PriorityHigh, PriorityMedium, PriorityLow:
	default:
		verrs.Add("priority", "La prioridad no es válida.")
	}

	return verrs, nil
}
