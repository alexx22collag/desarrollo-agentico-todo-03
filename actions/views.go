package actions

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"todo/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// viewNone identifica la vista de las tareas sin módulo.
const viewNone = "none"

// parseView interpreta el valor de la vista del listado: "" es «Todas», "none" es
// «Sin módulo» y un UUID es un módulo. Cualquier otro valor no es válido.
func parseView(raw string) (string, uuid.UUID, bool) {
	switch raw {
	case "":
		return "", uuid.Nil, true
	case viewNone:
		return viewNone, uuid.Nil, true
	}
	id, err := uuid.FromString(raw)
	if err != nil || id == uuid.Nil {
		return "", uuid.Nil, false
	}
	return id.String(), id, true
}

// listPath construye la URL del listado para una vista. La URL se reconstruye
// siempre desde constantes o un UUID parseado, nunca desde la entrada recibida.
func listPath(view string) string {
	view, id, ok := parseView(view)
	switch {
	case !ok || view == "":
		return "/"
	case view == viewNone:
		return "/?module=" + viewNone
	}
	return "/?module=" + id.String()
}

// taskViewAfterWrite decide a qué vista volver tras crear o actualizar una tarea:
// desde «Todas» se vuelve a «Todas»; desde un módulo o «Sin módulo», a la vista
// del módulo de la tarea para que siga visible.
func taskViewAfterWrite(view string, task *models.Task) string {
	view, _, ok := parseView(view)
	if !ok || view == "" {
		return ""
	}
	if v := task.ModuleValue(); v != "" {
		return v
	}
	return viewNone
}

// parseModuleParam interpreta el módulo elegido en un formulario: vacío es «Sin
// módulo» y un UUID es un módulo. Cualquier otro valor no es válido.
func parseModuleParam(raw string) (uuid.NullUUID, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.NullUUID{}, true
	}
	id, err := uuid.FromString(raw)
	if err != nil {
		return uuid.NullUUID{}, false
	}
	return uuid.NullUUID{UUID: id, Valid: true}, true
}

// sidebarItem es una entrada de la barra lateral.
type sidebarItem struct {
	Label   string
	Href    string
	Count   int
	Current bool
}

// selectOption es una opción de un selector.
type selectOption struct {
	Value string
	Label string
}

// setLayout prepara los datos de la barra lateral que necesita el layout.
func setLayout(c buffalo.Context, tx *pop.Connection, view string) error {
	view, _, ok := parseView(view)
	if !ok {
		view = ""
	}

	total, err := tx.Count(&models.Task{})
	if err != nil {
		return err
	}
	none, err := tx.Where("module_id IS NULL").Count(&models.Task{})
	if err != nil {
		return err
	}
	counts, err := models.ModuleCounts(tx)
	if err != nil {
		return err
	}

	items := []sidebarItem{{Label: "Todas", Href: "/", Count: total, Current: view == ""}}
	options := []selectOption{}
	for _, m := range counts {
		id := m.ID.String()
		items = append(items, sidebarItem{Label: m.Name, Href: listPath(id), Count: m.TaskCount, Current: view == id})
		options = append(options, selectOption{Value: id, Label: m.Name})
	}
	if none > 0 || view == viewNone {
		items = append(items, sidebarItem{Label: "Sin módulo", Href: listPath(viewNone), Count: none, Current: view == viewNone})
	}

	c.Set("sidebarItems", items)
	c.Set("moduleOptions", options)
	c.Set("view", view)
	if c.Value("moduleErrors") == nil {
		c.Set("moduleErrors", validate.NewErrors())
	}
	if c.Value("moduleName") == nil {
		c.Set("moduleName", "")
	}
	return nil
}

// resolveView comprueba la vista y devuelve su título y el módulo que filtra.
// found es false si la vista no es válida o el módulo no existe.
func resolveView(tx *pop.Connection, raw string) (view, title string, moduleID uuid.UUID, found bool, err error) {
	view, moduleID, ok := parseView(raw)
	if !ok {
		return "", "", uuid.Nil, false, nil
	}
	switch view {
	case "":
		return view, "Tareas", moduleID, true, nil
	case viewNone:
		return view, "Sin módulo", moduleID, true, nil
	}
	module := &models.Module{}
	if err := tx.Find(module, moduleID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", uuid.Nil, false, nil
		}
		return "", "", uuid.Nil, false, err
	}
	return view, module.Name, moduleID, true, nil
}

// viewOrAll devuelve la vista normalizada si existe y «Todas» ("") si no es
// válida o apunta a un módulo inexistente. Solo propaga errores de la base de datos.
// Úsala en los caminos que reciben la vista oculta de un formulario; el listado
// (TasksIndex) sigue respondiendo 404 con un módulo inexistente.
func viewOrAll(tx *pop.Connection, raw string) (string, error) {
	view, _, _, found, err := resolveView(tx, raw)
	if err != nil {
		return "", err
	}
	if !found {
		return "", nil
	}
	return view, nil
}

// newTaskForView devuelve una tarea vacía para el formulario de alta, con el
// módulo de la vista preseleccionado (ninguno en «Todas» y «Sin módulo»).
func newTaskForView(view string) *models.Task {
	task := &models.Task{Priority: models.PriorityMedium}
	if _, id, ok := parseView(view); ok && id != uuid.Nil {
		task.ModuleID = uuid.NullUUID{UUID: id, Valid: true}
	}
	return task
}

// renderTaskList muestra el listado filtrado por la vista, en el orden de prioridad.
func renderTaskList(c buffalo.Context, tx *pop.Connection, view string, task *models.Task, errs *validate.Errors, status int) error {
	view, title, moduleID, found, err := resolveView(tx, view)
	if err != nil {
		return err
	}
	if !found {
		return c.Error(http.StatusNotFound, fmt.Errorf("módulo no encontrado"))
	}

	q := tx.Q()
	switch {
	case view == viewNone:
		q = tx.Where("module_id IS NULL")
	case view != "":
		q = tx.Where("module_id = ?", moduleID)
	}
	tasks := models.Tasks{}
	if err := q.Order(models.TaskListOrder).All(&tasks); err != nil {
		return err
	}

	if err := setLayout(c, tx, view); err != nil {
		return err
	}
	c.Set("viewTitle", title)
	c.Set("tasks", tasks)
	c.Set("task", task)
	c.Set("errors", errs)
	return c.Render(status, r.HTML("tasks/index.plush.html"))
}
