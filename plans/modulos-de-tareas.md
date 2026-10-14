# Feature: módulos de tareas creados por el usuario en la barra lateral

## Encargo y criterios de aceptación

Encargo literal: «Segmentar las tareas en módulos para una mejor experiencia de usuario en la UI.»

Requisitos explícitos (decisión ya tomada por el usuario, con mockup elegido):

- R1. Los módulos los crea el usuario: listas/proyectos con nombre (p. ej. «Trabajo», «Personal»).
- R2. Cada tarea pertenece a un módulo.
- R3. Los módulos aparecen en la barra lateral (`<nav aria-label="Principal">` de `templates/application.plush.html`) con su contador.
- R4. Al pulsar un módulo se filtra el listado. Hay una entrada «Todas».
- R5. Se puede crear un módulo nuevo desde la barra lateral.
- R6. El formulario de alta permite elegir el módulo, preseleccionando el módulo que se está viendo.
- R7. Al filtrar se mantiene el orden por prioridad ya existente (`models.TaskListOrder`).

Mockup elegido por el usuario:

```
│ todo.        │ Trabajo                         │
│ ▸ Todas   5  │ ○ Enviar informe      [Alta]    │
│ ▸ Trabajo 3  │ ○ Preparar reunión   [Media]   │
│ ▸ Personal 2 │ ● Renovar dominio    [Baja]    │
│ + Nuevo      │ + Añadir una tarea… [Trabajo▾]  │
```

Decisiones de producto propuestas en la planificación (INFERENCIAS, no pedidas; el usuario debe validarlas antes de `/implement`). Para cada una se indica la alternativa descartada:

- D1. Módulo opcional en BD y modelo: `tasks.module_id` es `uuid NULL`. Las tareas existentes (las 3 del seed y cualquier otra) quedan sin módulo y se ven en «Todas» y en una entrada «Sin módulo» que solo aparece en la barra lateral cuando hay alguna. Esto relaja R2 («cada tarea pertenece a un módulo»): una tarea pertenece como mucho a un módulo.
  - Por qué: los tests existentes (que no deben modificarse) crean tareas con `&models.Task{Title: ...}` sin módulo (p. ej. `Test_TasksCreate_Valid`, `Test_Task_Create_Valid`, `Test_Task_Priority_DefaultOnPlainCreate`) y el fixture «tres tareas» no tiene módulo. Con `NOT NULL` todos fallarían. Un módulo por defecto «General» creado en la migración tampoco sirve en tests: `suite` trunca todas las tablas en cada `SetupTest`, así que esa fila desaparece y la FK fallaría.
  - Alternativa: `module_id NOT NULL` + módulo «General» creado en la migración y asignado a las tareas existentes, más un callback que cree/busque «General» al guardar una tarea sin módulo. Más fiel a R2, pero requiere modificar fixtures y añade lógica implícita en `BeforeSave`; queda fuera de este plan.
- D2. En el formulario de alta y de edición el selector incluye la opción «Sin módulo» (`value=""`). Preselección: el módulo que se está viendo; en «Todas» y en «Sin módulo», «Sin módulo».
- D3. Renombrar y borrar módulos quedan fuera (mínimo vertical: crear + listar + filtrar + asignar). La FK se crea sin `ON DELETE` (equivale a `NO ACTION`): si en el futuro se borra un módulo con tareas, la BD lo impedirá hasta que se decida la política (impedir, cascada o mover a «Sin módulo»).
- D4. Nombre del módulo: obligatorio tras recortar espacios, máximo 50 caracteres (runas), único sin distinguir mayúsculas/minúsculas (validación en el modelo + índice único `modules_name_lower_idx` sobre `lower(name)`). No se reservan los nombres «Todas» ni «Sin módulo».
  - Alternativa: unicidad exacta (sensible a mayúsculas) o sin unicidad; se descarta porque dos «Trabajo» indistinguibles en la barra lateral empeoran la UX que pide el encargo.
- D5. Orden de los módulos en la barra lateral: por fecha de creación (`created_at asc, id asc`). Evidencia: el mockup pone «Trabajo» antes que «Personal», que no es orden alfabético. Alternativa: alfabético.
- D6. Contador = número total de tareas del módulo, completadas incluidas. Evidencia: en el mockup «Trabajo 3» incluye «Renovar dominio», marcada como completada (●). «Todas» cuenta todas las tareas.
- D7. Esquema de URL del filtro: query string sobre el listado existente, `GET /?module=<uuid>` y `GET /?module=none` para «Sin módulo»; `GET /` es «Todas». Un `module` que no es UUID ni `none`, o un UUID inexistente, responde 404 (mismo criterio que `TasksEdit` con id inválido).
  - Alternativa: rutas `/modules/{id}`. Se descarta porque obligaría a duplicar la acción del listado; con query string `TasksIndex` sigue siendo la única acción de listado.
- D8. Vuelta a la vista actual: los formularios de crear, completar/reabrir, eliminar y editar llevan un campo oculto `view` (valor `""`, `none` o el UUID del módulo que se estaba viendo). La redirección se construye siempre en el servidor con `listPath(view)` (nunca se redirige a una URL recibida tal cual):
  - completar/reabrir y eliminar → vuelven a `listPath(view)`;
  - crear y actualizar → si `view` es «Todas» (`""`), vuelven a `/`; si se estaba en un módulo o en «Sin módulo», van a la vista del módulo **de la tarea** (para que la tarea creada/movida sea visible). Sin `view` (tests existentes) → `/`, como hoy.
  - El enlace de edición añade `?view=<view>` solo si la vista no es «Todas»; «Volver a la lista» y «Cancelar» usan `listPath(view)`.
- D9. La tarea puede cambiar de módulo desde la pantalla de edición (selector igual al del alta). Si el PUT no incluye `module_id` se conserva el actual (mismo patrón que `priority`); si llega vacío se quita el módulo.
- D10. Tras crear un módulo: flash «Módulo creado.» y redirección a `/?module=<id>` (el usuario aterriza en su módulo nuevo, vacío). Si hay error: 422 re-renderizando el listado de la vista en la que estaba (`view` oculto) con el formulario de módulo abierto y el error.
- D11. Encabezado del listado: «Tareas» en «Todas» (sin cambios), el nombre del módulo en su vista (como en el mockup) y «Sin módulo» en esa vista. El texto «N tareas en tu lista» se mantiene y cuenta las tareas filtradas.
- D12. Móvil (≤760px, la barra lateral es una franja superior): la marca queda en la primera fila y debajo, a todo el ancho, una fila horizontal desplazable (`overflow-x: auto`) con «Todas», los módulos y «Sin módulo» como pastillas con su contador; «+ Nuevo módulo» al final abre su formulario en una fila propia a todo el ancho. Alternativa: un `<select>` de módulos en móvil (descartada: duplicaría marcado).

Criterios de aceptación (observables):

1. Tras migrar existe la tabla `modules` (`id uuid PK`, `name varchar(50) NOT NULL`, `created_at`, `updated_at`) con el índice único `modules_name_lower_idx` sobre `lower(name)`, y `tasks.module_id uuid NULL` con FK `tasks_module_id_fkey` a `modules(id)` e índice `tasks_module_id_idx`. Las tareas existentes quedan con `module_id NULL`.
2. La barra lateral de toda página con layout (listado, edición y re-render 422) muestra: «Todas» con el total de tareas; cada módulo en orden de creación con su contador (completadas incluidas, 0 si no tiene); «Sin módulo» con su contador solo si es > 0; un control «+ Nuevo módulo».
3. El enlace de la vista activa lleva `aria-current="page"`; los demás no llevan `aria-current`.
4. `GET /?module=<id>` muestra solo las tareas de ese módulo, en el orden de `TaskListOrder`, con el nombre del módulo como `<h1>`, y el selector de módulo del alta con ese módulo seleccionado.
5. `GET /?module=none` muestra solo las tareas sin módulo. `GET /?module=<uuid inexistente>` y `GET /?module=abc` responden 404.
6. `POST /modules` con `name=" Trabajo "` crea el módulo «Trabajo» y redirige (303) a `/?module=<id>` con flash «Módulo creado.». Con nombre vacío/solo espacios, de más de 50 caracteres o repetido (sin distinguir mayúsculas) responde 422, no crea nada y muestra «El nombre del módulo es obligatorio.», «El nombre del módulo no puede superar los 50 caracteres.» o «Ya existe un módulo con ese nombre.».
7. Los nombres de módulo se escapan en la barra lateral, el `<h1>` y los selectores.
8. `POST /tasks` con `module_id=<id>` crea la tarea en ese módulo; sin `module_id` o con `module_id=""` la crea sin módulo; con un UUID inexistente o un valor no UUID responde 422 con «El módulo no es válido.» y no crea nada.
9. `GET /tasks/{id}/edit` muestra el selector con el módulo actual seleccionado. `PUT /tasks/{id}` con `module_id=<otro>` cambia el módulo; con `module_id=""` lo quita; sin `module_id` lo conserva; con valor inválido responde 422 y no cambia nada.
10. Redirecciones según D8; en concreto: `POST /tasks` con `view=<idA>` y `module_id=<idA>` → `Location: /?module=<idA>`; `PUT /tasks/{id}/toggle` con `view=<idA>` → `/?module=<idA>`; `DELETE /tasks/{id}` con `view=none` → `/?module=none`; un `view` manipulado (p. ej. `//evil.example`) → `/`.
11. Completar/reabrir (incluido el toggle optimista de `app.js`) no cambia el módulo ni la prioridad.
12. Todos los tests existentes siguen pasando sin modificarse; `fixtures/tasks.toml` no cambia.

## Comportamiento actual y propuesta

Actual (verificado en el código, rama `bacollaguazo/trabajo`, árbol limpio):

- Rutas en `actions/app.go`: `GET /` (`TasksIndex`), `POST /tasks`, `GET /tasks/{task_id}/edit`, `PUT /tasks/{task_id}`, `PUT /tasks/{task_id}/toggle`, `DELETE /tasks/{task_id}`, `GET /healthz`. Todas redirigen a `/` tras escribir.
- `actions/tasks.go`: `TasksIndex` y el re-render 422 de `TasksCreate` consultan `tx.Order(models.TaskListOrder).All(&tasks)` (consulta duplicada en dos sitios). `TasksUpdate` bindea a un struct anónimo `{Title, Priority}` (protección de mass assignment) y conserva la prioridad si no llega.
- `models/task.go`: `Task{ID, Title, Completed, Priority, CreatedAt, UpdatedAt}`; `BeforeValidate`/`BeforeSave` aplican la prioridad por defecto; `Validate` comprueba título y prioridad; `TaskListOrder` usa un `CASE`.
- Migraciones: `create_tasks` (fizz), `seed_tasks` (SQL, 3 tareas), `add_priority_to_tasks` (fizz + `sql()` para la `CHECK`). `migrations/embed.go` embebe `*.fizz *.sql`. La app migra al arrancar (`cmd/app/main.go` → `models.Migrate()`).
- `templates/application.plush.html`: barra lateral con la marca y `<nav aria-label="Principal">` con un único enlace «Tareas» (`aria-current="true"`); carga `app.css?v=4` y `app.js?v=3`.
- `templates/tasks/index.plush.html`: listado, formulario de alta con título + partial `tasks/priority_select.plush.html`; formularios de toggle (`data-task-toggle`, POST + `_method=PUT`) y borrado.
- `public/assets/app.css`: `.app-layout` en grid 14rem + contenido; a ≤760px la `.sidebar` pasa a fila flex (marca a la izquierda, nav a la derecha).
- `public/assets/app.js`: toggle optimista con `fetch(form.action, {method: "POST", body: new FormData(form)})`; reenvía todos los campos del formulario, por lo que un campo oculto `view` viaja solo.
- `github.com/gofrs/uuid` v4.4.0 (dependencia directa) incluye `uuid.NullUUID` con `Scan`/`Value`; no hace falta añadir `gobuffalo/nulls` (hoy solo indirecta).

Propuesta: una tabla `modules` nueva y una columna `tasks.module_id` opcional; un modelo `Module` con validación y una consulta de contadores; filtrado en `TasksIndex` mediante `?module=`; una acción `ModulesCreate`; un helper de barra lateral que todas las acciones que renderizan HTML llaman; una partial de selector de módulo reutilizada en alta y edición; y estilos de barra lateral para escritorio y móvil.

## Alcance

Incluido:

- Migración nueva: tabla `modules`, índice único sobre `lower(name)`, columna `tasks.module_id` con FK e índice.
- Modelo `Module` (validación de nombre) y consulta de contadores por módulo; campo `ModuleID` en `Task` y validación de existencia del módulo.
- Ruta `POST /modules` (`ModulesCreate`).
- Filtro del listado por `?module=<uuid>|none` manteniendo `TaskListOrder`.
- Barra lateral con «Todas», módulos con contador, «Sin módulo» (si > 0), estado activo y formulario «+ Nuevo módulo».
- Selector de módulo en alta (preseleccionado según la vista) y en edición.
- Campo `view` y helper `listPath` para volver a la vista tras crear/editar/completar/borrar.
- CSS de barra lateral (escritorio y ≤760px) y bump de `app.css?v=5`.
- Tests de modelo y de acciones nuevos.

Excluido (se deja para encargos posteriores):

- Renombrar, borrar, reordenar o archivar módulos; colores o iconos por módulo.
- Hacer el módulo obligatorio (`NOT NULL`) y módulo «General» por defecto (ver D1).
- Mostrar el nombre del módulo en cada fila del listado «Todas».
- Contadores de pendientes (solo totales) y actualización en vivo del contador al completar con el toggle optimista (no cambia, porque cuenta totales).
- Arrastrar tareas entre módulos, mover en bloque, filtros combinados (módulo + prioridad/estado), búsqueda.
- Cambios en `app.js`, el seed o `fixtures/tasks.toml`.
- Nombres reservados («Todas», «Sin módulo»).

## Diseño por capas

### Persistencia y migraciones

Archivo nuevo `migrations/20261015000000_create_modules.up.fizz` (timestamp posterior a `20261014000000_add_priority_to_tasks`):

```
create_table("modules") {
	t.Column("id", "uuid", {primary: true})
	t.Column("name", "string", {"size": 50})
	t.Timestamps()
}
sql("CREATE UNIQUE INDEX modules_name_lower_idx ON modules (lower(name))")
add_column("tasks", "module_id", "uuid", {"null": true})
sql("ALTER TABLE tasks ADD CONSTRAINT tasks_module_id_fkey FOREIGN KEY (module_id) REFERENCES modules (id)")
sql("CREATE INDEX tasks_module_id_idx ON tasks (module_id)")
```

Archivo nuevo `migrations/20261015000000_create_modules.down.fizz`:

```
sql("DROP INDEX IF EXISTS tasks_module_id_idx")
sql("ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_module_id_fkey")
drop_column("tasks", "module_id")
drop_table("modules")
```

Notas:

- Se usa `sql()` para la FK y los índices, igual que la `CHECK` de prioridades, para fijar nombres estables que los tests puedan buscar en el mensaje de error.
- En fizz, `string` sin `null: true` genera `NOT NULL`, como `title` en `create_tasks` (inferencia basada en el comportamiento por defecto de fizz; comprobar con `\d modules` tras migrar).
- Sin `ON DELETE`: decisión D3.
- No se toca el seed: las 3 tareas quedan con `module_id NULL` (D1).
- `migrations/embed.go` ya incluye `*.fizz`; no cambia.

### Modelo

`models/module.go` (nuevo):

- `const ModuleNameMaxLength = 50`.
- `type Module struct { ID uuid.UUID db:"id" form:"-"; Name string db:"name" form:"name"; CreatedAt, UpdatedAt }` con etiquetas `json`, y `type Modules []Module`.
- `func (m *Module) Validate(tx *pop.Connection) (*validate.Errors, error)`: recorta `Name` (`m.Name = strings.TrimSpace(m.Name)` en `BeforeValidate`, mismo patrón que la prioridad); errores en la clave `name`:
  - vacío → «El nombre del módulo es obligatorio.»
  - `len([]rune(name)) > ModuleNameMaxLength` → «El nombre del módulo no puede superar los 50 caracteres.»
  - `tx.Where("lower(name) = lower(?) AND id <> ?", m.Name, m.ID).Exists(&Module{})` → «Ya existe un módulo con ese nombre.» (en un alta `m.ID` es el UUID cero, así que la condición `id <>` no excluye nada).
- `const ModuleListOrder = "created_at asc, id asc"` (D5).
- `type ModuleCount struct { ID uuid.UUID db:"id"; Name string db:"name"; TaskCount int db:"task_count" }` y
  `func ModuleCounts(tx *pop.Connection) ([]ModuleCount, error)` con
  `tx.RawQuery("SELECT m.id, m.name, COUNT(t.id) AS task_count FROM modules m LEFT JOIN tasks t ON t.module_id = m.id GROUP BY m.id, m.name, m.created_at ORDER BY m.created_at asc, m.id asc").All(&counts)`.
  Nota: no usar `;` en la consulta (pop descarta `ORDER BY` con `;` en `Order()`; en `RawQuery` no aplica, pero se mantiene la convención).

`models/task.go` (modificar):

- Campo nuevo `ModuleID uuid.NullUUID \`json:"module_id" db:"module_id" form:"-"\``. `form:"-"` es deliberado: el valor se parsea a mano en la acción (un `module_id` no UUID en `c.Bind` provocaría error de binding/500 en lugar de 422; mismo razonamiento que la prioridad en texto).
- `func (t Task) ModuleValue() string`: `""` si `!t.ModuleID.Valid`, si no `t.ModuleID.UUID.String()`. Lo usan la partial del selector y las redirecciones.
- En `Validate`, añadir: si `t.ModuleID.Valid` y `!tx.Where("id = ?", t.ModuleID.UUID).Exists(&Module{})` → `verrs.Add("module_id", "El módulo no es válido.")`.
- `const ModuleInvalidMessage = "El módulo no es válido."` (en `task.go` o `module.go`) para reutilizarlo en la acción cuando el valor recibido no es UUID.
- No tocar `TaskListOrder`, callbacks ni la validación de prioridad.

### Acción y rutas

`actions/app.go`: añadir `app.POST("/modules", ModulesCreate)` junto a las rutas de tareas.

`actions/views.go` (nuevo, helpers compartidos):

- `const viewNone = "none"`.
- `func parseView(raw string) (view string, moduleID uuid.UUID, ok bool)`: `""` → vista «Todas»; `"none"` → «Sin módulo»; UUID válido → módulo; cualquier otra cosa → `ok=false`.
- `func listPath(view string) string`: `"/"` para `""` o valores no válidos; `"/?module=none"`; `"/?module=" + uuid.String()` (reconstruido desde el UUID parseado, nunca concatenando la entrada). Evita open redirect.
- `func taskViewAfterWrite(view string, task *models.Task) string`: implementa D8 para crear/actualizar (`""` → `""`; en otro caso `task.ModuleValue()` o `"none"` si no tiene módulo).
- `func parseModuleParam(raw string) (uuid.NullUUID, bool)`: `strings.TrimSpace(raw) == ""` → `NullUUID{}` válido; UUID → `{UUID, Valid: true}`; resto → `ok=false`.
- `type sidebarItem struct { Label, Href string; Count int; Current bool }` y
  `func setLayout(c buffalo.Context, tx *pop.Connection, view string) error`: calcula
  `total := tx.Count(&models.Task{})`, `none := tx.Where("module_id IS NULL").Count(&models.Task{})`, `models.ModuleCounts(tx)`; construye los items («Todas» → `/`; módulos → `/?module=<id>`; «Sin módulo» → `/?module=none` solo si `none > 0` **o** la vista actual es `none`) marcando `Current` según `view`; hace `c.Set("sidebarItems", items)`, `c.Set("view", view)` y, si no estaban ya en el contexto, `c.Set("moduleErrors", validate.NewErrors())` y `c.Set("moduleName", "")`. También `c.Set("moduleOptions", ...)` con la lista de módulos para los selectores (`[]selectOption{Value, Label}` construido desde los mismos `ModuleCounts`, evitando otra consulta).
  Decisión: helper explícito llamado por cada acción que renderiza HTML, en lugar de un middleware global, para no consultar la BD en `/healthz` y seguir el estilo explícito de `c.Set` del repositorio. Riesgo: olvidarlo en una acción → error de Plush por identificador desconocido; lo cubren los tests de 422 y de edición.
- `func renderTaskList(c buffalo.Context, tx *pop.Connection, view string, task *models.Task, errs *validate.Errors, status int) error`: resuelve la vista (404 si el módulo no existe), consulta las tareas filtradas con `Order(models.TaskListOrder)` — `q := tx.Q()`; para módulo `q = tx.Where("module_id = ?", id)`; para `none` `q = tx.Where("module_id IS NULL")` —, fija `viewTitle` («Tareas», nombre del módulo o «Sin módulo»), `tasks`, `task`, `errors`, llama a `setLayout` y renderiza `tasks/index.plush.html` con `status`. Sustituye las dos consultas duplicadas actuales (`TasksIndex` y 422 de `TasksCreate`) y la usa también `ModulesCreate` en su 422.

`actions/tasks.go` (modificar):

- `TasksIndex`: `view := c.Param("module")`; si `parseView` falla → `c.Error(http.StatusNotFound, ...)`. Tarea nueva `&models.Task{Priority: models.PriorityMedium}` con `ModuleID` = módulo de la vista (vacío en «Todas»/«Sin módulo»; D2). `renderTaskList(..., http.StatusOK)`.
- `TasksCreate`: tras `c.Bind(task)` y el `TrimSpace` del título, `view := c.Param("view")` (si `parseView` falla, tratar como `""`); `moduleID, ok := parseModuleParam(c.Param("module_id"))`; si `!ok` → no llamar a `ValidateAndCreate`; construir `verrs := validate.NewErrors()`, `verrs.Add("module_id", models.ModuleInvalidMessage)` y re-renderizar 422 sin crear nada; si `ok` → `task.ModuleID = moduleID` y `tx.ValidateAndCreate(task)` (el modelo valida que el módulo exista). Si hay errores → `renderTaskList(c, tx, view, task, verrs, 422)`. Éxito → flash y `c.Redirect(303, listPath(taskViewAfterWrite(view, task)))`.
- `TasksEdit`: `view := c.Param("view")` (inválido → `""`); `c.Set("backPath", listPath(view))`; `setLayout(c, tx, view)`.
- `TasksUpdate`: el struct de entrada añade `ModuleID string \`form:"module_id"\`` y `View string \`form:"view"\``. Para distinguir «no enviado» de «vacío» (D9): `_ = c.Request().ParseForm()` y `_, sent := c.Request().Form["module_id"]`. Si `sent`: `parseModuleParam`; inválido → 422 con el error (sin escribir); válido → asignar. En los 422 llamar a `setLayout` y fijar `backPath`. Éxito → `listPath(taskViewAfterWrite(view, task))`.
  No añadir `Completed` ni otros campos al struct (mass assignment; `Test_TasksUpdate_PersistsTitleOnly`).
- `TasksToggle` y `TasksDestroy`: redirigir a `listPath(c.Param("view"))`.

`actions/modules.go` (nuevo):

- `ModulesCreate`: bind a `input struct{ Name string form:"name"; View string form:"view" }` (no bindear el modelo directamente; mismo patrón que `TasksUpdate`), `module := &models.Module{Name: strings.TrimSpace(input.Name)}`, `tx.ValidateAndCreate(module)`. Errores → `c.Set("moduleErrors", verrs)`, `c.Set("moduleName", module.Name)` y `renderTaskList(c, tx, view, nuevaTareaParaLaVista, validate.NewErrors(), 422)` (si `view` no es válida o el módulo ya no existe, usar `""`). Éxito → flash «Módulo creado.» y `c.Redirect(303, "/?module="+module.ID.String())`.

### Renderizado (Plush)

`templates/application.plush.html`:

- Sustituir el enlace único «Tareas» del `<nav aria-label="Principal">` por una lista:

```
<ul class="module-nav">
  <%= for (item) in sidebarItems { %>
    <li><a href="<%= item.Href %>" class="nav-link"<%= if (item.Current) { %> aria-current="page"<% } %>>
      <span class="nav-label"><%= item.Label %></span>
      <span class="nav-count"><%= item.Count %><span class="sr-only"> tareas</span></span>
    </a></li>
  <% } %>
</ul>
<details class="new-module"<%= if (len(moduleErrors.Get("name")) > 0) { %> open<% } %>>
  <summary>+ Nuevo módulo</summary>
  <form action="/modules" method="POST" aria-label="Nuevo módulo">
    authenticity_token, <input type="hidden" name="view" value="<%= view %>">,
    <label for="module-name">Nombre del módulo</label>
    <input id="module-name" name="name" maxlength="50" value="<%= moduleName %>" autocomplete="off" + aria-invalid/aria-describedby si hay error; sin `required`, para que el error lo muestre el servidor como en el alta de tareas>
    <button type="submit" class="button button-primary">Crear</button>
    <p id="module-name-error" class="field-error" role="alert">…</p> (solo si hay error)
  </form>
</details>
```

  `<details>/<summary>` evita JavaScript y es accesible por teclado. El icono SVG actual puede mantenerse en «Todas».
- `app.css?v=4` → `?v=5`. `app.js` no cambia (sigue `?v=3`).

`templates/tasks/_module_select.plush.html` (nuevo, mismo patrón que `_priority_select`):

```
<select id="task-module" name="module_id" <%= if (len(errors.Get("module_id")) > 0) { %>aria-invalid="true" aria-describedby="module-error"<% } %>>
  <option value=""<%= if (task.ModuleValue() == "") { %> selected<% } %>>Sin módulo</option>
  <%= for (opt) in moduleOptions { %>
    <option value="<%= opt.Value %>"<%= if (task.ModuleValue() == opt.Value) { %> selected<% } %>><%= opt.Label %></option>
  <% } %>
</select>
```

`templates/tasks/index.plush.html`:

- `<h1><%= viewTitle %></h1>` (antes literal «Tareas»).
- En cada fila: formularios de toggle y borrado con `<input type="hidden" name="view" value="<%= view %>">`; enlace de edición `/tasks/<%= t.ID %>/edit<%= if (view != "") { %>?view=<%= view %><% } %>` (en «Todas» queda idéntico al actual; `Test_TasksIndex_EditLink` busca `/tasks/<id>/edit`).
- Mensaje de vacío: en una vista filtrada «No hay tareas en este módulo.»; en «Todas» se mantiene el texto actual («No hay tareas todavía…», lo comprueba `Test_TasksIndex_EmptyState`).
- Formulario de alta: campo oculto `view`, `<label for="task-module" class="sr-only">Módulo</label>` + `partial("tasks/module_select.plush.html")` entre el selector de prioridad y «Añadir»; bloque de error `module-error` igual que `priority-error`.

`templates/tasks/edit.plush.html`:

- Campo oculto `view`; «Volver a la lista» y «Cancelar» con `href="<%= backPath %>"`.
- `<label for="task-module">Módulo</label>` + la partial + bloque de error, tras la prioridad.

### CSS (`public/assets/app.css`)

- `.module-nav`: lista sin viñetas, `display: grid; gap: 0.15rem`.
- `.nav-link`: hoy tiene fondo rojizo fijo (era el único enlace, siempre activo). Pasar ese fondo/color a `.nav-link[aria-current="page"]`; el estado por defecto, transparente con color de texto; mantener `min-height: 44px`. `.nav-label` con `overflow: hidden; text-overflow: ellipsis; white-space: nowrap; flex: 1`; `.nav-count` alineado a la derecha, `font-variant-numeric: tabular-nums`, color atenuado.
- `.new-module summary` con aspecto de `.nav-link` (44px, cursor), formulario apilado con el input al 100%.
- Selector de módulo en `.new-task-row`: mismo estilo que el de prioridad; acotar ancho (`max-width: 9rem`) para nombres largos.
- `@media (max-width: 760px)` (D12): `.sidebar { flex-wrap: wrap; }`, `nav[aria-label="Principal"] { width: 100%; }`, `.module-nav { display: flex; overflow-x: auto; gap: 0.4rem; }` con `.nav-link { white-space: nowrap; }`, `.new-module { width: 100%; }`. Revisar que no haya scroll horizontal de página a 375px.
- `@media (max-width: 380px)`: la fila de alta ya hace `flex-wrap`; comprobar que los dos selectores caben.

## Archivos relevantes

- `migrations/20261015000000_create_modules.up.fizz` / `.down.fizz` (nuevos): esquema de módulos, FK e índices.
- `models/module.go` (nuevo): modelo `Module`, validación, `ModuleListOrder`, `ModuleCounts`.
- `models/task.go`: campo `ModuleID`, `ModuleValue()`, validación de existencia del módulo.
- `actions/app.go`: ruta `POST /modules`.
- `actions/views.go` (nuevo): `parseView`, `listPath`, `taskViewAfterWrite`, `parseModuleParam`, `setLayout`, `renderTaskList`.
- `actions/modules.go` (nuevo): `ModulesCreate`.
- `actions/tasks.go`: filtro, preselección, módulo en alta/edición, redirecciones con `view`.
- `templates/application.plush.html`: barra lateral con módulos, formulario de módulo nuevo, bump CSS.
- `templates/tasks/_module_select.plush.html` (nuevo): selector de módulo.
- `templates/tasks/index.plush.html`, `templates/tasks/edit.plush.html`: título de vista, campos `view`, selector y errores de módulo.
- `public/assets/app.css`: estilos de barra lateral, estado activo, móvil.
- `models/module_test.go` (nuevo), `models/task_test.go` (solo añadir tests), `actions/modules_test.go` (nuevo), `actions/tasks_test.go` (solo añadir tests).
- Sin cambios: `public/assets/app.js`, `fixtures/tasks.toml`, seed, `migrations/embed.go`, `go.mod`/`go.sum`.
- Referencia de patrones: `plans/prioridad-de-tareas.md` (selector en partial, struct de entrada en update, `CHECK` con `sql()`), `plans/check-no-persiste.md` (toggle por POST + `FormData`).

## Implementación paso a paso

1. Crear la migración up/down.
2. Crear `models/module.go`; modificar `models/task.go` (campo, `ModuleValue`, validación).
3. Escribir los tests de modelo (`models/module_test.go` y añadidos en `models/task_test.go`).
4. Crear `actions/views.go` con los helpers y `actions/modules.go`; registrar la ruta en `actions/app.go`.
5. Refactorizar `actions/tasks.go` para usar `renderTaskList`/`setLayout` y añadir módulo + `view` en cada acción.
6. Crear `templates/tasks/_module_select.plush.html`; modificar `application.plush.html`, `index.plush.html` y `edit.plush.html`.
7. Ajustar `public/assets/app.css` y subir a `?v=5`.
8. Escribir los tests de acciones (`actions/modules_test.go` y añadidos en `actions/tasks_test.go`).
9. `gofmt -w` sobre los `.go` tocados.
10. Migrar `todo_test` y validar en contenedor Linux (sección siguiente).
11. `docker compose up -d --build` y verificación manual.
12. `git diff --check`, `git status --short`. Sin commit.

## Tests y validación

### Tests de modelo

`models/module_test.go` (nuevo):

- `Test_Module_Create_Valid`: `ValidateAndCreate(&Module{Name: "  Trabajo  "})` → sin errores, `Name == "Trabajo"`, `ID` no cero.
- `Test_Module_Name_Required`: `""` y `"   "` → error «El nombre del módulo es obligatorio.» y `Count == 0`.
- `Test_Module_Name_MaxLength`: 50 `á` válido; 51 `á` → «El nombre del módulo no puede superar los 50 caracteres.».
- `Test_Module_Name_UniqueCaseInsensitive`: crear «Trabajo»; `ValidateAndCreate(&Module{Name: " trabajo "})` → «Ya existe un módulo con ese nombre.»; `Count == 1`.
- `Test_Module_Name_UniqueIndex`: crear «Trabajo»; `ms.DB.Create(&Module{Name: "TRABAJO"})` (sin validar) → error que contiene `modules_name_lower_idx`.
- `Test_Module_Update_KeepsOwnName`: crear «Trabajo» y `ValidateAndUpdate` sin cambiar el nombre → sin errores (la condición `id <>` excluye el propio registro).
- `Test_ModuleCounts`: crear «Trabajo» y luego «Personal» (con `CreatedAt` explícito creciente), 2 tareas en Trabajo (una completada), 1 en Personal, 1 sin módulo; crear «Vacío» → resultado `[Trabajo 2, Personal 1, Vacío 0]` en ese orden.

Añadidos en `models/task_test.go`:

- `Test_Task_Module_Optional`: `ValidateAndCreate(&Task{Title: "x"})` → `ModuleID.Valid == false` tras recargar.
- `Test_Task_Module_Persists`: tarea con `ModuleID` de un módulo existente → recargada con ese UUID.
- `Test_Task_Module_Nonexistent_Invalid`: `ModuleID` con UUID aleatorio → `verrs.Get("module_id")` contiene «El módulo no es válido.»; `Count == 0`.
- `Test_Task_Module_ForeignKey`: `ms.DB.Create(&Task{Title: "x", ModuleID: uuid.NullUUID{UUID: aleatorio, Valid: true}})` → error que contiene `tasks_module_id_fkey`.
- `Test_Task_ModuleValue`: `""` sin módulo; UUID en texto con módulo.

### Tests de acciones

Usar `url.Values` en todos los POST/PUT nuevos. Helpers nuevos en el test: `as.createModule(name string, createdAt time.Time) *models.Module` y `as.createTaskIn(title, priority string, moduleID uuid.NullUUID, createdAt time.Time) *models.Task` (los helpers existentes `createTask` y `taskForEditing` no se modifican).

`actions/modules_test.go` (nuevo):

- `Test_Sidebar_ListsModulesWithCounts`: módulos «Trabajo» (2 tareas, una completada) y «Personal» (1) creados en ese orden + 1 tarea sin módulo → `GET /` contiene `/?module=<idTrabajo>` antes que `/?module=<idPersonal>` (`strings.Index`), los contadores (comprobar el fragmento `nav-count">2<` etc., o el texto completo del `<a>`), «Todas» con 4, «Sin módulo» con 1, «Nuevo módulo» y `action="/modules"`.
- `Test_Sidebar_HidesNoModuleWhenEmpty`: solo tareas con módulo → no contiene `/?module=none`.
- `Test_Sidebar_TodasIsCurrentByDefault`: `GET /` → `href="/" class="nav-link" aria-current="page"` (ajustar al marcado final) y ningún otro `aria-current`.
- `Test_ModulesCreate_Valid`: POST `name=  Trabajo  ` → 303, `Location == "/?module=<id>"` del módulo creado, `Name == "Trabajo"`.
- `Test_ModulesCreate_EmptyName`: POST `name=   ` → 422, body contiene «El nombre del módulo es obligatorio.» y `<details class="new-module" open`; `Count(&models.Module{}) == 0`.
- `Test_ModulesCreate_Duplicate`: con «Trabajo» existente, POST `name=TRABAJO` → 422 «Ya existe un módulo con ese nombre.»; `Count == 1`.
- `Test_ModulesCreate_TooLong`: 51 runas → 422 y mensaje.
- `Test_ModulesCreate_EscapesName`: POST `name=<img src=x onerror=alert(1)>` → 303; `GET /` no contiene `<img src=x` y sí `&lt;img`.
- `Test_ModulesCreate_Invalid_KeepsView`: POST `name=` con `view=<idTrabajo>` → 422 y el `<h1>` es «Trabajo» (la vista se conserva).

Añadidos en `actions/tasks_test.go`:

- `Test_TasksIndex_FiltersByModule_KeepsPriorityOrder`: en «Trabajo» crear «T baja» (`baja`), «T alta» (`alta`), «T media» (`media`); en «Personal» «P alta»; `GET /?module=<idTrabajo>` → contiene las tres de Trabajo en orden alta < media < baja, NO contiene «P alta», `<h1>Trabajo</h1>`, `aria-current="page"` en el enlace de Trabajo y `<option value="<idTrabajo>" selected>`.
- `Test_TasksIndex_FilterNone`: una tarea con módulo y otra sin → `GET /?module=none` solo muestra la segunda; `<h1>Sin módulo</h1>`.
- `Test_TasksIndex_UnknownModule_NotFound`: `GET /?module=<uuid aleatorio>` → 404.
- `Test_TasksIndex_InvalidModule_NotFound`: `GET /?module=abc` → 404.
- `Test_TasksIndex_AllView_PreselectsNoModule`: `GET /` contiene `<option value="" selected>Sin módulo</option>`.
- `Test_TasksCreate_WithModule`: POST `title=X&module_id=<id>&view=<id>` → 303, `Location == "/?module=<id>"`, tarea con ese módulo.
- `Test_TasksCreate_FromAllView_StaysInAll`: POST `title=X&module_id=<id>&view=` → `Location == "/"`.
- `Test_TasksCreate_FromModuleView_OtherModule`: `view=<idA>`, `module_id=<idB>` → `Location == "/?module=<idB>"` (D8).
- `Test_TasksCreate_UnknownModule`: `module_id=<uuid aleatorio>` → 422, «El módulo no es válido.», `Count == 0`.
- `Test_TasksCreate_MalformedModule`: `module_id=abc` → 422, mismo mensaje, `Count == 0`.
- `Test_TasksEdit_PreselectsModule`: tarea en «Trabajo» → `GET /tasks/{id}/edit` contiene `<option value="<id>" selected>`; con `?view=<id>` el enlace «Cancelar» apunta a `/?module=<id>`.
- `Test_TasksUpdate_ChangesModule`: PUT `title=X&module_id=<idB>` → módulo B; prioridad y `Completed` intactos.
- `Test_TasksUpdate_ClearsModule`: PUT `title=X&module_id=` → `ModuleID.Valid == false`.
- `Test_TasksUpdate_WithoutModule_KeepsIt`: PUT solo `title` → conserva el módulo.
- `Test_TasksUpdate_InvalidModule`: PUT `module_id=abc` → 422 y BD sin cambios (título original).
- `Test_TasksToggle_ReturnsToView`: PUT toggle con `view=<id>` → `Location == "/?module=<id>"`; módulo intacto tras recargar.
- `Test_TasksToggle_FormPost_ReturnsToView`: POST con `_method=PUT` y `view=<id>` (como `app.js`) → 303 y misma `Location`.
- `Test_TasksDestroy_ReturnsToView`: DELETE con `view=none` → `Location == "/?module=none"`.
- `Test_ListPath_RejectsArbitraryValues`: test unitario de `listPath` (`""`, `"none"`, un UUID, `"//evil.example"`, `"/?module=x"`, `"https://x"`) → solo los tres primeros producen algo distinto de `/`.
- `Test_TasksIndex_ToggleAndDeleteFormsCarryView`: en `GET /?module=<id>` el body contiene `name="view" value="<id>"` y el enlace `/tasks/<idTarea>/edit?view=<id>`.

Los tests existentes no se modifican y deben seguir en verde. Puntos de atención: `Test_TasksCreate_Valid` y `Test_TasksUpdate_PersistsTitleOnly` (esperan `Location == "/"` sin `view`), `Test_TasksIndex_EditLink`, `Test_TasksIndex_EmptyState`, `Test_TasksIndex_FormDefaultsToMedium`, `Test_TasksUpdate_EmptyTitle` (`aria-describedby="title-error"`), `Test_AppJS_ToggleSubmitsFormAsPost`.

### Preparación del entorno y validación (Windows 11, Git Bash)

Contexto verificado por el usuario: Postgres en el contenedor `desarrollo-agentico-todo-03-db-1` (127.0.0.1:5432, postgres/postgres) con `todo_development` y `todo_test` (esta migrada hasta prioridades); `soda` instalado en `$(go env GOPATH)/bin/soda.exe`. En Windows los tests de `actions` no funcionan con partials en subcarpetas (Buffalo v1.1.4, `render/template.go:138` usa `filepath.Join`) y `core.autocrlf=true` rompe `gofmt` y un test de `adws`; por eso la validación fiable es en contenedor Linux, como CI.

```bash
REPO=/c/Users/bacollaguazo/Documents/GitHub/desarrollo-agentico-todo-03

# 1. Migrar todo_test con la migración nueva (como el paso «Migraciones» de .github/workflows/ci.yaml)
cd "$REPO" && GO_ENV=test "$(go env GOPATH)/bin/soda.exe" migrate -e test
# soda deja un migrations/schema.sql vacío (falta pg_dump): BORRARLO, nunca commitearlo,
# porque migrations/embed.go embebe *.sql y rompería las migraciones embebidas.
cd "$REPO" && rm -f migrations/schema.sql

# 2. Registrar los archivos nuevos para que «git ls-files» los vea y se normalicen sus CRLF
#    (sin esto gofmt dentro del contenedor marcará los .go nuevos).
cd "$REPO" && git add -N migrations/20261015000000_create_modules.up.fizz migrations/20261015000000_create_modules.down.fizz \
  models/module.go models/module_test.go actions/views.go actions/modules.go actions/modules_test.go \
  templates/tasks/_module_select.plush.html

# 3. Validación como CI en contenedor Linux
cd "$REPO" && MSYS_NO_PATHCONV=1 docker run --rm --network desarrollo-agentico-todo-03_default \
  -e GO_ENV=test -e DB_HOST=db -v "$(pwd -W)":/repo:ro -v todo03-gomod:/go/pkg/mod golang:1.26 \
  bash -c 'cp -r /repo /src && cd /src && git config --global --add safe.directory "*" && git ls-files | while read f; do sed -i "s/\r$//" "$f"; done && gofmt -l . && go vet ./... && go test -p 1 -count=1 ./... && go build ./...'
```

- `gofmt -l .` no debe imprimir nada. `-p 1` es obligatorio (`actions` y `models` comparten `todo_test` y se truncan mutuamente).
- `git add -N` solo registra la intención de añadir (no hace commit); es reversible con `git reset <archivo>`. Alternativa sin tocar el índice: sustituir `git ls-files` por `find . -type f \( -name "*.go" -o -name "*.fizz" -o -name "*.html" -o -name "*.toml" \) -not -path "./.git/*"` en el bucle de `sed`.
- Comprobar el esquema: `docker exec desarrollo-agentico-todo-03-db-1 psql -U postgres -d todo_test -c "\d modules" -c "\d tasks"` debe listar `modules_name_lower_idx`, `tasks_module_id_fkey` y `tasks_module_id_idx`.
- Probar el down/up de la migración en `todo_test`: `GO_ENV=test soda migrate down -e test` y de nuevo `migrate` (borrando `schema.sql` después de cada uno).

### Verificación manual

1. `docker compose up -d --build`; `docker compose logs app` sin errores de migración (migra `todo_development` al arrancar).
2. http://localhost:3000: la barra lateral muestra «Todas 3» y «Sin módulo 3» (las del seed) y «+ Nuevo módulo».
3. Crear «Trabajo» y «Personal»: aterriza en cada módulo vacío; aparecen en ese orden con contador 0. Intentar crear «trabajo» → error visible con el formulario abierto.
4. En «Trabajo», añadir tareas con prioridades distintas: el selector viene en «Trabajo», el listado se ordena Alta → Media → Baja y el contador sube. Editar una tarea del seed y moverla a «Personal»: al guardar se vuelve a la vista adecuada (D8) y los contadores cambian.
5. Completar/reabrir dentro de un módulo: no sale del módulo; al recargar persiste.
6. Navegación por teclado: Tab por los enlaces de la barra lateral, `aria-current` en el activo, «+ Nuevo módulo» se abre con Enter/Espacio.
7. A 375px y 760px: franja superior con la fila de módulos desplazable, sin scroll horizontal de la página; formulario de módulo a todo el ancho; la fila de alta con dos selectores no desborda.
8. Recarga forzada para confirmar `app.css?v=5`.

## Riesgos y recuperación

- D1 contradice parcialmente R2: si el usuario exige que toda tarea tenga módulo, este plan no lo cumple; el paso posterior (módulo «General» + `NOT NULL`) requerirá migración de datos, ajustar fixtures y un callback. Validar D1 antes de implementar.
- Distinguir «campo no enviado» de «vacío» en `TasksUpdate` depende de `c.Request().Form` tras `ParseForm` (con `_method=PUT` por POST el cuerpo está en `Form`; inferencia sobre Buffalo/net/http, cubierta por `Test_TasksUpdate_WithoutModule_KeepsIt` y `Test_TasksUpdate_ClearsModule`). Si fallara, alternativa: valor centinela `none` en la opción «Sin módulo».
- `uuid.NullUUID` con pop/sqlx: implementa `Scanner`/`Valuer`, así que debería leerse y escribirse como `NULL`/UUID (inferencia; cubierta por `Test_Task_Module_Optional`/`Persists`). Si pop no lo trata bien, alternativa: `nulls.UUID` de `github.com/gobuffalo/nulls` (hoy indirecta en `go.mod`; pasaría a directa con `go mod tidy`).
- Comparaciones en Plush: se comparan cadenas (`task.ModuleValue() == opt.Value`), no `uuid.UUID`, para evitar dudas sobre la igualdad de arrays en Plush.
- Olvidar `setLayout` en alguna acción que renderiza HTML provoca un error de Plush (500) por `sidebarItems` indefinido; los tests de 422 (crear tarea, actualizar, crear módulo) y de edición lo detectan.
- Condición de carrera en la unicidad: dos altas simultáneas con el mismo nombre pasan la validación y la segunda falla en el índice único con 500. Aceptable con un solo usuario; el índice garantiza la integridad.
- Contadores: se calculan con 3 consultas por página (total, sin módulo, por módulo). Irrelevante con el volumen actual; el índice `tasks_module_id_idx` cubre el `JOIN` y el filtro.
- Open redirect: `listPath` reconstruye la URL desde un UUID parseado o constantes; nunca concatena `view` recibido. Cubierto por `Test_ListPath_RejectsArbitraryValues`.
- XSS: nombres de módulo siempre con `<%= %>` (escapa); no usar `raw()`. Cubierto por `Test_ModulesCreate_EscapesName`.
- El toggle optimista no actualiza contadores (son totales y no cambian al completar) ni saca la tarea de la vista; coherente con D6.
- Cambio visual del `.nav-link`: hoy siempre tiene fondo rojizo; tras el cambio solo el activo. Comprobar contraste del estado no activo (texto oscuro sobre `--color-sidebar`).
- Plush y partials en subcarpetas en Windows: los tests de `actions` deben correr en el contenedor Linux; no interpretar un fallo local de Windows como fallo del cambio.
- `migrations/schema.sql` generado por soda: si se commitea vacío, `migrations/embed.go` lo embebe y `models.Migrate()` podría fallar o comportarse de forma inesperada. Borrarlo siempre tras `soda migrate`.
- Recuperación: `GO_ENV=<env> soda migrate down -e <env>` (o ejecutar a mano el contenido del `.down.fizz`: `DROP INDEX tasks_module_id_idx; ALTER TABLE tasks DROP CONSTRAINT tasks_module_id_fkey; ALTER TABLE tasks DROP COLUMN module_id; DROP TABLE modules;`) y revertir el código. Se pierden los módulos y las asignaciones; las tareas se conservan.
- Imagen Docker: la app embebe migraciones, templates y assets; sin `docker compose up -d --build` no se verá nada nuevo.
