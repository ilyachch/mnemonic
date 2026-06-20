# ТЗ для ИИ-агента: поэтапная реорганизация архитектуры проекта `mnemonic`

## 1. Вступление

Проект `mnemonic` — Go-приложение с CLI, MCP и Web-частью для локальной базы знаний на markdown-файлах. Сейчас проект уже функционален, но архитектура начала становиться дорогой в поддержке: одно изменение часто требует правок в нескольких местах, потому что бизнес-логика размазана между CLI-командами, MCP tools, Web manager и низкоуровневыми пакетами.

Основная цель этой работы — не добавить новые фичи, а привести код к архитектуре, где:

1. CLI, MCP и Web являются тонкими адаптерами.
2. Бизнес-сценарии живут в service/usecase-слое.
3. Работа с filesystem, registry, SQLite index и webauth DB живет в store-слое.
4. Markdown parsing/rendering живет в format-слое.
5. Config, paths, fs, lock, buildinfo живут в platform-слое.
6. `internal` перестает быть плоской кучей пакетов и становится набором понятных namespace-ов.
7. Каждый этап можно выполнить отдельным коммитом.
8. Можно продолжить работу с любого этапа, если предыдущие этапы уже выполнены.

Обратную совместимость внутренних Go-пакетов сохранять не нужно.

Пользовательское CLI-поведение желательно сохранить, но если в процессе реорганизации нужно поменять внутренние типы, DTO, package paths или helper-функции — это разрешено.

---

# 2. Общие инструкции для агента

Этот раздел нужно давать агенту каждый раз перед описанием конкретного этапа.

## 2.1. Как работать

Работай маленькими шагами.

Каждый этап из этого ТЗ должен быть отдельным логическим коммитом в одной ветке.

После завершения каждого этапа:

1. Запусти проверки.
2. Убедись, что проект компилируется.
3. Убедись, что нет незавершенных временных костылей без явного TODO.
4. Сделай один коммит с понятным сообщением.

Если на этапе обнаружилось, что часть работы уже выполнена, не делай ее повторно. Проверь фактическое состояние кода и доведи этап до критериев готовности.

Если этап частично выполнен, продолжай с места остановки.

Если этап оказался слишком большим, можно разбить его на несколько внутренних подшагов, но внешний результат этапа должен быть одним чистым коммитом, если возможно.

## 2.2. Основное архитектурное направление

Целевая архитектура:

```text
cmd/mnemonic
  -> internal/adapter/cli
  -> internal/app
  -> internal/service/*
  -> internal/store/*
  -> internal/domain/*
  -> internal/format/*
  -> internal/platform/*
```

Разрешенные зависимости:

```text
adapter -> app/service/domain/apperr
app -> service/store/domain/format/platform/apperr
service -> store/domain/format/platform/apperr
store -> domain/format/platform/apperr
format -> standard library / parser libraries
platform -> standard library / low-level third-party libraries
domain -> mostly standard library only
```

Запрещенные зависимости:

```text
service -> adapter
store -> service
domain -> service
domain -> store
domain -> adapter
platform -> app
platform -> service
platform -> adapter
```

`app` — это composition root. Он может импортировать разные слои, потому что его задача — собрать зависимости.

## 2.3. Что нужно делать

Нужно:

1. Реорганизовать пакеты внутри `internal`.
2. Убрать бизнес-логику из CLI.
3. Убрать бизнес-логику из MCP tools.
4. Убрать бизнес-логику из Web adapter.
5. Ввести service-слой для project, notes, query.
6. Ввести store-слой для registry, indexdb, markdown files, webauth.
7. Ввести domain-пакеты с чистыми типами.
8. Ввести `apperr` как общий пакет application errors.
9. Убрать скрытые глобальные зависимости.
10. Убрать package-level runtime hooks вроде глобальных registry parser variables.
11. Убрать прямое открытие SQLite из CLI/MCP/Web.
12. Убрать прямой project resolving из CLI/MCP/Web.
13. Убрать прямой filesystem deletion project/index/note state из CLI.
14. Сделать CLI-команды через constructors, а не через `init()`.
15. Сделать каждый этап отдельным понятным коммитом.

## 2.4. Что не нужно делать

Не нужно:

1. Добавлять новые пользовательские фичи.
2. Переписывать проект с нуля.
3. Менять markdown-формат заметок.
4. Менять TOML-формат config/manifest без отдельной необходимости.
5. Делать публичный Go API.
6. Сохранять старые internal import paths.
7. Делать миграции SQLite-схемы без причины.
8. Оптимизировать производительность.
9. Делать web UI.
10. Добавлять сложные абстракции ради абстракций.
11. Тащить в проект framework-style архитектуру, если простого service/store разделения достаточно.
12. Перемешивать рефакторинг и изменение поведения без явного указания.

## 2.5. Стиль реализации

Сохраняй Go-idiomatic стиль:

1. Пакеты называются коротко:
   - `projectsvc`
   - `notesvc`
   - `querysvc`
   - `registry`
   - `indexdb`
   - `markdownstore`
   - `markdown`
   - `paths`
   - `config`

2. Не используй длинные package names вроде:
   - `mnemonicprojectservice`
   - `internalprojectmanager`
   - `applicationusecaseservice`

3. Для service input/output используй простые DTO:
   - `CreateInput`
   - `CreateOutput`
   - `ListInput`
   - `ListOutput`
   - `ResolveInput`
   - `ResolveOutput`

4. Если контекст уже есть в package name, не дублируй его:
   - хорошо: `notesvc.CreateInput`
   - хуже: `notesvc.CreateNoteInput`, если в пакете только note usecases

5. Не используй `Manager` для всего подряд. Предпочитай:
   - `Service`
   - `Store`
   - `Resolver`
   - `Factory`, если это действительно factory

## 2.6. Что считать бизнес-логикой

Бизнес-логика — это все, что отвечает на вопросы:

1. Как выбрать проект?
2. Где находится memories root?
3. Где находится index.sqlite?
4. Нужно ли делать reindex?
5. Как создать заметку?
6. Как удалить заметку?
7. Как импортировать проект?
8. Как удалить проект?
9. Как выполнить doctor checks?
10. Как искать notes?
11. Как получить backlinks?
12. Как получить tags?
13. Как мапить registry errors в application errors?
14. Как работать с project kind central/local?

Эта логика должна быть в `internal/service/*` и частично в `internal/store/*`.

Она не должна быть в CLI/MCP/Web.

## 2.7. Что может остаться в CLI

CLI может содержать:

1. Cobra command definitions.
2. Flags.
3. Args validation на уровне CLI shape.
4. Чтение stdin.
5. Чтение файла, переданного через `--body-file`.
6. Human output formatting.
7. JSON output selection.
8. Exit code mapping.
9. Shell completion wiring.

CLI не должен содержать:

1. SQL.
2. `sql.Open`.
3. `registry.Resolve`.
4. `registry.Scan`.
5. `index.Path`.
6. `index.RebuildProjectIndex`.
7. `project.ResolveMemoriesRoot`.
8. Удаление registry/index/state файлов.
9. Doctor checks.
10. Markdown parsing/rendering бизнес-сценариев.
11. Reindex orchestration.

## 2.8. Что может остаться в MCP

MCP tools могут содержать:

1. Tool name.
2. Tool description.
3. Input schema.
4. Output schema.
5. Transport-level validation.
6. Mapping service output to MCP response.

MCP tools не должны содержать:

1. Markdown file operations.
2. SQLite open/query logic.
3. Reindex orchestration.
4. Project resolving.
5. Search SQL.
6. Дублирование create/edit/delete/list/show/search логики.

## 2.9. Что может остаться в Web

Web adapter может содержать:

1. HTTP routing.
2. Middleware.
3. Auth/session handling.
4. Permission checks.
5. Mapping HTTP/MCP-over-HTTP request to service call.

Web adapter не должен содержать:

1. Registry scanning logic.
2. Project root resolving logic.
3. Direct index DB opening.
4. Direct note file operations.
5. Duplicated MCP business logic.

---

# 3. Как проверять

Этот раздел нужно использовать после каждого этапа.

## 3.1. Базовая проверка

После каждого этапа выполнить:

```bash
go test ./...
go build ./cmd/mnemonic
go vet ./...
```

Если `go vet` падает из-за уже существующей проблемы, не исправляй ее в рамках этапа, если она не связана с текущим изменением. Зафиксируй в заметке к коммиту.

## 3.2. Проверка форматирования

Выполнить:

```bash
gofmt -w .
```

## 3.3. Проверка git diff

Перед коммитом выполнить:

```bash
git status
git diff
```

Проверить:

1. В diff нет случайных изменений.
2. Нет временных debug prints.
3. Нет закомментированных больших блоков старого кода.
4. Нет лишних файлов IDE.
5. Нет бинарников.
6. Нет временных файлов SQLite, lock, cache.
7. Нет сгенерированных мусорных файлов.

## 3.4. Ручной smoke-check CLI

Если проект можно запустить локально, после крупных этапов выполнить хотя бы:

```bash
go build -o /tmp/mnemonic ./cmd/mnemonic

/tmp/mnemonic version
/tmp/mnemonic config show
```

После этапов, связанных с notes/projects/query, проверить сценарий во временной директории:

```bash
TMPDIR="$(mktemp -d)"
export HOME="$TMPDIR/home"
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
export XDG_STATE_HOME="$TMPDIR/state"
export XDG_CACHE_HOME="$TMPDIR/cache"

mkdir -p "$HOME"

go build -o /tmp/mnemonic ./cmd/mnemonic

/tmp/mnemonic init TestProject
export MNEMONIC_PROJECT=testproject

/tmp/mnemonic project list
/tmp/mnemonic notes create --title "Hello" --tag test
/tmp/mnemonic notes list
/tmp/mnemonic notes show hello
/tmp/mnemonic project reindex testproject
/tmp/mnemonic notes search hello
/tmp/mnemonic tags list
/tmp/mnemonic project doctor testproject
```

Если какая-то команда не работает из-за текущего состояния проекта, не скрывай это. Зафиксируй, что именно не работает и почему.

## 3.5. Архитектурные grep-проверки

После этапов, где уже должны быть новые слои, выполнять выборочно:

```bash
grep -R "internal/adapter" -n internal/service internal/store internal/domain internal/platform || true
```

Ожидаемо: пусто.

```bash
grep -R "internal/service" -n internal/store internal/domain internal/platform || true
```

Ожидаемо: пусто.

```bash
grep -R "sql.Open" -n internal/adapter || true
```

Ожидаемо: пусто после миграции query/project/MCP/Web.

```bash
grep -R "func init()" -n internal/adapter/cli || true
```

Ожидаемо: пусто после этапа CLI namespace.

```bash
grep -R "var .*Cmd" -n internal/adapter/cli || true
```

Ожидаемо: пусто или только безопасные immutable declarations после этапа CLI namespace.

## 3.6. Правило коммитов

Каждый этап — отдельный коммит.

Формат сообщений:

```text
refactor: add application error package
refactor: move platform packages under internal/platform
refactor: introduce registry store
refactor: introduce project service session resolution
refactor: route notes cli through notes service
```

Не смешивай в одном коммите:

1. Перемещение пакетов.
2. Изменение поведения.
3. Добавление новых фич.
4. Массовое форматирование несвязанных файлов.

---

# 4. Реализация

## 4.1. Этап 0 — подготовить checklist и зафиксировать текущее поведение

### Цель

Создать опорный документ для ручной проверки поведения до и после реорганизации.

Этот этап не должен менять архитектуру.

### Что сделать

Создать файл:

```text
docs/refactor-checklist.md
```

В файл добавить:

````markdown
# Refactor checklist

## Build checks

```bash
go test ./...
go build ./cmd/mnemonic
```
````

## Basic CLI checks

```bash
go build -o /tmp/mnemonic ./cmd/mnemonic

/tmp/mnemonic version
/tmp/mnemonic config show
```

## Temporary environment

```bash
TMPDIR="$(mktemp -d)"
export HOME="$TMPDIR/home"
export XDG_CONFIG_HOME="$TMPDIR/config"
export XDG_DATA_HOME="$TMPDIR/data"
export XDG_STATE_HOME="$TMPDIR/state"
export XDG_CACHE_HOME="$TMPDIR/cache"

mkdir -p "$HOME"
```

## Project and notes smoke test

```bash
go build -o /tmp/mnemonic ./cmd/mnemonic

/tmp/mnemonic init TestProject
export MNEMONIC_PROJECT=testproject

/tmp/mnemonic project list
/tmp/mnemonic project show testproject
/tmp/mnemonic notes create --title "Hello" --tag test
/tmp/mnemonic notes list
/tmp/mnemonic notes show hello
/tmp/mnemonic project reindex testproject
/tmp/mnemonic notes search hello
/tmp/mnemonic tags list
/tmp/mnemonic project doctor testproject
/tmp/mnemonic notes edit hello --append "more text"
/tmp/mnemonic notes delete hello --dry-run
/tmp/mnemonic project remove testproject
```

## Notes

- If a command fails because of known current behavior, write it down here.
- Do not hide failures during refactoring.

````

Если markdown fenced code внутри markdown неудобен, оформи команды без вложенных fences.

### Что не делать

Не менять Go-код.

### Проверка

```bash
go test ./...
go build ./cmd/mnemonic
````

### Критерии готовности

1. Есть `docs/refactor-checklist.md`.
2. В документе есть build checks.
3. В документе есть smoke-check CLI.
4. Go-код не изменен.
5. Проверки проходят.

### Коммит

```text
docs: add refactor checklist
```

---

## 4.2. Этап 1 — вынести application errors в `internal/apperr`

### Цель

Убрать application errors из `internal/app`, потому что `app` должен быть composition root, а не общий пакет для всех слоев.

### Текущее состояние

Сейчас ошибки приложения находятся примерно в:

```text
internal/app/errors.go
```

Там есть:

1. `ErrCode`
2. `AppError`
3. `NewAppError`
4. `NewInternalError`
5. `NewCLIUsageError`
6. `NewNotFoundError`
7. `NewAmbiguousError`
8. `NewUnsafeError`
9. `NewCorruptedError`

Эти ошибки используются в CLI, notes, index, webauth и других пакетах.

### Что сделать

Создать новый пакет:

```text
internal/apperr/errors.go
```

Содержимое должно быть примерно таким:

```go
package apperr

import "fmt"

type Code int

const (
    CodeSuccess   Code = 0
    CodeInternal  Code = 1
    CodeCLIUsage  Code = 2
    CodeNotFound  Code = 3
    CodeAmbiguous Code = 4
    CodeUnsafe    Code = 5
    CodeCorrupted Code = 6
)

type Error struct {
    Code    Code
    Message string
    Err     error
}

func (e *Error) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Err)
    }
    return e.Message
}

func (e *Error) Unwrap() error {
    return e.Err
}

func New(code Code, msg string, err error) *Error {
    return &Error{
        Code:    code,
        Message: msg,
        Err:     err,
    }
}

func Internal(msg string, err error) *Error {
    return New(CodeInternal, msg, err)
}

func CLIUsage(msg string, err error) *Error {
    return New(CodeCLIUsage, msg, err)
}

func NotFound(msg string, err error) *Error {
    return New(CodeNotFound, msg, err)
}

func Ambiguous(msg string, err error) *Error {
    return New(CodeAmbiguous, msg, err)
}

func Unsafe(msg string, err error) *Error {
    return New(CodeUnsafe, msg, err)
}

func Corrupted(msg string, err error) *Error {
    return New(CodeCorrupted, msg, err)
}
```

Если хочется сохранить имя `ErrCode`, можно оставить:

```go
type ErrCode = Code
```

Но лучше выбрать одно имя и использовать его последовательно.

### Заменить использования

Заменить:

```go
app.NewInternalError(...)
app.NewCLIUsageError(...)
app.NewNotFoundError(...)
app.NewAmbiguousError(...)
app.NewUnsafeError(...)
app.NewCorruptedError(...)
```

на:

```go
apperr.Internal(...)
apperr.CLIUsage(...)
apperr.NotFound(...)
apperr.Ambiguous(...)
apperr.Unsafe(...)
apperr.Corrupted(...)
```

Заменить:

```go
var appErr *app.AppError
```

на:

```go
var appErr *apperr.Error
```

Заменить коды:

```go
app.CodeSuccess
app.CodeInternal
```

на:

```go
apperr.CodeSuccess
apperr.CodeInternal
```

### Обновить `ExitCodeForError`

Функция должна смотреть на `apperr.Error`.

Пример:

```go
func ExitCodeForError(err error) int {
    if err == nil {
        return int(apperr.CodeSuccess)
    }

    var appErr *apperr.Error
    if errors.As(err, &appErr) {
        return int(appErr.Code)
    }

    return int(apperr.CodeInternal)
}
```

### Удалить старый файл

Удалить:

```text
internal/app/errors.go
```

Если удаление невозможно за один проход из-за большого количества импортов, можно временно оставить aliases, но в конце этого же этапа aliases должны быть удалены.

### Что не делать

Не менять поведение ошибок.

Не менять тексты ошибок без необходимости.

Не менять exit code значения.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Дополнительно:

```bash
grep -R "NewCLIUsageError\|NewNotFoundError\|NewAmbiguousError\|NewUnsafeError\|NewCorruptedError\|AppError" -n internal || true
```

Ожидаемо: пусто или только в старых комментариях, которые лучше удалить.

### Критерии готовности

1. Есть пакет `internal/apperr`.
2. `internal/app/errors.go` удален.
3. Нет `app.AppError`.
4. Нет `app.NewCLIUsageError` и аналогичных constructors.
5. Exit code mapping работает через `apperr`.
6. Проверки проходят.

### Коммит

```text
refactor: add application error package
```

---

## 4.3. Этап 2 — перенести platform-пакеты в `internal/platform`

### Цель

Сгруппировать инфраструктурные пакеты, которые не являются бизнес-логикой.

### Перенести

Из:

```text
internal/buildinfo
internal/config
internal/paths
internal/fs
internal/lock
```

В:

```text
internal/platform/buildinfo
internal/platform/config
internal/platform/paths
internal/platform/fs
internal/platform/lock
```

### Что сделать

Выполнить переносы через `git mv`:

```bash
mkdir -p internal/platform

git mv internal/buildinfo internal/platform/buildinfo
git mv internal/config internal/platform/config
git mv internal/paths internal/platform/paths
git mv internal/fs internal/platform/fs
git mv internal/lock internal/platform/lock
```

Обновить импорты:

```go
"github.com/ilyachch/mnemonic/internal/buildinfo"
"github.com/ilyachch/mnemonic/internal/config"
"github.com/ilyachch/mnemonic/internal/paths"
"github.com/ilyachch/mnemonic/internal/fs"
"github.com/ilyachch/mnemonic/internal/lock"
```

на:

```go
"github.com/ilyachch/mnemonic/internal/platform/buildinfo"
"github.com/ilyachch/mnemonic/internal/platform/config"
"github.com/ilyachch/mnemonic/internal/platform/paths"
"github.com/ilyachch/mnemonic/internal/platform/fs"
"github.com/ilyachch/mnemonic/internal/platform/lock"
```

Package names оставить прежними:

```go
package buildinfo
package config
package paths
package fs
package lock
```

### Что не делать

Не менять код внутри пакетов без необходимости.

Не менять public API этих пакетов.

Не менять config discovery behavior.

Не менять path resolving behavior.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Проверить старые imports:

```bash
grep -R "internal/buildinfo\|internal/config\|internal/paths\|internal/fs\|internal/lock" -n . || true
```

Ожидаемо: пусто.

### Критерии готовности

1. Старых директорий `internal/buildinfo`, `internal/config`, `internal/paths`, `internal/fs`, `internal/lock` нет.
2. Новые директории есть под `internal/platform`.
3. Package names остались короткими.
4. Старых import paths нет.
5. Проверки проходят.

### Коммит

```text
refactor: move platform packages under internal platform
```

---

## 4.4. Этап 3 — перенести markdown-пакет в `internal/format/markdown`

### Цель

Отделить формат markdown от бизнес-логики notes/project/index.

### Что сделать

Перенести:

```text
internal/markdown
```

в:

```text
internal/format/markdown
```

Команды:

```bash
mkdir -p internal/format
git mv internal/markdown internal/format/markdown
```

Обновить импорты:

```go
"github.com/ilyachch/mnemonic/internal/markdown"
```

на:

```go
"github.com/ilyachch/mnemonic/internal/format/markdown"
```

Package name оставить:

```go
package markdown
```

### Что не делать

Не менять parsing behavior.

Не менять rendering behavior.

Не менять frontmatter behavior.

Не менять wikilink/tag/relation parsing behavior.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Проверить старый import:

```bash
grep -R "internal/markdown" -n . || true
```

Ожидаемо: пусто.

### Критерии готовности

1. `internal/markdown` удален.
2. `internal/format/markdown` существует.
3. Все импорты обновлены.
4. Проверки проходят.

### Коммит

```text
refactor: move markdown package under internal format
```

---

## 4.5. Этап 4 — создать domain-пакеты

### Цель

Добавить чистые domain-типы, которые можно будет использовать в service/store/adapter слоях без циклических импортов.

На этом этапе не нужно мигрировать весь код на новые типы. Достаточно добавить пакеты и начать использовать их там, где это безопасно.

### Создать структуру

```text
internal/domain/project
internal/domain/note
internal/domain/query
```

### `internal/domain/project/project.go`

Добавить:

```go
package project

type Kind string

const (
    KindCentral Kind = "central"
    KindLocal   Kind = "local"
)

type Record struct {
    ID           string `json:"id"`
    Name         string `json:"name"`
    Slug         string `json:"slug"`
    Kind         Kind   `json:"kind"`
    MemoriesPath string `json:"memories_path"`
}

type Resolution struct {
    MnemonicFilePath string `json:"mnemonic_file_path,omitempty"`
    RepoRootAbs      string `json:"repo_root_abs,omitempty"`
    MemoriesAbs      string `json:"memories_abs,omitempty"`
    ManifestAbs      string `json:"manifest_abs,omitempty"`
    Project          Record `json:"project"`
}

type Session struct {
    Resolution   Resolution `json:"resolution"`
    MemoriesRoot string     `json:"memories_root"`
    IndexPath    string     `json:"index_path"`
}
```

Если JSON tags не нужны, можно убрать, но DTO часто удобно возвращать наружу.

### `internal/domain/note/note.go`

Добавить:

```go
package note

type Summary struct {
    NoteID      string `json:"note_id"`
    Slug        string `json:"slug"`
    Title       string `json:"title"`
    Path        string `json:"path"`
    ContentHash string `json:"content_hash,omitempty"`
}

type Backlink struct {
    LinkID       string `json:"link_id"`
    NoteID       string `json:"note_id"`
    Slug         string `json:"slug"`
    Title        string `json:"title"`
    Path         string `json:"path"`
    RelationType string `json:"relation_type"`
    SourceLine   int    `json:"source_line"`
}

type TagSummary struct {
    Tag   string `json:"tag"`
    Count int    `json:"count"`
}
```

### `internal/domain/query/result.go`

Добавить:

```go
package query

type SearchResult struct {
    NoteID      string  `json:"note_id"`
    Slug        string  `json:"slug"`
    Title       string  `json:"title"`
    Path        string  `json:"path"`
    Score       float64 `json:"score"`
    Snippet     string  `json:"snippet,omitempty"`
    ContentHash string  `json:"content_hash,omitempty"`
}
```

Если текущий `search.Result` имеет другой набор полей, не ломай сейчас search. Позже можно адаптировать.

### Правила для domain-пакетов

Domain-пакеты не должны импортировать:

```text
internal/app
internal/adapter
internal/service
internal/store
internal/platform/config
internal/platform/paths
```

Допустимы:

1. standard library;
2. очень базовые типы, если без них нельзя.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Архитектурная проверка:

```bash
grep -R "internal/app\|internal/adapter\|internal/service\|internal/store" -n internal/domain || true
```

Ожидаемо: пусто.

### Критерии готовности

1. Есть `internal/domain/project`.
2. Есть `internal/domain/note`.
3. Есть `internal/domain/query`.
4. Domain-пакеты не зависят от app/service/store/adapter.
5. Проверки проходят.

### Коммит

```text
refactor: add domain packages
```

---

## 4.6. Этап 5 — перенести registry в `internal/store/registry` и убрать глобальные parser hooks

### Цель

Сделать registry явным store с dependency injection вместо глобальных переменных парсеров.

### Текущее состояние

Сейчас registry использует глобальные parser hooks, которые устанавливаются из `app.New`.

Это плохо, потому что:

1. `registry` выглядит самостоятельным, но работает правильно только после вызова `app.New`.
2. Тестировать registry отдельно сложнее.
3. Порядок инициализации становится скрытой зависимостью.

### Что сделать

Перенести пакет:

```bash
mkdir -p internal/store
git mv internal/registry internal/store/registry
```

Package name оставить:

```go
package registry
```

Обновить импорты:

```go
"github.com/ilyachch/mnemonic/internal/registry"
```

на:

```go
"github.com/ilyachch/mnemonic/internal/store/registry"
```

### Добавить `Store`

В `internal/store/registry` добавить типы:

```go
type ManifestParser func(path string) (ManifestData, error)
type PointerParser func(data []byte) (string, error)

type Store struct {
    MemoriesHome   string
    ManifestParser ManifestParser
    PointerParser  PointerParser
}
```

Добавить validate/helper:

```go
func (s Store) validate() error {
    if strings.TrimSpace(s.MemoriesHome) == "" {
        return fmt.Errorf("memories home is required")
    }
    if s.ManifestParser == nil {
        return fmt.Errorf("manifest parser is required")
    }
    if s.PointerParser == nil {
        return fmt.Errorf("pointer parser is required")
    }
    return nil
}
```

Если `PointerParser` нужен не во всех операциях, можно валидировать только там, где он нужен.

### Переписать API на методы Store

Целевые методы:

```go
func (s Store) Scan() ([]Entry, []Issue, error)
func (s Store) Resolve(selector string) (Entry, error)
func (s Store) Slugs() ([]string, error)
```

Если сейчас есть package-level функции:

```go
func Scan(memoriesHome string) ...
func Resolve(memoriesHome, selector string) ...
func Slugs(memoriesHome string) ...
```

нужно постепенно заменить их на методы.

В рамках этого этапа желательно удалить package-level функции. Если это слишком большой diff, можно временно оставить wrappers, но только если они не используют глобальные parser hooks.

Пример временного wrapper, если нужен:

```go
func ScanWithParsers(memoriesHome string, manifestParser ManifestParser, pointerParser PointerParser) (...) {
    return Store{
        MemoriesHome: memoriesHome,
        ManifestParser: manifestParser,
        PointerParser: pointerParser,
    }.Scan()
}
```

Но лучше сразу перевести call sites на `Store`.

### Удалить глобальные parser variables

Удалить:

```go
DefaultManifestParser
DefaultPointerParser
```

или любые аналоги.

Удалить функцию в `app`, которая мутирует глобальный registry state:

```go
wireRegistryParsers()
```

### Собрать registry store в `app.New`

В `internal/app/app.go` после resolving paths создать:

```go
registryStore := registry.Store{
    MemoriesHome: effective.MemoriesHome,
    ManifestParser: func(path string) (registry.ManifestData, error) {
        m, err := project.ParseMnemonicManifestFromFile(path)
        if err != nil {
            return registry.ManifestData{}, err
        }

        typ := "central"
        if m.IsLocal() {
            typ = "local"
        }

        return registry.ManifestData{
            ProjectID: m.ProjectID,
            Name:      m.Name,
            Slug:      m.Slug,
            Type:      typ,
        }, nil
    },
    PointerParser: func(data []byte) (string, error) {
        p, err := project.ParsePointerFile(data)
        if err != nil {
            return "", err
        }
        return p.ManifestPath, nil
    },
}
```

Если `project` позже будет перенесен, пока использовать текущий пакет.

### Временно обновить старые места

Все места, которые раньше делали:

```go
registry.Resolve(memoriesHome, selector)
registry.Scan(memoriesHome)
registry.Slugs(memoriesHome)
```

должны получить `registry.Store`.

Если место пока не имеет доступа к `app.Stores.Registry`, можно временно создать store локально, но лучше проводить через `app`.

### Обновить `app.App`

Добавить:

```go
type Stores struct {
    Registry registry.Store
}
```

И в `App`:

```go
Stores Stores
```

Пока можно оставить `Services` как есть.

### Что не делать

Не менять формат registry files.

Не менять manifest/pointer parsing behavior.

Не менять project list/show output без необходимости.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Проверить:

```bash
grep -R "DefaultManifestParser\|DefaultPointerParser\|wireRegistryParsers" -n internal || true
```

Ожидаемо: пусто.

```bash
grep -R "internal/registry" -n . || true
```

Ожидаемо: пусто.

### Критерии готовности

1. Registry находится в `internal/store/registry`.
2. Есть `registry.Store`.
3. Runtime parser dependencies передаются явно.
4. Нет `DefaultManifestParser`.
5. Нет `DefaultPointerParser`.
6. Нет `wireRegistryParsers`.
7. `app.New` создает `registry.Store`.
8. Старые imports обновлены.
9. Проверки проходят.

### Коммит

```text
refactor: introduce registry store
```

---

## 4.7. Этап 6 — перенести index в `internal/store/indexdb` и убрать скрытую зависимость от global paths

### Цель

Сделать SQLite index store с явным `StateHome`, чтобы index path не вычислялся через глобальный config/env внутри низкоуровневого пакета.

### Текущее состояние

Сейчас index path вычисляется через функцию вроде:

```go
index.Path(projectID)
```

и внутри она вызывает paths resolving.

Это создает скрытую зависимость низкоуровневого index-пакета от глобального окружения приложения.

### Что сделать

Перенести пакет:

```bash
git mv internal/index internal/store/indexdb
```

Переименовать package name:

```go
package indexdb
```

Обновить импорты:

```go
"github.com/ilyachch/mnemonic/internal/index"
```

на:

```go
"github.com/ilyachch/mnemonic/internal/store/indexdb"
```

### Добавить Store

Создать:

```go
type Store struct {
    StateHome string
}
```

### Реализовать path methods

```go
func (s Store) Path(projectID string) (string, error) {
    projectID = strings.TrimSpace(projectID)
    if projectID == "" {
        return "", fmt.Errorf("project id is required")
    }
    if strings.ContainsAny(projectID, `/\`) {
        return "", fmt.Errorf("project id must not contain path separators")
    }

    if strings.TrimSpace(s.StateHome) == "" {
        return "", fmt.Errorf("state home is required")
    }

    return filepath.Join(
        s.StateHome,
        "mnemonic",
        "projects",
        projectID,
        "index.sqlite",
    ), nil
}
```

Добавить:

```go
func (s Store) TempPath(projectID string) (string, error)
```

Логика:

```go
p, err := s.Path(projectID)
if err != nil {
    return "", err
}
return strings.TrimSuffix(p, ".sqlite") + ".new.sqlite", nil
```

### Обновить OpenDB

Было:

```go
func OpenDB(projectID string) (*sql.DB, error)
```

Стало:

```go
func (s Store) Open(projectID string) (*sql.DB, error)
```

Внутри использовать:

```go
dbPath, err := s.Path(projectID)
```

### Обновить Rebuild

Было:

```go
func RebuildProjectIndex(projectID string, root string) (ReindexResult, error)
```

Стало:

```go
func (s Store) Rebuild(projectID string, root string) (ReindexResult, error)
```

Внутри:

1. lock acquisition должен использовать store-aware lock path или временно существующий lock.
2. temp path должен идти через `s.TempPath`.
3. final path должен идти через `s.Path`.

Если lock package все еще сам использует global paths, оставить на следующий этап или исправить сейчас, если просто.

### Добавить Exists/OpenReadonly/Remove

Добавить методы:

```go
func (s Store) Exists(projectID string) (bool, error)
func (s Store) OpenReadonly(projectID string) (*sql.DB, error)
func (s Store) Remove(projectID string) error
```

`Exists`:

```go
p, err := s.Path(projectID)
if err != nil {
    return false, err
}
_, err = os.Stat(p)
if err == nil {
    return true, nil
}
if os.IsNotExist(err) {
    return false, nil
}
return false, err
```

`OpenReadonly`:

```go
p, err := s.Path(projectID)
if err != nil {
    return nil, err
}
return sql.Open(sqliteDriverName, "file:"+filepath.ToSlash(p)+"?mode=ro")
```

`Remove` удаляет:

```text
index.sqlite
index.sqlite-wal
index.sqlite-shm
```

### Собрать index store в `app.New`

В `app.New` создать:

```go
indexStore := indexdb.Store{
    StateHome: effective.StateHome,
}
```

Добавить в `App.Stores`:

```go
Index indexdb.Store
```

### Убрать package-level Path usage из нового кода

Все новые места должны использовать:

```go
app.Stores.Index.Path(projectID)
```

или service dependency:

```go
s.Index.Path(projectID)
```

Если пока остались старые package-level функции, они должны быть временными и не использовать global paths.

### Что не делать

Не менять SQLite schema.

Не менять search behavior.

Не менять FTS behavior.

Не менять reindex algorithm, кроме path dependency.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Проверить старый import:

```bash
grep -R "internal/index" -n . || true
```

Ожидаемо: пусто.

Проверить hidden paths dependency:

```bash
grep -R "GetMnemonicPaths" -n internal/store/indexdb || true
```

Ожидаемо: пусто.

### Критерии готовности

1. Index находится в `internal/store/indexdb`.
2. Package name — `indexdb`.
3. Есть `indexdb.Store`.
4. Path к index.sqlite считается через `Store.StateHome`.
5. `indexdb` не вызывает global paths resolving.
6. `app.New` создает `indexdb.Store`.
7. Проверки проходят.

### Коммит

```text
refactor: introduce index database store
```

---

## 4.8. Этап 7 — создать `projectsvc.Service` и единый `ProjectSession`

### Цель

Сделать единую точку выбора проекта и вычисления связанных путей:

```text
selector -> ProjectSession
```

После этого CLI/MCP/Web не должны самостоятельно решать:

1. Какой project выбран.
2. Где memories root.
3. Где manifest.
4. Где repo root.
5. Где index.sqlite.

### Создать пакет

```text
internal/service/projectsvc
```

### Service

Создать:

```go
package projectsvc

type Service struct {
    Registry registry.Store
    Index    indexdb.Store
    Paths    paths.EffectivePaths
}
```

Импорты:

```go
internal/domain/project
internal/store/registry
internal/store/indexdb
internal/platform/paths
internal/apperr
```

### DTO

```go
type ResolveInput struct {
    ProjectSelector  string
    EnvironmentValue string
}
```

Возвращаемый тип:

```go
project.Session
```

из `internal/domain/project`.

### Реализовать `Resolve`

Логика:

1. `selector := strings.TrimSpace(input.ProjectSelector)`
2. Если selector пустой, взять `input.EnvironmentValue`.
3. Если selector всё еще пустой, вернуть:

```go
apperr.CLIUsage("no project selected; specify --project or set MNEMONIC_PROJECT", nil)
```

4. Вызвать:

```go
entry, err := s.Registry.Resolve(selector)
```

5. Если registry вернул not found, замапить в:

```go
apperr.NotFound(err.Error(), nil)
```

6. Собрать `project.Resolution`.

7. Вычислить `memoriesRoot`.

Правило:

- Если `entry.MemoriesAbs` уже абсолютный и корректный, использовать его.
- Для local/central использовать существующую функцию project layout resolving, если она есть.
- Не дублировать эту логику по CLI.

8. Вычислить index path:

```go
indexPath, err := s.Index.Path(entry.ProjectID)
```

9. Вернуть:

```go
project.Session{
    Resolution: resolution,
    MemoriesRoot: memoriesRoot,
    IndexPath: indexPath,
}
```

### Добавить helper methods

```go
func (s *Service) Slugs(ctx context.Context) ([]string, error)
func (s *Service) List(ctx context.Context) (ListOutput, error)
func (s *Service) Show(ctx context.Context, selector string) (ShowOutput, error)
```

На этом этапе `List` и `Show` могут быть простыми wrappers над registry.

### Обновить `app.Services`

В `internal/app/services.go` или новом файле:

```go
type Services struct {
    Projects *projectsvc.Service
}
```

В `app.New`:

```go
projectsService := &projectsvc.Service{
    Registry: registryStore,
    Index:    indexStore,
    Paths:    effective,
}
```

### Временно оставить старый ProjectResolver?

Если старый `app.ProjectResolver` используется во многих местах, можно оставить его на время. Но новые места должны идти через `Services.Projects`.

Если возможно без большого diff, удалить:

```text
internal/app/resolver.go
```

и старые types:

```go
ProjectResolver
ProjectResolveInput
ProjectResolution
ProjectRecord
```

Если удалить пока сложно, оставь TODO:

```go
// TODO(refactor): remove after CLI/MCP migrate to projectsvc.
```

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

### Критерии готовности

1. Есть `internal/service/projectsvc`.
2. Есть `projectsvc.Service`.
3. Есть `Resolve`.
4. `Resolve` возвращает `domain/project.Session`.
5. `app.Services.Projects` создается в `app.New`.
6. Новые зависимости передаются явно.
7. Проверки проходят.

### Коммит

```text
refactor: introduce project service session resolution
```

---

## 4.9. Этап 8 — создать `markdownstore`

### Цель

Отделить файловое хранилище markdown-заметок от service-логики.

Сейчас пакет notes одновременно может содержать:

1. Бизнес-операции create/edit/delete.
2. Filesystem operations.
3. Resolve note by selector.
4. Walk notes.
5. Hash.
6. Trash/hard delete logic.

Цель — постепенно перенести filesystem-oriented операции в `store/markdownstore`.

### Создать пакет

```text
internal/store/markdownstore
```

### Store

```go
package markdownstore

type Store struct{}
```

Пока Store может быть stateless.

### Целевой API

Добавить методы по мере возможности:

```go
func (s Store) Walk(root string) ([]string, error)
func (s Store) Resolve(root string, selector string) (ResolvedNote, error)
func (s Store) Read(root string, selector string) (ReadResult, error)
func (s Store) Create(input CreateInput) (CreateResult, error)
func (s Store) Edit(input EditInput) (EditResult, error)
func (s Store) Delete(input DeleteInput) (DeleteResult, error)
func (s Store) HashBytes(data []byte) string
```

### Практичный подход

Чтобы не переписывать всё сразу, на этом этапе можно сделать `markdownstore` как thin wrapper над текущим пакетом notes.

Пример:

```go
func (s Store) Walk(root string) ([]string, error) {
    return notes.Walk(root)
}
```

```go
func (s Store) Create(input CreateInput) (CreateResult, error) {
    created, err := notes.Create(notes.CreateInput{
        RootDir: input.RootDir,
        Title: input.Title,
        Body: input.Body,
        Tags: input.Tags,
    })
    if err != nil {
        return CreateResult{}, err
    }

    return CreateResult{
        NoteID: created.NoteID,
        Slug: created.Slug,
        Path: created.Path,
        ContentHash: created.ContentHash,
    }, nil
}
```

Позже старый `notes` можно удалить или разделить.

### DTO

Создать DTO, совпадающие по смыслу с текущими notes input/output:

```go
type CreateInput struct {
    RootDir string
    Title   string
    Body    []byte
    Tags    []string
}

type CreateResult struct {
    NoteID      string
    Slug        string
    Path        string
    ContentHash string
}
```

Аналогично для edit/delete/list/show/read.

### Добавить в `app.Stores`

```go
type Stores struct {
    Registry registry.Store
    Index    indexdb.Store
    Notes    markdownstore.Store
}
```

В `app.New`:

```go
notesStore := markdownstore.Store{}
```

### Что не делать

Не менять формат markdown files.

Не менять create/edit/delete behavior.

Не менять trash behavior.

Не менять hash algorithm.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

### Критерии готовности

1. Есть `internal/store/markdownstore`.
2. Есть `markdownstore.Store`.
3. Store предоставляет wrappers для основных note file operations.
4. `app.Stores.Notes` создается.
5. Проверки проходят.

### Коммит

```text
refactor: introduce markdown note store
```

---

## 4.10. Этап 9 — создать `notesvc.Service` и перевести notes CLI

### Цель

Убрать бизнес-логику из notes CLI-команд.

После этапа notes CLI должен:

1. Читать flags/args/stdin/body-file.
2. Создавать input DTO.
3. Вызывать `app.Services.Notes`.
4. Печатать результат.

### Создать пакет

```text
internal/service/notesvc
```

### Service

```go
type Service struct {
    Projects *projectsvc.Service
    Notes    markdownstore.Store
    Index    indexdb.Store
}
```

### Общий ProjectSelector

Добавить тип, который можно использовать и в других сервисах.

Вариант A — в `notesvc`:

```go
type ProjectSelector struct {
    CLIValue string
    EnvValue string
}
```

Вариант B — лучше создать общий пакет:

```text
internal/service/selector
```

и тип:

```go
package selector

type Project struct {
    CLIValue string
    EnvValue string
}
```

Если не хочется добавлять пакет, можно пока использовать локальный тип в `notesvc`, а позже вынести.

### DTO

Добавить:

```go
type CreateInput struct {
    ProjectSelector ProjectSelector
    Title           string
    Body            []byte
    Tags            []string
}

type CreateOutput struct {
    NoteID      string `json:"note_id"`
    Slug        string `json:"slug"`
    Path        string `json:"path"`
    ContentHash string `json:"content_hash"`
}
```

```go
type EditInput struct {
    ProjectSelector ProjectSelector
    Selector         string
    Append           []byte
    Body             []byte
    HasBody          bool
    Set              map[string]string
    IfMatch          string
}

type EditOutput struct {
    NoteID      string `json:"note_id"`
    Slug        string `json:"slug"`
    Path        string `json:"path"`
    ContentHash string `json:"content_hash"`
    CreatedAt   string `json:"created_at"`
    UpdatedAt   string `json:"updated_at"`
}
```

```go
type DeleteInput struct {
    ProjectSelector ProjectSelector
    Selector         string
    DryRun           bool
    Hard             bool
    Yes              bool
}

type DeleteOutput struct {
    // mirror current notes.Delete output if possible
}
```

```go
type ListInput struct {
    ProjectSelector ProjectSelector
}

type ListOutput struct {
    Notes []note.Summary `json:"notes"`
}
```

```go
type ShowInput struct {
    ProjectSelector ProjectSelector
    Selector         string
}

type ShowOutput struct {
    Note ShowNote `json:"note"`
}

type ShowNote struct {
    NoteID      string         `json:"note_id"`
    Slug        string         `json:"slug"`
    Title       string         `json:"title"`
    Path        string         `json:"path"`
    Frontmatter map[string]any `json:"frontmatter"`
    Body        string         `json:"body"`
    ContentHash string         `json:"content_hash"`
    UpdatedAt   string         `json:"updated_at"`
    Raw         string         `json:"-"`
}
```

`Raw` нужен для human output `notes show`, если текущая команда печатает весь markdown file.

### Реализовать методы

```go
func (s *Service) Create(ctx context.Context, input CreateInput) (CreateOutput, error)
func (s *Service) Edit(ctx context.Context, input EditInput) (EditOutput, error)
func (s *Service) Delete(ctx context.Context, input DeleteInput) (DeleteOutput, error)
func (s *Service) List(ctx context.Context, input ListInput) (ListOutput, error)
func (s *Service) Show(ctx context.Context, input ShowInput) (ShowOutput, error)
```

Каждый метод должен:

1. Validate input.
2. Resolve project через `s.Projects.Resolve`.
3. Использовать `session.MemoriesRoot`.
4. Вызвать `s.Notes`.
5. Для write operations вызвать `s.Index.Rebuild`.

### Reindex rule

После успешных write operations:

1. `Create` вызывает reindex.
2. `Edit` вызывает reindex.
3. `Delete` вызывает reindex, если `DryRun == false`.

Если reindex fails, метод должен вернуть ошибку. Не скрывать ошибку.

### Обновить `app.Services`

```go
type Services struct {
    Projects *projectsvc.Service
    Notes    *notesvc.Service
}
```

### Перевести CLI notes commands

Команды:

1. `notes create`
2. `notes edit`
3. `notes delete`
4. `notes list`
5. `notes show`

должны использовать `notesvc`.

CLI может продолжать жить в старом `internal/cli` на этом этапе, если CLI namespace еще не перенесен. Главное — убрать бизнес-логику.

### Удалить helper `resolveNotesProjectRoot`

Если он больше не используется, удалить.

Если используется только в search/backlinks/tags, он будет удален на следующем этапе.

### Что не делать

Не переносить весь CLI namespace на этом этапе.

Не переписывать Cobra registration.

Не менять user-facing commands без необходимости.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Smoke-check:

```bash
/tmp/mnemonic init TestProject
export MNEMONIC_PROJECT=testproject
/tmp/mnemonic notes create --title "Hello"
/tmp/mnemonic notes list
/tmp/mnemonic notes show hello
/tmp/mnemonic notes edit hello --append "more"
/tmp/mnemonic notes delete hello --dry-run
```

Архитектурный grep:

```bash
grep -R "ResolveMemoriesRoot\|registry.Resolve\|indexdb.*Rebuild\|RebuildProjectIndex" -n internal/cli/notes* || true
```

Ожидаемо: notes create/edit/delete/list/show не должны это использовать.

### Критерии готовности

1. Есть `internal/service/notesvc`.
2. `app.Services.Notes` создается.
3. Notes CLI write/read commands используют `notesvc`.
4. `notes create/edit/delete` не решают project root самостоятельно.
5. `notes create/edit/delete` не вызывают reindex напрямую из CLI.
6. Проверки проходят.

### Коммит

```text
refactor: route notes commands through notes service
```

---

## 4.11. Этап 10 — создать `querysvc.Service` и перевести search/tags/backlinks

### Цель

Убрать SQL и SQLite-opening из CLI-команд:

1. `notes search`
2. `tags list`
3. `notes backlinks`

### Создать пакет

```text
internal/service/querysvc
```

### Service

```go
type Service struct {
    Projects *projectsvc.Service
    Index    indexdb.Store
}
```

### DTO

```go
type SearchInput struct {
    ProjectSelector notesvc.ProjectSelector
    Query           string
    Limit           int
    Tag             string
}

type SearchOutput struct {
    Hits []query.SearchResult `json:"hits"`
}
```

```go
type ListTagsInput struct {
    ProjectSelector notesvc.ProjectSelector
}

type ListTagsOutput struct {
    Tags []note.TagSummary `json:"tags"`
}
```

```go
type BacklinksInput struct {
    ProjectSelector notesvc.ProjectSelector
    Selector         string
}

type BacklinksOutput struct {
    Links []note.Backlink `json:"links"`
}
```

Если `ProjectSelector` вынесен в общий пакет, использовать общий тип.

### Перенести query logic

Лучший вариант:

1. SQL methods живут в `store/indexdb`.
2. `querysvc` только orchestrates.

В `indexdb.Store` добавить:

```go
func (s Store) Search(ctx context.Context, db *sql.DB, input SearchQuery) ([]query.SearchResult, error)
func (s Store) ListTags(ctx context.Context, db *sql.DB) ([]note.TagSummary, error)
func (s Store) FindIndexedNote(ctx context.Context, db *sql.DB, selector string) (IndexedNote, error)
func (s Store) Backlinks(ctx context.Context, db *sql.DB, targetNoteID string) ([]note.Backlink, error)
```

Если текущие пакеты `search` и `graph` уже содержат эту логику, можно временно вызывать их из `querysvc`, но CLI не должен импортировать их напрямую.

### Реализовать `Search`

Логика:

1. Validate `input.Query`.
2. Resolve project session.
3. Проверить index exists:
   - если нет, вернуть `apperr.NotFound("index missing; run `mnemonic project reindex`", nil)`.

4. Открыть readonly DB через `s.Index.OpenReadonly(projectID)`.
5. Выполнить search.
6. Вернуть hits.

### Реализовать `ListTags`

Логика:

1. Resolve project session.
2. Проверить index exists.
3. Open readonly DB.
4. Query tags.
5. Если tags nil, вернуть empty slice, не nil.

### Реализовать `Backlinks`

Логика:

1. Validate selector.
2. Resolve project session.
3. Проверить index exists.
4. Open readonly DB.
5. Find indexed note by selector.
6. Query backlinks.
7. Если links nil, вернуть empty slice, не nil.

### Обновить `app.Services`

```go
type Services struct {
    Projects *projectsvc.Service
    Notes    *notesvc.Service
    Query    *querysvc.Service
}
```

### Перевести CLI

Команды:

1. `notes search`
2. `tags list`
3. `notes backlinks`

должны вызывать `app.Services.Query`.

CLI должен оставить только:

1. Args.
2. Flags.
3. Human formatting.

### Что удалить из CLI

Удалить из CLI:

1. `sql.Open`.
2. SQL query helpers.
3. `queryIndexedNoteBySelector`, если он только в CLI.
4. `listTags`, если он только в CLI.
5. Direct imports `search`, `graph`, `indexdb`, если больше не нужны.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Smoke-check:

```bash
/tmp/mnemonic init TestProject
export MNEMONIC_PROJECT=testproject
/tmp/mnemonic notes create --title "Hello" --tag test
/tmp/mnemonic project reindex testproject
/tmp/mnemonic notes search hello
/tmp/mnemonic tags list
/tmp/mnemonic notes backlinks hello
```

Grep:

```bash
grep -R "sql.Open" -n internal/cli || true
grep -R "SELECT " -n internal/cli || true
```

Ожидаемо: нет business SQL в CLI.

### Критерии готовности

1. Есть `internal/service/querysvc`.
2. `app.Services.Query` создается.
3. `notes search` использует `querysvc`.
4. `tags list` использует `querysvc`.
5. `notes backlinks` использует `querysvc`.
6. В CLI нет direct `sql.Open`.
7. В CLI нет SQL для search/tags/backlinks.
8. Проверки проходят.

### Коммит

```text
refactor: route query commands through query service
```

---

## 4.12. Этап 11 — перенести project commands в `projectsvc.Service`

### Цель

Убрать бизнес-логику project lifecycle из CLI.

### Команды

Перевести:

1. `init NAME`
2. `project list`
3. `project show`
4. `project reindex`
5. `project import`
6. `project remove`
7. `project doctor`

### Добавить методы в `projectsvc`

```go
func (s *Service) Init(ctx context.Context, input InitInput) (InitOutput, error)
func (s *Service) Import(ctx context.Context, input ImportInput) (ImportOutput, error)
func (s *Service) List(ctx context.Context) (ListOutput, error)
func (s *Service) Show(ctx context.Context, input ShowInput) (ShowOutput, error)
func (s *Service) Reindex(ctx context.Context, input ReindexInput) (ReindexOutput, error)
func (s *Service) Remove(ctx context.Context, input RemoveInput) (RemoveOutput, error)
func (s *Service) Doctor(ctx context.Context, input DoctorInput) (DoctorOutput, error)
```

### Init

Input:

```go
type InitInput struct {
    Name        string
    Description string
    Local       bool
    CWD         string
}
```

Output:

```go
type InitOutput struct {
    ProjectID    string `json:"project_id"`
    Name         string `json:"name"`
    Slug         string `json:"slug"`
    MemoriesRoot string `json:"memories_root"`
    Indexed      bool   `json:"indexed"`
}
```

Логика:

1. Validate name.
2. Determine mode central/local.
3. Использовать current project init logic.
4. Rebuild index after init.
5. Return output.

CLI должен передавать `cwd`, потому что это transport/platform boundary:

```go
cwd, err := os.Getwd()
```

Это допустимо в CLI.

### List

Output должен соответствовать текущему JSON смыслу:

```go
type ListOutput struct {
    Projects []ListItem `json:"projects"`
}
```

`ListItem`:

```go
type ListItem struct {
    ProjectID    string `json:"project_id"`
    Name         string `json:"name"`
    Slug         string `json:"slug"`
    Type         string `json:"type"`
    MemoriesPath string `json:"memories_path"`
    StatePath    string `json:"state_path"`
    Status       string `json:"status"`
    Issue        string `json:"issue,omitempty"`
}
```

Логика registry scan/issues переезжает из CLI в service.

### Show

Input:

```go
type ShowInput struct {
    Selector string
}
```

Output:

```go
type ShowOutput struct {
    ProjectID string         `json:"project_id"`
    Name      string         `json:"name"`
    Slug      string         `json:"slug"`
    Type      string         `json:"type"`
    StateHome string         `json:"state_home"`
    Location  LocationOutput `json:"location"`
}
```

### Reindex

Input:

```go
type ReindexInput struct {
    Selector string
    All      bool
}
```

Rules:

1. If selector provided, reindex single project.
2. If All true, reindex all projects.
3. If neither selector nor All, keep current behavior if there is one. If current behavior reindexes all, keep that.
4. Partial project errors should be represented in output.

### Import

Input:

```go
type ImportInput struct {
    Path   string
    DryRun bool
}
```

Service handles:

1. Import project.
2. Reindex imported candidates if not dry-run.
3. Error mapping.

### Remove

Input:

```go
type RemoveInput struct {
    Selector string
    Wipe     bool
}
```

Service handles:

1. Resolve registry entry.
2. Remove registry entry.
3. Remove index files.
4. Remove project state dir.
5. Wipe markdown if requested.

CLI must not call `os.Remove`, `os.RemoveAll` for project/index removal.

### Doctor

Move all doctor logic into service:

1. Registry file check.
2. Project path exists.
3. Manifest parse.
4. Index exists.
5. SQLite quick_check.
6. Schema status.
7. Duplicate note UUIDs.
8. Duplicate slugs.
9. Unresolved links.
10. Trash ignored.
11. Stale temp files.

Doctor output:

```go
type DoctorOutput struct {
    Status string        `json:"status"`
    Checks []DoctorCheck `json:"checks"`
}

type DoctorCheck struct {
    Name   string `json:"name"`
    Status string `json:"status"`
    Detail string `json:"detail,omitempty"`
    Count  int    `json:"count,omitempty"`
}
```

### Update CLI

CLI project commands should only:

1. Parse flags/args.
2. Call project service.
3. Format human output.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Smoke-check:

```bash
/tmp/mnemonic init TestProject
export MNEMONIC_PROJECT=testproject
/tmp/mnemonic project list
/tmp/mnemonic project show testproject
/tmp/mnemonic project reindex testproject
/tmp/mnemonic project doctor testproject
/tmp/mnemonic project remove testproject
```

Grep:

```bash
grep -R "registry.Resolve\|registry.Scan\|RebuildProjectIndex\|os.RemoveAll\|os.Remove(" -n internal/cli || true
```

Ожидаемо: project commands не должны делать это напрямую.

### Критерии готовности

1. Project lifecycle logic находится в `projectsvc`.
2. `init` использует `projectsvc.Init`.
3. `project list` использует `projectsvc.List`.
4. `project show` использует `projectsvc.Show`.
5. `project reindex` использует `projectsvc.Reindex`.
6. `project import` использует `projectsvc.Import`.
7. `project remove` использует `projectsvc.Remove`.
8. `project doctor` использует `projectsvc.Doctor`.
9. CLI больше не содержит project business logic.
10. Проверки проходят.

### Коммит

```text
refactor: route project commands through project service
```

---

## 4.13. Этап 12 — перенести CLI в `internal/adapter/cli` и убрать `init()` registration

### Цель

Сделать CLI настоящим adapter namespace.

Сейчас CLI, вероятно, живет в одном package `internal/cli` с global `RootCmd`, global flags и `init()`-регистрацией команд.

Целевое состояние:

1. CLI находится в `internal/adapter/cli`.
2. Нет global mutable `RootCmd`.
3. Команды создаются через constructors.
4. Нет `func init()` для регистрации команд.
5. Shared flags хранятся в shared options object.
6. Commands сгруппированы по namespace-пакетам.

### Целевая структура

```text
internal/adapter/cli/
  root.go
  execute.go
  output.go
  flags.go
  completion.go

internal/adapter/cli/configcmd/
  command.go

internal/adapter/cli/projectcmd/
  command.go
  init.go
  list.go
  show.go
  import.go
  remove.go
  reindex.go
  doctor.go

internal/adapter/cli/notescmd/
  command.go
  create.go
  edit.go
  delete.go
  list.go
  show.go
  search.go
  backlinks.go

internal/adapter/cli/tagscmd/
  command.go
  list.go

internal/adapter/cli/mcpcmd/
  command.go

internal/adapter/cli/webcmd/
  command.go
  serve.go
  users.go
  perms.go

internal/adapter/cli/versioncmd/
  command.go
```

Если это слишком большой diff, можно сначала перенести в `internal/adapter/cli` без подпакетов, но итог этапа должен максимально приблизиться к структуре выше.

### SharedOptions

Создать:

```go
type SharedOptions struct {
    JSON    bool
    Project string
}
```

Добавить метод:

```go
func (o *SharedOptions) ProjectSelector() notesvc.ProjectSelector {
    return notesvc.ProjectSelector{
        CLIValue: o.Project,
        EnvValue: os.Getenv("MNEMONIC_PROJECT"),
    }
}
```

Если `ProjectSelector` вынесен в общий package, использовать его.

### Root constructor

Вместо:

```go
var RootCmd = &cobra.Command{...}

func init() {
    RootCmd.AddCommand(...)
}
```

сделать:

```go
func NewRootCommand(a *app.App) *cobra.Command {
    shared := &SharedOptions{}

    root := &cobra.Command{
        Use:   "mnemonic",
        Short: "mnemonic is a local-first personal knowledge base and search engine",
        Long:  "...",
        Run: func(cmd *cobra.Command, args []string) {
            _ = cmd.Help()
        },
    }

    root.PersistentFlags().BoolVar(&shared.JSON, "json", false, "output in JSON format")
    root.PersistentFlags().StringVarP(&shared.Project, "project", "p", "", "select a project by slug or UUID")

    // completion
    _ = root.RegisterFlagCompletionFunc("project", completeProjectNames(a))

    root.AddCommand(configcmd.New(a, shared))
    root.AddCommand(projectcmd.New(a, shared))
    root.AddCommand(notescmd.New(a, shared))
    root.AddCommand(tagscmd.New(a, shared))
    root.AddCommand(mcpcmd.New(a, shared))
    root.AddCommand(webcmd.New(a, shared))
    root.AddCommand(versioncmd.New(a, shared))

    return root
}
```

Чтобы избежать import cycles, можно вынести shared utilities в:

```text
internal/adapter/cli/clibase
```

Тогда subcommands импортируют `clibase`, а root импортирует subcommands.

### Execute

Сделать:

```go
func Execute(a *app.App) error {
    cmd := NewRootCommand(a)
    return cmd.Execute()
}
```

Не вызывай `os.Exit` глубоко в CLI, кроме `main`.

Лучше, чтобы `main` управлял exit.

### Output

Убрать global `jsonFlag`.

Сделать:

```go
func PrintOutput(w io.Writer, jsonEnabled bool, humanStr string, data any) error {
    if jsonEnabled {
        encoder := json.NewEncoder(w)
        encoder.SetIndent("", "  ")
        return encoder.Encode(data)
    }
    _, err := io.WriteString(w, humanStr)
    return err
}
```

Или:

```go
type Printer struct {
    JSON bool
    W    io.Writer
}
```

### Completion

`completeProjectNames` должен использовать `a.Services.Projects.Slugs`.

Не должен напрямую импортировать registry.

### Что не делать

Не менять service logic.

Не менять command behavior без необходимости.

Не пытаться одновременно переносить MCP/Web.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Grep:

```bash
grep -R "func init()" -n internal/adapter/cli || true
grep -R "var .*Cmd" -n internal/adapter/cli || true
grep -R "internal/cli" -n . || true
```

Ожидаемо:

1. Нет `func init()` в CLI.
2. Нет global mutable command vars.
3. Нет imports старого `internal/cli`.

### Критерии готовности

1. CLI находится в `internal/adapter/cli`.
2. Старый `internal/cli` удален.
3. Нет global `RootCmd`.
4. Нет command registration через `init()`.
5. Shared flags не global variables.
6. Project completion использует project service.
7. Проверки проходят.

### Коммит

```text
refactor: reorganize cli adapter
```

---

## 4.14. Этап 13 — обновить `cmd/mnemonic/main.go`

### Цель

Сделать `main.go` тонкой точкой входа:

1. Set build version.
2. Create app.
3. Run CLI.
4. Handle exit code.

### Целевой код

`cmd/mnemonic/main.go` должен быть примерно таким:

```go
package main

import (
    "fmt"
    "os"

    "github.com/ilyachch/mnemonic/internal/adapter/cli"
    "github.com/ilyachch/mnemonic/internal/app"
    "github.com/ilyachch/mnemonic/internal/apperr"
    "github.com/ilyachch/mnemonic/internal/platform/buildinfo"
)

var version = "dev"

func main() {
    buildinfo.SetVersion(version)

    application, err := app.New(app.Input{})
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(int(apperr.CodeInternal))
    }
    defer func() {
        _ = application.Close()
    }()

    if err := cli.Execute(application); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(cli.ExitCodeForError(err))
    }
}
```

Если `cli.Execute` уже сам печатает ошибку и вызывает `os.Exit`, изменить его так, чтобы он возвращал error.

### Что не делать

Не добавлять business logic в main.

Не читать config в main, если это уже делает `app.New`.

Не создавать services в main руками.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Проверить:

```bash
grep -R "internal/cli" -n cmd internal || true
```

Ожидаемо: пусто.

### Критерии готовности

1. `main.go` импортирует `internal/adapter/cli`.
2. `main.go` создает `app.App`.
3. `main.go` не содержит business logic.
4. `cli.Execute` возвращает error.
5. Проверки проходят.

### Коммит

```text
refactor: simplify mnemonic entrypoint
```

---

## 4.15. Этап 14 — перенести MCP в `internal/adapter/mcp` и перевести tools на services

### Цель

MCP должен использовать те же usecases, что CLI.

### Перенести пакет

```bash
mkdir -p internal/adapter
git mv internal/mcp internal/adapter/mcp
```

Package name можно оставить:

```go
package mcp
```

Обновить импорты.

### Обновить MCP dependencies

Старый MCP может иметь dependencies типа:

```go
GetMemoriesRoot()
GetIndexDB()
RebuildIndex()
```

Целевое состояние:

```go
type Dependencies struct {
    Projects *projectsvc.Service
    Notes    *notesvc.Service
    Query    *querysvc.Service
    Selector notesvc.ProjectSelector
    ReadOnly bool
}
```

Если dependencies передаются иначе, сохранить style проекта, но tools должны вызывать service слой.

### Перевести tools

#### create_note

Должен вызывать:

```go
deps.Notes.Create(ctx, notesvc.CreateInput{...})
```

Не должен:

1. Получать memories root.
2. Вызывать notes.Create напрямую.
3. Вызывать RebuildIndex напрямую.

#### edit_note

Должен вызывать:

```go
deps.Notes.Edit(ctx, notesvc.EditInput{...})
```

MCP-specific constraints вроде read-only или replace_body with if_match можно проверять в tool или service. Но бизнес-валидация должна быть в service.

#### delete_note

Должен вызывать:

```go
deps.Notes.Delete(ctx, notesvc.DeleteInput{...})
```

#### list_notes

Должен вызывать:

```go
deps.Notes.List(ctx, notesvc.ListInput{...})
```

#### read_note

Должен вызывать:

```go
deps.Notes.Show(ctx, notesvc.ShowInput{...})
```

#### search_notes

Должен вызывать:

```go
deps.Query.Search(ctx, querysvc.SearchInput{...})
```

#### list_tags

Должен вызывать:

```go
deps.Query.ListTags(ctx, querysvc.ListTagsInput{...})
```

#### list_backlinks

Должен вызывать:

```go
deps.Query.Backlinks(ctx, querysvc.BacklinksInput{...})
```

### Обновить CLI `mcp` command

CLI command `mcp` должен создавать MCP server через new adapter path и передавать services.

### Что не делать

Не менять MCP protocol behavior без необходимости.

Не менять tool names без необходимости.

Не менять JSON schemas без необходимости.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Grep:

```bash
grep -R "internal/mcp" -n . || true
grep -R "sql.Open\|RebuildProjectIndex\|GetIndexDB\|GetMemoriesRoot" -n internal/adapter/mcp || true
```

Ожидаемо: no direct business infra in tools.

### Критерии готовности

1. MCP находится в `internal/adapter/mcp`.
2. Старый `internal/mcp` удален.
3. MCP tools вызывают `notesvc`/`querysvc`/`projectsvc`.
4. MCP tools не открывают SQLite напрямую.
5. MCP tools не делают reindex напрямую.
6. Проверки проходят.

### Коммит

```text
refactor: route mcp tools through services
```

---

## 4.16. Этап 15 — перенести Web adapter и webauth store

### Цель

Сгруппировать Web как adapter, а webauth DB как store.

### Перенести Web

```bash
git mv internal/web internal/adapter/web
```

Package name оставить:

```go
package web
```

Обновить imports.

### Перенести webauth

```bash
git mv internal/webauth internal/store/webauth
```

Package name оставить:

```go
package webauth
```

Обновить imports.

### Обновить Web manager dependencies

Целевой constructor:

```go
type ManagerInput struct {
    Services       app.Services
    AuthStore      *webauth.Store
    SuperuserToken string
    ServeProjects  []string
}

func NewServerManager(input ManagerInput) (*ServerManager, error)
```

Если импорт `app.Services` из adapter/web создает неудобство, можно создать отдельный dependency struct:

```go
type Services struct {
    Projects *projectsvc.Service
    Notes    *notesvc.Service
    Query    *querysvc.Service
}
```

### Project validation

Web не должен напрямую использовать registry.

Заменить direct registry resolving/scanning на:

```go
services.Projects.Resolve(...)
services.Projects.Show(...)
services.Projects.List(...)
```

### Web CLI commands

Команды:

1. `web serve`
2. `web users add/list/revoke`
3. `web perms grant/revoke`

должны импортировать новые paths:

```go
internal/adapter/web
internal/store/webauth
```

Если web users/perms содержит бизнес-логику, можно создать:

```text
internal/service/websvc
```

Но если это простой CRUD over webauth store, допустимо временно оставить в CLI adapter, потому что это admin transport logic. Главное — не смешивать с project registry/index logic.

### Что не делать

Не менять auth token format.

Не менять DB schema webauth без необходимости.

Не менять HTTP endpoints без необходимости.

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Grep:

```bash
grep -R "internal/web" -n . || true
grep -R "internal/webauth" -n . || true
grep -R "internal/store/registry" -n internal/adapter/web || true
```

Ожидаемо:

1. Старых imports нет.
2. Web adapter не использует registry напрямую.

### Критерии готовности

1. Web adapter находится в `internal/adapter/web`.
2. Webauth store находится в `internal/store/webauth`.
3. Старые директории удалены.
4. Web использует services для project operations.
5. Проверки проходят.

### Коммит

```text
refactor: reorganize web adapter and auth store
```

---

## 4.17. Этап 16 — финальная чистка старых пакетов, wrappers и dependency audit

### Цель

Удалить временные совместимости, wrappers, старые helper-и и проверить направление зависимостей.

### Удалить старые пакеты, если остались

Проверить и удалить:

```text
internal/notes
internal/project
internal/search
internal/graph
```

Только если их логика уже перенесена.

Если часть пакета все еще нужна, определить правильное место:

1. Markdown parsing -> `internal/format/markdown`
2. Note file operations -> `internal/store/markdownstore`
3. Note usecases -> `internal/service/notesvc`
4. Project manifest parsing -> возможно `internal/domain/project` или `internal/store/registry` helper
5. Project usecases -> `internal/service/projectsvc`
6. Search SQL -> `internal/store/indexdb`
7. Graph/backlinks SQL -> `internal/store/indexdb`

Не удалять пакет, если это сломает функциональность. Но не оставлять package с неясной ответственностью.

### Удалить compatibility wrappers

Удалить package-level wrappers, если появились:

```go
registry.Resolve(...)
registry.Scan(...)
registry.Slugs(...)
indexdb.Path(...)
indexdb.RebuildProjectIndex(...)
```

Цель:

1. Registry operations идут через `registry.Store`.
2. Index operations идут через `indexdb.Store`.
3. Notes operations идут через `notesvc`/`markdownstore`.
4. Query operations идут через `querysvc`.

### Проверить отсутствие старых imports

Выполнить:

```bash
grep -R "internal/cli" -n . || true
grep -R "internal/mcp" -n . || true
grep -R "internal/web" -n . || true
grep -R "internal/registry" -n . || true
grep -R "internal/index" -n . || true
grep -R "internal/markdown" -n . || true
grep -R "internal/config" -n . || true
grep -R "internal/paths" -n . || true
grep -R "internal/fs" -n . || true
grep -R "internal/lock" -n . || true
grep -R "internal/buildinfo" -n . || true
grep -R "internal/webauth" -n . || true
```

Ожидаемо: пусто.

### Dependency audit

```bash
grep -R "internal/adapter" -n internal/service internal/store internal/domain internal/platform || true
```

Ожидаемо: пусто.

```bash
grep -R "internal/service" -n internal/store internal/domain internal/platform || true
```

Ожидаемо: пусто.

```bash
grep -R "internal/app" -n internal/service internal/store internal/domain internal/platform || true
```

Ожидаемо: пусто.

```bash
grep -R "sql.Open" -n internal/adapter || true
```

Ожидаемо: пусто.

```bash
grep -R "SELECT \|INSERT \|UPDATE \|DELETE " -n internal/adapter || true
```

Ожидаемо: нет business SQL.

```bash
grep -R "func init()" -n internal/adapter/cli || true
```

Ожидаемо: пусто.

```bash
grep -R "var .*Cmd" -n internal/adapter/cli || true
```

Ожидаемо: пусто.

### Создать архитектурную документацию

Создать:

```text
docs/architecture.md
docs/package-map.md
```

#### `docs/package-map.md`

Содержимое:

```markdown
# Package map

| Old package          | New package                   |
| -------------------- | ----------------------------- |
| `internal/cli`       | `internal/adapter/cli`        |
| `internal/mcp`       | `internal/adapter/mcp`        |
| `internal/web`       | `internal/adapter/web`        |
| `internal/registry`  | `internal/store/registry`     |
| `internal/index`     | `internal/store/indexdb`      |
| `internal/markdown`  | `internal/format/markdown`    |
| `internal/config`    | `internal/platform/config`    |
| `internal/paths`     | `internal/platform/paths`     |
| `internal/fs`        | `internal/platform/fs`        |
| `internal/lock`      | `internal/platform/lock`      |
| `internal/buildinfo` | `internal/platform/buildinfo` |
| `internal/webauth`   | `internal/store/webauth`      |
```

#### `docs/architecture.md`

Документ должен описывать:

1. Layers.
2. Dependency direction.
3. Adapter responsibilities.
4. Service responsibilities.
5. Store responsibilities.
6. Domain responsibilities.
7. Platform responsibilities.
8. How to add a CLI command.
9. How to add an MCP tool.
10. How to add a query.
11. Error handling rules.
12. Output DTO rules.

Добавить раздел:

```markdown
## How to add `notes rename`

1. Add service method in `internal/service/notesvc`.
2. Add low-level file operation in `internal/store/markdownstore` if needed.
3. Rebuild index from service after successful write.
4. Add CLI command in `internal/adapter/cli/notescmd`.
5. Add MCP tool in `internal/adapter/mcp/tools` if needed.
6. Do not resolve project root in adapter.
7. Do not open SQLite in adapter.
```

### Проверка

```bash
gofmt -w .
go test ./...
go build ./cmd/mnemonic
```

Run smoke checklist from `docs/refactor-checklist.md`.

### Критерии готовности

1. Старые package paths не используются.
2. Временные wrappers удалены.
3. Direction dependencies проверены.
4. SQL отсутствует в adapters.
5. CLI больше не использует `init()` registration.
6. Есть `docs/architecture.md`.
7. Есть `docs/package-map.md`.
8. Smoke checklist пройден или failures явно задокументированы.
9. Проверки проходят.

### Коммит

```text
refactor: clean up architecture boundaries
```

---

# 5. Ожидаемый итоговый результат

После выполнения всех этапов структура проекта должна быть примерно такой:

```text
cmd/
  mnemonic/
    main.go

internal/
  app/
    app.go
    services.go
    stores.go

  apperr/
    errors.go

  domain/
    project/
      project.go
    note/
      note.go
    query/
      result.go

  service/
    projectsvc/
      service.go
      session.go
      init.go
      import.go
      list.go
      show.go
      remove.go
      reindex.go
      doctor.go
    notesvc/
      service.go
      create.go
      edit.go
      delete.go
      list.go
      show.go
    querysvc/
      service.go
      search.go
      tags.go
      backlinks.go

  store/
    registry/
      store.go
      scan.go
      resolve.go
      types.go
    indexdb/
      store.go
      paths.go
      open.go
      schema.go
      reindex.go
      queries.go
      check.go
      lock.go
    markdownstore/
      store.go
      walk.go
      resolve.go
      create.go
      edit.go
      delete.go
    webauth/
      store.go

  format/
    markdown/
      ast.go
      frontmatter.go
      note.go
      render.go
      tag.go
      wikilink.go
      relation.go
      observation.go

  adapter/
    cli/
      root.go
      execute.go
      output.go
      flags.go
      completion.go
      configcmd/
      projectcmd/
      notescmd/
      tagscmd/
      mcpcmd/
      webcmd/
      versioncmd/
    mcp/
      server.go
      tools/
    web/
      manager.go
      middleware.go
      handlers.go

  platform/
    buildinfo/
    config/
    paths/
    fs/
    lock/
```

## Итоговые свойства архитектуры

1. `cmd/mnemonic/main.go` тонкий.
2. `app.New` собирает stores и services.
3. `app` не является dumping ground для общих типов.
4. Application errors живут в `apperr`.
5. CLI не содержит business logic.
6. MCP tools не содержат business logic.
7. Web adapter не содержит business logic.
8. Project resolving живет в `projectsvc`.
9. Notes usecases живут в `notesvc`.
10. Search/tags/backlinks живут в `querysvc`.
11. Registry filesystem state живет в `store/registry`.
12. SQLite index state живет в `store/indexdb`.
13. Markdown files state живет в `store/markdownstore`.
14. Markdown format parsing/rendering живет в `format/markdown`.
15. Config/paths/fs/lock/buildinfo живут в `platform`.
16. Нет глобальных registry parser hooks.
17. Index path не вычисляется через global paths внутри indexdb.
18. CLI commands создаются через constructors.
19. Нет CLI command registration через `init()`.
20. Нет direct SQL в adapters.
21. Нет direct `sql.Open` в adapters.
22. Нет direct project root resolving в adapters.
23. Добавление новой фичи требует изменения service/store и тонкого adapter, а не правок в 7 несвязанных местах.

## Итоговый критерий успеха

Если завтра нужно добавить фичу `notes rename`, разработчик должен сделать примерно так:

1. Добавить `notesvc.Rename`.
2. При необходимости добавить low-level operation в `markdownstore`.
3. В `notesvc.Rename` вызвать reindex после успешного изменения.
4. Добавить CLI command `notescmd rename`.
5. При необходимости добавить MCP tool.
6. Не трогать registry resolving в CLI.
7. Не открывать SQLite в CLI.
8. Не дублировать root/index/path logic.

Если это возможно — реорганизация достигла цели.
