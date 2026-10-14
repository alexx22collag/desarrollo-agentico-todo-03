package actions

import (
	"net/url"
	"strings"
	"time"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_TasksIndex_EmptyState() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksIndex_ListsTasks() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "Completada")
}

func (as *ActionSuite) Test_TasksCreate_Valid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "  Nueva tarea de prueba  "})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Nueva tarea de prueba", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksCreate_EmptyTitle_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: ""})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_OnlySpaces_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "     "})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksToggle_CompletesAndReopens() {
	task := &models.Task{Title: "Pendiente de completar"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s/toggle", id).Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/toggle").Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksDestroy() {
	task := &models.Task{Title: "Tarea que se elimina"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Delete()
	as.Equal(303, res.Code)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksDestroy_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s", id).Delete()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskForEditing(completed bool) *models.Task {
	task := &models.Task{Title: "Título original", Completed: completed}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksIndex_EditLink() {
	task := as.taskForEditing(false)
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "/tasks/"+task.ID.String()+"/edit")
	as.Contains(res.Body.String(), "Editar tarea: Título original")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsTitle() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="Título original"`)
	as.Contains(res.Body.String(), `value="PUT"`)
	as.Contains(res.Body.String(), "Guardar cambios")
	as.Contains(res.Body.String(), "Cancelar")
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksEdit_NotFound() {
	res := as.HTML("/tasks/%s/edit", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/edit").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_EscapesTitle() {
	task := &models.Task{Title: `<img src=x onerror="alert(1)">`}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `<img src=x`)
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsTitleOnly() {
	task := as.taskForEditing(true)
	id, createdAt, updatedAt := task.ID, task.CreatedAt, task.UpdatedAt
	res := as.HTML("/tasks/%s", id).Put(url.Values{
		"title": {"  Título actualizado  "}, "completed": {"false"},
		"id": {uuid.Must(uuid.NewV4()).String()}, "created_at": {"2000-01-01T00:00:00Z"},
	})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.Equal("Título actualizado", task.Title)
	as.Equal(id, task.ID)
	as.True(task.Completed)
	as.True(task.CreatedAt.Equal(createdAt))
	as.False(task.UpdatedAt.Before(updatedAt))
	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_TasksUpdate_PendingTaskRemainsPending() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo título"}, "completed": {"true"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo título", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_EmptyTitle() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {""}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), `aria-describedby="title-error"`)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_OnlySpaces() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_TitleTooLong() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título no puede superar los 200 caracteres.")
	as.Contains(res.Body.String(), title)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_TitleAtLimit() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(title, task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_NotFound() {
	res := as.HTML("/tasks/%s", uuid.Must(uuid.NewV4())).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid").Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) createTask(title, priority string, completed bool, createdAt time.Time) *models.Task {
	task := &models.Task{Title: title, Priority: priority, Completed: completed, CreatedAt: createdAt}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	return task
}

func (as *ActionSuite) Test_TasksIndex_ShowsPriority() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "task-priority-alta")
	as.Contains(body, "task-priority-baja")
	as.Contains(body, "task-priority-media")
	as.Contains(body, "Alta")
	as.Contains(body, "Baja")
	as.Contains(body, "Media")
	as.Contains(body, "Prioridad: ")
}

func (as *ActionSuite) Test_TasksIndex_FormDefaultsToMedium() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `name="priority"`)
	as.Contains(res.Body.String(), `<option value="media" selected>`)
}

func (as *ActionSuite) Test_TasksCreate_WithPriority() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Con prioridad"}, "priority": {"alta"}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal(models.PriorityHigh, task.Priority)
}

func (as *ActionSuite) Test_TasksCreate_WithoutPriority_DefaultsToMedium() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"Sin prioridad"}})
	as.Equal(303, res.Code)

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal(models.PriorityMedium, task.Priority)
}

func (as *ActionSuite) Test_TasksCreate_InvalidPriority() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"X"}, "priority": {"urgente"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "La prioridad no es válida.")
	as.NotContains(res.Body.String(), `aria-describedby="title-error"`)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksEdit_PreselectsPriority() {
	task := as.createTask("Para editar", models.PriorityLow, false, time.Time{})
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `<option value="baja" selected>`)
}

func (as *ActionSuite) Test_TasksUpdate_ChangesPriority() {
	task := as.createTask("Igual", models.PriorityMedium, true, time.Time{})
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Igual"}, "priority": {"alta"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(models.PriorityHigh, task.Priority)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_WithoutPriority_KeepsIt() {
	task := as.createTask("Conserva prioridad", models.PriorityLow, false, time.Time{})
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título nuevo", task.Title)
	as.Equal(models.PriorityLow, task.Priority)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidPriority() {
	task := as.createTask("Original", models.PriorityMedium, false, time.Time{})
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo"}, "priority": {"urgente"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "La prioridad no es válida.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Original", task.Title)
	as.Equal(models.PriorityMedium, task.Priority)
}

func (as *ActionSuite) Test_TasksToggle_KeepsPriority() {
	task := as.createTask("Importante", models.PriorityHigh, false, time.Time{})
	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
	as.Equal(models.PriorityHigh, task.Priority)
}

func (as *ActionSuite) Test_TasksIndex_OrdersByPriority() {
	base := time.Now().Add(-time.Hour)
	as.createTask("Tarea baja", models.PriorityLow, false, base)
	as.createTask("Tarea alta antigua", models.PriorityHigh, false, base.Add(1*time.Second))
	as.createTask("Tarea media", models.PriorityMedium, false, base.Add(2*time.Second))
	as.createTask("Tarea alta reciente", models.PriorityHigh, true, base.Add(3*time.Second))

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	highOld := strings.Index(body, "Tarea alta antigua")
	highNew := strings.Index(body, "Tarea alta reciente")
	medium := strings.Index(body, "Tarea media")
	low := strings.Index(body, "Tarea baja")
	as.True(highOld >= 0 && highNew >= 0 && medium >= 0 && low >= 0)
	as.Less(highOld, highNew)
	as.Less(highNew, medium)
	as.Less(medium, low)
}

func (as *ActionSuite) Test_TasksCreate_Invalid_KeepsPriorityOrder() {
	base := time.Now().Add(-time.Hour)
	as.createTask("Tarea baja", models.PriorityLow, false, base)
	as.createTask("Tarea alta", models.PriorityHigh, false, base.Add(time.Second))

	res := as.HTML("/tasks").Post(url.Values{"title": {""}})
	as.Equal(422, res.Code)

	body := res.Body.String()
	as.Less(strings.Index(body, "Tarea alta"), strings.Index(body, "Tarea baja"))
}

// El toggle optimista de app.js envía el formulario por POST con _method=PUT.
func (as *ActionSuite) Test_TasksToggle_FormPostWithMethodOverride() {
	task := &models.Task{Title: "Marcar desde el navegador"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Post(url.Values{"_method": {"PUT"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Post(url.Values{"_method": {"PUT"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

// Un GET (lo que hacía el fetch sin opciones) no es una ruta válida y no persiste nada.
func (as *ActionSuite) Test_TasksToggle_GetIsNotARoute() {
	task := &models.Task{Title: "No se marca por GET"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Get()
	as.Equal(404, res.Code)
	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_AppJS_ToggleSubmitsFormAsPost() {
	res := as.HTML("/assets/app.js").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, `method: "POST"`)
	as.Contains(body, "new FormData(form)")
}

func (as *ActionSuite) Test_TasksIndex_ToggleFormCarriesOverrideAndToken() {
	task := &models.Task{Title: "Con formulario de toggle"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "data-task-toggle")
	as.Contains(body, `name="_method" value="PUT"`)
	as.Contains(body, `name="authenticity_token"`)
}

func (as *ActionSuite) Test_TasksIndex_FiltersByModule_KeepsPriorityOrder() {
	base := time.Now().Add(-time.Hour)
	work := as.createModule("Trabajo", base)
	personal := as.createModule("Personal", base.Add(time.Second))
	as.createTaskIn("T baja", models.PriorityLow, inModule(work), base)
	as.createTaskIn("T alta", models.PriorityHigh, inModule(work), base.Add(time.Second))
	as.createTaskIn("T media", models.PriorityMedium, inModule(work), base.Add(2*time.Second))
	as.createTaskIn("P alta", models.PriorityHigh, inModule(personal), base.Add(3*time.Second))

	res := as.HTML("/?module=%s", work.ID).Get()
	as.Equal(200, res.Code)
	body := res.Body.String()

	high := strings.Index(body, "T alta")
	medium := strings.Index(body, "T media")
	low := strings.Index(body, "T baja")
	as.True(high >= 0 && medium >= 0 && low >= 0)
	as.Less(high, medium)
	as.Less(medium, low)
	as.NotContains(body, "P alta")
	as.Contains(body, "<h1>Trabajo</h1>")
	as.Contains(body, `href="/?module=`+work.ID.String()+`" class="nav-link" aria-current="page"`)
	as.Contains(body, `<option value="`+work.ID.String()+`" selected>`)
}

func (as *ActionSuite) Test_TasksIndex_FilterNone() {
	work := as.createModule("Trabajo", time.Time{})
	as.createTaskIn("Con módulo", models.PriorityMedium, inModule(work), time.Time{})
	as.createTaskIn("Sin asignar", models.PriorityMedium, uuid.NullUUID{}, time.Time{})

	res := as.HTML("/?module=none").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	as.Contains(body, "Sin asignar")
	as.NotContains(body, "Con módulo")
	as.Contains(body, "<h1>Sin módulo</h1>")
}

func (as *ActionSuite) Test_TasksIndex_UnknownModule_NotFound() {
	res := as.HTML("/?module=%s", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksIndex_InvalidModule_NotFound() {
	res := as.HTML("/?module=abc").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksIndex_AllView_PreselectsNoModule() {
	as.createModule("Trabajo", time.Time{})

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `<option value="" selected>Sin módulo</option>`)
}

func (as *ActionSuite) Test_TasksCreate_WithModule() {
	work := as.createModule("Trabajo", time.Time{})

	res := as.HTML("/tasks").Post(url.Values{"title": {"X"}, "module_id": {work.ID.String()}, "view": {work.ID.String()}})
	as.Equal(303, res.Code)
	as.Equal("/?module="+work.ID.String(), res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.True(task.ModuleID.Valid)
	as.Equal(work.ID, task.ModuleID.UUID)
}

func (as *ActionSuite) Test_TasksCreate_FromAllView_StaysInAll() {
	work := as.createModule("Trabajo", time.Time{})

	res := as.HTML("/tasks").Post(url.Values{"title": {"X"}, "module_id": {work.ID.String()}, "view": {""}})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
}

func (as *ActionSuite) Test_TasksCreate_FromModuleView_OtherModule() {
	base := time.Now().Add(-time.Hour)
	a := as.createModule("A", base)
	b := as.createModule("B", base.Add(time.Second))

	res := as.HTML("/tasks").Post(url.Values{"title": {"X"}, "module_id": {b.ID.String()}, "view": {a.ID.String()}})
	as.Equal(303, res.Code)
	as.Equal("/?module="+b.ID.String(), res.Header().Get("Location"))
}

func (as *ActionSuite) Test_TasksCreate_UnknownModule() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"X"}, "module_id": {uuid.Must(uuid.NewV4()).String()}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El módulo no es válido.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_MalformedModule() {
	res := as.HTML("/tasks").Post(url.Values{"title": {"X"}, "module_id": {"abc"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El módulo no es válido.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksEdit_PreselectsModule() {
	work := as.createModule("Trabajo", time.Time{})
	task := as.createTaskIn("En trabajo", models.PriorityMedium, inModule(work), time.Time{})

	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `<option value="`+work.ID.String()+`" selected>`)

	res = as.HTML("/tasks/%s/edit?view=%s", task.ID, work.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `<a href="/?module=`+work.ID.String()+`" class="button">Cancelar</a>`)
}

func (as *ActionSuite) Test_TasksUpdate_ChangesModule() {
	base := time.Now().Add(-time.Hour)
	a := as.createModule("A", base)
	b := as.createModule("B", base.Add(time.Second))
	task := as.createTaskIn("X", models.PriorityHigh, inModule(a), time.Time{})
	task.Completed = true
	as.NoError(as.DB.Update(task))

	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"X"}, "module_id": {b.ID.String()}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(b.ID, task.ModuleID.UUID)
	as.Equal(models.PriorityHigh, task.Priority)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_ClearsModule() {
	work := as.createModule("Trabajo", time.Time{})
	task := as.createTaskIn("X", models.PriorityMedium, inModule(work), time.Time{})

	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"X"}, "module_id": {""}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.False(task.ModuleID.Valid)
}

func (as *ActionSuite) Test_TasksUpdate_WithoutModule_KeepsIt() {
	work := as.createModule("Trabajo", time.Time{})
	task := as.createTaskIn("X", models.PriorityMedium, inModule(work), time.Time{})

	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo", task.Title)
	as.True(task.ModuleID.Valid)
	as.Equal(work.ID, task.ModuleID.UUID)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidModule() {
	work := as.createModule("Trabajo", time.Time{})
	task := as.createTaskIn("Original", models.PriorityMedium, inModule(work), time.Time{})

	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo"}, "module_id": {"abc"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El módulo no es válido.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Original", task.Title)
	as.Equal(work.ID, task.ModuleID.UUID)
}

func (as *ActionSuite) Test_TasksToggle_ReturnsToView() {
	work := as.createModule("Trabajo", time.Time{})
	task := as.createTaskIn("X", models.PriorityMedium, inModule(work), time.Time{})

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{"view": {work.ID.String()}})
	as.Equal(303, res.Code)
	as.Equal("/?module="+work.ID.String(), res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
	as.Equal(work.ID, task.ModuleID.UUID)
}

func (as *ActionSuite) Test_TasksToggle_FormPost_ReturnsToView() {
	work := as.createModule("Trabajo", time.Time{})
	task := as.createTaskIn("X", models.PriorityHigh, inModule(work), time.Time{})

	res := as.HTML("/tasks/%s/toggle", task.ID).Post(url.Values{"_method": {"PUT"}, "view": {work.ID.String()}})
	as.Equal(303, res.Code)
	as.Equal("/?module="+work.ID.String(), res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
	as.Equal(work.ID, task.ModuleID.UUID)
	as.Equal(models.PriorityHigh, task.Priority)
}

func (as *ActionSuite) Test_TasksDestroy_ReturnsToView() {
	task := as.createTaskIn("X", models.PriorityMedium, uuid.NullUUID{}, time.Time{})

	res := as.HTML("/tasks/%s", task.ID).Post(url.Values{"_method": {"DELETE"}, "view": {"none"}})
	as.Equal(303, res.Code)
	as.Equal("/?module=none", res.Header().Get("Location"))

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_ListPath_RejectsArbitraryValues() {
	id := uuid.Must(uuid.NewV4())
	as.Equal("/", listPath(""))
	as.Equal("/?module=none", listPath("none"))
	as.Equal("/?module="+id.String(), listPath(id.String()))
	for _, v := range []string{"//evil.example", "/?module=x", "https://x"} {
		as.Equal("/", listPath(v))
	}
}

func (as *ActionSuite) Test_TasksIndex_ToggleAndDeleteFormsCarryView() {
	work := as.createModule("Trabajo", time.Time{})
	task := as.createTaskIn("X", models.PriorityMedium, inModule(work), time.Time{})

	res := as.HTML("/?module=%s", work.ID).Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	as.GreaterOrEqual(strings.Count(body, `name="view" value="`+work.ID.String()+`"`), 3)
	as.Contains(body, "/tasks/"+task.ID.String()+"/edit?view="+work.ID.String())
}
