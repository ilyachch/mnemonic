# Epic: завершение `fixes-and-improvements`

## Общие ограничения

Для всех задач действуют следующие правила:

* Не добавлять миграции и обратную совместимость.
* Не поддерживать старые MCP-контракты.
* Не добавлять новые версии форматов.
* Не менять unrelated functionality.
* Ошибки должны сохранять существующую классификацию через `apperr`.
* После выполнения каждой задачи:

  * запустить `gofmt`;
  * запустить тесты изменённых пакетов;
  * запустить `go test ./...`.

---

# MNEMONIC-101 — Сделать порядок related notes детерминированным ~~[DONE]~~

## Цель

Гарантировать, что `search_notes` возвращает одинаковые related notes в одинаковом порядке при одинаковом состоянии индекса.

Сейчас related notes ограничиваются тремя элементами и выбираются по приоритету, но исходный SQL не задаёт стабильный порядок.

## Требования

Перед ограничением related notes необходимо упорядочить их по следующим правилам:

1. Ссылки из `relations_section`.
2. Остальные исходящие ссылки.
3. Входящие ссылки.
4. Внутри группы:

   * `relation_type`;
   * `slug`;
   * `note_id`.

Не полагаться на естественный порядок строк SQLite.

Дедупликация должна выполняться по `note_id`.

Если одна и та же заметка связана несколькими ссылками, сохранять связь с наивысшим приоритетом.

## Условия приёмки

* Два одинаковых запроса возвращают related notes в одном порядке.
* `relations_section` имеет приоритет над обычным исходящим wikilink.
* Исходящая ссылка имеет приоритет над backlink.
* Результат содержит не более трёх related notes.
* Одна заметка не появляется дважды.
* Есть unit tests минимум для:

  * разного порядка входных строк;
  * дубликатов;
  * разных `source_kind`;
  * разных `direction`.

## Ключевые файлы

* `internal/store/sqliteindex/store.go`
* `internal/store/sqliteindex/*_test.go`

---

# MNEMONIC-102 — Исправить классификацию ошибок в `ShowMany` ~~[DONE]~~

## Цель

Не считать любую ошибку чтения отсутствующей заметкой.

Сейчас `ShowMany` помещает selector в `Missing` при любой ошибке `Show()`.

## Требования

Разделить результаты batch-read на:

* успешно прочитанные заметки;
* отсутствующие selectors;
* ошибки чтения.

Только ошибка `not found` должна попадать в `Missing`.

Ошибки следующих типов не должны маскироваться как отсутствующая заметка:

* ambiguous selector;
* invalid frontmatter;
* corrupted Markdown;
* permission error;
* ошибка файловой системы;
* внутренняя ошибка.

Добавить структуру результата:

```go
type ReadManyIssue struct {
    Selector string `json:"selector"`
    Kind     string `json:"kind"`
    Message  string `json:"message"`
}
```

Допустимые `Kind` минимум:

```text
ambiguous
corrupted
io_error
internal
```

Поведение запроса:

* ошибки отдельных selectors не должны отменять успешно прочитанные заметки;
* системная ошибка самого batch-read может завершать весь вызов.

## Условия приёмки

* Несуществующая заметка попадает в `missing`.
* Ambiguous selector попадает в `issues`, а не в `missing`.
* Повреждённая заметка попадает в `issues`.
* Успешные заметки возвращаются даже при наличии `missing` и `issues`.
* Порядок успешно прочитанных заметок соответствует порядку selectors.
* MCP output содержит `notes`, `missing` и `issues`.

## Ключевые файлы

* `internal/service/notesvc/service.go`
* `internal/adapter/stdio/tools.go`
* `internal/service/notesvc/*_test.go`
* `internal/adapter/stdio/*_test.go`

---

# MNEMONIC-103 — Добавить typed errors для полей frontmatter ~~[DONE]~~

## Цель

Позволить diagnostics отличать неверный timestamp от общей ошибки YAML/frontmatter.

## Требования

Добавить тип ошибки для некорректного canonical field:

```go
type FrontmatterFieldError struct {
    Field string
    Kind  string
    Err   error
}
```

Минимальные `Kind`:

```text
invalid_type
invalid_timestamp
invalid_string
invalid_string_list
```

Парсер должен возвращать `FrontmatterFieldError` для ошибок в:

* `created_at`;
* `updated_at`;
* `tags`;
* `aliases`;
* строковых canonical fields.

Общий YAML syntax error должен оставаться `invalid_frontmatter`.

Diagnostics должен использовать `errors.As`, а не разбор текста ошибки.

## Условия приёмки

Для заметки:

```yaml
created_at: yesterday
```

возвращается diagnostic:

```json
{
  "kind": "invalid_timestamp",
  "field": "created_at"
}
```

Для синтаксически неверного YAML возвращается:

```json
{
  "kind": "invalid_frontmatter"
}
```

Для:

```yaml
tags: 123
```

ошибка классифицируется как ошибка поля `tags`, а не как timestamp или generic YAML error.

Добавлены unit tests для каждого поддержанного типа ошибки.

## Ключевые файлы

* `internal/format/markdown/note.go`
* новый файл `internal/format/markdown/errors.go`
* `internal/service/indexsvc/diagnostics.go`
* `internal/format/markdown/*_test.go`
* `internal/service/indexsvc/*_test.go`

---

# MNEMONIC-104 — Запретить подмену отсутствующих timestamps временем файла ~~[DONE]~~

## Цель

Сохранить корректную семантику временного поиска.

`created_at` и `updated_at` должны означать значения из frontmatter, а не filesystem modification time.

## Требования

При индексировании заметки без `created_at` или `updated_at`:

* не подставлять mtime файла;
* сохранить отсутствующее значение как `NULL` или `0`;
* time-filtered search не должен считать такую заметку созданной или обновлённой недавно.

Diagnostics должен возвращать отдельные issues:

```text
missing_timestamp: created_at
missing_timestamp: updated_at
```

Обычный текстовый поиск может продолжать находить такую заметку, если принято решение индексировать её с неполной metadata.

## Условия приёмки

* Заметка без `created_at` не попадает в `created_since=24h`.
* Заметка без `updated_at` не попадает в `updated_since=24h`.
* Изменение filesystem mtime не влияет на `created_at`/`updated_at` в индексе.
* Diagnostics возвращает missing timestamp.
* Заметка остаётся доступна обычному FTS-поиску, если остальные обязательные поля валидны.
* Добавлены integration tests с реальным временным файлом.

## Ключевые файлы

* `internal/store/sqliteindex/scan.go`
* `internal/store/sqliteindex/rebuild.go`
* `internal/store/sqliteindex/schema.go`
* `internal/store/sqliteindex/store.go`
* `internal/service/indexsvc/diagnostics.go`

---

# MNEMONIC-105 — Расширить structured diagnostics ~~[DONE]~~

## Цель

Дать агенту достаточно структурированной информации для автоматического исправления заметок.

## Требования

Расширить `DiagnosticIssue` полями:

```go
Field      string
SourceLine int
SourceKind string
LinkStyle  string
Target     string
```

Правила заполнения:

* `Field` — имя проблемного frontmatter field;
* `SourceLine` — строка broken/ambiguous link;
* `SourceKind` — `wikilink`, `relations_section` и другие существующие значения;
* `LinkStyle` — `wiki` или `regular`;
* `Target` — исходная цель ссылки.

Human-readable `Detail` оставить, но не использовать для программного анализа.

Добавить `missing_timestamp` в:

* MCP tool description;
* CLI help;
* список допустимых diagnostic kinds;
* документацию.

## Условия приёмки

Broken link возвращает:

```json
{
  "kind": "unresolved_link",
  "target": "some-note",
  "source_line": 14,
  "source_kind": "relations_section",
  "link_style": "wiki"
}
```

Ошибка timestamp возвращает `field`.

Ни MCP adapter, ни CLI adapter не парсят `Detail`.

## Ключевые файлы

* `internal/service/indexsvc/diagnostics.go`
* `internal/store/sqliteindex/store.go`
* `internal/adapter/stdio/tools.go`
* `internal/adapter/cli/project_doctor.go`

---

# MNEMONIC-106 — Перенести suggestions для broken links в diagnostics service ~~[DONE]~~

## Цель

Удалить N+1 search calls и бизнес-логику из MCP adapter.

## Требования

Candidate suggestions должны строиться внутри diagnostics/service layer.

Нельзя выполнять отдельный FTS search для каждой issue.

Нужно:

1. Собрать уникальные unresolved/ambiguous targets.
2. Выполнить поиск candidates пакетно либо с переиспользованием одного открытого индекса.
3. Добавить максимум три candidates на target.
4. Вернуть candidates внутри `DiagnosticIssue`.

MCP adapter должен только передать:

```go
IncludeSuggestions bool
```

и сериализовать готовый результат.

Удалить:

* `findLinkCandidates` из stdio adapter;
* использование legacy `SearchInput` для diagnostics.

## Условия приёмки

* При 50 одинаковых broken links поиск candidates выполняется один раз для уникального target.
* MCP adapter не вызывает search service самостоятельно.
* `include_suggestions=false` не запускает поиск candidates.
* `include_suggestions=true` возвращает максимум три candidates.
* Ошибка candidate search не скрывает основные diagnostic issues.

## Ключевые файлы

* `internal/service/indexsvc/diagnostics.go`
* `internal/service/searchsvc/service.go`
* `internal/adapter/stdio/tools.go`
* при необходимости новый `internal/service/diagnosticsvc/`

---

# MNEMONIC-107 — Удалить legacy search API и compatibility aliases ~~[DONE]~~

## Цель

Оставить один актуальный API без compatibility-кода.

## Требования

Удалить неиспользуемые:

* `SearchInput`;
* старый `SearchResult`;
* `Service.Search()`;
* `MnemonicManifest` aliases;
* `MnemonicManifestLayout`;
* `MnemonicGenerator`;
* `NewMnemonicManifest`;
* `Permalink`;
* fallback `EffectiveSlug()` на `permalink`;
* комментарии с `legacy`, `compatibility`, `migration`.

Все call sites перевести на актуальные типы и методы.

Не добавлять transitional wrappers.

## Условия приёмки

* Поиск по репозиторию не находит старые API names.
* В canonical note model отсутствует `Permalink`.
* Parser не читает `permalink`.
* Renderer не пишет `permalink`.
* `go test ./...` проходит.
* Документация не упоминает старые имена.

## Ключевые файлы

* `internal/service/searchsvc/service.go`
* `internal/format/manifest/manifest.go`
* `internal/format/markdown/note.go`
* `internal/format/markdown/render.go`
* все найденные call sites

---

# MNEMONIC-108 — Завершить dependency injection логгера ~~[DONE]~~

## Цель

Передавать logger через constructors, а не устанавливать его вручную после создания сервисов.

## Требования

Добавить logger в runtime input:

```go
type RuntimeInput struct {
    Config *config.Config
    KB     kb.KnowledgeBase
    Logger *slog.Logger
}
```

Передавать logger в:

* `notesvc`;
* `searchsvc`;
* `indexsvc`;
* maintenance runtime factory;
* web MCP server;
* stdio MCP server.

Удалить прямую мутацию:

```go
runtime.Services.Notes.Logger = logger
runtime.Services.Search.Logger = logger
runtime.Services.Index.Logger = logger
```

Добавить агрегированный log для `ShowMany`:

```text
requested_count
found_count
missing_count
issue_count
duration
```

Не логировать:

* полный body;
* полный frontmatter;
* секреты;
* access tokens.

## Условия приёмки

* Обычный CLI runtime использует configured logger.
* `project doctor --all` использует тот же logger.
* Web MCP использует configured logger.
* Stdio MCP пишет logs только в stderr.
* `read_notes` создаёт одно агрегированное info/debug событие.
* Нет ручной мутации public Logger fields в adapters.

## Ключевые файлы

* `internal/app/app.go`
* `internal/app/runtime.go`
* `internal/adapter/cli/runtime.go`
* `internal/adapter/cli/web_serve.go`
* `internal/adapter/cli/stdio.go`
* constructors сервисов

---

# MNEMONIC-109 — Синхронизировать CLI search с MCP search contract ~~[DONE]~~

## Цель

CLI и MCP должны использовать одинаковую структуру search result.

## Требования

CLI `notes search` должен возвращать:

* `note_id`;
* `slug`;
* `title`;
* `summary`;
* `tags`;
* `snippet`;
* `matched_queries`;
* `related_notes`, если запрошены.

При debug дополнительно:

* `path`;
* `score`;
* `content_hash`;
* path для related notes.

Default limit CLI установить в `10`, как в MCP.

Related note должен содержать:

* `relation_type`;
* `source_kind`;
* `direction`.

Избегать отдельного дублирующего DTO mapping, если можно использовать общий adapter DTO или mapper.

## Условия приёмки

* CLI JSON output содержит те же основные поля, что MCP.
* Без `--debug` внутренние поля отсутствуют.
* С `--debug` score присутствует даже при значении `0`.
* Default limit равен 10.
* Human-readable output не становится чрезмерно подробным.
* Добавлены tests для JSON output.

## Ключевые файлы

* `internal/adapter/cli/notes_search.go`
* при необходимости общий mapper package

---

# MNEMONIC-110 — Показывать `links_style` в `project show` ~~[DONE]~~

## Цель

Дать пользователю возможность увидеть effective link style проекта.

## Требования

Добавить `links_style` в:

* JSON output `project show`;
* human-readable output;
* при необходимости project list/details DTO.

Значение должно быть effective:

```text
wiki
regular
```

Если поле отсутствует в manifest, показывать default `wiki`.

## Условия приёмки

Для manifest:

```toml
[format]
links_style = "regular"
```

`project show --json` возвращает:

```json
{
  "links_style": "regular"
}
```

При отсутствии `[format]` возвращается `wiki`.

## Ключевые файлы

* `internal/adapter/cli/project_show.go`
* `internal/service/catalogsvc/service.go`
* `internal/domain/kb/knowledge_base.go`

---

# MNEMONIC-111 — Экранировать wildcard-символы в tag filters ~~[DONE]~~

## Цель

Не трактовать пользовательские `%` и `_` как SQL wildcard.

## Требования

Перед использованием tag в `LIKE` экранировать:

* `\`;
* `%`;
* `_`.

SQL должен использовать:

```sql
LIKE ? ESCAPE '\'
```

Одинаковое поведение должно быть в:

* text search с tag filters;
* filter-only search;
* любых других tag query paths.

Дополнительно удалить пустые tags после `TrimSpace`.

Повторяющиеся tags дедуплицировать.

## Условия приёмки

Tag `foo_bar` ищет literal underscore.

Tag `100%` ищет literal percent.

Два одинаковых tag filters не создают два одинаковых `EXISTS`.

Пустой tag игнорируется или возвращает validation error — выбранное поведение зафиксировано тестом.

## Ключевые файлы

* `internal/store/sqliteindex/store.go`
* `internal/service/searchsvc/service.go`
* `internal/store/sqliteindex/*_test.go`

---

# MNEMONIC-112 — Добавить ограничения размеров MCP-запросов ~~[DONE]~~

## Цель

Защитить MCP server от случайно чрезмерных запросов и слишком больших ответов.

## Требования

Ввести константы:

```text
max search limit
max query count
max query length
max read_notes identifiers
max body chars
max diagnostics limit
max diagnostic kinds
```

Рекомендуемые стартовые значения:

```text
search limit: 100
query count: 8
query length: 500 chars
read identifiers: 50
max_body_chars: 100000
diagnostics limit: 200
```

Если значение превышено — возвращать validation error.

Не обрезать input молча.

## Условия приёмки

* `limit=10000` возвращает понятную ошибку.
* 1000 identifiers не запускают чтение.
* Слишком длинный query не передаётся в SQLite.
* Значения на границе допустимого диапазона работают.
* Ограничения описаны в MCP tool descriptions.

## Ключевые файлы

* `internal/adapter/stdio/tools.go`
* желательно общий validation package или service input validation
* `internal/adapter/stdio/*_test.go`

---

# MNEMONIC-113 — Обновить пользовательскую и developer-документацию ~~[DONE]~~

## Цель

Синхронизировать документацию с фактическими командами и контрактами.

## Требования

Обновить:

* `README.md`;
* `PROMPTS.md`;
* `ARCHITECTURE.md`;
* `AGENTS.md`;
* `internal/format/markdown/README.md`;
* generated CLI reference при наличии.

Документация должна отражать:

* `read_notes`, а не `read_note`;
* multi-query search;
* default limit 10;
* `summary`, `tags`, `matched_queries`;
* `links_style`;
* wiki и regular links;
* `missing_timestamp`;
* structured diagnostics;
* реальные CLI commands;
* реальные logging settings;
* graph-aware boost, но не PageRank, если PageRank не реализован.

Удалить упоминания:

* compatibility;
* migration;
* legacy API;
* permalink fallback;
* старые команды.

## Условия приёмки

* Все команды из README реально существуют.
* Все MCP tools и параметры совпадают с кодом.
* `[format] links_style` присутствует в manifest example.
* Список diagnostic kinds полный.
* Поиск по документации не находит `read_note`.
* Сгенерированный CLI reference обновлён.

## Ключевые файлы

* `README.md`
* `PROMPTS.md`
* `ARCHITECTURE.md`
* `AGENTS.md`
* `README.cli.md`
* `internal/format/markdown/README.md`

---

# MNEMONIC-114 — Добавить regression tests для retrieval ~~[DONE]~~

## Цель

Зафиксировать ожидаемое поведение поиска и предотвратить повторное появление ошибок ранжирования.

## Требования

Создать тестовый SQLite index минимум с 8–12 заметками и связями.

Покрыть:

* RRF;
* query deduplication;
* candidate pool больше result limit;
* matched queries;
* snippet от лучшего rank;
* graph boost;
* ограничение graph boost;
* tag AND;
* tag escaping;
* summary fallback;
* normalized tags;
* related notes ordering;
* related notes deduplication;
* filter-only search;
* missing timestamps в time filters.

Тесты должны проверять не только наличие документа, но и его позицию.

## Условия приёмки

Есть тесты вида:

```text
expected note is rank 1
expected note is within top 3
duplicate query does not change rank
graph-only result does not beat strong textual match
```

Тесты детерминированы и не зависят от текущего времени — используется фиксированный clock.

## Ключевые файлы

* `internal/store/sqliteindex/search_test.go`
* `internal/service/searchsvc/*_test.go`
* test fixtures package при необходимости

---

# MNEMONIC-115 — Добавить contract tests для MCP и diagnostics ~~[DONE]~~

## Цель

Зафиксировать внешний JSON contract.

## Требования

Покрыть:

### `read_notes`

* default fields;
* custom fields;
* unknown field;
* пустые requested tags/aliases;
* missing selectors;
* per-selector issues;
* rune-safe truncation.

### `search_notes`

* default limit;
* compact output;
* debug output;
* zero score в debug;
* related notes без path;
* related notes с path в debug.

### `diagnose_notes`

* missing timestamp;
* invalid timestamp;
* unresolved link;
* structured target/line/style/source;
* suggestions enabled/disabled;
* cursor за пределами результата.

## Условия приёмки

* JSON output сравнивается с ожидаемой структурой.
* Отсутствующие optional fields действительно отсутствуют.
* Запрошенные пустые arrays сериализуются как `[]`.
* Ошибки validation имеют стабильное сообщение или код.
* Tests не требуют запуска реального MCP transport.

## Ключевые файлы

* `internal/adapter/stdio/tools_test.go`
* `internal/service/indexsvc/diagnostics_test.go`

---

# Рекомендуемый порядок выполнения

## Сначала — корректность

1. `MNEMONIC-101` — deterministic related notes
2. `MNEMONIC-102` — batch-read error classification
3. `MNEMONIC-103` — typed frontmatter errors
4. `MNEMONIC-104` — timestamp indexing semantics
5. `MNEMONIC-105` — structured diagnostics

## Затем — архитектура

6. `MNEMONIC-106` — batch suggestions
7. `MNEMONIC-108` — logger DI
8. `MNEMONIC-107` — legacy cleanup

## Затем — внешние интерфейсы

9. `MNEMONIC-109` — CLI search contract
10. `MNEMONIC-110` — project show
11. `MNEMONIC-111` — tag escaping
12. `MNEMONIC-112` — request limits

## В завершение

13. `MNEMONIC-114` — retrieval regression tests
14. `MNEMONIC-115` — MCP/diagnostics contract tests
15. `MNEMONIC-113` — documentation
