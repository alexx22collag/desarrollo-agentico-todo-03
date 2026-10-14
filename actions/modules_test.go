package actions

import (
	"net/url"
	"strings"
	"time"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) createModule(name string, createdAt time.Time) *models.Module {
	module := &models.Module{Name: name, CreatedAt: createdAt}
	verrs, err := as.DB.ValidateAndCreate(module)
	as.NoError(err)
	as.False(verrs.HasAny())
	return module
}

func (as *ActionSuite) createTaskIn(title, priority string, moduleID uuid.NullUUID, createdAt time.Time) *models.Task {
	task := &models.Task{Title: title, Priority: priority, ModuleID: moduleID, CreatedAt: createdAt}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	return task
}

func inModule(m *models.Module) uuid.NullUUID {
	return uuid.NullUUID{UUID: m.ID, Valid: true}
}

func (as *ActionSuite) Test_Sidebar_ListsModulesWithCounts() {
	base := time.Now().Add(-time.Hour)
	work := as.createModule("Trabajo", base)
	personal := as.createModule("Personal", base.Add(time.Second))
	as.createTaskIn("T1", models.PriorityMedium, inModule(work), time.Time{})
	done := as.createTaskIn("T2", models.PriorityMedium, inModule(work), time.Time{})
	done.Completed = true
	as.NoError(as.DB.Update(done))
	as.createTaskIn("P1", models.PriorityMedium, inModule(personal), time.Time{})
	as.createTaskIn("Suelta", models.PriorityMedium, uuid.NullUUID{}, time.Time{})

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()

	workIdx := strings.Index(body, `href="/?module=`+work.ID.String()+`"`)
	personalIdx := strings.Index(body, `href="/?module=`+personal.ID.String()+`"`)
	as.True(workIdx >= 0 && personalIdx >= 0)
	as.Less(workIdx, personalIdx)

	as.Contains(body, `nav-label">Todas</span><span class="nav-count">4<`)
	as.Contains(body, `nav-label">Trabajo</span><span class="nav-count">2<`)
	as.Contains(body, `nav-label">Personal</span><span class="nav-count">1<`)
	as.Contains(body, `href="/?module=none"`)
	as.Contains(body, `nav-label">Sin módulo</span><span class="nav-count">1<`)
	as.Contains(body, "Nuevo módulo")
	as.Contains(body, `action="/modules"`)
}

func (as *ActionSuite) Test_Sidebar_HidesNoModuleWhenEmpty() {
	work := as.createModule("Trabajo", time.Time{})
	as.createTaskIn("T1", models.PriorityMedium, inModule(work), time.Time{})

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), "/?module=none")
}

func (as *ActionSuite) Test_Sidebar_TodasIsCurrentByDefault() {
	as.createModule("Trabajo", time.Time{})

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	body := res.Body.String()
	as.Contains(body, `href="/" class="nav-link" aria-current="page"`)
	as.Equal(1, strings.Count(body, "aria-current"))
}

func (as *ActionSuite) Test_ModulesCreate_Valid() {
	res := as.HTML("/modules").Post(url.Values{"name": {"  Trabajo  "}})
	as.Equal(303, res.Code)

	module := &models.Module{}
	as.NoError(as.DB.First(module))
	as.Equal("Trabajo", module.Name)
	as.Equal("/?module="+module.ID.String(), res.Header().Get("Location"))
}

func (as *ActionSuite) Test_ModulesCreate_EmptyName() {
	res := as.HTML("/modules").Post(url.Values{"name": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El nombre del módulo es obligatorio.")
	as.Contains(res.Body.String(), `<details class="new-module" open`)

	count, err := as.DB.Count(&models.Module{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_ModulesCreate_Duplicate() {
	as.createModule("Trabajo", time.Time{})

	res := as.HTML("/modules").Post(url.Values{"name": {"TRABAJO"}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "Ya existe un módulo con ese nombre.")

	count, err := as.DB.Count(&models.Module{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_ModulesCreate_TooLong() {
	res := as.HTML("/modules").Post(url.Values{"name": {strings.Repeat("á", models.ModuleNameMaxLength+1)}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El nombre del módulo no puede superar los 50 caracteres.")

	count, err := as.DB.Count(&models.Module{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_ModulesCreate_EscapesName() {
	res := as.HTML("/modules").Post(url.Values{"name": {"<img src=x onerror=alert(1)>"}})
	as.Equal(303, res.Code)

	res = as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), "<img src=x")
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_ModulesCreate_Invalid_KeepsView() {
	work := as.createModule("Trabajo", time.Time{})

	res := as.HTML("/modules").Post(url.Values{"name": {""}, "view": {work.ID.String()}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "<h1>Trabajo</h1>")
}
