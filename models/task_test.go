package models

import (
	"strings"
	"time"
)

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) Test_Task_Priority_DefaultsToMedium() {
	task := &Task{Title: "Sin prioridad"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
	ms.Equal(PriorityMedium, task.Priority)

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.Equal(PriorityMedium, reloaded.Priority)
}

func (ms *ModelSuite) Test_Task_Priority_AcceptsValidValues() {
	for _, p := range []string{PriorityHigh, PriorityMedium, PriorityLow} {
		task := &Task{Title: "Tarea " + p, Priority: p}

		verrs, err := ms.DB.ValidateAndCreate(task)
		ms.NoError(err)
		ms.False(verrs.HasAny())

		reloaded := &Task{}
		ms.NoError(ms.DB.Find(reloaded, task.ID))
		ms.Equal(p, reloaded.Priority)
	}
}

func (ms *ModelSuite) Test_Task_Priority_Invalid() {
	task := &Task{Title: "Con prioridad inválida", Priority: "urgente"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("priority"), "La prioridad no es válida.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Priority_DefaultOnPlainCreate() {
	task := &Task{Title: "Creada sin validar"}
	ms.NoError(ms.DB.Create(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.Equal(PriorityMedium, reloaded.Priority)
}

func (ms *ModelSuite) Test_Task_PriorityLabel() {
	ms.Equal("Alta", Task{Priority: PriorityHigh}.PriorityLabel())
	ms.Equal("Media", Task{Priority: PriorityMedium}.PriorityLabel())
	ms.Equal("Baja", Task{Priority: PriorityLow}.PriorityLabel())
}

func (ms *ModelSuite) Test_Task_Priority_CheckConstraint_Create() {
	err := ms.DB.Create(&Task{Title: "Saltando la validación", Priority: "urgente"})
	ms.Error(err)
	ms.Contains(err.Error(), "tasks_priority_check")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Priority_CheckConstraint_Update() {
	task := &Task{Title: "Válida", Priority: PriorityLow}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Priority = "urgente"
	err = ms.DB.Update(task)
	ms.Error(err)
	ms.Contains(err.Error(), "tasks_priority_check")

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.Equal(PriorityLow, reloaded.Priority)
}

func (ms *ModelSuite) Test_Task_ListOrder() {
	base := time.Now().Add(-time.Hour)
	for i, t := range []Task{
		{Title: "B1", Priority: PriorityLow},
		{Title: "A1", Priority: PriorityHigh},
		{Title: "M1", Priority: PriorityMedium},
		{Title: "A2", Priority: PriorityHigh},
	} {
		t.CreatedAt = base.Add(time.Duration(i) * time.Second)
		verrs, err := ms.DB.ValidateAndCreate(&t)
		ms.NoError(err)
		ms.False(verrs.HasAny())
	}

	tasks := Tasks{}
	ms.NoError(ms.DB.Order(TaskListOrder).All(&tasks))

	titles := []string{}
	for _, t := range tasks {
		titles = append(titles, t.Title)
	}
	ms.Equal([]string{"A1", "A2", "M1", "B1"}, titles)
}
