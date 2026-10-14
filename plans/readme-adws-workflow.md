# Chore: documentar en `adws/README.md` el workflow `/adw`, el ciclo de etiquetas y los permisos

## Encargo y resultado esperado

Issue #4. `adws/README.md` afirma que «el trigger no contiene el pipeline ni proporciona `/adw`» y pide crear ese workflow. Desde la PR #1 el workflow existe (`.claude/skills/adw/SKILL.md`) y el trigger está configurado para invocarlo en modo desatendido (`config/adw.yml`, `prompt` con `--auto`), pero el README no lo refleja.

Resultado esperado: **solo** `adws/README.md` cambia (además de este plan) para:

1. Sustituir el párrafo que dice que `/adw` no existe por una sección «Workflow `/adw`» (fases y diferencia interactivo / `--auto`).
2. Añadir «Ciclo de etiquetas» (`adw` → `adw-en-curso` → `adw-hecho` | `adw-bloqueado`, quién pone cada una, reactivación manual).
3. Añadir «Permisos» (`claude -p` necesita permisos en `.claude/settings.local.json`, no versionado; qué tipo de comandos requiere `/adw`; el agente no fusiona PRs ni toca `docker compose`), sin copiar el archivo ni rutas personales.
4. Ajustar «Configurar» para que el ejemplo de `prompt` incluya `--auto`.
5. Ajustar «Límites de esta versión» para que no contradiga lo anterior (mínimo cambio).

Regla de redacción: toda afirmación nueva del README debe poder trazarse a `.claude/skills/adw/SKILL.md`, `config/adw.yml` o `adws/trigger.go` (ver tabla de trazabilidad en «Tests y validación»). No se inventan comandos, flags ni comportamientos.

## Estado actual y restricciones

Verificado en el repositorio (rama `chore/actualizar-readme-adws`, árbol limpio, `git status --short --branch` → `## chore/actualizar-readme-adws...origin/main`):

- `adws/README.md` (67 líneas). Líneas afectadas:
  - L9: «El trigger no contiene el pipeline ni proporciona `/adw`. Debes crear ese workflow antes o elegir otro prompt…» → **obsoleta**.
  - L11: «Las piezas disponibles son `/feature`, `/bug`, `/chore` y `/implement`… Tu workflow debe coordinar esas fases…» → obsoleta en su segunda mitad (el workflow ya existe).
  - L17-23: ejemplo YAML con `interval_seconds: 20` y `prompt` **sin** `--auto`.
  - L25: «`repo`: … Se entrega vacío para que lo configures antes de arrancar.» → ya no es cierto: `config/adw.yml` L2 trae un repositorio concreto.
  - L30: «`origin` y `main` son argumentos del `/adw` del ejemplo…» → hay que añadir `--auto`.
  - L54-58: «Límites de esta versión». L54 (no modifica etiquetas / no recuerda issues), L55 (no cambia de rama), L57 (no interpreta el resultado) y L58 (permisos) siguen siendo ciertos **del trigger en Go**, pero ahora `/adw --auto` gestiona etiquetas y crea rama; hay que aclararlo sin contradicción.
- `.claude/skills/adw/SKILL.md`:
  - L9-13: argumentos `<descripción | URL de issue> [--remote origin] [--base main] [--auto]`; sin `--base`, la base la decide el usuario en la Fase 5; sin `--auto`, modo interactivo con puertas obligatorias.
  - L17-29: modo `--auto`: etiquetas (L21-24), sin navegador ni preguntas (L25), no reconstruir la app con `docker compose` y verificación manual pendiente (L26), PowerShell en `claude -p` sobre Windows y comando de validación en contenedor (L27-29); «si un comando necesario es denegado por permisos, no busques rodeos» → bloqueado con el comando exacto (L29).
  - Fases 0-5 (L31-86): rama actual en interactivo (L40-41), `git switch -c <tipo>/<slug-kebab> <remote>/<base>` en `--auto` (L42); plan en subagente (L46); puerta humana / parada en `--auto` (L64-65); `/implement` sin commit y validación de CI (L69-71); reconstrucción y verificación en la app (L75-78); commit Conventional Commits en español, push, PR con `gh pr create --base`, `Refs #N` por defecto en `--auto` (L82-86).
  - L40 contiene un nombre de rama con usuario concreto: **no copiarlo** al README.
- `config/adw.yml`: `label: adw`, `interval_seconds: 60`, `prompt: /adw {{issue_url}} --remote origin --base main --auto`; L2 `repo` con un owner concreto (**no copiarlo**).
- `adws/trigger.go`: consulta `gh issue list --state open --label <label> --search sort:created-desc --limit 100` (L72-74), sustituye `{{issue_url}}` (L85), ejecuta `claude -p <prompt>` con stdin cerrado (L87-89), termina si `gh` o `claude` fallan (L77-79, L90-92), no toca etiquetas ni ramas. `readConfig` exige `{{issue_url}}` en `prompt` (L45-47).
- `adws/trigger-github.sh`: `cd` a la raíz y `exec go run ./adws "$@"`. Sin cambios.
- `adws/trigger_test.go`: tests con `gh`/`claude`/`sleep` simulados como scripts Bash y `/bin/sleep` → en Windows nativo no son fiables; se ejecutan en contenedor Linux. `TestTriggerProjectConfig` lee `../config/adw.yml` (no afectado: no se toca la configuración). No necesitan BD.
- `.claude/settings.local.json`: existe localmente, **no está versionado** (`git ls-files .claude` no lo lista) y está ignorado **solo por el ignore global del usuario** (`git check-ignore -v` → `~/.config/git/ignore`), no por el `.gitignore` del repo. Por categorías permite: lectura/búsqueda (`Read`, `Glob`, `Grep`), edición dentro del repo, subagentes y skills, `git`, `gh`, `go`, `soda`, `docker`, `curl` y utilidades de lectura de texto, tanto en forma Bash como PowerShell. Deniega: editar `.claude/settings*.json` y `.git/**`, `git push --force`/`-f`, push a `main`, `git reset --hard`, `git clean`, `gh pr merge`, `gh repo delete`/`edit`, `docker compose`, `docker volume rm`, `docker system prune`, `rm -rf` y `Remove-Item -Recurse`. Contiene además una ruta personal a un binario: **nada de esto se copia literalmente**.
- Etiquetas: las cuatro (`adw`, `adw-en-curso`, `adw-hecho`, `adw-bloqueado`) existen en el repositorio configurado (`gh label list --repo <repo de config/adw.yml>`); no existen en `upstream`. No se documenta en el README porque no es trazable a las tres fuentes (ver Riesgos).

Restricciones: ningún cambio en código, tests, configuración, skills ni `.gitignore`. Sin rutas personales (`C:\Users\…`, `/c/Users/…`) ni usuarios/owners concretos en el README.

## Alcance

Incluido:

- Editar `adws/README.md`: sustituir L9-11 por tres secciones nuevas; ajustar el ejemplo y las viñetas de «Configurar» (L17-30); ajustar viñetas de «Límites de esta versión» (L54-58).

Excluido:

- Cambiar el comportamiento del trigger (`adws/trigger.go`, `adws/trigger-github.sh`) o de la skill (`.claude/skills/adw/SKILL.md`), aunque se detecten incoherencias (ver Riesgos).
- Modificar `config/adw.yml`, `.claude/settings.local.json`, `.gitignore` o `plans/README.md`.
- Crear etiquetas en GitHub.
- Secciones «Arrancar» (L34-50) y «Comprobar el lanzador» (L60-66): sin cambios.

## Archivos relevantes

| Ruta | Responsabilidad | Acción |
|---|---|---|
| `adws/README.md` | Documentación del trigger | **Editar** |
| `plans/readme-adws-workflow.md` | Este plan | Crear (ya creado) |
| `.claude/skills/adw/SKILL.md` | Fuente de verdad del workflow `/adw` | Solo lectura |
| `config/adw.yml` | Configuración real del trigger | Solo lectura |
| `adws/trigger.go` | Comportamiento del lanzador | Solo lectura |
| `adws/trigger-github.sh` | Envoltorio Bash | Solo lectura |
| `adws/trigger_test.go` | Tests del lanzador | Solo lectura; se ejecuta |
| `.claude/settings.local.json` | Permisos locales (no versionado) | Solo lectura, no se cita literalmente |

## Implementación paso a paso

Orden: (1) Preparación → (2) Configurar → (3) Límites. Cada paso es una edición localizada; no reformatear el resto del archivo ni cambiar finales de línea.

### Paso 1 — Sustituir L9-11 («El trigger no contiene el pipeline…» y «Las piezas disponibles…») por tres secciones

Borrar exactamente los dos párrafos de L9 y L11 (la L7 de «Preparación» se conserva intacta) e insertar en su lugar, antes de `## Configurar`, el texto siguiente:

~~~markdown
El trigger solo lanza Claude: el pipeline lo implementa la skill `/adw` (`.claude/skills/adw/SKILL.md`), que es lo que invoca el `prompt` de `config/adw.yml`.

## Workflow `/adw`

`/adw` coordina las skills de `.claude/skills/` de punta a punta; no las sustituye. Sus argumentos son:

```text
/adw <descripción de la tarea | URL de issue> [--remote origin] [--base main] [--auto]
```

- `--remote` (por defecto `origin`): remoto al que se sube la rama.
- `--base`: rama contra la que se abre la PR. Si no se indica, la decide el usuario en la fase de commit y PR.
- `--auto`: modo desatendido, el que usa el trigger. Sin `--auto`, el modo es interactivo.

### Fases

0. **Encargo y rama.** Si recibe una URL, lee la issue (`gh issue view`) y la usa como encargo literal. Comprueba el árbol de trabajo: si hay cambios sin commitear ajenos al encargo, se detiene. Clasifica el encargo como bug, feature o chore.
1. **Plan.** Ejecuta `/bug`, `/feature` o `/chore` en un subagente, que escribe el plan en `plans/`, y lo revisa críticamente contra el código.
2. **Revisión del plan.** Presenta un resumen del plan (problema, propuesta, cambios, tests, fuera de alcance, decisiones a validar, riesgos).
3. **Implementación.** Ejecuta `/implement plans/<archivo>.md`, fiel al plan y sin commit, y la validación de CI: `gofmt -l .`, `go vet ./...`, `go test -p 1 ./...`, `go build ./...`. En Windows se ejecuta en un contenedor Linux.
4. **Verificación en la app.** Reconstruye la app y reproduce el escenario del encargo como lo haría un usuario.
5. **Commit y PR.** Commit en Conventional Commits en español con la ruta del plan, push y PR con `gh pr create --base <base>`, cuya descripción incluye el plan resumido.

### Interactivo y `--auto`

| | Interactivo (sin `--auto`) | `--auto` (trigger) |
|---|---|---|
| Rama | Trabaja en la rama actual; solo crea otra si el usuario lo pide | Crea `<tipo>/<slug>` desde `<remote>/<base>` (`git fetch` + `git switch -c`): cada issue aislada |
| Revisión del plan (fase 2) | Puerta humana: no implementa sin aprobación explícita | Sin puerta humana: continúa solo si el plan no tiene decisiones de producto abiertas ni riesgos bloqueantes; si los tiene, comenta el resumen en la issue, la marca `adw-bloqueado` y termina |
| Verificación en la app (fase 4) | Reconstruye la app, indica qué comprobar y espera confirmación | No reconstruye ni reinicia la app; se limita a la validación de CI en contenedor y deja indicado en la PR y en la issue que la verificación manual queda pendiente |
| Commit y PR (fase 5) | Solo tras confirmación del usuario; abre el commit en el navegador | Sin preguntas ni navegador; la PR enlaza la issue con `Refs #N` (`Closes #N` solo si el usuario lo ha pedido) |
| Etiquetas | No las toca | Gestiona el ciclo descrito abajo |

En `--auto` se pasa siempre `--base`, porque la rama de trabajo se crea desde `<remote>/<base>`.

## Ciclo de etiquetas

Solo en `--auto`. El trigger no recuerda qué issues ha procesado: vuelve a lanzar cualquier issue abierta que conserve la etiqueta configurada. `/adw --auto` evita las repeticiones cambiando la etiqueta:

```text
adw  →  adw-en-curso  →  adw-hecho
                      ↘  adw-bloqueado
```

| Etiqueta | Quién la pone | Cuándo |
|---|---|---|
| `adw` | El usuario, a mano | Para autorizar el encargo. Es la `label` que consulta el trigger |
| `adw-en-curso` | `/adw --auto`, quitando `adw` | Lo primero de todo. Si este cambio falla, termina sin hacer nada más |
| `adw-hecho` | `/adw --auto`, quitando `adw-en-curso` | Al terminar con la PR abierta; comenta en la issue el enlace a la PR y el resumen del plan |
| `adw-bloqueado` | `/adw --auto`, quitando `adw-en-curso` | Si se detiene por cualquier motivo (árbol sucio, plan con decisiones abiertas, tests en rojo, permiso denegado, error); comenta el motivo y qué necesita del usuario |

`/adw` nunca vuelve a poner `adw`: **reactivar una issue es volver a ponerle `adw` a mano**, por ejemplo tras resolver lo que pedía el comentario de bloqueo. Una issue que se quede en `adw-en-curso` (proceso interrumpido) no vuelve a lanzarse sola por el mismo motivo.

Los nombres de las etiquetas están fijados en `.claude/skills/adw/SKILL.md`; si cambias `label` en `config/adw.yml`, `/adw --auto` seguirá quitando `adw` y el ciclo no coincidirá.

## Permisos

El trigger ejecuta `claude -p` sin nadie delante, así que nadie puede aprobar permisos durante la ejecución. Las acciones de `/adw` deben estar permitidas de antemano en `.claude/settings.local.json`, un archivo local de cada máquina que **no se versiona**: créalo o revísalo antes de arrancar el trigger. Si un comando necesario se deniega, `/adw --auto` no busca rodeos: marca la issue `adw-bloqueado` indicando el comando exacto.

`/adw` necesita:

- Leer, buscar y editar archivos del repositorio, y lanzar subagentes y skills.
- `git`: estado, `fetch`, crear rama, `add`, `commit` y `push` de la rama de trabajo.
- `gh`: leer, etiquetar y comentar issues, y crear la PR.
- `go` y `docker run` para la validación de CI en contenedor.

En `claude -p` sobre Windows la herramienta de shell suele ser PowerShell, no Bash: `/adw` ejecuta cada comando en su propia llamada, empezando por el programa (`git …`, `gh …`, `go …`, `docker …`), para que coincida con reglas escritas por programa. Las reglas deben cubrir la herramienta de shell que se use.

Límites que la configuración debe imponer (reglas de denegación):

- **Fusionar PRs** (`gh pr merge`): el workflow termina en `gh pr create`; la fusión es humana.
- **`docker compose`**: la app en marcha es la instancia que usa el usuario; `/adw --auto` no la reconstruye ni la reinicia.
- Push forzado o directo a `main`, `git reset --hard`, `git clean`, borrados recursivos y editar los propios archivos de configuración de Claude.
~~~

Notas para quien implemente:

- El bloque `text` con argumentos y el diagrama son literales de este plan; no añadir flags ni etiquetas adicionales.
- No incluir la rama de ejemplo de SKILL.md L40 (contiene un usuario), ni el comando `docker run` completo de SKILL.md L28 (contiene el nombre de la red de compose y no es necesario aquí), ni reglas literales del archivo de permisos.

### Paso 2 — Ajustar «Configurar» (L17-30)

2a. Sustituir el bloque YAML de L17-23 por:

~~~markdown
```yaml
repo: OWNER/REPO
label: adw
interval_seconds: 60
prompt: >-
  /adw {{issue_url}} --remote origin --base main --auto
```
~~~

(`interval_seconds: 20` → `60` para coincidir con `config/adw.yml`; `repo` sigue como `OWNER/REPO` para no publicar un owner concreto.)

2b. L25, sustituir:

> - `repo`: repositorio donde buscar las issues. Se entrega vacío para que lo configures antes de arrancar.

por:

> - `repo`: repositorio donde buscar las issues (`OWNER/REPO`). Cámbialo por el tuyo antes de arrancar.

2c. L26, sustituir:

> - `label`: etiqueta que activa el trabajo.

por:

> - `label`: etiqueta que activa el trabajo. `/adw --auto` asume `adw` (ver «Ciclo de etiquetas»).

2d. L30, sustituir:

> `origin` y `main` son argumentos del `/adw` del ejemplo, no opciones del trigger. Cámbialos según tu workflow. El bloque `>-` permite escribir un prompt largo en varias líneas: YAML las une con espacios.

por:

> `--remote origin`, `--base main` y `--auto` son argumentos de `/adw`, no opciones del trigger: el trigger solo sustituye `{{issue_url}}`. Cambia el remoto y la base según tu repositorio. Mantén `--auto`: sin él `/adw` funciona en modo interactivo y espera aprobaciones que nadie puede dar en `claude -p`. El bloque `>-` permite escribir un prompt largo en varias líneas: YAML las une con espacios.

L28 (`prompt`…) y L32 («Usa issues revisadas…») se conservan sin cambios.

### Paso 3 — Ajustar «Límites de esta versión» (L54-58)

Decisión: mantener la sección como límites **del trigger** (siguen siendo ciertos de `adws/trigger.go`) y añadir a cada viñeta afectada una aclaración de qué cubre `/adw --auto`. Cambio mínimo: se reescriben L54, L55, L57 y L58; L56 queda igual.

- L54, sustituir por:

> - El trigger no modifica etiquetas, no cierra issues ni recuerda las que ha ejecutado. Si una issue sigue abierta y conserva la etiqueta, puede volver a ejecutarse en la siguiente consulta o al reiniciar el trigger. Con `/adw --auto` las etiquetas las cambia el workflow (ver «Ciclo de etiquetas»), así que la repetición solo ocurre si no llega a cambiarlas. Abrir un PR no cierra la issue por sí solo: `/adw --auto` la enlaza con `Refs #N`.

- L55, sustituir por:

> - El trigger no crea entornos, no cambia de rama ni limpia el checkout entre encargos. `/adw --auto` crea una rama por issue, pero trabaja en el mismo checkout: el estado que deja un encargo afecta al siguiente y, si quedan cambios sin commitear, la siguiente issue se detiene en la fase 0 y queda `adw-bloqueado`.

- L56: sin cambios.

- L57, sustituir por:

> - El trigger no interpreta el resultado de Claude. Un código cero no demuestra que exista un PR; el resultado de cada issue lo indican la etiqueta final (`adw-hecho` o `adw-bloqueado`) y el comentario que deja `/adw --auto`.

- L58, sustituir por:

> - El trigger no desactiva los permisos de Claude. Tu workflow y su configuración deben permitir las acciones previstas (ver «Permisos»); sin una respuesta humana, una ambigüedad o un permiso pendiente puede impedir completar el encargo.

### Paso 4 — Revisión final del diff

`git diff adws/README.md` y confirmar que solo cambian: L9-11 (sustituidas por las tres secciones), el bloque YAML, L25, L26, L30, L54, L55, L57 y L58. Ningún otro cambio de espaciado ni de finales de línea.

## Tests y validación

Ejecutar en PowerShell desde la raíz del repo, **un comando por llamada**.

### Antes del cambio (línea base)

1. `git status --short --branch` → solo `?? plans/readme-adws-workflow.md` además de la cabecera de rama.
2. Tests del lanzador en contenedor Linux (los tests usan scripts Bash y `/bin/sleep`; no necesitan BD ni la red de compose):

   ```powershell
   docker run --rm -v "${PWD}:/repo:ro" -v todo03-gomod:/go/pkg/mod golang:1.26 bash -c 'cp -r /repo /src && cd /src && git config --global --add safe.directory "*" && git ls-files | while read f; do sed -i "s/\r$//" "$f"; done && go test -p 1 -count=1 ./adws'
   ```

   Esperado: `ok` para el paquete `adws`. Si se dispone de Bash y entorno Unix, `go test -p 1 ./adws` es equivalente.

### Después del cambio

3. Repetir el comando 2 → `ok` (debe seguir en verde: la documentación no afecta al código y `config/adw.yml` no cambia).
4. `git diff --stat` → solo `adws/README.md`.
5. `git status --short` → solo ` M adws/README.md` y `?? plans/readme-adws-workflow.md`.
6. Búsqueda de rutas personales (esperado: sin salida):

   ```powershell
   Select-String -Path adws/README.md -SimpleMatch -Pattern 'C:\Users', '/c/Users', 'Users\', '/Users/'
   ```

7. Búsqueda de usuarios concretos (esperado: sin salida):

   ```powershell
   Select-String -Path adws/README.md -SimpleMatch -Pattern $env:USERNAME, (git config user.name), ((git remote get-url origin).Split('/')[3])
   ```

8. Que no se ha copiado el archivo de permisos (esperado: sin salida):

   ```powershell
   Select-String -Path adws/README.md -SimpleMatch -Pattern '"permissions"', 'Bash(', 'PowerShell(', 'Edit(./'
   ```

9. Que el texto obsoleto desaparece y el nuevo está (esperado: sin salida y luego varias coincidencias):

   ```powershell
   Select-String -Path adws/README.md -SimpleMatch -Pattern 'no contiene el pipeline', 'Debes crear ese workflow'
   ```

   ```powershell
   Select-String -Path adws/README.md -Pattern '^## (Workflow `/adw`|Ciclo de etiquetas|Permisos)$', '--auto$'
   ```

10. Revisión manual de trazabilidad: cada afirmación nueva debe corresponder a una fila de esta tabla.

| Afirmación del README | Fuente |
|---|---|
| Argumentos `--remote`, `--base`, `--auto` y sus valores por defecto | `.claude/skills/adw/SKILL.md` L9-13 |
| Fase 0: `gh issue view`, árbol limpio, clasificación | SKILL.md L33-38 |
| Rama actual (interactivo) / `<tipo>/<slug>` desde `<remote>/<base>` (`--auto`) | SKILL.md L40-42 |
| Plan en subagente y revisión crítica | SKILL.md L46-47 |
| Resumen del plan; puerta humana / parada en `--auto` | SKILL.md L51-65 |
| `/implement` sin commit; validación CI; contenedor en Windows | SKILL.md L69-71, L27-28 |
| Reconstrucción y verificación en la app (interactivo) | SKILL.md L75-78 |
| No reconstruir en `--auto`; verificación manual pendiente | SKILL.md L26 |
| Commit, push, `gh pr create --base`, navegador solo interactivo | SKILL.md L82-85, L25 |
| `Refs #N` por defecto, `Closes #N` solo si se pide | SKILL.md L86 |
| Ciclo de etiquetas, comentarios, nunca reponer `adw` | SKILL.md L19-24 |
| Etiqueta en curso se queda si el proceso se interrumpe y no se relanza | Inferencia: SKILL.md L21 + `adws/trigger.go` L72-74 (solo consulta `label`) |
| Nombres de etiqueta fijados en la skill | SKILL.md L21-23 vs `config/adw.yml` L3 |
| Comando denegado → bloqueado con el comando exacto; PowerShell en Windows | SKILL.md L27-29 |
| `claude -p` sin humano, stdin cerrado | `adws/trigger.go` L87-89; SKILL.md L13 |
| Trigger solo sustituye `{{issue_url}}`; no toca etiquetas ni ramas | `adws/trigger.go` L45-47, L72-92 |
| Ejemplo YAML (`label`, `interval_seconds`, `prompt` con `--auto`) | `config/adw.yml` L3-10 |
| Sin `--auto` espera aprobaciones que nadie da en `claude -p` | Inferencia: SKILL.md L13, L64 + `adws/trigger.go` L89 |
| Archivo de permisos local y no versionado; categorías y denegaciones | `git ls-files` / `git check-ignore`; SKILL.md L27 lo nombra; contenido descrito por categorías |

## Riesgos y recuperación

- **Incoherencia SKILL.md vs permisos (fuera de alcance):** la Fase 4 interactiva de SKILL.md (L75) ejecuta `docker compose up -d --build`, pero la configuración local deniega `docker compose`. El README no la resuelve. Seguimiento recomendado (otra issue): alinear SKILL.md Fase 4 con esa regla.
- **`settings.local.json` ignorado solo por el ignore global del usuario**, no por el `.gitignore` del repo: en otra máquina sin esa regla podría versionarse con rutas personales. Fuera de alcance (es configuración); seguimiento recomendado: añadir `.claude/settings.local.json` al `.gitignore` del repo en otra issue.
- **Etiquetas inexistentes:** si `adw-en-curso` no existe en el repositorio, el primer `gh issue edit` probablemente falla (incierto: no verificado en este entorno), `/adw` termina sin cambiar `adw` y el trigger la relanza en cada consulta. Hoy las cuatro existen en el repo configurado. No se documenta en el README por no ser trazable a las tres fuentes; mencionarlo requeriría ampliar el criterio.
- **Deriva de la documentación:** si cambian SKILL.md o `config/adw.yml`, el README queda desactualizado. Mitigación: el README remite a las rutas de las fuentes.
- **Fugas de datos personales:** mitigado por las búsquedas 6-8.
- **Tests en Windows nativo:** pueden fallar por entorno (scripts Bash, `/bin/sleep`), no por el cambio; usar el contenedor del paso 2.
- **Recuperación:** `git restore adws/README.md` deshace el cambio. Si se descarta también el plan, borrar a mano `plans/readme-adws-workflow.md` (no usar `git clean`, que está denegado).
