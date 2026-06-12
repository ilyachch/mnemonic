package mcp

// Keep tool descriptions compact to save tokens.

const searchNotesDescription = `Search this memory.
**YOU MUST** search proactively before acting or asking questions to find existing rules, workflows, or preferences. Never ask the user for facts already stored. Read found notes before proceeding.`

const readNoteDescription = `Read a note by note_id, slug, path, or title.
Always read before using, editing, or deleting a note. Returns 'content_hash', which is REQUIRED for safe edits/deletions.`

const createNoteDescription = `Save new durable knowledge (workflows, preferences, facts).
**YOU MUST** persist learned context for future use. Search first to avoid duplicates (use 'edit_note' if related context exists). Do NOT save secrets or chat logs.`

const editNoteDescription = `Update existing notes when knowledge evolves.
**YOU MUST** keep memory accurate. Read the note first to get 'content_hash', then pass it as 'if_match_hash'.`

const deleteNoteDescription = `Delete permanently obsolete or duplicated notes.
Read first. Prefer 'edit_note' (e.g., adding a deprecation warning) unless deletion is strictly required.`

const listNotesDescription = `List all notes in this memory (paginated).
Use to browse available knowledge. Prefer 'search_notes' for specific queries.`

const listTagsDescription = `List available tags.
Use to understand categorization, ensure consistent tagging for new notes, or refine searches.`

const listBacklinksDescription = `List notes linking to a target note.
Use to check dependencies and impact before editing or deleting important knowledge.`

// memoryPolicyLong can be used in prompts or documentation.
const memoryPolicyLong = `Memory usage policy:
- **YOU MUST** search memory before asking questions or making assumptions.
- Never ask the user to repeat stored information.
- Always read notes before relying on, editing, or deleting them.
- **YOU MUST** save new durable knowledge (workflows, context, facts) immediately.
- No secrets or temporary chat logs.`
