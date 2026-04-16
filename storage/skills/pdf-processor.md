---
name: pdf-processor
description: Extracts, summarizes, and restructures content from PDFs. Use when the user is working with a PDF document and wants to extract text, summarize sections, find specific information, or reformat the content.
user-invocable: false
---

# PDF Processor

You are working with PDF content. Apply the appropriate mode based on what the user needs.

## Modes

**Extract** — pull specific content
- Identify the relevant sections by heading or topic
- Return verbatim text with section labels
- Note page numbers if available

**Summarize** — condense the document
- Lead with a one-paragraph executive summary
- Follow with section-by-section bullet summaries
- Flag any sections that appear important but are unclear or truncated

**Restructure** — reformat for a new purpose
- Ask the user what format they need if not specified (table, list, outline, narrative)
- Preserve all factual content; reorder for clarity
- Note any content that doesn't fit the new structure

**Find** — locate specific information
- Search for the requested information across all sections
- Return the exact text plus surrounding context
- If not found, say so clearly

## Guidelines

- PDFs often have extraction artifacts (line breaks mid-sentence, garbled tables). Silently clean these.
- If the PDF content appears to be a scanned image rather than text, note this and describe what's visible
- For large documents, summarize by section rather than attempting a single summary
- Preserve numeric data, dates, and proper nouns exactly as written
