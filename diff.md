## Общая оценка

Направление правильное, и сделано заметно больше, чем косметическая правка API. Есть рабочий каркас для:

* multi-query поиска;
* временных фильтров;
* `summary` и `aliases`;
* batch-read;
* regular Markdown links;
* diagnostics;
* конфигурируемого логирования.

Но **повторный тест на реальных данных я бы пока не запускал**. В ранжировании есть критическая ошибка, несколько ключевых контрактов фактически не реализованы, а `links_style` пока является неработающей настройкой.

Условно:

* архитектурное направление: **7/10**;
* соответствие согласованным требованиям: **5/10**;
* готовность к реальному evaluation: **4/10**.

## Что сделано хорошо

### Индекс стал содержательнее

В SQLite появились отдельные `summary`, `aliases`, integer timestamps, индексы по времени и отдельные FTS-поля для title/summary/tags/aliases/body. Это правильная основа.

### Временной поиск реализован адекватно

Есть абсолютные границы и относительные интервалы `m/h/d`; допускается поиск без текстового запроса, если передан tag или временной фильтр. Это соответствует задаче автоматической чистки.

### Batch API и diagnostics появились

`read_note` заменён на `read_notes`, добавлен `diagnose_notes`, расширен `doctor`, появилась пагинация. Само направление верное.

### Logging имеет нормальную базу

Есть:

```text
CLI > env > config > defaults
```

поддержаны text/json и debug/info/warn/error, вывод идёт в stderr. Это хорошая база для stdio MCP.

---

# Критические проблемы

## 1. Multi-query boost работает в обратную сторону

Это главный баг.

SQLite FTS5 BM25 ранжируется так: **меньшее значение лучше**, на практике scores обычно отрицательные. Код тоже сортирует ascending.

Но multi-query boost делает:

```go
r.Score = r.Score * (1.0 - 0.1*float64(r.MatchCount-1))
if r.Score < 0 {
    r.Score = 0
}
```

Допустим:

```text
документ A: score = -10, найден двумя queries
документ B: score = -5, найден одним query
```

После boost:

```text
A → -9 → 0
B → -5
```

При сортировке ascending документ B окажется выше. То есть документ, найденный несколькими запросами, **штрафуется и почти наверняка уезжает вниз**.

Graph boost имеет ту же проблему:

```go
score = score / (1 + boost)
```

Для отрицательного score это делает его ближе к нулю, то есть хуже.  

### Как исправить

Лучше вообще не выполнять арифметику над raw BM25. Использовать RRF:

```text
score = Σ 1 / (k + rank)
score += graph_bonus
```

И сортировать descending.

Это одновременно решит проблему несопоставимости BM25 между разными запросами.

---

## 2. Tag filter заявлен как AND, но реализован как OR

Документация и MCP description говорят:

```text
tags: every listed tag, AND logic
```

Но SQL строится так:

```sql
nt.tag LIKE ? OR nt.tag LIKE ? OR nt.tag LIKE ?
```

Поэтому запрос:

```json
{"tags": ["spei", "deposit"]}
```

вернёт заметки с `spei` **или** `deposit`, а не с обоими тегами.

Та же ошибка есть в search-only-by-filters и FTS search.

Нужен либо отдельный `EXISTS` на каждый tag, либо:

```sql
GROUP BY note_id
HAVING COUNT(DISTINCT normalized_tag) = ?
```

---

## 3. Search-result всё ещё не решает исходную проблему

Главная цель была сделать search-result самодостаточным, чтобы агент мог понять, какие заметки читать. Но сейчас результат содержит:

```json
{
  "note_id": "...",
  "slug": "...",
  "title": "...",
  "snippet": "..."
}
```

В нём по-прежнему нет:

* `summary`;
* `tags`;
* `matched_queries`.

Хотя `summary` и aliases уже индексируются, наружу они не передаются. `AdvancedSearchResult` всё ещё повторяет старый минимальный payload.

Кроме того, snippet всегда строится из FTS-колонки body:

```sql
snippet(notes_fts, 5, ...)
```

Если заметка найдена по title, summary или alias, snippet может быть пустым или не объяснять совпадение.

### Что нужно вернуть

```json
{
  "note_id": "...",
  "slug": "...",
  "title": "...",
  "summary": "...",
  "tags": ["..."],
  "snippet": "...",
  "matched_queries": ["...", "..."]
}
```

Fallback для snippet:

```text
matched body excerpt
→ summary
→ title
```

Default limit остался `20`, хотя задача была уменьшить payload. Для MCP разумнее `8` или `10`.

---

## 4. `links_style` сейчас ничего не делает

Настройка добавлена в manifest:

```toml
[format]
links_style = "wiki"
```

и существует `FormatLink()`, но:

* `KnowledgeBase` не содержит `LinksStyle`;
* manifest value не передаётся runtime-сервисам;
* `FormatLink()` нигде не вызывается;
* ни create, ни edit, ни MCP instructions не учитывают настройку.

Фактически это мёртвая конфигурация. В коде `FormatLink` встречается только в месте определения.

Нужно протянуть:

```text
manifest.Format.LinksStyle
→ kb.KnowledgeBase.LinksStyle
→ notes service / MCP instructions / link renderer
```

И решить, где именно приложение генерирует ссылки. Пока приложение лишь парсит пользовательский body, `links_style` не имеет наблюдаемого эффекта.

---

## 5. Title-based wiki links по-прежнему резолвятся

Мы договорились, что wiki target — это slug/machine identifier. Однако resolver всё ещё принимает:

```go
doc.Title == target
```

и затем выполняет normalized-title fallback.

То есть:

```md
[[Chargebacks - Process & Provider Handling]]
```

по-прежнему может успешно резолвиться по title. Следовательно:

* старый неявный формат фактически продолжает поддерживаться;
* diagnostics не покажет его как проблему;
* переход на slug-based links не обеспечен.

Нужно отделить:

### Note selector

Может принимать:

* id;
* slug;
* path;
* title.

### Link resolver

Должен принимать только:

* slug;
* возможно note_id;
* regular-link path;
* явно оговорённые aliases.

Без title/fuzzy auto-resolution.

---

## 6. `relation_type` теряется при индексировании

`RelationRef` содержит:

```go
RelationType string
Source       RelationSource
LinkStyle    string
```

Но при построении `linkRow` сохраняются:

```go
RawTarget
Label
LinkStyle
SourceKind
Line
```

`RelationType` туда не копируется.

Позже `populateRelatedNotes` берёт `links.source_kind` и возвращает его как:

```json
"relation_type": "relations_section"
```

или:

```json
"relation_type": "wikilink"
```

То есть агент ожидает:

```text
depends_on
relates_to
```

а получает место, откуда ссылка была извлечена.

Нужно хранить отдельно:

```sql
source_kind
relation_type
link_style
```

И добавить `direction`:

```json
{
  "relation_type": "depends_on",
  "direction": "outgoing"
}
```

---

# Проблемы `read_notes`

## 7. Параметр `fields` полностью игнорируется

Контракт объявляет:

```json
{
  "fields": ["title", "summary", "body"]
}
```

Но handler всегда возвращает:

* note_id;
* slug;
* title;
* path;
* весь frontmatter;
* body;
* content_hash;
* updated_at.

То есть исходная задача сокращения токенов не выполнена.

Более того, `summary`, `tags` и `aliases` не представлены отдельными управляемыми полями.

### Правильнее

Вынести это из stdio adapter в:

```go
notesvc.ReadMany(input)
```

А adapter только преобразует DTO.

Default fields я бы сделал:

```text
note_id
slug
title
summary
tags
body
```

И только по явному запросу:

```text
path
frontmatter
content_hash
aliases
timestamps
```

## 8. `max_body_chars` режет строку по байтам

Сейчас:

```go
body = body[:input.MaxBodyChars]
```

Это может разрезать UTF-8 символ посередине и вернуть невалидный текст, особенно на русской базе.

Нужно либо:

```go
[]rune(body)
```

либо безопасное ограничение по UTF-8 boundary.

## 9. Batch-read выполняется последовательно внутри adapter

Один MCP call экономится, но каждую заметку `deps.Notes.Show()` читает отдельно и последовательно.

Это пока терпимо, однако бизнес-сценарий batch-read должен находиться в service layer. Там можно:

* дедуплицировать identifiers;
* читать параллельно с ограничением;
* сохранять порядок;
* централизованно применять fields/truncation;
* логировать duration и found/missing counts.

---

# Diagnostics

## 10. Cursor может вызвать panic

В `Diagnose`:

```go
cursor := input.Cursor
end := cursor + limit
if end > len(issues) {
    end = len(issues)
}

issues[cursor:end]
```

Если `cursor > len(issues)`, получится slice вроде:

```go
issues[100:10]
```

и процесс упадёт.

Нужно:

```go
if cursor >= len(issues) {
    return empty page
}
```

## 11. Missing timestamps не диагностируются

`missing_required_field` проверяет только:

* `mnemonic_note_id`;
* `title`;
* `slug`.

Но `created_at` и `updated_at` не проверяются. При индексировании отсутствующий timestamp бесшумно заменяется на mtime файла.

Для твоего cron-cleanup это опасно: «создана заметка» превращается в «последний mtime файла», что семантически другое.

Я бы сделал timestamps обязательными и:

* missing → `missing_required_field`;
* wrong type / `<= 0` → `invalid_timestamp`;
* не индексировать такую заметку как валидную либо явно помечать её degraded.

## 12. `invalid_timestamp` часто станет `invalid_frontmatter`

Если timestamp записан строкой, `ParseNote()` завершится ошибкой, а diagnostics классифицирует весь файл как `invalid_frontmatter`. До `invalid_timestamp` выполнение не дойдёт.

Если отдельная diagnostic category нужна, форматный parser должен возвращать структурированную ошибку с полем/kind, а не только текст.

## 13. Suggestions построены слишком хрупко

MCP adapter:

1. извлекает target парсингом строки `Detail`;
2. ожидает конкретный английский текст `target "..."`;
3. вызывает legacy `Search`;
4. делает отдельный поиск на каждую issue.

Это:

* business logic в adapter;
* зависимость от human-readable error string;
* N+1 searches;
* использование старого search API.

Лучше, чтобы issue содержала:

```go
Target string
Line   int
Column int
```

А suggestions строились внутри diagnostics service одним batch-запросом.

---

# Logging

Основа есть, но заявленные условия выполнены частично.

### Уже хорошо

* CLI/env/config precedence;
* stderr;
* text/json;
* level validation.

### Не хватает

* `read_notes` не логируется;
* notes create/edit/delete не логируются;
* search log не содержит duration;
* diagnostics suggestions не логируются;
* logger не внедряется в notes service;
* web MCP SDK server не получает CLI logger;
* runtime-сервисы, созданные через `Maint.RuntimeFactory` для `--all`, не получают logger.

То есть текущий DI:

```text
CLI adapter вручную мутирует Search.Logger и Index.Logger
```

работает только для части execution paths.

Лучше передавать logger в `app.RuntimeInput` и конструкторы сервисов.

---

# Prompt instructions сейчас слишком агрессивны

Сейчас агенту говорится:

```text
ALWAYS provide multiple distinct query variants
ALWAYS use read_notes
Run diagnose_notes periodically
```

Это может увеличить именно те расходы, которые ты пытаешься сократить.

Например, для вопроса «что такое CLABE?» один точный query может быть достаточен. А `diagnose_notes` вообще не относится к обычному QA.

Я бы заменил на:

```text
- Search the knowledge base before answering questions within its scope.
- Use 2–4 query variants when the first formulation may be ambiguous or incomplete.
- Batch-read all selected notes in one read_notes call.
- Use diagnose_notes only for repository maintenance, cleanup, or repair tasks.
```

---

# Остатки того, что ты просил не делать

В коде всё ещё присутствуют:

* `PRAGMA user_version = 2`;
* schema version 2;
* `legacy search payload`;
* старые `Search` / `SearchInput`;
* `permalink` fallback;
* “Backward-compatible aliases … during migration”;
* compatibility alias `App = Bootstrap`.

Это не обязательно ломает поведение, но прямо противоречит последнему решению «никаких v2, compatibility и migration». Код стал одновременно новым и частично старым. Лучше удалить эти ветки сейчас, пока API не опубликован.

---

# Документация заметно разошлась с кодом

В README указано:

```bash
mnemonic notes diagnose
mnemonic index rebuild
mnemonic doctor
```

Но command tree содержит:

```text
mnemonic project doctor
mnemonic project reindex
```

и не содержит `notes diagnose`.  

Также:

* manifest example не содержит `[format] links_style`;
* README заявляет PageRank, хотя реализован локальный boost по числу связей;
* `PROMPTS.md` всё ещё описывает `read_note` и старый `search_notes(query, tag)`;
* документация заявляет tag AND, код реализует OR.

Перед следующим evaluation лучше генерировать MCP reference из Go schemas или хотя бы добавить consistency tests.

# Рекомендуемый порядок исправлений

## P0 — до любого нового теста

1. Заменить BM25 arithmetic на RRF.
2. Исправить tag filtering на AND.
3. Добавить в search output `summary`, `tags`, `matched_queries`.
4. Реализовать `read_notes.fields`.
5. Исправить diagnostics cursor panic.
6. Перестать выдавать `source_kind` как `relation_type`.

## P1 — чтобы проверить исходную гипотезу о токенах

1. Default search limit → 8–10.
2. Summary fallback для snippet.
3. Убрать path из related notes либо отдавать только в debug.
4. Default `read_notes` без полного frontmatter и hash.
5. Rune-safe body truncation.
6. Ограничить prompt-инструкции без безусловных multi-query/diagnostics.

## P2 — закончить модель ссылок и инфраструктуру

1. Протянуть `links_style` в runtime.
2. Удалить title-based link resolution.
3. Хранить `relation_type`, `source_kind`, `direction` отдельно.
4. Перенести batch-read и suggestions из adapter в services.
5. Завершить logger DI.
6. Удалить legacy/v2/compatibility остатки.
7. Синхронизировать документацию.

## Итог

Это не плохая реализация. Основные компоненты уже появились, и структура проекта не развалилась. Но сейчас есть неприятная комбинация:

* новая функциональность присутствует;
* центральные контракты реализованы не полностью;
* ranking boosts математически работают наоборот;
* часть настроек декоративна.

После исправления P0 можно снова прогонять архив реальных вопросов и измерять:

```text
recall@5
rank expected note
MCP call count
search payload bytes
total response tokens
answer completeness
```

Сейчас результаты такого evaluation будут искажены ошибкой ранжирования, поэтому на их основе ещё рано делать выводы о качестве подхода.
