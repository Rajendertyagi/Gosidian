# memory_init_agent — modalità augment (Cursor)

Hai ricevuto il payload dal tool MCP `memory_init_agent`. **Ruolo**:
integrare le regole gosidian nelle Cursor Rules del workspace preservando
tutto il contenuto esistente.

Il server ti ha passato:

- `gosidian_block` — markdown parametrico. Placeholder non risolti:
  `{{LANGUAGE}}`, `{{CODE_LANGUAGE}}`, `{{PROJECT_TYPE}}`, `{{STACK}}`,
  `{{HOT_FILES}}`.
- `needs_scaffold` — se `true`, crea il progetto vault prima del merge.
- `mode: "augment"` — è presente `existing_content`.

## Step 1 — Determina filename

Cursor usa due convenzioni:

- **Nuovo formato** (Cursor ≥ 0.45): `.cursor/rules/<nome>.mdc` — file
  multipli, ciascuno con frontmatter `description` + `globs`.
- **Formato legacy**: `.cursorrules` nella root (unico file,
  deprecato).

Se `filename_hint` è presente, rispettalo. Altrimenti:

- Se esiste `.cursor/rules/` → nuovo formato. Crea
  `.cursor/rules/gosidian-memory.mdc` per la sezione gosidian, lascia
  gli altri file intatti.
- Se esiste solo `.cursorrules` → formato legacy, fai merge dentro.
- Se esistono entrambi → chiedi all'utente quale usare.

## Step 2 — Pre-check vault

Se `needs_scaffold=true`, chiama `mcp__gosidian__memory_project_scaffold`
con `project="{{PROJECT}}"` e `template="karpathy-wiki"`.

## Step 3 — Raccogli placeholder

Raccogli `{{LANGUAGE}}`, `{{CODE_LANGUAGE}}`, `{{PROJECT_TYPE}}`,
`{{STACK}}`, `{{HOT_FILES}}` da `existing_content` o chiedili
all'utente (Cursor non ha `AskUserQuestion`: usa una normale domanda
in chat, con opzioni numerate chiare).

## Step 4 — Merge

**Nuovo formato `.mdc`**: crea un file nuovo `gosidian-memory.mdc` con
frontmatter:

```mdc
---
description: Gosidian memory layer — session bootstrap, ingest rules, workflow end-of-task
globs:
  - "**/*"
alwaysApply: true
---
```

Seguito dal `gosidian_block` con placeholder risolti. Gli altri file di
regole Cursor esistenti non vengono toccati.

**Formato legacy `.cursorrules`**: apre il file esistente, appendi
`gosidian_block` (con placeholder risolti) in coda, preceduto da una
riga separatrice. Non toccare regole esistenti. Conflitti strutturali
→ fermati e chiedi (stesso protocollo del profilo Claude).

## Step 5 — Materializza

Scrivi il file nel formato scelto. Nel nuovo formato, un solo file
`.mdc` dedicato a gosidian. Nel legacy, il `.cursorrules` completo
riscritto.

## Step 6 — Primo ingest (facoltativo)

Se dallo scan emerge struttura, fai un seed in
`{{PROJECT}}/memory/architecture.md` e `{{PROJECT}}/hot.md` via i tool
`mcp__gosidian__memory_edit`.

## Step 7 — Conferma all'utente

Sommario breve: file scritto (path), formato scelto (mdc o legacy),
scaffold vault eseguito, placeholder risolti, conflitti risolti.
