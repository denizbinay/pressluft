<!-- Context: core/structure | Priority: critical | Version: 1.1 | Updated: 2026-02-22 -->

# Context Structure Standard

Function-based structure keeps context discoverable and predictable. Organize files by purpose (`concepts`, `examples`, `guides`, `lookup`, `errors`), not by ad-hoc topic names.

---

## Core Rule

<rule id="function_structure" enforcement="strict">
Every context category MUST include:
- `navigation.md`
- `concepts/`
- `examples/`
- `guides/`
- `lookup/`
- `errors/`
</rule>

```text
{category}/
├── navigation.md
├── concepts/
├── examples/
├── guides/
├── lookup/
└── errors/
```

---

## Folder Intent

- `concepts/`: what something is, definitions, principles.
- `examples/`: minimal working snippets.
- `guides/`: step-by-step execution.
- `lookup/`: quick references (tables, commands, paths).
- `errors/`: recurring failures and fixes.

---

## Categorization Quick Test

Use this decision map before placing a file:

| Question | Folder |
|---|---|
| Explains what/why? | `concepts/` |
| Shows working code? | `examples/` |
| Teaches how-to steps? | `guides/` |
| Used as quick reference? | `lookup/` |
| Documents a failure/fix? | `errors/` |

---

## navigation.md Requirements

Each `navigation.md` should include:
- 1-sentence purpose
- structure snapshot
- quick routes table
- loading strategy

Keep it concise and link to files instead of duplicating content.

---

## Minimal Example

```markdown
# Payments Navigation
**Purpose**: Payment integration rules and workflows.

| Task | Path |
|---|---|
| **Handle webhook errors** | `errors/webhooks.md` |
```

---

## Validation Checklist

- [ ] Category has `navigation.md`
- [ ] Files are placed in function folders
- [ ] No broken links in navigation tables
- [ ] File remains under 200 lines

---

## Related

- `mvi.md`
- `templates.md`
- `../guides/creation.md`
