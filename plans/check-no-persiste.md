# Bug: el check de una tarea no persiste al recargar la página

## Encargo y resultado esperado

Bug reportado por el usuario: "no permite hacer check en las tareas; se marca la tarea como completada pero al recargar la página se desmarca".

Resultado esperado tras el arreglo:

- Al pulsar el botón de completar/reabrir de una tarea en `/`, la interfaz cambia al instante (comportamiento optimista actual) **y** el cambio queda guardado en la columna `tasks.completed`.
- Al recargar la página, la tarea conserva el estado elegido.
- Si el servidor rechaza o no recibe la petición, la interfaz no se queda mostrando un estado que no se ha guardado: se recarga para volver a mostrar lo que hay en la BD.
- No cambia nada más: rutas, acción `TasksToggle`, plantilla del formulario, CSRF y resto de formularios quedan igual.

## Evidencia y causa raíz

### Hechos observados (código del repositorio, rama `bug/check-no-persiste` desde `origin/main`)

1. `templates/tasks/index.plush.html:16-18`: cada tarea tiene un formulario `action="/tasks/<id>/toggle" method="POST" data-task-toggle` con los campos ocultos `authenticity_token` y `_method=PUT`. Si se envía como formulario HTML normal, la petición es correcta.
2. `public/assets/app.js:9-13`: el manejador global de `submit` intercepta los formularios `[data-task-toggle]` y llama a `event.preventDefault()`, así que el navegador **no** envía el formulario.
3. `public/assets/app.js:15-32`: cambia la interfaz de forma optimista (clase `task-completed`, etiqueta, `aria-pressed`, insignia "Completada").
4. `public/assets/app.js:34`: la única petición al servidor es `fetch(form.action);`. Sin segundo argumento, `fetch` hace un **GET** sin cuerpo: no envía `_method`, ni `authenticity_token`, ni la cabecera `X-CSRF-Token`. Tampoco mira la respuesta.
5. `actions/app.go:53`: la ruta solo existe como `app.PUT("/tasks/{task_id}/toggle", TasksToggle)`. No hay GET para esa ruta, así que el GET acaba en `app.ServeFiles("/", ...)` (`actions/app.go:57`) y devuelve 404.
6. La sustitución de método de Buffalo (`buffalo@v1.1.4/method_override.go:15-20`) solo actúa si la petición es `POST` y trae `_method`. Un GET nunca se convierte en PUT.
7. El middleware CSRF (`actions/app.go:45`, `gobuffalo/middleware@v1.0.0/csrf/csrf.go:84-112`) exige un token en los métodos que no son seguros. Lo busca primero en la cabecera `X-CSRF-Token` (`csrf.go:27,225`) y, si no está, en el campo de formulario `authenticity_token` (`csrf.go:229-237`, también en multipart). Con `GO_ENV=test` no se ejecuta (`csrf.go:50-55`), así que los tests de acciones no comprueban CSRF.
8. Historia: el `fetch(form.action)` llegó en `5fdd6c3 feat(ui): improve task completion feedback`. Antes de ese commit el formulario se enviaba de forma nativa. `d8dc18d` solo cambió las etiquetas y los atributos ARIA. Así que el bug se introdujo en `5fdd6c3`.
9. Los tests actuales (`actions/tasks_test.go:62-90`) hacen `PUT` directo contra `/tasks/{id}/toggle` y pasan: cubren el servidor, no la petición real que hace el navegador. No hay tests de JavaScript en el repositorio (no hay `package.json` ni un runner JS).

### Reproducción por HTTP contra la app en marcha (http://localhost:3000)

Primero se comprobó que el `app.js` servido es idéntico al de esta rama: `curl -s http://localhost:3000/assets/app.js | diff - public/assets/app.js` no da diferencias. La imagen de Docker se construyó desde otra rama (`bacollaguazo/trabajo`, que añade prioridades y la columna `priority` con valor por defecto), pero esa rama no toca `app.js` ni `TasksToggle`.

Se creó una tarea temporal en `todo_development` (después se borró), se cogieron la cookie de sesión y el token de `GET /`, y se lanzaron estas peticiones:

| Petición | Equivale a | HTTP | `completed` en BD |
|---|---|---|---|
| `GET /tasks/{id}/toggle` con cookie, `Accept: */*` | **lo que hace hoy `fetch(form.action)`** | **404** | **sigue `f`** (no persiste) |
| `POST` multipart con `authenticity_token` + `_method=PUT`, con cookie | `fetch(form.action, {method:"POST", body:new FormData(form)})` | 303 → `/` | pasa a `t` |
| `POST` urlencoded con el mismo token + `_method=PUT` | envío nativo del formulario | 303 | vuelve a `f` (alterna) |
| `POST` multipart solo con `_method=PUT`, sin token | | 403 | sin cambio |
| `PUT` sin token | | 403 | sin cambio |
| `PUT` con cabecera `X-CSRF-Token` | otra opción | 303 | alterna |

Comandos usados (en Git Bash; `$J` es un cookie jar y `$TOK` el token sacado del HTML de `/`):

```bash
curl -s -c $J -b $J http://localhost:3000/ -o /dev/null
curl -s -i -b $J -H "Accept: */*" http://localhost:3000/tasks/$ID/toggle            # 404, no persiste
curl -s -o /dev/null -w "%{http_code}\n" -b $J -F "authenticity_token=$TOK" -F "_method=PUT" \
     http://localhost:3000/tasks/$ID/toggle                                           # 303, persiste
docker exec desarrollo-agentico-todo-03-db-1 psql -U postgres -d todo_development -tc \
     "select completed from tasks where id='$ID'"
```

### Causa raíz

`public/assets/app.js:34` (`fetch(form.action);`) sustituye el envío nativo del formulario por un GET que no coincide con ninguna ruta (404). El servidor nunca recibe el PUT con CSRF, así que la BD no cambia. La interfaz se actualizó antes de forma optimista y no mira la respuesta, por eso el usuario ve la tarea marcada hasta que recarga la página.

La acción `TasksToggle`, la ruta y el CSRF funcionan bien: la petición correcta persiste (tabla de arriba).

### Incertidumbres

- No se ha probado en un navegador real, solo por HTTP imitando a `fetch`. Hay que hacerlo en la verificación manual. Por especificación, `fetch` con `body: FormData` envía `multipart/form-data` e incluye las cookies del mismo origen (`credentials: "same-origin"` por defecto). Eso es lo que se reprodujo con `curl -F`.
- Caché del navegador: la plantilla carga `/assets/app.js?v=2` (`templates/application.plush.html:30`). Si no se sube el parámetro de versión, un navegador con el JS viejo en caché podría seguir fallando. No se ha medido qué cabeceras de caché pone `ServeFiles`. Por prudencia se sube a `?v=3`.
- `fetch` sigue el 303 hacia `GET /` y descarga el HTML del índice, que se tira. Es un coste menor que se acepta para no tocar el servidor.

## Alcance

Incluido:

- Corregir la petición de `public/assets/app.js` para que replique el formulario: `POST` con `new FormData(form)`, que lleva `_method=PUT` y `authenticity_token`.
- Comprobar la respuesta: si no es correcta o falla la red, recargar la página para volver al estado real.
- Subir la versión del script en `templates/application.plush.html` (`?v=2` → `?v=3`).
- Añadir tests de regresión en `actions/tasks_test.go`.

Excluido:

- Cambios en `actions/tasks.go`, `actions/app.go`, rutas, CSRF o method override. El servidor funciona.
- Responder JSON o 204 en `TasksToggle` para ahorrarse el redirect.
- Añadir un framework de tests JS (Node, jsdom, Playwright).
- El trabajo de prioridades de la rama `bacollaguazo/trabajo` y el directorio `.claude/skills/adw/` sin trackear.
- Cambiar el comportamiento de `data-confirm` del mismo manejador.

## Archivos relevantes

| Ruta | Responsabilidad |
|---|---|
| `public/assets/app.js` | Manejador `submit`: confirmación y toggle optimista. **Aquí está el bug (línea 34).** |
| `templates/application.plush.html` | Layout. Carga `app.js?v=2` (línea 30); hay que subir la versión. |
| `templates/tasks/index.plush.html` | Formulario `data-task-toggle` con `_method=PUT` y `authenticity_token` (líneas 16-18). Solo se lee; el JS depende de él. |
| `actions/app.go` | Ruta `PUT /tasks/{task_id}/toggle` y middleware CSRF. Solo se lee. |
| `actions/tasks.go` | `TasksToggle` (líneas 105-120): alterna y responde 303. Solo se lee. |
| `actions/tasks_test.go` | Tests de acciones. Aquí van los tests nuevos. |
| `public/public.go` | Embebe `public/` en el binario, así que el JS corregido solo llega a Docker al reconstruir la imagen. |

## Implementación paso a paso

1. **`public/assets/app.js`**: sustituir la línea 34 (`fetch(form.action);`) por una petición que replique el formulario y compruebe el resultado:

   ```js
   fetch(form.action, {
     method: "POST",
     body: new FormData(form),
     credentials: "same-origin"
   })
     .then(function (response) {
       if (!response.ok) {
         window.location.reload();
       }
     })
     .catch(function () {
       window.location.reload();
     });
   ```

   - `method: "POST"` con `FormData(form)` envía `_method=PUT` y `authenticity_token`. Buffalo lo convierte en PUT y el CSRF lo valida (reproducción: 303 y persiste).
   - `credentials: "same-origin"` es el valor por defecto; se pone explícito para dejar claro que la cookie de sesión hace falta para el CSRF.
   - Si falla, se recarga en vez de revertir el DOM a mano. Así la interfaz vuelve a lo que diga la BD sin duplicar la lógica de etiquetas, y no hay doble toggle (no se reenvía nada).
   - Se mantiene el resto del manejador tal cual: lógica optimista, `data-confirm` y `event.preventDefault()`.

2. **`templates/application.plush.html:30`**: cambiar `src="/assets/app.js?v=2"` por `src="/assets/app.js?v=3"`.

3. **`actions/tasks_test.go`**: añadir tests (ver la sección siguiente). Si hace falta, importar `strings`.

4. Ejecutar la validación (sección siguiente) y la verificación manual en el navegador tras reconstruir el contenedor.

## Tests y validación

### Límite de la cobertura automática (léelo antes de dar el bug por cerrado)

El fallo está en JavaScript del navegador y el repositorio no tiene tests de JS. Ningún test de Go puede ejecutar `fetch`. Por eso la cobertura se reparte así:

1. **`Test_TasksToggle_FormPostWithMethodOverride`** (contrato del servidor que usa el `fetch` corregido). Crea una tarea pendiente y hace `as.HTML("/tasks/%s/toggle", id).Post(url.Values{"_method": {"PUT"}})`. Comprueba 303 y que `completed` pasa a `true`; repite y comprueba que vuelve a `false`. **Pasa ya antes del arreglo** porque el servidor funciona; protege el contrato del que depende el JS. Limitación: con `GO_ENV=test` el CSRF no se ejecuta, así que este test no lo cubre.
2. **`Test_TasksToggle_GetIsNotARoute`**. Hace `as.HTML("/tasks/%s/toggle", id).Get()`, comprueba 404 y que la tarea sigue sin completar. Deja escrito por qué un GET (lo que hacía el bug) no sirve. También pasa antes del arreglo.
3. **`Test_AppJS_ToggleSubmitsFormAsPost`** (la guarda que **falla antes del arreglo**). Hace `as.HTML("/assets/app.js").Get()` (el asset embebido, igual que lo recibe el navegador), espera 200 y comprueba que el cuerpo contiene `method: "POST"` y `new FormData(form)`. Con el código actual falla porque solo existe `fetch(form.action);`. Es una comprobación de texto, frágil ante refactors de estilo; se acepta a sabiendas porque es la única forma de tener un test rojo antes del arreglo sin meter un runner JS. Si al implementar resulta que `ServeFiles` no responde dentro de la suite, se lee con `fs.ReadFile(public.FS(), "assets/app.js")`.
4. **`Test_TasksIndex_ToggleFormCarriesOverrideAndToken`** (opcional, muy barato). Crea una tarea, hace GET `/` y comprueba que el HTML contiene `data-task-toggle`, `name="_method" value="PUT"` y `name="authenticity_token"`: los campos que el `FormData` necesita.

### Validación como CI (en contenedor Linux)

En Windows con `core.autocrlf=true`, `gofmt -l .` lista todos los `.go` por los CRLF. Por eso se valida en un contenedor Linux contra la BD `todo_test`, que ya existe y está migrada. Desde la raíz del repo, en Git Bash:

```bash
MSYS_NO_PATHCONV=1 docker run --rm --network desarrollo-agentico-todo-03_default -e GO_ENV=test -e DB_HOST=db \
  -v "$(pwd -W)":/repo:ro -v todo03-gomod:/go/pkg/mod golang:1.26 \
  bash -c 'cp -r /repo /src && cd /src && git config --global --add safe.directory "*" && git ls-files | while read f; do sed -i "s/\r$//" "$f"; done && gofmt -l . && go vet ./... && go test -p 1 -count=1 ./... && go build ./...'
```

Criterio: `gofmt -l .` sin salida, `vet` limpio, todos los tests en verde y build correcto. Antes del arreglo, el test 3 debe fallar con este mismo comando; conviene comprobarlo antes de tocar `app.js`.

Ojo: `git ls-files` no incluye archivos nuevos sin `git add`. Si los tests nuevos van en `actions/tasks_test.go`, que ya está trackeado, no hay problema.

### Verificación manual en el navegador (obligatoria)

1. Reconstruir la app con el código de esta rama: `docker compose up -d --build`. El contenedor actual se construyó desde `bacollaguazo/trabajo` y los assets van embebidos en el binario.
   - Ojo: la BD de desarrollo tiene la columna `priority` de esa otra rama, con valor por defecto. El modelo de esta rama no la usa, así que inserts y updates deberían funcionar (incierto, no verificado con la imagen de esta rama). Si algo falla por eso, es ajeno a este bug.
2. Abrir http://localhost:3000, crear una tarea, abrir DevTools > Network y pulsar el check.
   - Esperado: `POST /tasks/<id>/toggle` con `multipart/form-data`, respuesta 303 y luego `GET /` con 200. Que no haya ningún GET a `/toggle` ni ningún 404.
3. Recargar (F5): la tarea sigue completada. Volver a pulsar y recargar: sigue pendiente.
4. Confirmar en BD: `docker exec desarrollo-agentico-todo-03-db-1 psql -U postgres -d todo_development -c "select title, completed from tasks order by updated_at desc limit 3"`.
5. Comprobar que el HTML servido carga `app.js?v=3`.
6. Prueba de fallo: en DevTools, poner la red en "Offline" y pulsar el check. La página intenta recargar, así que no se queda un estado falso. Borrar las tareas de prueba al terminar.

## Riesgos y recuperación

- **Caché del JS viejo**: se mitiga con `?v=3`. Si alguien sigue viendo el fallo, recargar sin caché (Ctrl+F5).
- **Recarga si algo falla**: en un 403 de CSRF (por ejemplo, la sesión caducó) la página se recarga y se pierde el cambio optimista. Es lo correcto (no se guardó) y la recarga renueva el token.
- **Doble clic rápido**: cada clic manda un toggle y el servidor alterna en cada petición. Con dos peticiones en paralelo, el estado final lo decide el orden en que lleguen. Ya pasaba con el envío nativo y queda fuera de alcance; se puede tratar después deshabilitando el botón mientras la petición está en curso.
- **Test de texto (test 3) frágil**: si alguien reescribe el `fetch` con otro estilo, el test fallará aunque la petición sea correcta. Se arregla actualizando las cadenas que busca, o sustituyéndolo por un test JS real si algún día se añade un runner.
- **Recuperación**: el cambio son unas pocas líneas en `app.js`, una en la plantilla y tests nuevos, sin migraciones ni cambios en el servidor. Para revertir basta `git revert` del commit y `docker compose up -d --build`.
