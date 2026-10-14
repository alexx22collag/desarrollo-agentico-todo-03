package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// ModuleNameMaxLength es el número máximo de caracteres permitido en el nombre de un módulo.
const ModuleNameMaxLength = 50

// ModuleListOrder ordena los módulos por fecha de creación.
const ModuleListOrder = "created_at asc, id asc"

// ModuleInvalidMessage es el error que se muestra cuando el módulo de una tarea no es válido.
const ModuleInvalidMessage = "El módulo no es válido."

// Module agrupa tareas en una lista con nombre creada por el usuario.
type Module struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Name      string    `json:"name" db:"name" form:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Modules es una colección de Module.
type Modules []Module

// BeforeValidate se ejecuta antes de los métodos "pop.Validate*".
func (m *Module) BeforeValidate(tx *pop.Connection) error {
	m.Name = strings.TrimSpace(m.Name)
	return nil
}

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (m *Module) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	name := strings.TrimSpace(m.Name)
	if name == "" {
		verrs.Add("name", "El nombre del módulo es obligatorio.")
		return verrs, nil
	}
	if len([]rune(name)) > ModuleNameMaxLength {
		verrs.Add("name", fmt.Sprintf("El nombre del módulo no puede superar los %d caracteres.", ModuleNameMaxLength))
		return verrs, nil
	}

	exists, err := tx.Where("lower(name) = lower(?) AND id <> ?", name, m.ID).Exists(&Module{})
	if err != nil {
		return verrs, err
	}
	if exists {
		verrs.Add("name", "Ya existe un módulo con ese nombre.")
	}

	return verrs, nil
}

// ModuleCount es un módulo con el número total de tareas que contiene.
type ModuleCount struct {
	ID        uuid.UUID `db:"id"`
	Name      string    `db:"name"`
	TaskCount int       `db:"task_count"`
}

// ModuleCounts devuelve los módulos en orden de creación con su número de tareas
// (completadas incluidas).
func ModuleCounts(tx *pop.Connection) ([]ModuleCount, error) {
	counts := []ModuleCount{}
	err := tx.RawQuery("SELECT m.id, m.name, COUNT(t.id) AS task_count FROM modules m LEFT JOIN tasks t ON t.module_id = m.id GROUP BY m.id, m.name, m.created_at ORDER BY m.created_at asc, m.id asc").All(&counts)
	return counts, err
}
