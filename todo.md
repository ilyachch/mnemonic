# Оставшиеся задачи для ветки `fixes-and-improvements`

## Общие ограничения

При выполнении задач соблюдать следующие правила:

- не добавлять миграции SQLite;
- не добавлять поддержку старых форматов и контрактов;
- не определять версию существующего индекса;
- не запускать reindex автоматически;
- считать SQLite-индекс полностью пересоздаваемым производным артефактом;
- не классифицировать ошибки через анализ `err.Error()`;
- не менять существующую семантику без необходимости;
- обновить или добавить тестовые сценарии для каждого изменённого поведения.

---

# MNEMONIC-301 — Подключить structural validation SQLite-индекса

**Приоритет:** P0, блокирует merge.

## Цель

Проверять структуру существующего SQLite-индекса до выполнения обычных поисковых запросов.

## Проблема

Функция `ValidateSchema` уже реализована, но не вызывается при открытии существующего индекса.

Из-за этого несовместимый индекс может привести к низкоуровневой SQL-ошибке:

```text
no such column: l.relation_type
```

вместо понятного сообщения с требованием выполнить явный reindex.

Кроме того, таблица `note_aliases` используется текущим индексом, но отсутствует в списке обязательных таблиц.

## Что нужно сделать

### 1. Подключить проверку к `OpenReadonly`

После успешных `sql.Open` и `Ping` вызвать:

```go
ValidateSchema(db)
```

Если проверка завершилась ошибкой:

1. закрыть открытое соединение;
2. вернуть типизированную application error;
3. не выполнять никаких SQL-запросов поиска;
4. не изменять файл индекса;
5. не запускать rebuild.

Рекомендуемый тип ошибки:

```go
apperr.Corrupted(...)
```

Сообщение верхнего уровня должно быть стабильным:

```text
index is invalid; run `mnemonic project reindex`
```

Детали отсутствующей таблицы или колонки можно сохранить во вложенной ошибке.

### 2. Дополнить structural contract

Добавить в `requiredTables` таблицу:

```text
note_aliases
```

с колонками:

```text
note_id
alias
```

Обязательный набор должен включать таблицы, которые реально используются обычными read/search operations:

- `notes`;
- `note_tags`;
- `note_aliases`;
- `observations`;
- `links`;
- `notes_fts`.

`index_runs` не добавлять в обязательный read contract, если она используется только как служебная информация процесса rebuild.

### 3. Не добавлять версии схемы

Запрещено возвращать:

- `schema version mismatch`;
- номер найденной версии;
- номер ожидаемой версии.

Запрещено использовать:

```sql
PRAGMA user_version
```

или отдельное поле `schema_version`.

## Как проверить

Добавить или обновить тестовые сценарии:

1. Полностью валидный индекс успешно открывается.

2. Индекс без таблицы `note_aliases` отклоняется.

3. Индекс без колонки `links.relation_type` отклоняется.

4. Индекс без `notes_fts` отклоняется.

5. Ошибка распознаётся через:

   ```go
   errors.As(err, *apperr.Error)
   ```

   и имеет код `CodeCorrupted`.

6. Текст верхнего уровня содержит команду:

   ```text
   mnemonic project reindex
   ```

7. После ошибки:

   - индекс не пересоздан;
   - timestamp и содержимое файла не изменены;
   - новые таблицы или колонки не появились.

8. После явного вызова `mnemonic project reindex` операция поиска успешно выполняется.

## Ключевые файлы

- `internal/store/sqliteindex/schema_check.go`
- `internal/store/sqliteindex/store.go`
- `internal/store/sqliteindex/schema.go`
- `internal/adapter/cli/query.go`
- тесты `sqliteindex` и CLI error mapping

---

# MNEMONIC-302 — Удалить классификацию ошибок по тексту сообщений

**Приоритет:** P0, блокирует merge.

## Цель

Классифицировать ошибки только по типам, sentinel errors или `apperr.Code`.

## Проблема

В проекте остаются места, где тип ошибки определяется через проверки вида:

```go
strings.HasPrefix(err.Error(), ...)
strings.HasSuffix(err.Error(), ...)
strings.Contains(err.Error(), ...)
```

Примеры текущих мест:

- `wrapAddError`;
- `wrapImportError`;
- обработка результата `LookupNoteByIdentifier` в `searchsvc.Backlinks`;
- возможно другие аналогичные места.

Такой код зависит от текста сообщения и ломается при изменении формулировок.

## Что нужно сделать

### 1. Найти все подобные места

Проверить проект на конструкции:

```text
err.Error()
HasPrefix
HasSuffix
Contains
```

Нужно исправить только те места, где результат `err.Error()` используется для определения категории ошибки.

Логирование или вывод уже классифицированной ошибки не запрещены.

### 2. Типизировать ошибки в месте возникновения

В месте, где причина уже известна, возвращать подходящий тип:

- отсутствующий файл или проект — `apperr.NotFound`;
- неоднозначный selector или конфликтующий slug — `apperr.Ambiguous`;
- некорректный пользовательский ввод — `apperr.CLIUsage`;
- повреждённый формат — `apperr.Corrupted`;
- ошибка файловой системы — `apperr.IO`;
- внутренний сбой — исходная ошибка или `apperr.Internal`.

Например, `LookupNoteByIdentifier` при `sql.ErrNoRows` должен возвращать типизированный `NotFound`, а не строку `"note ... not found"`.

### 3. Упростить adapters

После типизации ошибок:

- удалить `wrapAddError`, если он больше ничего не делает;
- удалить `wrapImportError`, если он больше ничего не делает;
- не создавать новую ошибку только из `err.Error()` с `nil` в качестве cause;
- передавать уже типизированную ошибку наверх без повторной классификации.

### 4. Исправить `wrapSyncError`

Не превращать любую ошибку sync в `CLIUsage`.

Ошибка должна сохранять исходный тип.

Если конкретная ошибка действительно вызвана пользовательским вводом, она должна быть превращена в `CLIUsage` там, где причина известна.

## Как проверить

Добавить или обновить тестовые сценарии:

1. Отсутствующий путь в `project add` возвращает `CodeNotFound`.
2. Отсутствующий manifest возвращает `CodeNotFound`.
3. Дублирующийся slug возвращает `CodeAmbiguous`.
4. Неизвестная заметка в backlinks возвращает `CodeNotFound`.
5. Ошибка чтения файла сохраняет `CodeIO`.
6. Вложенная исходная ошибка доступна через `errors.Is` или `errors.As`.
7. Изменение текста сообщения ошибки не меняет её exit code.
8. Поиск по production-коду не находит условной классификации через `err.Error()`.

Пример команды для ручной проверки:

```bash
rg 'err\.Error\(\)' internal
```

Каждое найденное место проверить вручную: допустимы логирование и вывод, недопустима классификация по строке.

## Ключевые файлы

- `internal/adapter/cli/project_add.go`
- `internal/adapter/cli/project_import.go`
- `internal/adapter/cli/project_sync.go`
- `internal/service/catalogsvc/service.go`
- `internal/service/searchsvc/service.go`
- `internal/store/sqliteindex/store.go`
- `internal/apperr/errors.go`

---

# MNEMONIC-303 — Поддержать очистку `tags` и `aliases`

**Приоритет:** P0, блокирует merge.

## Цель

Позволить явно заменить `tags` или `aliases` пустым списком.

## Проблема

Текущий MCP-контракт использует:

```go
Tags    []string
Aliases []string
```

и определяет наличие параметра через:

```go
len(input.Tags) > 0
len(input.Aliases) > 0
```

Поэтому невозможно отличить:

- поле не передано;
- передан пустой массив для очистки.

Аналогичная проблема есть в CLI: факт передачи флага определяется через длину списка.

## Что нужно сделать

### 1. Изменить MCP input

Использовать presence-aware поля:

```go
Tags    *[]string `json:"tags,omitempty"`
Aliases *[]string `json:"aliases,omitempty"`
```

Семантика:

- `nil` — поле не изменять;
- `&[]string{}` — очистить поле;
- `&[]string{"one", "two"}` — заменить поле указанным списком.

### 2. Обновить validation edit mode

Metadata edit mode считается выбранным, если:

```go
input.Tags != nil || input.Aliases != nil
```

а не если длина списка больше нуля.

Должна быть разрешена одна операция, одновременно меняющая оба поля:

```json
{
  "identifier": "note",
  "tags": [],
  "aliases": ["old-name"]
}
```

### 3. Обновить преобразование в `notesvc.EditInput`

Передавать указатели без проверки длины:

```go
editInput.Tags = input.Tags
editInput.Aliases = input.Aliases
```

### 4. Добавить явную очистку в CLI

Рекомендуемый CLI-контракт:

```text
--set-tags TAG
--set-aliases ALIAS
--clear-tags
--clear-aliases
```

Правила:

- `--clear-tags` устанавливает `Tags = &[]string{}`;
- `--clear-aliases` устанавливает `Aliases = &[]string{}`;
- `--set-tags` нельзя комбинировать с `--clear-tags`;
- `--set-aliases` нельзя комбинировать с `--clear-aliases`;
- разрешено одновременно изменить tags и aliases;
- наличие `--set-tags` и `--set-aliases` определять через `cmd.Flags().Changed(...)`, а не через длину результата.

Не использовать пустую строку как специальный marker очистки.

### 5. Сохранить strict list contract

Поля `tags` и `aliases` должны по-прежнему записываться только как YAML arrays.

Не добавлять scalar-to-list conversion.

Generic операции:

```text
--set tags=...
--set aliases=...
merge_frontmatter.tags
merge_frontmatter.aliases
```

должны оставаться запрещёнными.

## Как проверить

### MCP

1. Поле отсутствует — существующие tags не изменяются.
2. `"tags": []` — все tags удаляются.
3. `"aliases": []` — все aliases удаляются.
4. Непустой массив полностью заменяет старое значение.
5. Tags и aliases можно изменить одним вызовом.
6. После повторного чтения frontmatter содержит YAML list.
7. Попытка использовать `merge_frontmatter` для `tags` или `aliases` возвращает `CodeCLIUsage`.

### CLI

Проверить сценарии:

```bash
mnemonic notes edit note --clear-tags
mnemonic notes edit note --clear-aliases
mnemonic notes edit note --set-tags one --set-tags two
mnemonic notes edit note --set-aliases first --set-aliases second
mnemonic notes edit note --clear-tags --set-aliases current-name
```

Должны завершаться ошибкой:

```bash
mnemonic notes edit note --clear-tags --set-tags one
mnemonic notes edit note --clear-aliases --set-aliases one
```

После очистки `read_notes` должен вернуть:

```json
{
  "tags": []
}
```

или:

```json
{
  "aliases": []
}
```

## Ключевые файлы

- `internal/adapter/stdio/tools.go`
- `internal/adapter/cli/notes_edit.go`
- `internal/service/notesvc/service.go`
- `internal/store/markdownstore/store.go`
- тесты CLI и MCP edit contracts

---

# MNEMONIC-304 — Удалить неиспользуемый `Bootstrap.Logger`

**Приоритет:** P1.

## Цель

Окончательно закрепить immutable logger wiring.

## Проблема

Command-specific logger уже передаётся явным аргументом в runtime, catalog и maintenance operations.

Однако в `app.Bootstrap` всё ещё осталось поле:

```go
Logger *slog.Logger
```

Оно больше не используется и создаёт возможность случайно вернуть mutable setter-style wiring.

## Что нужно сделать

1. Удалить поле `Logger` из `app.Bootstrap`.
2. Удалить ненужный импорт `log/slog` из `internal/app/app.go`, если после этого он не используется.
3. Не менять `RuntimeInput.Logger`.
4. Не менять параметр logger в `Bootstrap.Runtime`.
5. Не менять logger-параметр `maintsvc.RuntimeFactory`.
6. Убедиться, что нигде нет:

   - `boot.Logger = ...`;
   - чтения `boot.Logger`;
   - мутации logger в уже созданных service instances.

## Как проверить

1. Проект компилируется после удаления поля.

2. Поиск не возвращает обращений к полю:

   ```bash
   rg '\.Logger' internal/app internal/adapter/cli
   ```

   Допустимы поля logger у runtime services и adapters, но не у `Bootstrap`.

3. Два runtime, созданные с разными logger, получают соответствующие logger.

4. Создание второго runtime не меняет logger первого.

5. `go test -race ./...` не обнаруживает гонок вокруг logger wiring.

## Ключевые файлы

- `internal/app/app.go`
- `internal/app/runtime.go`
- `internal/service/maintsvc/service.go`
- `internal/adapter/cli/runtime.go`

---

# MNEMONIC-305 — Валидировать отрицательные limits для tags и backlinks

**Приоритет:** P1.

## Цель

Не принимать отрицательный `limit` как неявное отсутствие ограничения.

## Проблема

`list_tags` и `list_backlinks` сейчас не отклоняют отрицательные значения.

В результате `limit = -1` может молча означать «вернуть всё».

## Что нужно сделать

### Общая семантика

Для обеих операций:

```text
limit == 0  → не применять явное ограничение
limit > 0   → вернуть не больше limit элементов
limit < 0   → вернуть CodeCLIUsage
```

Существующую семантику `0` сохранить.

### `list_tags`

Перед выполнением запроса проверить `input.Limit`.

При отрицательном значении вернуть:

```go
apperr.CLIUsage("limit must be >= 0", nil)
```

Ограничение должно применяться до формирования ответа.

Предпочтительно перенести limit в service input, чтобы adapter не был единственным владельцем validation.

Например:

```go
type ListTagsInput struct {
    Limit int
}
```

### `list_backlinks`

Добавить validation в `searchsvc.Backlinks`, а не только в MCP adapter:

```go
if input.Limit < 0 {
    return nil, apperr.CLIUsage("limit must be >= 0", nil)
}
```

Store не должен самостоятельно превращать отрицательное значение в отсутствие `LIMIT`.

## Как проверить

Для обеих операций проверить:

1. `limit = -1` возвращает `CodeCLIUsage`.
2. `limit = 0` сохраняет текущее поведение без явного ограничения.
3. `limit = 1` возвращает не более одного элемента.
4. Service validation нельзя обойти прямым вызовом сервиса.
5. MCP и прямой вызов сервиса возвращают одинаковую категорию ошибки.

## Ключевые файлы

- `internal/adapter/stdio/tools.go`
- `internal/service/searchsvc/service.go`
- `internal/store/sqliteindex/store.go`

---

# MNEMONIC-306 — Не мутировать входной slice в candidate search

**Приоритет:** P2.

## Цель

Не менять входные аргументы store-метода.

## Проблема

`SearchCandidatesByTargets` выполняет:

```go
sort.Strings(targets)
```

Это сортирует исходный slice, переданный caller-ом.

Сейчас caller создаёт отдельный список, поэтому видимой ошибки нет, но контракт функции неожиданно мутирует входные данные.

## Что нужно сделать

Перед сортировкой создать копию:

```go
sortedTargets := append([]string(nil), targets...)
sort.Strings(sortedTargets)
```

Далее использовать `sortedTargets`.

Не менять остальные свойства запроса:

- один batch query;
- не более `limitPerTarget` результатов;
- сортировка по `score`, `slug`, `note_id`;
- детерминированный порядок targets.

## Как проверить

1. Передать slice:

   ```go
   targets := []string{"z", "a", "m"}
   ```

2. Вызвать `SearchCandidatesByTargets`.

3. После вызова `targets` должен остаться:

   ```go
   []string{"z", "a", "m"}
   ```

4. SQL-результат должен оставаться детерминированным.

5. Одинаковый набор targets в разном входном порядке должен возвращать эквивалентный результат.

## Ключевой файл

- `internal/store/sqliteindex/store.go`

---

# MNEMONIC-307 — Синхронизировать документацию с финальными контрактами

**Приоритет:** P2, выполнить до merge.

## Цель

Зафиксировать итоговое поведение индекса, edit API и application errors.

## Что нужно сделать

### 1. Документировать disposable index

Добавить в пользовательскую или архитектурную документацию:

- индекс является производным артефактом;

- несовместимый индекс не мигрируется;

- приложение не выполняет автоматический rebuild;

- пользователь должен явно выполнить:

  ```bash
  mnemonic project reindex
  ```

- стандартное сообщение:

  ```text
  index is invalid; run `mnemonic project reindex`
  ```

### 2. Обновить документацию `edit_note`

Указать, что edit поддерживает:

- `append`;
- `replace_body`;
- `merge_frontmatter`;
- typed `tags`;
- typed `aliases`.

Добавить семантику:

```text
поле отсутствует → не изменять
пустой массив → очистить
непустой массив → заменить
```

Указать, что `tags` и `aliases` запрещено передавать через `merge_frontmatter`.

### 3. Обновить CLI reference

Добавить новые flags:

```text
--set-tags
--set-aliases
--clear-tags
--clear-aliases
```

### 4. Обновить developer guide

В `AGENTS.md`:

- добавить `apperr.IO()` в список стандартных error helpers;
- явно указать, что запрещено классифицировать ошибки по `err.Error()`;
- указать, что structural validation не является системой версий или миграций.

### 5. Задокументировать limit semantics

Для `list_tags` и `list_backlinks` указать:

```text
0 — без явного ограничения
положительное значение — maximum result count
отрицательное значение — validation error
```

## Как проверить

Выполнить поиск по актуальной документации:

```bash
rg 'schema version|automatic rebuild|auto.?rebuild|permalink|read_note\b|PageRank'
```

В актуальных документах продукта не должно остаться описаний:

- версий SQLite-схемы;
- автоматической миграции;
- автоматического rebuild;
- `permalink`;
- старого инструмента `read_note`;
- PageRank.

Исторические diff или release notes можно не изменять.

Также проверить:

```bash
rg 'edit_note|tags|aliases|project reindex|apperr.IO' README.md ARCHITECTURE.md AGENTS.md README.cli.md
```

Новые контракты должны быть отражены в документации.

## Ключевые файлы

- `README.md`
- `README.cli.md`
- `ARCHITECTURE.md`
- `AGENTS.md`
- `internal/format/markdown/README.md`
- MCP tool descriptions в `internal/adapter/stdio/tools.go`

---

# Общая финальная проверка

После выполнения всех задач запустить:

```bash
go test ./...
go test -race ./...
golangci-lint run
```

Дополнительно проверить отсутствие запрещённых подходов:

```bash
rg 'PRAGMA user_version|schema_version' .
rg 'HasPrefix\(.*Error\(\)|HasSuffix\(.*Error\(\)|Contains\(.*Error\(\)' internal
rg 'boot\.Logger|Bootstrap.*Logger' internal
```

Критерий готовности ветки:

- structural validation вызывается на обычном read-path;
- несовместимый индекс требует только явного reindex;
- ошибки не классифицируются по строкам;
- `tags` и `aliases` можно устанавливать и очищать;
- command logger не хранится в mutable Bootstrap;
- отрицательные limits отклоняются;
- тесты, race-check и linter проходят.
