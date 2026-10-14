---
name: adw
description: Orquesta una tarea de punta a punta (clasificar → planificar con /feature, /bug o /chore → revisión humana del plan → /implement → verificación en la app → commit y PR). Usar cuando el usuario encargue una tarea, bug o mejora para este repositorio, o cuando el trigger de adws/ invoque `/adw <issue_url>`.
allowed-tools: Bash, Read, Glob, Grep, Write, Edit, Agent, AskUserQuestion
---

# ADW: flujo de trabajo de una tarea

`$ARGUMENTS`: `<descripción de la tarea | URL de issue> [--remote origin] [--base main] [--auto]`

- `--remote` (por defecto `origin`): remoto al que se sube la rama.
- `--base`: rama contra la que se abre la PR. Si no se indica, la decide el usuario en la Fase 5.
- `--auto`: modo desatendido (lo usa `adws/trigger-github.sh`, que ejecuta `claude -p` sin humano). Sin `--auto` el modo es **interactivo** y las puertas de revisión son obligatorias.

Esta skill **coordina** las skills del repositorio; no las sustituye. Cada fase usa la skill indicada siguiendo su `SKILL.md` al pie de la letra (si no están cargadas como comandos, leer `.claude/skills/<nombre>/SKILL.md`).

## Modo `--auto`: etiquetas de la issue (obligatorio)

El trigger no recuerda qué issues ha procesado: vuelve a lanzar cualquier issue abierta que conserve la etiqueta `adw`. Por eso, en `--auto`:

1. **Lo primero de todo**, antes de cualquier otra acción: `gh issue edit <url> --remove-label adw --add-label adw-en-curso`. Si falla, termina sin hacer nada más.
2. Al terminar con PR abierta: `gh issue edit <url> --remove-label adw-en-curso --add-label adw-hecho` y comenta en la issue el enlace a la PR y el resumen del plan.
3. Si te detienes por cualquier motivo (árbol sucio, plan con decisiones abiertas, tests en rojo, permiso denegado, error): `gh issue edit <url> --remove-label adw-en-curso --add-label adw-bloqueado` y comenta el motivo concreto y qué necesitas del usuario.
4. Nunca vuelvas a poner la etiqueta `adw`: la reactivación es decisión del usuario.
5. No abras el navegador (`cmd //c start`) ni hagas preguntas (`AskUserQuestion`): no hay nadie delante.
6. **No reconstruyas ni reinicies la app** (`docker compose up/down/restart`): es la instancia que usa el usuario. En `--auto` la Fase 4 se limita a la validación de CI en contenedor; indica en la PR y en la issue que la verificación manual en la app queda pendiente del usuario.
7. **Shell:** en `claude -p` sobre Windows la herramienta disponible suele ser **PowerShell**, no Bash. Ejecuta cada comando en su propia llamada (sin encadenar con `&&` ni `;`), empezando directamente por el programa (`git ...`, `gh ...`, `go ...`, `docker ...`): así coinciden con las reglas de `.claude/settings.local.json`. Validación de CI en contenedor Linux desde PowerShell (sin `env`/`MSYS_NO_PATHCONV`, que son de Bash):
   `docker run --rm --network desarrollo-agentico-todo-03_default -e GO_ENV=test -e DB_HOST=db -v "${PWD}:/repo:ro" -v todo03-gomod:/go/pkg/mod golang:1.26 bash -c 'cp -r /repo /src && cd /src && git config --global --add safe.directory "*" && git ls-files | while read f; do sed -i "s/\r$//" "$f"; done && gofmt -l . && go vet ./... && go test -p 1 -count=1 ./... && go build ./...'`
   (antes, `git add -N` de los archivos nuevos). Si un comando necesario es denegado por permisos, no busques rodeos: aplica el punto 3 indicando el comando exacto.

## Fase 0 — Encargo y rama

1. Si el argumento es una URL de issue: `gh issue view <url> --json title,body,labels,comments` y úsalo como encargo literal.
2. `git status --short --branch`. Si hay cambios sin commitear ajenos al encargo, **detente** y avisa (no los mezcles ni los descartes). En `--auto`, aplica el punto 3 de las etiquetas.
3. Clasifica el encargo y justifícalo en una línea:
   - **bug**: algo que debería funcionar y no funciona → `/bug`
   - **feature**: comportamiento nuevo visible → `/feature`
   - **chore**: mantenimiento sin cambio funcional → `/chore`
4. Rama de trabajo:
   - **Interactivo (por defecto): trabaja en la rama actual**, que acumula el trabajo en curso (p. ej. `bacollaguazo/trabajo`). Las tareas se construyen unas sobre otras: **no** partas de `main`, porque perderías funcionalidades aún no fusionadas (como las prioridades) y la app dejaría de mostrarlas. Cada tarea queda en su propio commit.
   - Solo crea una rama nueva si el usuario lo pide; en ese caso, pártela de la rama actual salvo que diga otra base.
   - `--auto`: `git fetch <remote> && git switch -c <tipo>/<slug-kebab> <remote>/<base>` (cada issue aislada).

## Fase 1 — Plan (`/feature`, `/bug` o `/chore`)

1. Lanza la skill de planificación en un **subagente** (contexto limpio, separa planificar de revisar), pasándole el encargo y, si aplica, la ruta `plans/<slug>.md`. Incluye en el prompt el contexto del entorno que necesite para documentar los tests (BD de test, cómo arrancar la app).
2. Revisa el plan **críticamente** contra el código antes de enseñarlo: verifica las afirmaciones clave (rutas, funciones, estado actual, tests existentes), busca huecos de alcance, decisiones de producto implícitas y riesgos no cubiertos. Si encuentras fallos técnicos, pide la corrección al mismo subagente.

## Fase 2 — Revisión del plan (puerta humana)

Presenta un **resumen** del plan (no el documento entero), con este formato:

```
Plan: plans/<archivo>.md  ·  tipo: <bug|feature|chore>  ·  rama: <rama>
Problema / objetivo: <1-2 líneas>
Causa raíz (bug) o propuesta (feature): <1-3 líneas, con archivo:línea>
Cambios: <lista corta de archivos y qué cambia en cada uno>
Tests: <test(s) de regresión / nuevos y qué demuestran>
Fuera de alcance: <lo excluido>
Decisiones a validar: <inferencias de producto que debe confirmar el usuario>
Riesgos: <los relevantes>
```

- **Interactivo**: pregunta con `AskUserQuestion` (aprobar / pedir cambios). Si pide cambios, envíalos al subagente planificador, vuelve a revisar y a resumir. **No pases a la Fase 3 sin aprobación explícita.**
- **`--auto`**: continúa solo si el plan no tiene decisiones de producto abiertas ni riesgos bloqueantes; si los tiene, detente, comenta el resumen en la issue (`gh issue comment`) y termina.

## Fase 3 — Implementación (`/implement`)

Ejecuta `/implement plans/<archivo>.md` respetando sus reglas: fiel al plan, sin rediseñar ni ampliar alcance, sin commit. Si el plan resulta inviable, detente y vuelve a la Fase 2 con el bloqueo.

Validación (la misma que CI, `.github/workflows/ci.yaml`): `gofmt -l .`, `go vet ./...`, `go test -p 1 ./...`, `go build ./...`. En Windows, ejecútala en un contenedor Linux sobre la red de compose y con los ficheros normalizados a LF (Buffalo resuelve partials con `filepath.Join` y `core.autocrlf` convierte a CRLF): ver la skill de usuario `lanzar-repo` y el historial del repo.

## Fase 4 — Verificación en la app

1. Reconstruye la app (`docker compose up -d --build`): el contenedor embebe código, plantillas y migraciones; sin `--build` se ve la versión antigua.
2. Reproduce el escenario del encargo **como lo haría el usuario** (formularios con CSRF, rutas reales; si el bug depende de JavaScript del navegador, compruébalo en navegador o explica qué parte no se puede verificar por HTTP).
3. Limpia los datos de prueba.
4. Interactivo: indica al usuario qué comprobar en http://localhost:3000 (con Ctrl+F5 si cambian assets) y espera su confirmación.

## Fase 5 — Commit y PR

- **Interactivo**: solo cuando el usuario lo confirme. Pregunta contra qué repositorio/rama va la PR si no está claro (fork vs. upstream).
- Commit en estilo del repo (Conventional Commits en español: `fix(tasks): …`, `feat(tasks): …`), con cuerpo detallado y la ruta del plan.
- Tras el commit, súbelo y **ábrelo en el navegador** para revisarlo (`git push`, luego `cmd //c start "" https://github.com/<owner>/<repo>/commit/<sha>` en Windows).
- PR: `gh pr create --base <base>` con descripción detallada: resumen, causa raíz/propuesta, cambios por capa, tests y validación ejecutada (con resultados), verificación manual, riesgos, y el **plan resumido** con enlace a `plans/<archivo>.md`.
- En `--auto`, enlaza la issue (`Closes #N` solo si el usuario lo ha pedido; por defecto `Refs #N`).

## Informe final

Rama, plan, archivos cambiados, comandos ejecutados con su resultado, verificación en la app, desviaciones respecto al plan y riesgos residuales. Distingue lo verificado de lo inferido.
