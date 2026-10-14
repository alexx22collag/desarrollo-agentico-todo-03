package actions

import (
	"fmt"
	"net/http"
	"strings"

	"todo/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
)

// TasksIndex muestra la lista de tareas, filtrada por módulo con ?module=<id>|none,
// y el formulario de creación.
func TasksIndex(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	view, _, ok := parseView(c.Param("module"))
	if !ok {
		return c.Error(http.StatusNotFound, fmt.Errorf("módulo no válido"))
	}

	return renderTaskList(c, tx, view, newTaskForView(view), validate.NewErrors(), http.StatusOK)
}

// TasksCreate crea una tarea nueva a partir del formulario.
func TasksCreate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := c.Bind(task); err != nil {
		return err
	}
	task.Title = strings.TrimSpace(task.Title)

	view, _, ok := parseView(c.Param("view"))
	if !ok {
		view = ""
	}

	moduleID, ok := parseModuleParam(c.Param("module_id"))
	if !ok {
		verrs := validate.NewErrors()
		verrs.Add("module_id", models.ModuleInvalidMessage)
		return renderTaskList(c, tx, view, task, verrs, http.StatusUnprocessableEntity)
	}
	task.ModuleID = moduleID

	verrs, err := tx.ValidateAndCreate(task)
	if err != nil {
		return err
	}

	if verrs.HasAny() {
		return renderTaskList(c, tx, view, task, verrs, http.StatusUnprocessableEntity)
	}

	c.Flash().Add("success", "Tarea creada.")
	return c.Redirect(http.StatusSeeOther, listPath(taskViewAfterWrite(view, task)))
}

// TasksEdit muestra el formulario de edición de una tarea existente.
func TasksEdit(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	view, _, ok := parseView(c.Param("view"))
	if !ok {
		view = ""
	}

	if err := setLayout(c, tx, view); err != nil {
		return err
	}
	c.Set("backPath", listPath(view))
	c.Set("task", task)
	c.Set("errors", validate.NewErrors())
	return c.Render(http.StatusOK, r.HTML("tasks/edit.plush.html"))
}

// TasksUpdate actualiza el título, la prioridad y el módulo, conservando los demás
// datos de la tarea. Si no se envía prioridad o módulo, se mantiene el actual.
func TasksUpdate(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	input := struct {
		Title    string `form:"title" json:"title"`
		Priority string `form:"priority" json:"priority"`
		ModuleID string `form:"module_id" json:"module_id"`
		View     string `form:"view" json:"view"`
	}{}
	if err := c.Bind(&input); err != nil {
		return err
	}
	view, _, ok := parseView(input.View)
	if !ok {
		view = ""
	}

	task.Title = strings.TrimSpace(input.Title)
	if p := strings.TrimSpace(input.Priority); p != "" {
		task.Priority = p
	}

	_ = c.Request().ParseForm()
	if _, sent := c.Request().Form["module_id"]; sent {
		moduleID, ok := parseModuleParam(input.ModuleID)
		if !ok {
			verrs := validate.NewErrors()
			verrs.Add("module_id", models.ModuleInvalidMessage)
			return renderTaskEdit(c, tx, view, task, verrs)
		}
		task.ModuleID = moduleID
	}

	verrs, err := tx.ValidateAndUpdate(task)
	if err != nil {
		return err
	}
	if verrs.HasAny() {
		return renderTaskEdit(c, tx, view, task, verrs)
	}

	c.Flash().Add("success", "Tarea actualizada.")
	return c.Redirect(http.StatusSeeOther, listPath(taskViewAfterWrite(view, task)))
}

// renderTaskEdit vuelve a mostrar el formulario de edición con errores (422).
func renderTaskEdit(c buffalo.Context, tx *pop.Connection, view string, task *models.Task, verrs *validate.Errors) error {
	if err := setLayout(c, tx, view); err != nil {
		return err
	}
	c.Set("backPath", listPath(view))
	c.Set("task", task)
	c.Set("errors", verrs)
	return c.Render(http.StatusUnprocessableEntity, r.HTML("tasks/edit.plush.html"))
}

// TasksToggle alterna una tarea entre completada y pendiente.
func TasksToggle(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	task.Completed = !task.Completed
	if err := tx.Update(task); err != nil {
		return err
	}

	return c.Redirect(http.StatusSeeOther, listPath(c.Param("view")))
}

// TasksDestroy elimina una tarea.
func TasksDestroy(c buffalo.Context) error {
	tx := c.Value("tx").(*pop.Connection)

	task := &models.Task{}
	if err := tx.Find(task, c.Param("task_id")); err != nil {
		return c.Error(http.StatusNotFound, err)
	}

	if err := tx.Destroy(task); err != nil {
		return err
	}

	c.Flash().Add("success", "Tarea eliminada.")
	return c.Redirect(http.StatusSeeOther, listPath(c.Param("view")))
}
