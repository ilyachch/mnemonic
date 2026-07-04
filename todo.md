## Итог

Стало существенно лучше. Большая часть критики из прошлого ревью исправлена корректно:

* multi-query переведён на RRF;
* tag-фильтры стали AND;
* search-result получил `summary`, `tags`, `matched_queries`;
* `read_notes.fields` реализован;
* UTF-8 truncation исправлен;
* default limit снижен до 10;
* `relation_type`, `source_kind`, `direction` разделены;
* title-based link resolution удалён;
* diagnostics cursor больше не падает;
* MCP instructions стали менее агрессивными.

По одному только diff это уже примерно **7.5/10**. Я не запускал компиляцию и тесты: загружен diff, а не актуальное дерево репозитория.

# Что исправлено хорошо

## RRF

Переход от арифметики над отрицательным BM25 к:

```go
entry.result.Score += 1.0 / (rrfK + float64(rank+1))
```

правильный. Результаты разных запросов теперь можно объединять без предположения, что их BM25 scores сопоставимы.

## Search payload

Теперь агент получает достаточно информации для выбора документов:

```json
{
  "title": "...",
  "summary": "...",
  "tags": ["..."],
  "snippet": "...",
  "matched_queries": ["..."]
}
```

Это прямо решает значительную часть исходной проблемы с лишними `read_note` вызовами.

## `read_notes`

Хорошо сделаны:

* компактные default fields;
* optional expensive fields;
* rune-safe truncation;
* pointers для условно возвращаемых scalar-полей.

## Links

Удаление:

```go
doc.Title == target
normalizeTitleSlug(doc.Title)
```

из link resolver — нужное изменение. Note selector всё ещё может работать по title, но wiki-link больше не должен автоматически разрешаться по нему.

## Related metadata

Теперь:

```text
relation_type = depends_on / relates_to
source_kind   = wikilink / relations_section
direction     = incoming / outgoing
```

не смешиваются в одном поле. Это правильная модель.

---

# Что ещё нужно исправить

## 1. Graph boost теперь правильного знака, но огромного масштаба

RRF score имеет примерно такой масштаб:

```text
rank 1, один query:  1 / 61 ≈ 0.0164
rank 1, четыре query:         ≈ 0.0656
```

Graph boost:

```go
results[i].Score += 0.1 * float64(conn)
```

Одна связь даёт `+0.1`, то есть перевешивает даже четыре идеальных текстовых совпадения.

В результате слабый документ с одной связью может подняться выше лучшего FTS-result.

### Лучше

Например:

```go
const graphBoostPerConnection = 0.002
const maxGraphBoost = 0.01
```

или мультипликативно:

```go
connections := min(conn, 3)
score *= 1.0 + 0.05*float64(connections)
```

Второй вариант безопаснее: graph влияет на порядок близких результатов, но не уничтожает текстовую релевантность.

**Это основной оставшийся blocker перед evaluation.**

---

## 2. Search tags возвращаются во внутреннем формате

В индекс tags записываются как:

```text
frontmatter:payment
inline:payment
observation:payment
```

Новый `populateSearchTags()` возвращает `tag` напрямую:

```sql
SELECT note_id, tag FROM note_tags
```

Поэтому MCP, вероятно, получит:

```json
{
  "tags": [
    "frontmatter:payment",
    "inline:spei"
  ]
}
```

а не:

```json
{
  "tags": [
    "payment",
    "spei"
  ]
}
```

Кроме того, одна и та же tag может повториться из разных источников.

Используй ту же нормализацию, что уже есть в `ListTags`:

```sql
CASE
    WHEN instr(tag, ':') > 0
    THEN substr(tag, instr(tag, ':') + 1)
    ELSE tag
END
```

и дедупликацию:

```sql
SELECT DISTINCT note_id, normalized_tag
```

---

## 3. `links_style` всё ещё не влияет на поведение

Теперь значение протянуто:

```text
manifest
→ KnowledgeBase.LinksStyle
```

Но дальше оно нигде не используется.

В частности:

* MCP instructions всё ещё говорят только `[[Wiki-Links]]`;
* `FormatLink()` не вызывается;
* create/edit tools не знают, какой формат предпочитать;
* regular mode не отражается в prompt.

То есть настройка уже не полностью мёртвая, но всё ещё **не имеет наблюдаемого эффекта**.

Минимально нужно динамически формировать instructions:

```text
wiki:
Link related notes using [[target-slug|Display Label]].

regular:
Link related notes using [Display Label](target-slug.md).
```

Причём стандартные links можно продолжать парсить в обоих режимах.

### Дополнительно

`resolveLinksStyle()` не должен молча возвращать `wiki` при ошибке чтения manifest:

```go
m, err := ...
if err != nil {
    return "wiki"
}
```

Это скрывает повреждённую конфигурацию. Лучше распарсить manifest один раз и передавать значение из уже валидированной структуры.

---

## 4. Related notes всё ещё могут раздуть payload

`populateRelatedNotes()` возвращает все incoming и outgoing links для каждого результата.

Проблемы:

* нет лимита, например 3–5;
* нет дедупликации;
* один target может появиться несколько раз через разные ссылки;
* при 10 search hits payload снова может стать большим.

Я бы возвращал максимум 3 related notes на hit по умолчанию:

```text
1. explicit relations_section
2. outgoing regular/wiki links
3. backlinks
```

Либо добавить:

```json
{
  "include_related": true,
  "related_limit": 3
}
```

Но отдельный параметр, вероятно, избыточен. Фиксированного небольшого лимита достаточно.

---

# Меньшие замечания

## `matched_queries` недетерминирован

Список собирается из Go map:

```go
for q := range e.matchedQueries {
    queries = append(queries, q)
}
```

Порядок будет случайным. Нужно:

```go
sort.Strings(queries)
```

Либо лучше сохранять порядок исходного `opts.Queries`.

Также стоит дедуплицировать входные queries. Сейчас:

```json
{
  "queries": ["chargeback", "chargeback"]
}
```

удвоит RRF contribution.

---

## RRF candidate pool слишком узкий

Каждый query получает только:

```go
runFTSSearch(..., opts.Limit, ...)
```

При итоговом limit 10 RRF объединяет только top-10 каждого запроса. Документ, стоящий на 11-м месте по нескольким queries, вообще не попадёт в fusion.

Лучше:

```go
candidateLimit := max(opts.Limit*3, 30)
```

а уже после fusion обрезать до `opts.Limit`.

---

## `read_notes.fields` имеет не совсем честный контракт

Даже при:

```json
{
  "fields": ["body"]
}
```

ответ всегда содержит:

* `note_id`;
* `slug`;
* `title`.

Я считаю это разумным: identity должна присутствовать всегда. Но описание сейчас говорит, что `fields` выбирает поля вообще.

Лучше зафиксировать:

```text
note_id, slug and title are always returned.
fields controls optional fields.
```

Также неизвестные поля сейчас молча игнорируются:

```json
{
  "fields": ["summery"]
}
```

Лучше вернуть ошибку со списком допустимых значений.

---

## Empty slices с `omitempty`

Для:

```go
Tags []string `json:"tags,omitempty"`
```

даже явно созданный `[]string{}` будет удалён из JSON.

Если нужно различать:

* поле не запрошено;
* поле запрошено, но список пуст;

используй:

```go
Tags *[]string `json:"tags,omitempty"`
```

То же относится к `Aliases`.

Это не blocker, но pointers уже используются для строк, поэтому модель можно сделать последовательной.

---

## Debug score может исчезнуть

```go
Score float64 `json:"score,omitempty"`
```

Для filter-only search score равен `0`, поэтому даже при `debug=true` поле будет скрыто.

Надёжнее:

```go
Score *float64 `json:"score,omitempty"`
```

или убрать `omitempty`, если debug DTO отделён.

---

## Snippet хранится от первого совпавшего query

При RRF `SearchResult` инициализируется первым query, где встретилась заметка. Последующие более качественные hits увеличивают RRF score, но snippet не обновляют.

Можно сохранять snippet от query, где документ имел минимальный rank. Не критично, поскольку теперь есть summary fallback.

---

# Что осталось без изменений из прошлого ревью

Этот diff не закрывает:

* missing `created_at` / `updated_at` diagnostics;
* отдельную классификацию `invalid_timestamp`;
* N+1 suggestions в `diagnose_notes`;
* parsing target из human-readable `Detail`;
* перенос batch-read из stdio adapter в service;
* полноценный logging для read/create/edit/delete;
* legacy/compatibility остатки;
* рассинхронизацию README/PROMPTS с командами и контрактами.

Это можно делать после retrieval evaluation, кроме diagnostics timestamps — они важны для запланированной cron-cleanup логики.

# Приоритет

Перед повторным тестом я бы исправил только четыре пункта:

1. уменьшить и ограничить graph boost;
2. нормализовать tags в search output;
3. ограничить и дедуплицировать related notes;
4. сделать `matched_queries` детерминированным.

После этого уже имеет смысл прогнать те же реальные сценарии и сравнить:

```text
expected note rank
количество search/read calls
размер search payload
общее число токенов
качество итогового ответа
```

`links_style` можно завершить параллельно, но на качество текущего retrieval-теста он почти не влияет.
