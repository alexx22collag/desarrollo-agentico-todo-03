# Trigger de issues

Este lanzador consulta las issues abiertas con la etiqueta configurada, de más recientes a más antiguas por **fecha de creación**, e invoca Claude de una en una. Espera a que termine cada proceso antes de lanzar el siguiente. Cada consulta recoge hasta 100 issues; al terminar la lista espera el intervalo configurado y vuelve a consultar.

## Preparación

Necesitas Go (la versión del proyecto), Bash, GitHub CLI (`gh`) y Claude Code (`claude`) en `PATH`, autenticados, y el entorno de desarrollo del proyecto ya disponible. El lanzador reutiliza la librería YAML que ya incluye el proyecto. `--jq` usa el formateador integrado de `gh`: no hace falta instalar `jq`.

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

## Configurar

Edita **`config/adw.yml`**. No necesitas editar el código ni establecer variables de entorno:

```yaml
repo: OWNER/REPO
label: adw
interval_seconds: 60
prompt: >-
  /adw {{issue_url}} --remote origin --base main --auto
```

- `repo`: repositorio donde buscar las issues (`OWNER/REPO`). Cámbialo por el tuyo antes de arrancar.
- `label`: etiqueta que activa el trabajo. `/adw --auto` asume `adw` (ver «Ciclo de etiquetas»).
- `interval_seconds`: espera entre consultas, una vez terminada la lista anterior.
- `prompt`: texto que recibe Claude. Puede invocar tu propia skill o comando; `{{issue_url}}` se sustituye por la URL de cada issue. No se evalúa como código de shell.

`--remote origin`, `--base main` y `--auto` son argumentos de `/adw`, no opciones del trigger: el trigger solo sustituye `{{issue_url}}`. Cambia el remoto y la base según tu repositorio. Mantén `--auto`: sin él `/adw` funciona en modo interactivo y espera aprobaciones que nadie puede dar en `claude -p`. El bloque `>-` permite escribir un prompt largo en varias líneas: YAML las une con espacios.

Usa issues revisadas y listas para ejecutar; poner la etiqueta configurada significa que autorizas el encargo dentro de los límites de tu workflow.

## Arrancar

Desde la raíz del repositorio:

```bash
bash adws/trigger-github.sh
```

Para consultar y procesar la lista una sola vez:

```bash
bash adws/trigger-github.sh --once
```

También puedes ejecutar `go run ./adws` directamente. El script sitúa la ejecución en este checkout; no prepara otro entorno. Ambos admiten `--config ruta/al/config.yml` para elegir otro fichero. `--once` ejecuta Claude: no es una simulación.

Cada invocación equivale a `claude -p "<prompt configurado con la URL>"`. El trigger no se arranca automáticamente. Detén el sondeo con `Ctrl+C`. Si `gh` o `claude` devuelve un código de error, el lanzador termina.

## Límites de esta versión

- El trigger no modifica etiquetas, no cierra issues ni recuerda las que ha ejecutado. Si una issue sigue abierta y conserva la etiqueta, puede volver a ejecutarse en la siguiente consulta o al reiniciar el trigger. Con `/adw --auto` las etiquetas las cambia el workflow (ver «Ciclo de etiquetas»), así que la repetición solo ocurre si no llega a cambiarlas. Abrir un PR no cierra la issue por sí solo: `/adw --auto` la enlaza con `Refs #N`.
- El trigger no crea entornos, no cambia de rama ni limpia el checkout entre encargos. `/adw --auto` crea una rama por issue, pero trabaja en el mismo checkout: el estado que deja un encargo afecta al siguiente y, si quedan cambios sin commitear, la siguiente issue se detiene en la fase 0 y queda `adw-bloqueado`.
- No gestiona feedback, reintentos, recuperación ni coordinación con otros procesos.
- El trigger no interpreta el resultado de Claude. Un código cero no demuestra que exista un PR; el resultado de cada issue lo indican la etiqueta final (`adw-hecho` o `adw-bloqueado`) y el comentario que deja `/adw --auto`.
- El trigger no desactiva los permisos de Claude. Tu workflow y su configuración deben permitir las acciones previstas (ver «Permisos»); sin una respuesta humana, una ambigüedad o un permiso pendiente puede impedir completar el encargo.

## Comprobar el lanzador

```bash
go test -p 1 ./adws
```

Los tests usan `gh` y `claude` simulados: no consultan GitHub ni ejecutan un agente real. Comprueban la configuración YAML, la consulta, el orden secuencial, la sustitución del prompt, las paradas por error y el sondeo sin memoria. No sustituyen un ensayo completo del workflow hasta el PR.
