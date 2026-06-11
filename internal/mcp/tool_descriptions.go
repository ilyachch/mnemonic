package mcp

// Keep tool descriptions compact. They are usually included in the model/tool
// selection context by MCP clients. Long policy text should live in prompts,
// resources, or documentation instead of being duplicated across tools.

const searchNotesDescription = `Search project memory notes.

Use this proactively before project-specific work when prior context may exist: architecture, decisions, conventions, bugs, APIs, integrations, files, tasks, or previous findings.

Prefer searching memory before asking the user to repeat project facts or rediscovering known information.

If relevant notes are found, read the most relevant notes before relying on them or modifying memory. If no relevant notes are found, continue investigation and later create or update notes with durable findings.`

const readNoteDescription = `Read a project memory note by note_id, slug, path, or title.

Use after search_notes, list_notes, or list_backlinks returns a relevant note.

Read notes before relying on them, editing them, deleting them, or using them as evidence. The result includes content_hash for safe edit_note and delete_note calls.`

const createNoteDescription = `Create a project memory note for durable, reusable project knowledge discovered during work.

Use for stable facts that will help future agents: decisions, architecture, conventions, setup steps, API contracts, debugging findings, constraints, or task outcomes.

Before creating, search existing notes to avoid duplicates. Prefer edit_note when a related note already exists. Do not store secrets, credentials, guesses, temporary chat details, or unverified assumptions.`

const editNoteDescription = `Edit an existing project memory note when new information corrects, extends, confirms, or supersedes stored project knowledge.

Use after project work when decisions, implementation details, bug status, conventions, or task outcomes changed.

Read the note first and use content_hash as if_match_hash when possible. Prefer editing over creating duplicate notes.`

const deleteNoteDescription = `Delete a project memory note only when it is clearly obsolete, duplicated, misleading, accidentally created, or harmful to keep.

Before deleting, read the note and consider whether edit_note or merging would be safer.

Do not delete notes merely because they are old. Use hard_delete only when explicitly requested or when the note must not remain in memory.`

const listNotesDescription = `List project memory notes, optionally paginated.

Use to orient yourself, inspect available memory, or find notes when search terms are unclear.

Prefer search_notes for targeted context lookup.`

const listTagsDescription = `List tags used in project memory.

Use to understand how knowledge is organized, discover available topics, choose consistent tags, or refine broad searches.`

const listBacklinksDescription = `List notes that link to or reference a specific project memory note.

Use to understand related decisions, dependencies, affected features, and consequences before editing or deleting an important note.`

// projectMemoryPolicyLong is the full policy text for project memory usage.
// It is not automatically prepended to tool descriptions. Use it in MCP
// prompts, resources, README, or documentation instead.
const projectMemoryPolicyLong = `Project memory usage policy:
- Use these tools as persistent memory for the current project.
- Before project-specific work, search memory when relevant prior context may exist.
- Prefer searching memory before asking the user to repeat project facts or independently rediscovering known information.
- Read relevant notes before relying on them, editing them, or deleting them.
- When durable project knowledge is discovered, create or update notes.
- After completing project work, update memory if decisions, conventions, architecture, APIs, bugs, setup steps, constraints, or task outcomes changed.
- Do not store secrets, credentials, temporary conversation details, guesses, or unverified assumptions.`
