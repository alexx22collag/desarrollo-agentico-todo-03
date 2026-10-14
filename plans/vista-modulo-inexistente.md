# Bug: una vista con un módulo inexistente da 404 en vez de volver a «Todas»

## Encargo y resultado esperado

Issue #3. Cuando el parámetro oculto `view` contiene un UUID con formato válido pero sin fila en `modules`, varias pantallas responden 404 o generan enlaces/redirecciones a `/?module=<uuid>`, que dan 404.

Decisiones ya tomadas (del encargo):

- Una vista que apunta a un módulo inexistente se trata como «Todas» (`view = ""`) en **todos** los caminos que reciben `view`.
- `GET /?module=<uuid inexistente>` **sigue** dando 404 (`TasksIndex` no cambia).

Resultado esperado tras el arreglo, con `view=00000000-0000-4000-8000-000000000000` (a partir de aquí, «vista fantasma»):

| Petición | Hoy | Tras el arreglo |
|---|---|---|
| `POST /tasks` con título vacío | 404 «módulo no encontrado» | 422 con «El título es obligatorio.», vista «Todas» (`<h1>Tareas</h1>`) |
| `POST /tasks` válido | 303 a `/?module=none` o al módulo de la tarea | 303 a `/` |
| `GET /tasks/{id}/edit?view=…` | 200, «Cancelar» y «Volver a la lista» a `/?module=<fantasma>`; campo oculto `view` con el fantasma | 200, ambos enlaces a `/`; campo oculto `view=""`; «Todas» marcada en la barra lateral |
| `PUT /tasks/{id}` inválido (422) | 422, enlaces a `/?module=<fantasma>` | 422, enlaces a `/`, `view=""` |
| `PUT /tasks/{id}` válido | 303 a `/?module=none` o al módulo de la tarea | 303 a `/` |
| `PUT /tasks/{id}/toggle` | 303 a `/?module=<fantasma>` (que da 404) | 303 a `/` |
| `DELETE /tasks/{id}` | 303 a `/?module=<fantasma>` (que da 404) | 303 a `/` |
| `POST /modules` inválido | 422 en «Todas» (ya correcto) | Igual (refactor sin cambio de comportamiento) |
| `GET /?module=<fantasma>` | 404 | 404 (sin cambios) |

Criterios de aceptación del issue: (1) test de regresión del 422 de `POST /tasks` que falle antes del arreglo; (2) test de los enlaces de vuelta de `GET /tasks/{id}/edit`; (3) `Test_TasksIndex_UnknownModule_NotFound` en verde; (4) sin cambios en rutas ni plantillas salvo lo imprescindible (este plan no toca ninguna de las dos).

## Evidencia y causa raíz

### Hechos observados en el código (rama `bug/vista-modulo-inexistente`, árbol limpio, igual a `origin/main`)

- `actions/views.go`
  - `parseView` (l. 23-35) solo valida **formato**: `""`, `"none"` o un UUID distinto de `uuid.Nil`. No consulta la base de datos, así que la vista fantasma es «válida» para él.
  - `listPath` (l. 39-48) construye `/?module=<uuid>` para cualquier UUID con formato válido.
  - `taskViewAfterWrite` (l. 53-62): si la vista no es `""`, vuelve al módulo de la tarea o a `none`. Con vista fantasma y tarea sin módulo devuelve `none`.
  - `setLayout` (l. 93-133) normaliza solo por formato y hace `c.Set("view", view)`; ese valor se propaga a los campos ocultos `name="view"` de `templates/application.plush.html:25`, `templates/tasks/index.plush.html:23,51,67`, `templates/tasks/edit.plush.html:9` y al enlace de edición `index.plush.html:43`. Con vista fantasma, el fantasma se perpetúa en todos los formularios.
  - `resolveView` (l. 137-156) sí comprueba la existencia: devuelve `found=false` si el formato no es válido o si `tx.Find` da `sql.ErrNoRows`; otros errores se propagan.
  - `renderTaskList` (l. 169-198) responde **404 «módulo no encontrado»** cuando `resolveView` no encuentra la vista (l. 174-176).
- `actions/tasks.go`
  - `TasksIndex` (l. 17-26): pasa `?module=` a `renderTaskList`; el 404 con módulo inexistente es el comportamiento deseado.
  - `TasksCreate` (l. 38-41): normaliza `view` solo con `parseView` y la pasa a `renderTaskList` en los 422 (l. 47 y 57) → **404**. En éxito, `listPath(taskViewAfterWrite(view, task))` (l. 61).
  - `TasksEdit` (l. 73-81): `setLayout(c, tx, view)` y `backPath = listPath(view)` sin comprobar existencia → enlaces a `/?module=<fantasma>`.
  - `TasksUpdate` (l. 106-109): misma normalización solo por formato; `renderTaskEdit` (l. 140-148) reproduce el problema de `backPath` y del campo oculto en los 422; el éxito redirige con `taskViewAfterWrite` (l. 136).
  - `TasksToggle` (l. 164) y `TasksDestroy` (l. 181): `c.Redirect(303, listPath(c.Param("view")))` → redirección a `/?module=<fantasma>`, que da 404 al seguirla.
- `actions/modules.go` `ModulesCreate` (l. 32-42): en el 422 ya usa `resolveView` y cae a `""` si `!found`. Es el patrón correcto, escrito en línea.
- `actions/app.go`: no hay ruta para borrar módulos. Por tanto, hoy la vista fantasma solo aparece por un parámetro manipulado/obsoleto (p. ej. un módulo borrado a mano en la base de datos, o una URL antigua), no por un flujo normal de la UI. Esto no cambia la decisión, pero explica por qué no se detectó antes.
- Tests existentes en `actions/tasks_test.go` y `actions/modules_test.go`: estilo `as.HTML(path).Post/Put/Get/Delete(url.Values{...})`, helpers `createModule`, `createTaskIn`, `inModule`, `taskForEditing`, `createTask`. Los tests no envían `authenticity_token` y obtienen 303/422, así que el middleware CSRF no bloquea en `GO_ENV=test` (estoy haciendo una inferencia a partir de que esos tests pasan en CI; no he leído el código de `github.com/gobuffalo/middleware/csrf`). Los nuevos tests deben seguir el mismo estilo, sin token.
- `Test_TasksIndex_UnknownModule_NotFound` (`tasks_test.go:472`) ya cubre el criterio 3.

### Causa raíz

La vista se valida en dos niveles distintos y los handlers de escritura solo aplican el primero: `parseView` (formato) en lugar de `resolveView` (existencia). La comprobación de existencia solo ocurre dentro de `renderTaskList`, y ahí se traduce en 404 (correcto para `GET /`, incorrecto para los re-render de formularios). Los caminos que no pasan por `renderTaskList` (`TasksEdit`, `renderTaskEdit`, redirecciones de `TasksCreate`/`TasksUpdate`/`TasksToggle`/`TasksDestroy`) nunca comprueban existencia y construyen URLs hacia la vista fantasma.

### Reproducción mínima (no ejecutada en esta planificación: no se arranca la app ni docker)

1. `POST /tasks` con `title=` y `view=00000000-0000-4000-8000-000000000000` → 404 (hoy).
2. Crear una tarea y abrir `GET /tasks/<id>/edit?view=00000000-0000-4000-8000-000000000000` → enlaces a `/?module=00000000-…`.
3. `PUT /tasks/<id>/toggle` con el mismo `view` → `Location: /?module=00000000-…`.

Los tests de la sección «Tests y validación» automatizan estas reproducciones.

### Incertidumbres

- No he ejecutado los tests ni la app; los resultados «hoy» de la tabla se derivan de leer el código (estoy haciendo una inferencia, aunque directa).
- Comportamiento exacto del middleware CSRF en test: inferido de los tests existentes (ver arriba).

## Alcance

### Incluido

- Nueva función auxiliar en `actions/views.go` que resuelve la vista y devuelve `""` si no existe (ver paso 1).
- Usarla en `TasksCreate`, `TasksEdit`, `TasksUpdate` (cubre `renderTaskEdit`), `TasksToggle`, `TasksDestroy` y `ModulesCreate`.
- Tests de regresión en `actions/tasks_test.go` y `actions/modules_test.go`.

Justificación de incluir toggle/borrar: la decisión dice «todos los caminos que reciben `view`»; además, tras una acción con éxito el usuario acabaría en un 404, que es el mismo defecto. Justificación de incluir `ModulesCreate`: es un refactor de 6 líneas a 3 que elimina la única copia del patrón y evita que la lógica diverja; no cambia su comportamiento y se protege con un test nuevo.

### Excluido

- `TasksIndex` y `renderTaskList`: sin cambios (el 404 de `GET /?module=<fantasma>` y de `?module=abc` se mantiene).
- `parseView`, `listPath`, `taskViewAfterWrite`, `setLayout`, `resolveView`: sin cambios de firma ni de comportamiento.
- Rutas (`actions/app.go`) y plantillas (`templates/**`): sin cambios. Al normalizar `view` antes de `setLayout`, los campos ocultos y enlaces ya reciben `""` sin tocar las plantillas.
- Validación de `module_id` inexistente (ya cubierta por `Test_TasksCreate_UnknownModule`).
- Borrado de módulos u otras funcionalidades.

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `actions/views.go` | Helpers de vista (`parseView`, `listPath`, `resolveView`, `setLayout`, `renderTaskList`). Aquí va la nueva función auxiliar. |
| `actions/tasks.go` | Handlers `TasksCreate`, `TasksEdit`, `TasksUpdate`, `renderTaskEdit`, `TasksToggle`, `TasksDestroy`. Se cambia cómo obtienen `view`. |
| `actions/modules.go` | `ModulesCreate`: pasa a usar la función auxiliar. |
| `actions/tasks_test.go` | Tests de regresión de tareas. |
| `actions/modules_test.go` | Test de regresión de `ModulesCreate` con vista fantasma. |
| `templates/tasks/edit.plush.html`, `templates/tasks/index.plush.html`, `templates/application.plush.html` | Solo lectura: consumen `view` y `backPath`. No se modifican. |

## Implementación paso a paso

### 1. Función auxiliar en `actions/views.go`

Añadir, justo después de `resolveView`:

```go
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
```

Nota: `resolveView` ya devuelve `view = ""` cuando `!found`, pero se mantiene la comprobación explícita para no depender de ese detalle.

### 2. `TasksCreate` (`actions/tasks.go`)

Sustituir:

```go
view, _, ok := parseView(c.Param("view"))
if !ok {
	view = ""
}
```

por:

```go
view, err := viewOrAll(tx, c.Param("view"))
if err != nil {
	return err
}
```

Atención: más abajo existe `verrs, err := tx.ValidateAndCreate(task)`; como `verrs` es nueva, `:=` sigue compilando. La variable `ok` se sigue declarando en `moduleID, ok := parseModuleParam(...)`; comprobar que compila (si `ok` deja de estar declarado antes, esa línea con `:=` lo declara sin problema).

Efecto: los dos `renderTaskList(..., http.StatusUnprocessableEntity)` reciben `""` → 422 en «Todas»; el redirect de éxito pasa a ser `/` (porque `taskViewAfterWrite("", task)` devuelve `""`).

### 3. `TasksEdit`

Sustituir el bloque `parseView` (l. 73-76) por `view, err := viewOrAll(tx, c.Param("view"))` con su `if err != nil { return err }`, manteniéndolo **después** del `tx.Find` de la tarea (el 404 por tarea inexistente conserva prioridad). `setLayout` y `backPath = listPath(view)` quedan igual y ya reciben `""`.

### 4. `TasksUpdate`

Sustituir el bloque `parseView(input.View)` (l. 106-109) por `view, err := viewOrAll(tx, input.View)` + `if err != nil { return err }`, en el mismo sitio (después de `c.Bind`). Más abajo `verrs, err := tx.ValidateAndUpdate(task)` sigue siendo válido con `:=` porque `verrs` es nueva. `renderTaskEdit` no se toca: recibe ya la vista normalizada.

### 5. `TasksToggle` y `TasksDestroy`

Tras el `tx.Find` de la tarea (para conservar el 404 por tarea inexistente) y antes de la mutación, añadir:

```go
view, err := viewOrAll(tx, c.Param("view"))
if err != nil {
	return err
}
```

y cambiar el redirect final a `listPath(view)`. Se coloca antes de `tx.Update`/`tx.Destroy` para que, si la consulta falla, no se haya mutado nada (aunque `popmw.Transaction` haría rollback igualmente). Ojo con `if err := tx.Update(task); err != nil` / `if err := tx.Destroy(task); err != nil`: usan `:=` en ámbito del `if`, no chocan con el `err` exterior.

### 6. `ModulesCreate` (`actions/modules.go`)

Sustituir:

```go
view, _, _, found, err := resolveView(tx, input.View)
if err != nil {
	return err
}
if !found {
	view = ""
}
```

por:

```go
view, err := viewOrAll(tx, input.View)
if err != nil {
	return err
}
```

Dentro del bloque `if verrs.HasAny()`, `err` ya existe en el ámbito exterior (`verrs, err := tx.ValidateAndCreate(module)`); `view, err := ...` dentro del bloque declara `view` nueva y sombrea `err` — compila, pero para evitar el aviso de sombreado se puede escribir `view, verr := ...`. Elegir lo que deje `go vet` limpio (`go vet` por defecto no comprueba sombreado, así que cualquiera de las dos vale; preferir no sombrear).

### 7. No tocar

`TasksIndex`, `renderTaskList`, `setLayout`, plantillas y rutas.

## Tests y validación

### Tests nuevos

Declarar en `actions/tasks_test.go` (nivel de paquete):

```go
// unknownView es un UUID con formato válido que no corresponde a ningún módulo.
const unknownView = "00000000-0000-4000-8000-000000000000"
```

Tests en `actions/tasks_test.go` (estilo del archivo, sin `authenticity_token`):

1. `Test_TasksCreate_UnknownView_Invalid_FallsBackToAll` — **criterio 1**. `POST /tasks` con `url.Values{"title": {""}, "view": {unknownView}}`. Esperado: `422`; cuerpo contiene `El título es obligatorio.` y `<h1>Tareas</h1>`; `NotContains(body, unknownView)` (ningún campo oculto lo perpetúa); recuento de tareas = 0. **Hoy falla**: responde 404.
2. `Test_TasksEdit_UnknownView_LinksToAll` — **criterio 2**. Crear tarea con `taskForEditing(false)`; `GET /tasks/%s/edit?view=` + `unknownView`. Esperado: `200`; contiene `<a href="/" class="back-link">` y `<a href="/" class="button">Cancelar</a>`; contiene `name="view" value=""`; contiene `href="/" class="nav-link" aria-current="page"` («Todas» marcada); `NotContains(body, unknownView)`. **Hoy falla**: los enlaces apuntan a `/?module=00000000-…`.
3. `Test_TasksUpdate_UnknownView_Invalid_LinksToAll`. `taskForEditing(false)`; `PUT /tasks/%s` con `{"title": {""}, "view": {unknownView}}`. Esperado: `422`; `El título es obligatorio.`; `<a href="/" class="button">Cancelar</a>`; `NotContains(body, unknownView)`; título en BD sin cambios. **Hoy falla** (enlaces y campo oculto con el fantasma).
4. `Test_TasksUpdate_UnknownView_RedirectsToAll`. Tarea sin módulo; `PUT` con `{"title": {"Nuevo"}, "view": {unknownView}}`. Esperado: `303`, `Location: /`. **Hoy falla**: `Location: /?module=none`.
5. `Test_TasksCreate_UnknownView_RedirectsToAll`. `POST /tasks` con `{"title": {"X"}, "view": {unknownView}}`. Esperado: `303`, `Location: /`; tarea creada. **Hoy falla**: `Location: /?module=none`.
6. `Test_TasksToggle_UnknownView_RedirectsToAll`. Tarea creada; `PUT /tasks/%s/toggle` con `{"view": {unknownView}}`. Esperado: `303`, `Location: /`; tarea completada. **Hoy falla**: `Location: /?module=00000000-…`.
7. `Test_TasksDestroy_UnknownView_RedirectsToAll`. Tarea creada; `POST /tasks/%s` con `{"_method": {"DELETE"}, "view": {unknownView}}` (igual que `Test_TasksDestroy_ReturnsToView`). Esperado: `303`, `Location: /`; recuento 0. **Hoy falla**: `Location: /?module=00000000-…`.
8. `Test_ViewOrAll` (prueba directa de la función, como `Test_ListPath_RejectsArbitraryValues`). Con un módulo creado `work`: `viewOrAll(as.DB, "")` → `""`; `"none"` → `"none"`; `work.ID.String()` → `work.ID.String()`; `unknownView` → `""`; `"abc"` → `""`; sin error en todos. **Hoy no compila** (la función no existe), lo que cuenta como fallo previo.

Test en `actions/modules_test.go`:

9. `Test_ModulesCreate_Invalid_UnknownView_FallsBackToAll`. `POST /modules` con `{"name": {""}, "view": {"00000000-0000-4000-8000-000000000000"}}` (o reutilizar `unknownView`, que está en el mismo paquete). Esperado: `422`, contiene `El nombre del módulo es obligatorio.` y `<h1>Tareas</h1>`. **Hoy pasa**: es una red de seguridad para el refactor del paso 6, no un test de regresión.

Tests existentes que deben seguir en verde sin modificarse (en especial): `Test_TasksIndex_UnknownModule_NotFound` (**criterio 3**), `Test_TasksIndex_InvalidModule_NotFound`, `Test_TasksEdit_PreselectsModule`, `Test_TasksCreate_WithModule`, `Test_TasksCreate_FromModuleView_OtherModule`, `Test_TasksToggle_ReturnsToView`, `Test_TasksToggle_FormPost_ReturnsToView`, `Test_TasksDestroy_ReturnsToView`, `Test_ModulesCreate_Invalid_KeepsView`.

Orden recomendado para demostrar el fallo previo: añadir primero los tests 1-7 y 9 (sin el 8), ejecutar y comprobar que 1-7 fallan y 9 pasa; después implementar los pasos 1-6, añadir el test 8 y ejecutar la validación completa.

### Validación (la de CI)

En Windows (PowerShell), desde la raíz del repo, con el contenedor de PostgreSQL del proyecto en marcha (red `desarrollo-agentico-todo-03_default`). Si hay archivos nuevos, antes `git add -N <archivo>` para que `git ls-files` los incluya (en este plan no se crean archivos de código nuevos; solo se modifican existentes):

```powershell
docker run --rm --network desarrollo-agentico-todo-03_default -e GO_ENV=test -e DB_HOST=db -v "${PWD}:/repo:ro" -v todo03-gomod:/go/pkg/mod golang:1.26 bash -c 'cp -r /repo /src && cd /src && git config --global --add safe.directory "*" && git ls-files | while read f; do sed -i "s/\r$//" "$f"; done && gofmt -l . && go vet ./... && go test -p 1 -count=1 ./... && go build ./...'
```

Criterio de éxito: `gofmt -l .` sin salida, `go vet` sin avisos, todos los tests en verde, build correcto.

Comprobación manual opcional (en la app ya arrancada, sin reiniciarla): repetir los pasos 2 y 3 de la reproducción del issue y verificar 422 en «Todas» y enlaces a `/`; `GET /?module=00000000-0000-4000-8000-000000000000` sigue en 404.

## Riesgos y recuperación

- **Cambio de comportamiento en redirecciones de éxito** (`TasksCreate`/`TasksUpdate` con vista fantasma pasan de `/?module=none` o el módulo de la tarea a `/`). Es consecuencia directa de la decisión «tratar como Todas en todos los caminos»; queda cubierto por los tests 4 y 5. Con vistas válidas no cambia nada (tests existentes).
- **Una consulta extra** (`SELECT` por PK en `modules`) en create/update/toggle/destroy/edit, solo cuando `view` es un UUID. Coste despreciable.
- **Errores de BD en la consulta de la vista** ahora hacen fallar toggle/destroy/edit con 500, cuando antes no consultaban. Se coloca antes de la mutación y dentro de la transacción de `popmw`, así que no hay escrituras parciales.
- **Sombreado de `err`** en `ModulesCreate` y `TasksCreate`: revisar que compila y que `go vet` queda limpio (paso 2 y 6).
- **Regresión del 404 de `GET /`**: mitigada al no tocar `TasksIndex` ni `renderTaskList`; `Test_TasksIndex_UnknownModule_NotFound` lo vigila.
- **Recuperación**: el cambio es local a `actions/views.go`, `actions/tasks.go`, `actions/modules.go` y sus tests; se revierte con `git revert` del commit sin migraciones ni cambios de datos.

## Decisiones abiertas

Ninguna que bloquee. Puntos que he decidido por inferencia a partir de «todos los caminos» y que conviene que el revisor confirme de un vistazo:

1. Redirecciones de éxito de alta/edición con vista fantasma → `/` (en vez de saltar al módulo de la tarea o a «Sin módulo»).
2. Redirecciones de alternar/borrar con vista fantasma → `/`.
3. `ModulesCreate` pasa a usar la función auxiliar (refactor sin cambio de comportamiento).
