# Feature: prioridad de las tareas visible en el listado

## Encargo y criterios de aceptación

Encargo literal: «El listado de tareas no tiene prioridades; hay que añadir esta funcionalidad: que cada tarea tenga una prioridad visible en el listado.»

Requisitos explícitos:

- R1. Cada tarea tiene una prioridad persistida.
- R2. La prioridad de cada tarea se ve en el listado (`GET /`).

Decisiones de producto tomadas en la planificación (inferencias, no pedidas; revisar antes de `/implement`):

- D1. Tres niveles cerrados: `alta`, `media`, `baja` (valores almacenados en minúscula, etiquetas visibles «Alta», «Media», «Baja»).
- D2. Valor por defecto `media`, tanto para tareas nuevas sin prioridad como para todas las tareas existentes (incluidas las tres del seed), que la reciben al añadir la columna con `DEFAULT 'media'`.
- D3. La prioridad se elige al crear (selector en el formulario de alta, preseleccionado en «Media») y se puede cambiar en la pantalla de edición.
- D4. (Decisión del usuario, sustituye a la inferencia previa) El listado se ordena por prioridad Alta → Media → Baja y, dentro de cada nivel, por `created_at asc, id asc`. Filtrar por prioridad queda fuera de alcance.
- D7. (Decisión del usuario) La BD impone los valores válidos con una restricción `CHECK` con nombre `tasks_priority_check`, además de la validación del modelo.
- D5. La prioridad se muestra como etiqueta de texto (no solo color) junto al título, con prefijo «Prioridad:» solo para lectores de pantalla.
- D6. El seed no se modifica; sus tareas quedan en `media`.

Criterios de aceptación (observables):

1. Tras migrar, la tabla `tasks` tiene una columna `priority` `VARCHAR(10) NOT NULL DEFAULT 'media'`, y las filas existentes quedan con `media`.
2. `GET /` muestra en cada tarea una etiqueta con su prioridad («Alta», «Media» o «Baja») y una clase `task-priority-<valor>`.
3. El formulario de alta incluye un `<select name="priority">` con las tres opciones y «Media» seleccionada por defecto.
4. `POST /tasks` con `priority=alta` crea la tarea con prioridad `alta`; sin campo `priority` la crea con `media`.
5. `POST /tasks` con una prioridad fuera de la lista (p. ej. `urgente`) responde 422, no crea la tarea y muestra «La prioridad no es válida.».
6. `GET /tasks/{id}/edit` muestra el selector con la prioridad actual seleccionada.
7. `PUT /tasks/{id}` con `priority=baja` cambia la prioridad; sin campo `priority` la conserva; con valor inválido responde 422, muestra el error y no cambia nada en BD.
8. Completar/reabrir una tarea (incluido el cambio optimista de `app.js`) no altera ni oculta la prioridad.
9. `GET /` (y el re-render 422 de `POST /tasks`) lista primero las tareas `alta`, luego `media` y luego `baja`; dentro de cada nivel, de más antigua a más reciente (`created_at asc, id asc`). El estado completada/pendiente no influye en el orden.
10. La tabla `tasks` tiene la restricción `tasks_priority_check` (`CHECK (priority IN ('alta','media','baja'))`): un `INSERT`/`UPDATE` con otro valor falla en BD aunque se salte la validación del modelo.
11. Todos los tests existentes siguen pasando sin modificarse (salvo los ajustes de fixtures descritos abajo).

## Comportamiento actual y propuesta

Actual (verificado en el código):

- `migrations/20260901000000_create_tasks.up.fizz`: `tasks` tiene `id`, `title`, `completed`, timestamps. No hay prioridad.
- `models/task.go`: `Task` con `Title` (`form:"title"`), `Completed` (`form:"-"`); `Validate` solo valida el título.
- `actions/tasks.go`:
  - `TasksIndex` lista con `Order("created_at asc, id asc")` y pasa `task` vacío y `errors`.
  - `TasksCreate` hace `c.Bind(task)` sobre el modelo y `ValidateAndCreate`; si hay errores re-renderiza `tasks/index.plush.html` con 422.
  - `TasksUpdate` bindea a un struct anónimo solo con `Title` (protección contra mass assignment, cubierta por `Test_TasksUpdate_PersistsTitleOnly`) y hace `ValidateAndUpdate`.
  - `TasksToggle` hace `tx.Update` sin validar.
- `templates/tasks/index.plush.html`: cada `<li class="task">` es un grid de 3 columnas (toggle, `.task-title`, `.task-actions`); `.task-status` es `sr-only` (posición absoluta, no ocupa celda). El título del input usa `errors.HasAny()` para `aria-invalid`.
- `public/assets/app.js`: el toggle optimista lee `.task-title` para el `aria-label` e inserta `.task-status` antes de `.task-actions`.
- `templates/application.plush.html` carga `/assets/app.css?v=3` (cache-busting manual por versión).
- La app aplica migraciones al arrancar (`models.Migrate()` en `cmd/app/main.go`) con las migraciones embebidas (`migrations/embed.go`, patrón `*.fizz *.sql`).

Propuesta: añadir la columna `priority` (texto, valores cerrados, protegida por `CHECK`) con migración fizz nueva, ordenar el listado por prioridad mediante una expresión `CASE` compartida, campo y validación en el modelo con valor por defecto aplicado en callbacks de pop, aceptación del campo en alta y edición, una partial con el selector reutilizada en ambos formularios, y una etiqueta visible en cada fila del listado con estilos nuevos en `app.css`.

Por qué texto y no entero (decisión técnica): con un `int`, un valor no numérico enviado a `c.Bind` provoca error de binding (500) en lugar de un 422 de validación; además los valores son legibles en BD y fixtures. Coste: el orden por prioridad necesita un `CASE` en el `ORDER BY` (ver «Acción»).

## Alcance

Incluido:

- Migración nueva que añade `priority` con default `media` y la restricción `CHECK` `tasks_priority_check`.
- Modelo: constantes, campo, validación, valor por defecto, etiqueta y constante de orden del listado.
- Ordenar el listado por prioridad (Alta → Media → Baja, desempate `created_at asc, id asc`).
- Acciones `TasksIndex`, `TasksCreate` (orden en el re-render 422), `TasksUpdate`.
- Templates: listado (etiqueta), formulario de alta y de edición (selector), partial nueva.
- CSS de la etiqueta y del selector; bump de `app.css?v=3` a `?v=4`.
- Fixtures y tests de modelo y acciones.

Excluido:

- Filtrar o agrupar visualmente (cabeceras por nivel) el listado por prioridad.
- Reordenar en cliente: tras cambiar la prioridad en edición, el nuevo orden se ve al volver a `/` (redirect 303), no hay JS nuevo.
- Cambiar la prioridad desde el listado sin entrar en edición.
- Modificar migraciones existentes o el seed (`20260901000001_seed_tasks.*`).
- Índice sobre `priority` o sobre la expresión `CASE` (listado sin paginación y de tamaño pequeño; ver Riesgos).
- API JSON, cambios de rutas, cambios en `app.js`.

## Diseño por capas

### Persistencia y migraciones

Nuevo par de migraciones fizz con timestamp posterior a las existentes:

- `migrations/20261014000000_add_priority_to_tasks.up.fizz`:
  ```
  add_column("tasks", "priority", "string", {"default": "media", "size": 10})
  sql("ALTER TABLE tasks ADD CONSTRAINT tasks_priority_check CHECK (priority IN ('alta', 'media', 'baja'))")
  ```
  Fizz (v1.14.4, verificado en `translators/postgres.go`) traduce la primera línea a `ALTER TABLE "tasks" ADD COLUMN "priority" VARCHAR (10) NOT NULL DEFAULT 'media';`. PostgreSQL rellena las filas existentes con el default antes de que se añada la constraint, así que la constraint valida filas ya conformes. La función `sql(...)` existe en fizz v1.14.4 (verificado: `bubbler.go` registra `"sql": f.RawSQL`). El orden importa: la constraint va después de la columna.
- `migrations/20261014000000_add_priority_to_tasks.down.fizz`:
  ```
  sql("ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_priority_check")
  drop_column("tasks", "priority")
  ```
  PostgreSQL elimina automáticamente las constraints de tabla que solo involucran la columna borrada (documentación de `ALTER TABLE ... DROP COLUMN`: «Indexes and table constraints involving the column will be automatically dropped as well»), así que `drop_column` bastaría. Aun así se elimina explícitamente para que la down sea legible, simétrica y no dependa de ese comportamiento.

No requiere tocar `migrations/embed.go` (el patrón `*.fizz` ya las incluye).

### Modelo (`models/task.go`)

- Constantes: `PriorityHigh = "alta"`, `PriorityMedium = "media"`, `PriorityLow = "baja"`.
- Campo: `Priority string \`json:"priority" db:"priority" form:"priority"\``.
- `func (t Task) PriorityLabel() string`: devuelve «Alta», «Media», «Baja»; para cualquier otro valor devuelve el valor tal cual (defensivo, no debería ocurrir).
- `func (t *Task) applyDefaultPriority()`: si `strings.TrimSpace(t.Priority) == ""` asigna `PriorityMedium`; en otro caso deja `t.Priority = strings.TrimSpace(t.Priority)`.
- Callbacks de pop v6.4.1 (interfaces `BeforeValidateable` y `BeforeSaveable`, verificadas en `callbacks.go`; `ValidateAndCreate` llama a `beforeValidate` antes de validar, y `Create`/`Update` llaman a `beforeSave`):
  - `func (t *Task) BeforeValidate(tx *pop.Connection) error { t.applyDefaultPriority(); return nil }`
  - `func (t *Task) BeforeSave(tx *pop.Connection) error { t.applyDefaultPriority(); return nil }`
  El `BeforeSave` es necesario porque un `tx.Create(task)` sin validar con `Priority == ""` insertaría `''` explícitamente (pop envía todas las columnas, el `DEFAULT` de BD no se aplicaría).
- `Validate`: además del título, si `t.Priority` no es uno de los tres valores, `verrs.Add("priority", "La prioridad no es válida.")`.
- Constante de orden del listado, junto a las de prioridad (vive en `models` porque depende del nombre de columna y de los valores, y ya es el paquete que exporta constantes de dominio como `TitleMaxLength`):
  ```go
  // TaskListOrder ordena por prioridad (alta, media, baja) y, dentro de cada nivel, por antigüedad.
  const TaskListOrder = "CASE priority WHEN 'alta' THEN 0 WHEN 'media' THEN 1 ELSE 2 END, created_at asc, id asc"
  ```
  No debe contener `;`: pop v6.4.1 (`sql_builder.go`, `regexpMatchNames = (?i).*;+.*`) descarta en silencio TODO el `ORDER BY` si lo encuentra, solo con un warning en el log. Gracias al `CHECK`, `ELSE 2` solo puede corresponder a `baja`.

### Acción (`actions/tasks.go`)

- Las dos consultas del listado (`TasksIndex` y el re-render 422 de `TasksCreate`) pasan de `tx.Order("created_at asc, id asc")` a `tx.Order(models.TaskListOrder)`. Son las únicas apariciones de esa cadena (verificado con grep en `actions/`). No se crea un helper de consulta: el repo no tiene ninguno y la constante elimina la duplicación relevante.
- `TasksIndex`: además, `c.Set("task", &models.Task{Priority: models.PriorityMedium})` para que el selector salga en «Media».
- `TasksCreate`: además del orden, `c.Bind(task)` ya recoge `priority` por la etiqueta `form`. En el re-render 422, `task.Priority` conserva el valor enviado (si era inválido, ninguna opción quedará marcada y el navegador mostrará la primera; aceptable).
- `TasksUpdate`: ampliar el struct de entrada con `Priority string \`form:"priority" json:"priority"\``. Tras asignar el título: `if p := strings.TrimSpace(input.Priority); p != "" { task.Priority = p }`. Así una petición sin `priority` conserva la actual (mantiene verde `Test_TasksUpdate_PersistsTitleOnly` y compañía) y se sigue sin aceptar `completed`, `id`, `created_at`. Actualizar el comentario de la función («actualiza el título y la prioridad…»).
- `TasksToggle` y `TasksDestroy`: sin cambios (la tarea cargada de BD ya tiene prioridad válida).
- Rutas (`actions/app.go`): sin cambios.

### Renderizado (Plush)

- Nueva partial `templates/tasks/_priority_select.plush.html` (se invoca como `<%= partial("tasks/priority_select.plush.html") %>`, mismo mecanismo que `partial("flash.plush.html")` → `_flash.plush.html`). Usa `task` y `errors` del contexto. La partial NO incluye el `<label>`; cada formulario pone el suyo (en alta `sr-only`, en edición visible). Contenido de la partial:
  ```html
  <select
    id="task-priority"
    name="priority"
    <%= if (len(errors.Get("priority")) > 0) { %>aria-invalid="true" aria-describedby="priority-error"<% } %>
  >
    <option value="alta"<%= if (task.Priority == "alta") { %> selected<% } %>>Alta</option>
    <option value="media"<%= if (task.Priority == "media") { %> selected<% } %>>Media</option>
    <option value="baja"<%= if (task.Priority == "baja") { %> selected<% } %>>Baja</option>
  </select>
  ```
  Orden de opciones Alta/Media/Baja.
- `templates/tasks/index.plush.html`:
  - En cada `<li>`, justo después de `<span class="task-title">…</span>` y antes del bloque `task-status`:
    ```html
    <span class="task-priority task-priority-<%= t.Priority %>"><span class="sr-only">Prioridad: </span><%= t.PriorityLabel() %></span>
    ```
    Va fuera de `.task-title` para que el `aria-label` que construye `app.js` con `.task-title.textContent` no incluya la prioridad. `app.js` inserta `.task-status` antes de `.task-actions`, es decir, después de la prioridad: coherente con el orden del template, no hay que tocar JS.
  - Orden y `app.js`: revisado `public/assets/app.js` (35 líneas). El toggle optimista solo cambia clases, textos y `aria-*` del `<li>` afectado y hace `fetch(form.action)`; no mueve nodos, no ordena ni depende de la posición por fecha. Como el orden no depende de `completed`, completar/reabrir no debe reordenar, así que no hay desajuste entre la vista optimista y la recarga. Ningún template ni test existente asume orden por fecha (`Test_TasksIndex_ListsTasks` solo usa `Contains`).
  - En `.new-task-row`, entre el input de título y el botón «Añadir»: `<label for="task-priority" class="sr-only">Prioridad</label>` + partial.
  - Cambiar en el input de título la condición `errors.HasAny()` por `len(errors.Get("title")) > 0` (para `autofocus aria-invalid aria-describedby`) y envolver el `<p id="title-error">` con la misma condición, para no marcar el título como inválido cuando el error es solo de prioridad.
  - Añadir tras el bloque de error del título:
    ```html
    <%= if (len(errors.Get("priority")) > 0) { %>
      <p id="priority-error" class="field-error" role="alert"><%= for (msg) in errors.Get("priority") { %><%= msg %><% } %></p>
    <% } %>
    ```
- `templates/tasks/edit.plush.html`:
  - Mismo cambio de condición `errors.HasAny()` → `len(errors.Get("title")) > 0` en el input y en el bloque `title-error` (el test `Test_TasksUpdate_EmptyTitle` sigue encontrando `aria-describedby="title-error"`).
  - Tras el bloque de error del título: `<label for="task-priority">Prioridad</label>`, la partial y el bloque `priority-error`.
  - Cambiar el texto de ayuda a «Hasta 200 caracteres.» (la frase «Los demás datos de la tarea no cambian» deja de ser cierta). Comprobado: ningún test busca esa frase.

### CSS (`public/assets/app.css`) y layout

- `.task`: `grid-template-columns: 44px minmax(0, 1fr) auto auto;` (y en `@media (max-width: 760px)`: `36px minmax(0, 1fr) auto auto;`). La nueva columna aloja la etiqueta; `.task-status` sigue siendo `sr-only` absoluto y no ocupa celda.
- `.task-priority`: `align-self: center; padding: 0.1rem 0.5rem; border-radius: 999px; font-size: 0.75rem; font-weight: 600; line-height: 1.5; white-space: nowrap; border: 1px solid var(--color-border);`.
- Variantes: `.task-priority-alta` (texto `var(--color-primary)`, fondo `#fdf0ee`, borde `#f1c9c3`), `.task-priority-media` (texto `var(--color-text)`, fondo `var(--color-sidebar)`), `.task-priority-baja` (texto `var(--color-muted)`, fondo transparente). Verificar contraste AA (≥ 4.5:1) del texto sobre fondo; `--color-primary` #b73526 sobre #fdf0ee cumple holgadamente (inferencia; comprobar con una herramienta).
- `.task-completed .task-priority { opacity: 0.6; }` para acompañar el atenuado de completadas.
- Estilo de `select` coherente con `input[type="text"]` (misma altura mínima 44px, borde `#92928a`, radio 6px, `font: inherit`, fondo blanco) y `select[aria-invalid="true"]` igual que el input inválido. En `.new-task-row select { flex-shrink: 0; width: auto; }`. En el bloque `@media (max-width: 380px)` comprobar que el `flex-wrap` existente coloca el selector y el botón en la segunda línea sin desbordar; si no, añadir `.new-task-row select { margin-left: 44px; }` o equivalente.
- `templates/application.plush.html`: `app.css?v=3` → `app.css?v=4`. `app.js` no cambia, su versión tampoco.

## Archivos relevantes

| Ruta | Responsabilidad / cambio |
|---|---|
| `migrations/20261014000000_add_priority_to_tasks.up.fizz` | Nuevo. Añade columna `priority` con default `media`. |
| `migrations/20261014000000_add_priority_to_tasks.down.fizz` | Nuevo. Elimina la columna. |
| `migrations/embed.go` | Sin cambios (ya embebe `*.fizz`). |
| `models/task.go` | Constantes, `TaskListOrder`, campo `Priority`, `PriorityLabel`, callbacks `BeforeValidate`/`BeforeSave`, validación. |
| `actions/tasks.go` | `TasksIndex` (orden y default en formulario), `TasksCreate` (orden en re-render 422), `TasksUpdate` (acepta prioridad opcional). |
| `.github/workflows/ci.yaml` | Sin cambios; referencia del comando de migraciones de test (líneas 51-52). |
| `templates/tasks/_priority_select.plush.html` | Nuevo. Selector reutilizable. |
| `templates/tasks/index.plush.html` | Etiqueta de prioridad por tarea, selector en alta, errores por campo. |
| `templates/tasks/edit.plush.html` | Selector en edición, errores por campo, texto de ayuda. |
| `templates/application.plush.html` | Bump `app.css?v=4`. |
| `public/assets/app.css` | Grid de `.task`, estilos de `.task-priority*` y `select`. |
| `fixtures/tasks.toml` | Añadir `priority` a dos filas del escenario «tres tareas». |
| `models/task_test.go` | Tests de modelo de prioridad. |
| `actions/tasks_test.go` | Tests de listado, alta y edición con prioridad. |
| `public/assets/app.js` | Sin cambios; verificar que el toggle optimista sigue funcionando. |

## Implementación paso a paso

1. `git status --short --branch`; confirmar que el árbol está limpio salvo este plan.
2. Crear las dos migraciones fizz del apartado «Persistencia».
3. Modificar `models/task.go` según «Modelo» (incluida `TaskListOrder`). Ejecutar `gofmt -w models/task.go`.
4. Modificar `actions/tasks.go`: `tx.Order(models.TaskListOrder)` en `TasksIndex` y en el re-render 422 de `TasksCreate`; default del formulario en `TasksIndex`; `TasksUpdate` y su comentario. Comprobar con `grep -n "created_at asc" actions/` que no queda ninguna consulta con el orden antiguo.
5. Crear `templates/tasks/_priority_select.plush.html`.
6. Modificar `templates/tasks/index.plush.html` y `templates/tasks/edit.plush.html` según «Renderizado».
7. Modificar `public/assets/app.css` y el bump en `templates/application.plush.html`.
8. Actualizar `fixtures/tasks.toml`: `priority = "alta"` en «Comprar material de oficina», `priority = "baja"` en «Llamar al gestor», sin `priority` en «Publicar la oferta de empleo» (así se ejercita el default de BD).
9. Añadir los tests de la sección siguiente.
10. Preparar la BD de test y ejecutar las validaciones (sección «Tests y validación»).
11. Reconstruir la app (`docker compose up -d --build`), que migrará `todo_development` al arrancar, y comprobar a mano en http://localhost:3000.
12. `git diff --check`, `git diff`, `git status --short`. Sin commit.

## Tests y validación

### Tests de modelo (`models/task_test.go`)

- `Test_Task_Priority_DefaultsToMedium`: `ValidateAndCreate(&Task{Title: "x"})` → sin errores, `task.Priority == PriorityMedium`, y recargando desde BD también `media`.
- `Test_Task_Priority_AcceptsValidValues`: para `alta`, `media`, `baja`, `ValidateAndCreate` sin errores y valor persistido.
- `Test_Task_Priority_Invalid`: `Priority: "urgente"` → `verrs.Get("priority")` contiene «La prioridad no es válida.» y `Count == 0`.
- `Test_Task_Priority_DefaultOnPlainCreate`: `ms.DB.Create(&Task{Title: "x"})` (sin validar) → recargada tiene `media` (demuestra `BeforeSave`).
- `Test_Task_PriorityLabel`: `Task{Priority: "alta"}.PriorityLabel() == "Alta"`, etc.
- `Test_Task_Priority_CheckConstraint_Create`: `ms.DB.Create(&Task{Title: "x", Priority: "urgente"})` (sin validar; `BeforeSave` no toca valores no vacíos) → `ms.Error(err)` y `ms.Contains(err.Error(), "tasks_priority_check")`; después `Count == 0`.
- `Test_Task_Priority_CheckConstraint_Update`: crear válida con `ValidateAndCreate`, poner `task.Priority = "urgente"` y `ms.DB.Update(task)` → error que menciona `tasks_priority_check`; recargada sigue con la prioridad original.
  Nota: `suite.Model` usa `ms.DB` sin transacción envolvente por test (trunca en `SetupTest`), así que un statement fallido no debería abortar las consultas siguientes del mismo test (inferencia). Si en la práctica aparece «current transaction is aborted», separar la comprobación posterior en otro test en lugar de hacerla tras el error.
- `Test_Task_ListOrder`: crear con `ValidateAndCreate` en este orden «B1» (`baja`), «A1» (`alta`), «M1» (`media`), «A2» (`alta`); `ms.DB.Order(TaskListOrder).All(&tasks)` → títulos `A1, A2, M1, B1`. Para que el desempate por `created_at` sea determinista, asignar `CreatedAt` explícito y creciente (`time.Now().Add(time.Duration(i) * time.Second)`) antes de crear; pop respeta un `CreatedAt` no cero (inferencia; si no, crear y luego fijar `created_at` con `ms.DB.RawQuery("UPDATE tasks SET created_at = ? WHERE id = ?", ...)`).

### Tests de acciones (`actions/tasks_test.go`)

Usar `url.Values` en los POST/PUT con prioridad para no depender del codificador de structs.

- `Test_TasksIndex_ShowsPriority`: `LoadFixture("tres tareas")`; el body contiene `task-priority-alta`, `task-priority-baja`, `task-priority-media`, y los textos `Alta`, `Baja`, `Media`, `Prioridad: `.
- `Test_TasksIndex_FormDefaultsToMedium`: `GET /` contiene `name="priority"` y `<option value="media" selected>`.
- `Test_TasksCreate_WithPriority`: POST `title=Con prioridad&priority=alta` → 303 y la tarea en BD tiene `alta`.
- `Test_TasksCreate_WithoutPriority_DefaultsToMedium`: POST solo `title` → 303 y `media`. (El existente `Test_TasksCreate_Valid` puede ampliarse con `as.Equal(models.PriorityMedium, task.Priority)`; preferible test nuevo para no tocar los existentes.)
- `Test_TasksCreate_InvalidPriority`: POST `title=X&priority=urgente` → 422, body contiene «La prioridad no es válida.», `Count == 0`, y el body NO contiene `aria-describedby="title-error"`.
- `Test_TasksEdit_PreselectsPriority`: tarea creada con `baja` → `GET /tasks/{id}/edit` contiene `<option value="baja" selected>`.
- `Test_TasksUpdate_ChangesPriority`: PUT `title=Igual&priority=alta` sobre tarea `media` → 303, `alta`, `Completed` intacto.
- `Test_TasksUpdate_WithoutPriority_KeepsIt`: tarea `baja`, PUT solo `title` → 303 y sigue `baja`.
- `Test_TasksUpdate_InvalidPriority`: PUT `title=Nuevo&priority=urgente` → 422, error visible, recargada conserva título y prioridad originales.
- `Test_TasksToggle_KeepsPriority`: tarea `alta`, toggle → sigue `alta`.
- `Test_TasksIndex_OrdersByPriority`: crear con `CreatedAt` explícito creciente las tareas «Tarea baja» (`baja`), «Tarea alta antigua» (`alta`), «Tarea media» (`media`), «Tarea alta reciente» (`alta`, completada). `GET /` y, con `strings.Index(body, título)`, comprobar que todas aparecen (índice ≥ 0) y que `alta antigua < alta reciente < media < baja`. Demuestra el orden por nivel, el desempate por fecha y que `completed` no influye. Usar títulos que no sean prefijo unos de otros.
- `Test_TasksCreate_Invalid_KeepsPriorityOrder`: crear «Tarea baja» (`baja`) y luego «Tarea alta» (`alta`); POST `title=` (vacío) → 422 y en el body `strings.Index("Tarea alta") < strings.Index("Tarea baja")`. Cubre la segunda consulta del listado.

Los tests existentes no se modifican; deben seguir verdes. Atención especial a `Test_TasksUpdate_PersistsTitleOnly`, `Test_TasksUpdate_EmptyTitle` (busca `aria-describedby="title-error"`) y `Test_TasksIndex_ListsTasks`.

### Preparación del entorno local (Windows 11, Git Bash, sin Buffalo CLI)

Fuente del procedimiento: el job `test` de `.github/workflows/ci.yaml` define el servicio `postgres:17.6-alpine` con `POSTGRES_DB: todo_test` y `GO_ENV: test`, y migra en el paso «Migraciones» (líneas 51-52: `go install github.com/gobuffalo/pop/v6/soda@v6.4.1` y `soda migrate -e test`) antes de `go test -p 1 ./...`. Los pasos locales replican eso.

Los tests usan `GO_ENV=test`, que según `database.yml` conecta a `todo_test` en `DB_HOST` (por defecto `127.0.0.1`), usuario/clave `postgres/postgres`. `suite.NewActionWithFixtures` / `NewModelWithFixtures` NO migran: el esquema debe existir ya (CI lo hace con `soda migrate -e test`). `suite` trunca todas las tablas en cada `SetupTest`, así que las filas del seed no interfieren.

PostgreSQL corre en el contenedor `desarrollo-agentico-todo-03-db-1` publicado en `127.0.0.1:5432`.

```bash
REPO=/c/Users/bacollaguazo/Documents/GitHub/desarrollo-agentico-todo-03

# 1. Crear la BD de test si no existe (idempotente)
docker exec desarrollo-agentico-todo-03-db-1 psql -U postgres -tAc "SELECT 1 FROM pg_database WHERE datname='todo_test'" | grep -q 1 \
  || docker exec desarrollo-agentico-todo-03-db-1 createdb -U postgres todo_test

# 2. Aplicar migraciones de test con soda (mismo comando que .github/workflows/ci.yaml, líneas 51-52)
go install github.com/gobuffalo/pop/v6/soda@v6.4.1
cd "$REPO" && GO_ENV=test "$(go env GOPATH)/bin/soda" migrate -e test

# 3. Validaciones
cd "$REPO" && gofmt -l .                     # debe no imprimir nada
cd "$REPO" && go vet ./...
cd "$REPO" && GO_ENV=test go test -p 1 ./...
cd "$REPO" && go build ./...
```

Notas:

- Si `go install` de soda falla en Windows (posible por dependencias cgo de SQLite; incierto), alternativa: migrar arrancando la app contra test en otro puerto, p. ej. `cd "$REPO" && GO_ENV=test PORT=3001 go run ./cmd/app`, esperar a que aparezca el log de arranque y detenerla (Ctrl+C). `models.Migrate()` se ejecuta antes de servir. Que Buffalo lea `PORT` es una inferencia; si no, comprobar `ADDR`/`PORT` en buffalo y evitar el conflicto con el contenedor `app` en 3000.
- Si ya se habían aplicado migraciones antiguas en `todo_test`, el paso 2 solo aplica la nueva; es idempotente.
- `-p 1` es obligatorio: los paquetes `actions` y `models` comparten `todo_test` y se truncan mutuamente.

### Verificación manual

1. `docker compose up -d --build`; esperar healthcheck; `docker compose logs app` sin errores de migración.
2. En http://localhost:3000: las tres tareas del seed muestran «Media»; crear una con «Alta» (debe aparecer la primera) y otra con «Baja» (la última); editar una y cambiar su prioridad (al volver al listado cambia de posición); completar/reabrir una tarea y ver que la etiqueta persiste, el estado se actualiza y la tarea no cambia de sitio al recargar.
3. Comprobar la constraint: `docker exec desarrollo-agentico-todo-03-db-1 psql -U postgres -d todo_development -c "\d tasks"` debe listar `tasks_priority_check`.
4. Revisar a 375px y 760px de ancho que la fila no desborda y el formulario de alta se reparte bien.
5. Recarga forzada para confirmar que se sirve `app.css?v=4`.

## Riesgos y recuperación

- Default solo en BD vs. inserción explícita: pop inserta todas las columnas, por lo que `Priority == ""` sobrescribiría el `DEFAULT`. Mitigado con `BeforeSave`; cubierto por `Test_Task_Priority_DefaultOnPlainCreate`. Si `BeforeSave` fallara, la `CHECK` rechazaría `''` con error 500 en vez de guardar un valor inválido.
- `CHECK` y errores 500: cualquier escritura que salte la validación con un valor inválido devuelve error de BD (500 en la acción). Es el comportamiento deseado como última barrera; las rutas normales validan antes y devuelven 422. Si en el futuro se añade un nivel, hay que cambiar en la misma entrega: constantes, `Validate`, `TaskListOrder`, la partial, el CSS y una migración que sustituya la constraint (`DROP CONSTRAINT` + `ADD CONSTRAINT`).
- `ORDER BY` descartado en silencio: pop v6.4.1 omite todo el `ORDER BY` si la cadena contiene `;` (solo un warning en log). `TaskListOrder` no debe llevarlo; `Test_TasksIndex_OrdersByPriority` lo detectaría.
- Duplicación del orden: si alguien añade otra consulta del listado con un orden literal distinto, la vista normal y la de error serían incoherentes. Mitigado con la constante y con `Test_TasksCreate_Invalid_KeepsPriorityOrder`.
- Rendimiento del `CASE`: impide usar un índice para ordenar; irrelevante con el volumen actual (listado completo sin paginación). Si crece, índice de expresión sobre el `CASE` o pasar a entero.
- Cambio de comportamiento percibido: las tareas ya no salen en orden de creación; una tarea nueva `media` o `baja` puede aparecer en mitad de la lista y no al final, junto al formulario. Es la decisión D4 del usuario.
- Binding: si `c.Bind` no rellena `Priority` (etiqueta `form` mal escrita), las tareas se crearían siempre en `media` sin fallo visible; lo detecta `Test_TasksCreate_WithPriority`.
- Mass assignment en `TasksUpdate`: se añade solo `Priority` al struct de entrada; no añadir `Completed` ni otros campos.
- Accesibilidad: cambiar `errors.HasAny()` por condiciones por campo altera el marcado de error; `Test_TasksUpdate_EmptyTitle` y `Test_TasksCreate_InvalidPriority` lo protegen. Dos elementos con `role="alert"` simultáneos son posibles si fallan título y prioridad a la vez; aceptable.
- Layout: la nueva columna del grid puede estrechar títulos largos en móvil; verificación manual a 375px.
- Caché del navegador: sin el bump `?v=4` se verían etiquetas sin estilo.
- Migración en producción/dev: `ADD COLUMN ... NOT NULL DEFAULT` es instantánea en PostgreSQL ≥ 11 para defaults constantes (fuente: notas de versión de PostgreSQL 11; no verificado en este repo). Añadir la `CHECK` recorre la tabla para validar las filas existentes (todas `media` tras el default), con un bloqueo breve en tablas pequeñas. Recuperación: `soda migrate down -e <env>` (o aplicar `ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_priority_check;` y `ALTER TABLE tasks DROP COLUMN priority;`) y revertir el código; perder la columna borra las prioridades asignadas.
- Imagen Docker: la app embebe migraciones y templates; si el contenedor no se reconstruye (`--build`), seguirá sin la columna y sin cambios visuales.
