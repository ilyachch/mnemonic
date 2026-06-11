package mcp

const projectMemoryUsagePolicy = `Project memory usage policy:
- Use these tools as persistent memory for the current project.
- Before project-specific work, search memory when relevant prior context may exist.
- Prefer searching memory before asking the user to repeat project facts or independently rediscovering known information.
- Read relevant notes before relying on them, editing them, or deleting them.
- When durable project knowledge is discovered, create or update notes.
- After completing project work, update memory if decisions, conventions, architecture, APIs, bugs, setup steps, constraints, or task outcomes changed.
- Do not store secrets, credentials, temporary conversation details, guesses, or unverified assumptions.`

const listNotesDescription = `List project memory notes, optionally paginated.

Use this to orient yourself in available project memory, discover note structure, inspect recent or broad memory coverage, or find notes when search terms are unclear.

Prefer search_notes for targeted context lookup. Use list_notes when starting in an unfamiliar project area or when you do not yet know what to search for.`

const listTagsDescription = `List tags used in project memory.

Use this to understand how project knowledge is organized, discover available topics, choose consistent tags for new notes, or refine broad memory searches.

Use before create_note or edit_note when tag consistency matters.`

const searchNotesDescription = projectMemoryUsagePolicy + `

Search project memory notes by natural language query, keywords, tags, entities, file paths, task names, decisions, bugs, features, integrations, or architecture concepts.

Use this tool proactively as the first step when project context may be missing, stale, ambiguous, or useful for the current task.

Use it before asking the user to repeat project details or before independently rediscovering facts that may already be documented.

Use it when the user refers to prior work, existing decisions, conventions, implementation details, bugs, features, services, APIs, modules, files, tickets, people, or project-specific terminology.

If relevant notes are found, read the most relevant notes before making project-specific claims or changing memory.

If no relevant notes are found, continue gathering information from available sources, then create or update notes with durable facts learned during the work.`

const readNoteDescription = projectMemoryUsagePolicy + `

Read a specific project memory note by note_id, slug, path, or title.

Use this after search_notes, list_notes, or list_backlinks returns a potentially relevant note.

Read notes before relying on their contents, editing them, deleting them, or using them as evidence for project decisions.

The result includes the full body, frontmatter, updated_at, and content_hash. Use content_hash as if_match_hash for safe edit_note or delete_note operations.`

const listBacklinksDescription = `List notes that link to or reference a specific project memory note.

Use this to understand related decisions, dependencies, affected features, architectural relationships, and consequences before editing or deleting an important note.

Use backlinks when a note appears central, outdated, contradicted, or connected to a broader project area.`

const createNoteDescription = projectMemoryUsagePolicy + `

Create a new project memory note for durable, reusable knowledge discovered during the task.

Use this proactively when you learn stable project knowledge that will likely help future agents or future sessions: architecture, domain rules, coding conventions, setup steps, API contracts, decisions, constraints, recurring problems, important file locations, integration details, debugging findings, or task outcomes.

Before creating a note, search existing notes to avoid duplicates. If a related note already exists, use edit_note instead.

Do not create notes for temporary chat details, obvious facts, guesses, unverified assumptions, secrets, credentials, or private user information unrelated to the project.`

const editNoteDescription = projectMemoryUsagePolicy + `

Edit an existing project memory note when new information changes, corrects, extends, confirms, or supersedes stored project knowledge.

Use this after completing project work if the task produced durable knowledge, changed implementation details, invalidated old assumptions, resolved a bug, clarified a decision, or updated the status of a feature/task.

Prefer editing an existing note over creating a duplicate. Read the note first and use its content_hash as if_match_hash when possible.

Preserve useful historical context when it helps future agents understand why something changed. Remove or clearly mark outdated information when keeping it would mislead future work.`

const deleteNoteDescription = `Delete a project memory note only when it is clearly obsolete, duplicated, misleading, accidentally created, or harmful to keep.

Use this cautiously. Before deleting, read the note and consider whether edit_note or merging into another note would be safer.

Do not delete notes merely because they are old. Delete only when keeping the note would likely confuse future project work.

Use hard_delete only when the user explicitly requested permanent deletion or when the note contains information that must not remain in memory.`
