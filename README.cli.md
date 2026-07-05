# CLI Reference

## Command Directory

- [mnemonic](#mnemonic)
  - [config](#mnemonic-config)
    - [show](#mnemonic-config-show)
  - [notes](#mnemonic-notes)
    - [backlinks](#mnemonic-notes-backlinks)
    - [create](#mnemonic-notes-create)
    - [delete](#mnemonic-notes-delete)
    - [edit](#mnemonic-notes-edit)
    - [list](#mnemonic-notes-list)
    - [search](#mnemonic-notes-search)
    - [show](#mnemonic-notes-show)
  - [project](#mnemonic-project)
    - [add](#mnemonic-project-add)
    - [doctor](#mnemonic-project-doctor)
    - [import](#mnemonic-project-import)
    - [init](#mnemonic-project-init)
    - [list](#mnemonic-project-list)
    - [reindex](#mnemonic-project-reindex)
    - [remove](#mnemonic-project-remove)
    - [show](#mnemonic-project-show)
    - [sync](#mnemonic-project-sync)
  - [stdio](#mnemonic-stdio)
  - [tags](#mnemonic-tags)
    - [list](#mnemonic-tags-list)
  - [version](#mnemonic-version)
  - [web](#mnemonic-web)
    - [serve](#mnemonic-web-serve)

---

## `mnemonic`

mnemonic is a local-first personal knowledge base and search engine

### Synopsis

mnemonic is a local-first CLI tool and MCP server that indexes markdown notes, provides graph-aware reranking, structures wiki-links, and searches utilizing SQLite FTS5.

```
mnemonic [flags]
```

### Options

```
  -h, --help                help for mnemonic
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic config`

Inspect configuration

```
mnemonic config [flags]
```

### Options

```
  -h, --help   help for config
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic config help`

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type config help [path to command] for full details.

```
mnemonic config help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic config show`

Show effective config and paths

```
mnemonic config show [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic help`

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type mnemonic help [path to command] for full details.

```
mnemonic help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes`

Manage notes

```
mnemonic notes [flags]
```

### Options

```
  -h, --help   help for notes
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes backlinks`

Show backlinks for a note

```
mnemonic notes backlinks SELECTOR [flags]
```

### Options

```
  -h, --help   help for backlinks
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes create`

Create a note

```
mnemonic notes create [flags]
```

### Options

```
      --body-file string   read the note body from a file
  -h, --help               help for create
      --stdin              read the note body from stdin
      --tag stringArray    add a note tag
      --title string       note title
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes delete`

Delete a note

```
mnemonic notes delete SELECTOR [flags]
```

### Options

```
      --dry-run   show what would be deleted without making changes
      --hard      delete the note file instead of moving it to trash
  -h, --help      help for delete
      --yes       confirm deletion without prompting
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes edit`

Edit a note

```
mnemonic notes edit SELECTOR [flags]
```

### Options

```
      --append string             append text to the note body
      --body-file string          replace the note body with the contents of a file
      --clear-aliases             clear all aliases
      --clear-tags                clear all tags
  -h, --help                      help for edit
      --if-match string           only update if the current content hash matches
      --set stringArray           set a frontmatter field
      --set-aliases stringArray   replace the aliases list
      --set-tags stringArray      replace the tags list
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes help`

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type notes help [path to command] for full details.

```
mnemonic notes help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes list`

List notes

```
mnemonic notes list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes search`

Search notes

```
mnemonic notes search [flags]
```

### Options

```
      --created-after int      filter by creation time (Unix timestamp)
      --created-before int     filter by creation time (Unix timestamp)
      --created-since string   filter by creation time (relative, e.g. 24h)
      --debug                  show debug fields (path, score, content_hash)
  -h, --help                   help for search
      --include-related        include related notes in results
      --limit int              maximum number of results (default 10)
  -q, --query strings          search query (repeatable)
  -t, --tag strings            filter by tag (repeatable)
      --updated-after int      filter by update time (Unix timestamp)
      --updated-before int     filter by update time (Unix timestamp)
      --updated-since string   filter by update time (relative, e.g. 24h)
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic notes show`

Show a note

```
mnemonic notes show SELECTOR [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project`

Manage projects

```
mnemonic project [flags]
```

### Options

```
  -h, --help   help for project
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project add`

Register an existing structured project

```
mnemonic project add [PATH] [flags]
```

### Options

```
  -h, --help   help for add
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project doctor`

Run project health checks

```
mnemonic project doctor [PROJECT] [flags]
```

### Options

```
      --all            run doctor across all active projects
  -h, --help           help for doctor
      --kind strings   filter diagnostics by kind (invalid_frontmatter, missing_required_field, missing_summary, missing_timestamp, invalid_timestamp, duplicate_slug, duplicate_alias, unresolved_link, ambiguous_link, empty_body)
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project help`

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type project help [path to command] for full details.

```
mnemonic project help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project import`

Onboard a raw directory of markdown notes

```
mnemonic project import [PATH] [flags]
```

### Options

```
      --dry-run   Show what would be imported without making changes
  -h, --help      help for import
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project init`

Initialize a project

```
mnemonic project init NAME [flags]
```

### Options

```
      --description string   optional description of this memory's knowledge scope
  -h, --help                 help for init
      --local                create a local project
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project list`

List registered projects

```
mnemonic project list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project reindex`

Rebuild project indexes

```
mnemonic project reindex [PROJECT] [flags]
```

### Options

```
      --all    reindex all active projects
  -h, --help   help for reindex
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project remove`

Remove a project

### Synopsis

By default, removes the registry entry while keeping markdown notes intact.

  remove SLUG
      Delete the registry entry (pointer file or manifest).

  remove --wipe SLUG
      Delete the registry entry and all markdown notes.

Index and lock files are always cleaned up.

```
mnemonic project remove SLUG [flags]
```

### Options

```
  -h, --help   help for remove
      --wipe   also remove all markdown notes
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project show`

Show a registered project

```
mnemonic project show SLUG [flags]
```

### Options

```
  -h, --help   help for show
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic project sync`

Reconcile metadata for new or updated notes

```
mnemonic project sync PROJECT_SELECTOR [FILES...] [flags]
```

### Options

```
      --dry-run   Show what would be synced without making changes
  -h, --help      help for sync
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic stdio`

Run the MCP stdio adapter

```
mnemonic stdio [flags]
```

### Options

```
  -h, --help        help for stdio
      --read-only   run the MCP server without write tools
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic tags`

Manage tags

```
mnemonic tags [flags]
```

### Options

```
  -h, --help   help for tags
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic tags help`

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type tags help [path to command] for full details.

```
mnemonic tags help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic tags list`

List tags

```
mnemonic tags list [flags]
```

### Options

```
  -h, --help   help for list
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic version`

Prints the version of mnemonic

```
mnemonic version [flags]
```

### Options

```
  -h, --help   help for version
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic web`

Serve the web MCP server

```
mnemonic web [flags]
```

### Options

```
  -h, --help   help for web
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic web help`

Help about any command

### Synopsis

Help provides help for any command in the application.
Simply type web help [path to command] for full details.

```
mnemonic web help [command] [flags]
```

### Options

```
  -h, --help   help for help
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```

---

## `mnemonic web serve`

Serve MCP over HTTP/SSE

```
mnemonic web serve [flags]
```

### Options

```
  -h, --help          help for serve
      --port string   listen port for the web server
```

### Options inherited from parent commands

```
      --json                output in JSON format
      --log-format string   log format (text, json)
      --log-level string    log level (debug, info, warn, error)
  -p, --project string      select a project by slug or UUID
```
