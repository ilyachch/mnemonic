# Оставшиеся доработки `fixes-and-improvements`

## Общие правила

* Не добавлять миграции.
* Не поддерживать старые контракты и форматы.
* Не добавлять compatibility wrappers.
* Не различать конкретные старые версии данных.
* Индекс SQLite считать производным артефактом, который можно полностью пересоздать.
* Не классифицировать ошибки по тексту сообщения.
* Не использовать прямой `time.Now()`, если в проекте предусмотрен clock abstraction.

---

# MNEMONIC-201 — Удалить version-based обновление SQLite-индекса

## Цель

Убрать механизм, который распознаёт старые версии индекса и автоматически адаптирует существующие установки.

## Контекст

Сейчас schema version увеличена до `3`, а `CheckSchemaStatus` возвращает `NeedsRebuild`, если найден другой номер версии.

Это создаёт upgrade path для ранее созданных индексов, хотя проект не требует обратной совместимости.

## Требования

Удалить использование:

```sql
PRAGMA user_version
```

Удалить запись:

```text
meta.schema_version
```

Удалить:

* комментарии вида `schema version 3`;
* сравнение текущей версии с предыдущими версиями;
* логику `version != N → rebuild`;
* формулировки `upgrade existing installation`.

`ApplySchema` должен создавать только текущую схему в новой пустой базе.

`reindex` должен:

1. создать новую временную базу;
2. применить текущую схему;
3. полностью заполнить индекс;
4. атомарно заменить старый файл.

Приложение не должно пытаться преобразовать существующий индекс.

При несовместимом или повреждённом индексе разрешается вернуть общую ошибку:

```text
index is invalid; rebuild it
```

Но нельзя определять, какая именно старая версия была обнаружена, и выполнять отдельную логику для неё.

## Условия приёмки

* В коде отсутствует `PRAGMA user_version`.
* В `meta` не хранится `schema_version`.
* В коде нет числовых версий SQLite schema.
* Нет веток для старой схемы или upgrade.
* `reindex` всегда строит индекс с нуля.
* Новый индекс содержит поле `relation_type`.
* Повреждённый или несовместимый индекс не мигрируется автоматически.

## Ключевые файлы

* `internal/store/sqliteindex/schema.go`
* `internal/store/sqliteindex/rebuild.go`
* места использования `CheckSchemaStatus`
* документация, описывающая rebuild

---

# MNEMONIC-202 — Завершить dependency injection логгера

## Цель

Передавать configured logger во все runtime paths без `nil` и без последующей мутации сервисов.

## Проблемы

Web adapter создаёт stdio/MCP server так:

```go
stdio.NewServer(..., input.ReadOnly, nil)
```

`catalogsvc` создаёт index service с `nil` logger при:

* `project add`;
* `project init`;
* import/reindex после импорта.

## Требования

Добавить logger в input web server:

```go
type ServerInput struct {
    // существующие поля
    Logger *slog.Logger
}
```

Передавать его в:

```go
stdio.NewServer(...)
```

При создании `indexsvc.Service` внутри `catalogsvc` использовать `s.Logger`:

```go
indexsvc.New(resolved, s.Logger)
```

Configured logger должен передаваться через constructors в:

* CLI runtime;
* stdio MCP;
* web MCP;
* maintenance runtime;
* catalog operations;
* notes service;
* search service;
* index service.

Не устанавливать logger через изменение public fields после создания сервиса.

## Условия приёмки

* Web MCP получает configured logger.
* `project add`, `project init` и import используют configured logger.
* В runtime-коде нет вызова `New(..., nil)`, если logger доступен у вызывающего сервиса.
* Stdio logs не попадают в stdout.
* Logger устанавливается только при создании объекта.

## Ключевые файлы

* `internal/adapter/web/manager.go`
* команда запуска web server
* `internal/service/catalogsvc/service.go`
* `internal/app/app.go`
* `internal/app/runtime.go`

---

# MNEMONIC-203 — Убрать классификацию ошибок `ShowMany` по строкам

## Цель

Сделать классификацию ошибок batch-read устойчивой к изменению текстов сообщений.

## Проблема

Сейчас используются проверки:

```go
strings.Contains(msg, "parse note")
strings.Contains(msg, "parse frontmatter")
strings.Contains(msg, "read note")
```

Изменение формулировки ошибки изменит внешний контракт `read_notes`.

## Требования

`markdownstore.Show()` должен возвращать typed `apperr.Error` для всех ожидаемых ошибок.

Минимальная классификация:

```text
CodeNotFound   → missing
CodeAmbiguous  → issue kind "ambiguous"
CodeCorrupted  → issue kind "corrupted"
CodeIO         → issue kind "io_error"
остальные      → issue kind "internal"
```

Если отдельного `CodeIO` сейчас нет, добавить его или использовать другой существующий код с однозначной семантикой.

В `notesvc.classifyShowError` разрешается использовать только:

```go
errors.As
errors.Is
```

Не использовать:

* `strings.Contains`;
* сравнение `err.Error()`;
* предположения о тексте ошибки.

## Условия приёмки

* Not found попадает только в `missing`.
* Ambiguous selector попадает в `issues` как `ambiguous`.
* Ошибка parsing попадает в `issues` как `corrupted`.
* Ошибка чтения или permission попадает в `issues` как `io_error`.
* Изменение текста сообщения не меняет классификацию.
* В `classifyShowError` отсутствует анализ строк.

## Ключевые файлы

* `internal/store/markdownstore/store.go`
* `internal/service/notesvc/service.go`
* `internal/apperr/*`

---

# MNEMONIC-204 — Добавить duration в batch-read logging

## Цель

Завершить operational logging для `read_notes`.

## Требования

В начале `ShowMany` получить текущее время через используемый в проекте clock abstraction.

После выполнения записать одно агрегированное событие:

```text
batch read completed
requested_count
found_count
missing_count
issue_count
duration
```

Не логировать:

* body;
* frontmatter;
* полный текст заметок.

Не создавать отдельное info-событие на каждый selector.

Отдельные ошибки selector можно логировать на debug.

## Условия приёмки

* В агрегированном событии присутствует `duration`.
* Duration вычисляется через project clock.
* Один batch-read создаёт одно агрегированное operational событие.
* Содержимое заметок не попадает в log.

## Ключевые файлы

* `internal/service/notesvc/service.go`
* clock package проекта

---

# MNEMONIC-205 — Удалить прямой `time.Now()` из diagnostics suggestions

## Цель

Использовать единый источник времени и сделать поведение сервиса детерминированным.

## Проблема

`searchCandidatesByTarget` использует:

```go
now := time.Now()
```

## Требования

Передать clock в `indexsvc.Service` либо использовать существующий project clock.

Возможный интерфейс:

```go
type Clock interface {
    Now() time.Time
}
```

`indexsvc.New` должен получать clock через dependency или использовать общий `clock.NowUTC()` проекта.

Время должно быть UTC.

Нельзя вызывать:

```go
time.Now()
```

непосредственно внутри diagnostics service.

## Условия приёмки

* В diagnostics отсутствует прямой `time.Now()`.
* Candidate search получает время из injected clock.
* Производственный runtime использует реальный clock.
* Поведение можно воспроизвести с фиксированным временем.

## Ключевые файлы

* `internal/service/indexsvc/service.go`
* `internal/service/indexsvc/diagnostics.go`
* `internal/app/runtime.go`
* clock package проекта

---

# MNEMONIC-206 — Сделать `tags` и `aliases` строго списками

## Цель

Оставить один canonical frontmatter format без поддержки альтернативного старого синтаксиса.

## Проблема

Сейчас принимается одиночная строка:

```yaml
tags: payment
```

и преобразуется в:

```go
[]string{"payment"}
```

## Требования

Поля:

```text
tags
aliases
```

должны принимать только YAML list of strings:

```yaml
tags:
  - payment
  - spei
```

Следующие варианты должны считаться невалидными:

```yaml
tags: payment
aliases: "old title"
tags: 123
tags:
  - payment
  - 123
```

При ошибке возвращать `FrontmatterFieldError`:

```text
Kind = invalid_string_list
Field = tags | aliases
```

Не выполнять автоматическое преобразование scalar → list.

## Условия приёмки

* List of strings успешно парсится.
* Одиночная строка отклоняется.
* Число отклоняется.
* Список со значением не-string отклоняется.
* Renderer всегда пишет list.
* Diagnostics возвращает имя поля.

## Ключевые файлы

* `internal/format/markdown/note.go`
* `internal/format/markdown/render.go`
* `internal/format/markdown/errors.go`

---

# MNEMONIC-207 — Централизовать ограничения входных данных

## Цель

Применять одинаковые ограничения в MCP, CLI и service layer.

## Проблемы

Сейчас большая часть ограничений находится только в stdio adapter.

Не обработаны полностью:

* отрицательные значения;
* количество diagnostic kinds;
* Unicode query length;
* чрезмерные CLI limits.

## Требования

Вынести validation в service layer или общий пакет.

### Search

Проверять:

```text
1 <= limit <= 100
queries count <= 8
query length <= 500 Unicode characters
```

Для отсутствующего limit применять default `10`.

Длину считать через:

```go
utf8.RuneCountInString
```

### Read notes

Проверять:

```text
identifiers count: 1..50
max_body_chars: 0..100000
```

Отрицательное `max_body_chars` должно возвращать ошибку.

### Diagnostics

Проверять:

```text
limit: 1..200
cursor >= 0
число kinds не больше числа поддерживаемых kinds
```

Неизвестный kind должен возвращать validation error.

### Общие правила

* Не обрезать значения молча.
* MCP и CLI должны получать одинаковую ошибку.
* Adapter может проверять input раньше, но service layer остаётся источником истины.

## Условия приёмки

* CLI не может обойти ограничения MCP.
* Русский запрос длиной 500 символов принимается.
* Русский запрос длиной 501 символ отклоняется.
* Отрицательный `max_body_chars` отклоняется.
* Отрицательный cursor отклоняется.
* Неизвестный diagnostic kind отклоняется.
* Ошибки имеют стабильный application error code.

## Ключевые файлы

* `internal/service/searchsvc/service.go`
* `internal/service/notesvc/service.go`
* `internal/service/indexsvc/service.go`
* `internal/adapter/stdio/tools.go`
* `internal/adapter/cli/notes_search.go`
* `internal/adapter/cli/project_doctor.go`

---

# MNEMONIC-208 — Выполнять candidate suggestions одним batch-запросом

## Цель

Не выполнять отдельный SQLite search для каждого уникального broken-link target.

## Проблема

Targets уже дедуплицируются и база открывается один раз, но для каждого target выполняется отдельный:

```go
SearchAdvanced(...)
```

## Требования

Добавить store-level метод:

```go
SearchCandidatesByTargets(
    db *sql.DB,
    targets []string,
    limitPerTarget int,
    now time.Time,
) (map[string][]SearchResult, error)
```

Метод должен:

* принять все уникальные targets;
* выполнить один составной SQL-запрос либо ограниченное фиксированное количество запросов;
* вернуть не более трёх candidates на target;
* сохранить детерминированный порядок;
* не применять related-notes expansion;
* не выполнять graph reranking, если оно не нужно для suggestions.

`indexsvc` должен один раз вызвать этот метод.

Удалить цикл с отдельным `SearchAdvanced` для каждого target.

## Условия приёмки

* Количество SQL search operations не растёт линейно от количества targets.
* Для каждого target возвращается не более трёх candidates.
* Один и тот же target ищется один раз.
* Ошибка suggestions не удаляет основные diagnostics.
* Порядок candidates стабилен.

## Ключевые файлы

* `internal/store/sqliteindex/store.go`
* `internal/service/indexsvc/diagnostics.go`

---

# MNEMONIC-209 — Возвращать исходные `matched_queries`

## Цель

Показывать агенту исходные query-варианты, а не внутренний FTS syntax.

## Проблема

`dedupQueries` возвращает значение после `sanitizeFTSQuery`, и оно попадает в:

```json
matched_queries
```

## Требования

Использовать отдельную структуру:

```go
type normalizedQuery struct {
    Original string
    FTS      string
}
```

Правила:

1. `Original` — trimmed исходный пользовательский query.
2. `FTS` — результат sanitization.
3. Дедупликация выполняется по `FTS`.
4. Если несколько original queries дают один FTS query, сохраняется первый.
5. В SQLite передаётся `FTS`.
6. В `matched_queries` возвращается `Original`.

Не показывать внутренние escaping и FTS operators, добавленные приложением.

## Условия приёмки

Input:

```json
{
  "queries": [
    "chargeback process",
    " chargeback process "
  ]
}
```

выполняет один FTS search и возвращает:

```json
{
  "matched_queries": ["chargeback process"]
}
```

`matched_queries` не содержит внутренний sanitized syntax.

## Ключевые файлы

* `internal/store/sqliteindex/store.go`
* `internal/service/searchsvc/service.go`

---

# MNEMONIC-210 — Унифицировать timestamp contract в MCP

## Цель

Использовать один формат для одноимённых timestamp fields.

## Проблема

Frontmatter, индекс и search filters используют Unix seconds, но `read_notes` возвращает RFC3339 strings.

## Требования

Изменить DTO:

```go
CreatedAt *int64 `json:"created_at,omitempty"`
UpdatedAt *int64 `json:"updated_at,omitempty"`
```

Заполнять:

```go
createdAt := resolved.Note.CreatedAt.Unix()
updatedAt := resolved.Note.UpdatedAt.Unix()
```

Не форматировать даты через RFC3339.

Одноимённые поля во всех внешних контрактах должны означать Unix timestamp seconds.

## Условия приёмки

`read_notes` возвращает:

```json
{
  "created_at": 1783024200,
  "updated_at": 1783025200
}
```

Значения имеют тип JSON number.

Tool description не упоминает RFC3339.

## Ключевые файлы

* `internal/adapter/stdio/tools.go`
* CLI note output, если содержит те же поля
* пользовательская документация

---

# MNEMONIC-211 — Завершить удаление legacy search API

## Цель

Удалить оставшийся старый single-query API из SQLite store.

## Контекст

`searchsvc.Search` удалён, но в `sqliteindex.Store` всё ещё существует старый метод:

```go
Search(db, query, limit, tag)
```

## Требования

Найти все вызовы `sqliteindex.Store.Search`.

Если вызовов нет:

* удалить метод;
* удалить связанные private helpers, используемые только им;
* удалить старый payload и комментарии.

Если вызовы остаются:

* перевести их на `SearchAdvanced`;
* затем удалить старый метод.

Не оставлять wrappers для старого API.

## Условия приёмки

* В `sqliteindex.Store` отсутствует single-query `Search`.
* Все search call sites используют `SearchAdvanced` или специализированный актуальный метод.
* В коде нет терминов `legacy search`.
* Нет adapters старого search contract.

## Ключевые файлы

* `internal/store/sqliteindex/store.go`
* все call sites, найденные поиском

---

# MNEMONIC-212 — Обновить документацию под текущие контракты

## Цель

Синхронизировать документацию с фактической реализацией.

## Требования

Обновить:

* `README.md`;
* `PROMPTS.md`;
* `ARCHITECTURE.md`;
* `AGENTS.md`;
* CLI reference;
* markdown format documentation.

Документация должна описывать:

* `search_notes.queries`;
* RRF и graph-aware reranking;
* default limit 10;
* `summary`, `tags`, `matched_queries`;
* `read_notes.fields`;
* `read_notes.missing` и `read_notes.issues`;
* Unix timestamps;
* strict list format для tags и aliases;
* `links_style`;
* `missing_timestamp`;
* structured diagnostic fields;
* актуальные CLI commands;
* request limits;
* полный rebuild индекса без upgrade/migration path.

Удалить упоминания:

* `read_note`;
* permalink;
* PageRank;
* schema upgrade;
* migrations;
* compatibility;
* старый single-query API;
* schema version 2/3.

## Условия приёмки

* Все команды из документации существуют.
* Все параметры MCP совпадают с DTO.
* Нет утверждения об автоматическом upgrade существующего индекса.
* Нет schema version history.
* Manifest и note examples соответствуют parser.
* Timestamp examples используют Unix seconds.

## Ключевые файлы

* `README.md`
* `PROMPTS.md`
* `ARCHITECTURE.md`
* `AGENTS.md`
* `README.cli.md`
* `internal/format/markdown/README.md`

---

# Рекомендуемый порядок

1. `MNEMONIC-201` — убрать schema upgrade/versioning
2. `MNEMONIC-203` — typed errors для `ShowMany`
3. `MNEMONIC-206` — strict tags/aliases
4. `MNEMONIC-207` — общая validation
5. `MNEMONIC-202` — logger DI
6. `MNEMONIC-204` — batch-read duration
7. `MNEMONIC-205` — clock в diagnostics
8. `MNEMONIC-209` — original matched queries
9. `MNEMONIC-210` — Unix timestamps в MCP
10. `MNEMONIC-208` — batch suggestions
11. `MNEMONIC-211` — удалить legacy store search
12. `MNEMONIC-212` — документация
