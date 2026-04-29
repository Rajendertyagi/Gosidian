# memory_init_agent — modalità augment (Aider / CONVENTIONS.md)

Hai ricevuto il payload dal tool MCP `memory_init_agent`. **Ruolo**:
integrare le regole gosidian nel file `CONVENTIONS.md` (il file che
Aider carica via `--read CONVENTIONS.md` o nel `.aider.conf.yml`)
preservando tutto il contenuto esistente.

- `gosidian_block` con placeholder non risolti: `{{LANGUAGE}}`,
  `{{CODE_LANGUAGE}}`, `{{PROJECT_TYPE}}`, `{{STACK}}`, `{{HOT_FILES}}`.
- `needs_scaffold` bool, `mode: "augment"`.

## Step 1 — Determina filename

**Default**: `CONVENTIONS.md` nella root. Se `filename_hint` diverso
(es. `.aider/rules.md`), rispettalo. Se l'utente ha una setup diversa
(es. conventions in un subfolder), chiedi dove scrivere prima di agire.

## Step 2 — Pre-check vault

Se `needs_scaffold=true`, chiama
`mcp__gosidian__memory_project_scaffold(project="{{PROJECT}}",
template="karpathy-wiki")`.

## Step 3 — Raccogli placeholder

Da `existing_content` o chiedi in chat (Aider è CLI, usa un menu
numerato testuale semplice).

## Step 4 — Merge

- Non sovrascrivere sezioni esistenti.
- Appendi `gosidian_block` come sezione `## Memory & workflow (gosidian)`.
- Risolvi i placeholder prima del merge.
- Conflitti → fermati e chiedi.

## Step 5 — Materializza

Scrivi il file completo (existing + gosidian).

**Nota importante**: ricorda all'utente di aggiungere il file al
`.aider.conf.yml` se non già presente:

```yaml
read:
  - CONVENTIONS.md
```

Oppure usarlo inline con `aider --read CONVENTIONS.md`.

## Step 6 — Primo ingest (facoltativo)

Seed `architecture.md` e `hot.md` via `mcp__gosidian__memory_edit`.

## Step 7 — Conferma all'utente

Sommario + reminder sulla configurazione Aider.
