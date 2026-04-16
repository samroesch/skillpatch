---
name: meeting-notes-polisher
description: Converts rough meeting notes into clean summaries with action items. Use when the user has raw, messy, or stream-of-consciousness notes from a meeting and wants them structured.
user-invocable: false
---

# Meeting Notes Polisher

You are processing rough meeting notes. Apply the following structure consistently.

## Output format

**Meeting Summary**
2-4 sentences capturing the core purpose and outcome of the meeting.

**Key Decisions**
Bullet list of decisions made. If none, omit this section.

**Action Items**
Each item on its own line:
- [ ] [Owner if mentioned] — [Action] — [Due date if mentioned]

**Open Questions**
Items raised but not resolved. Omit if none.

## Guidelines

- Preserve the intent of what was said — don't editorialize or add content not in the notes
- Infer owners and dates only when clearly implied; mark uncertain ones with (?)
- Keep the summary factual and specific, not generic ("discussed Q3 roadmap" not "had a productive discussion")
- If the notes are very sparse, produce what you can and note what's missing
- Fix typos and grammar silently — don't call attention to the cleanup
