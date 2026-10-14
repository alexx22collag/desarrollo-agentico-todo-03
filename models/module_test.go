package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

func (ms *ModelSuite) Test_Module_Create_Valid() {
	module := &Module{Name: "  Trabajo  "}

	verrs, err := ms.DB.ValidateAndCreate(module)
	ms.NoError(err)
	ms.False(verrs.HasAny())
	ms.Equal("Trabajo", module.Name)
	ms.NotZero(module.ID)
}

func (ms *ModelSuite) Test_Module_Name_Required() {
	for _, name := range []string{"", "   "} {
		verrs, err := ms.DB.ValidateAndCreate(&Module{Name: name})
		ms.NoError(err)
		ms.Contains(verrs.Get("name"), "El nombre del módulo es obligatorio.")
	}

	count, err := ms.DB.Count(&Module{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Module_Name_MaxLength() {
	verrs, err := ms.DB.ValidateAndCreate(&Module{Name: strings.Repeat("á", ModuleNameMaxLength)})
	ms.NoError(err)
	ms.False(verrs.HasAny())

	verrs, err = ms.DB.ValidateAndCreate(&Module{Name: strings.Repeat("á", ModuleNameMaxLength+1)})
	ms.NoError(err)
	ms.Contains(verrs.Get("name"), "El nombre del módulo no puede superar los 50 caracteres.")
}

func (ms *ModelSuite) Test_Module_Name_UniqueCaseInsensitive() {
	verrs, err := ms.DB.ValidateAndCreate(&Module{Name: "Trabajo"})
	ms.NoError(err)
	ms.False(verrs.HasAny())

	verrs, err = ms.DB.ValidateAndCreate(&Module{Name: " trabajo "})
	ms.NoError(err)
	ms.Contains(verrs.Get("name"), "Ya existe un módulo con ese nombre.")

	count, err := ms.DB.Count(&Module{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Module_Name_UniqueIndex() {
	verrs, err := ms.DB.ValidateAndCreate(&Module{Name: "Trabajo"})
	ms.NoError(err)
	ms.False(verrs.HasAny())

	err = ms.DB.Create(&Module{Name: "TRABAJO"})
	ms.Error(err)
	ms.Contains(err.Error(), "modules_name_lower_idx")
}

func (ms *ModelSuite) Test_Module_Update_KeepsOwnName() {
	module := &Module{Name: "Trabajo"}
	verrs, err := ms.DB.ValidateAndCreate(module)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	verrs, err = ms.DB.ValidateAndUpdate(module)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_ModuleCounts() {
	base := time.Now().Add(-time.Hour)
	create := func(name string, createdAt time.Time) *Module {
		module := &Module{Name: name, CreatedAt: createdAt}
		verrs, err := ms.DB.ValidateAndCreate(module)
		ms.NoError(err)
		ms.False(verrs.HasAny())
		return module
	}
	work := create("Trabajo", base)
	personal := create("Personal", base.Add(time.Second))

	for _, t := range []*Task{
		{Title: "T1", ModuleID: nullUUID(work)},
		{Title: "T2", ModuleID: nullUUID(work), Completed: true},
		{Title: "P1", ModuleID: nullUUID(personal)},
		{Title: "Sin módulo"},
	} {
		verrs, err := ms.DB.ValidateAndCreate(t)
		ms.NoError(err)
		ms.False(verrs.HasAny())
	}
	create("Vacío", base.Add(2*time.Second))

	counts, err := ModuleCounts(ms.DB)
	ms.NoError(err)

	got := []string{}
	for _, c := range counts {
		got = append(got, fmt.Sprintf("%s %d", c.Name, c.TaskCount))
	}
	ms.Equal([]string{"Trabajo 2", "Personal 1", "Vacío 0"}, got)
	ms.Equal(work.ID, counts[0].ID)
}

// nullUUID devuelve el identificador del módulo como valor de Task.ModuleID.
func nullUUID(m *Module) uuid.NullUUID {
	return uuid.NullUUID{UUID: m.ID, Valid: true}
}
