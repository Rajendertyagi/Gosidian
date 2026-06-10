<!-- gosidian:directives v={{DIRECTIVES_VERSION}} -->
<!--
  directives.tmpl.md — direttive operative gosidian, servite da
  memory_bootstrap nel campo `directives_block` (parametrico solo su
  {{PROJECT}} e {{DIRECTIVES_VERSION}}). NON è un file da scrivere su
  disco: l'agente le legge fresche a ogni bootstrap. La personalizzazione
  per-progetto (stack, hot files, lingua) vive nello stub locale, non qui.
-->

## Direttive operative (gosidian) — v{{DIRECTIVES_VERSION}}

Queste regole sono servite da `memory_bootstrap` (campo `directives_block`)
e valgono per il lavoro nel progetto `{{PROJECT}}`. **Leggile e seguile a
ogni sessione**: non tenerne una copia nel file di istruzioni — si
aggiornano da sole quando il server cambia versione.

### Auto-conversione del file di istruzioni (one-time)

Controlla **subito** il file di istruzioni di questo progetto
(CLAUDE.md/AGENTS.md/… già caricato in contesto): se **non** contiene il
marker `<!-- gosidian:stub -->`, è un file *pre-stub* (probabilmente con
queste stesse direttive duplicate al suo interno). Convertilo **una-tantum**
a stub sottile:

```
mcp__gosidian__memory_init_agent({project: "{{PROJECT}}", existing_content: "<contenuto del file>"})
```

(modalità augment) — innesta lo stub fra i marker `gosidian:stub`, **preserva
le sezioni locali** del repo e **rimuovi** le direttive operative duplicate
(ora le servo io). Da quel momento il file resta sottile e queste direttive
arrivano sempre fresche dal bootstrap. Idempotente: se il marker è già
presente, non fare nulla. Procedura completa: skill `rollout direttive`.

### Mappa delle cartelle del vault

| Path vault | Scopo | Quando leggere |
|---|---|---|
| `{{PROJECT}}/memory/` | Conoscenza stabile: architecture, ADR, conventions, glossary, environments. Il "wiki" nel senso Karpathy. | Inizio task per contesto |
| `{{PROJECT}}/agents/` | Ruoli specializzati (un agent = un dominio ricorrente) | Quando il task ricade nello scope di un agent |
| `{{PROJECT}}/plans/` | Piani di task non banali, formato `YYYYMMDD-<slug>.md`, con `Outcome` post-esecuzione | Prima di implementare task large |
| `{{PROJECT}}/skills/` | Procedure ripetibili | Prima di eseguire operazioni ricorrenti |
| `{{PROJECT}}/docs/` | Q&A, open questions, improvements backlog, bug tracker | Quando cerchi decisioni passate / side findings |
| `{{PROJECT}}/hot.md` | Session cache aggiornata fine-task | **Sempre** al bootstrap |
| `{{PROJECT}}/log.md` | Log append-only di attività | Append a fine task |

### Quando scrivere in memoria (ingest rules)

Durante il lavoro, quando scopri qualcosa che sopravvivrà al task
corrente, aggiorna il vault:

| Discovery | Dove scrivere | Come |
|---|---|---|
| Fatto sul codice / sistema non documentato | `{{PROJECT}}/memory/architecture.md` | `memory_edit` della sezione rilevante |
| Decisione tecnica vincolante | `{{PROJECT}}/memory/decisions.md` | `memory_append` di un nuovo `## ADR-NNN` |
| Termine di dominio nuovo | `{{PROJECT}}/memory/glossary.md` | `memory_append` |
| Nuova convenzione di codice / test / ops | `{{PROJECT}}/memory/conventions.md` | `memory_edit` |
| Cambio infra / deploy / env | `{{PROJECT}}/memory/environments.md` | `memory_edit` |
| Task non banale che sta per cominciare | `{{PROJECT}}/plans/<YYYYMMDD>-<slug>.md` | `memory_create` **prima** di toccare il codice |
| Procedura ripetuta ≥2 volte nella stessa sessione | `{{PROJECT}}/skills/<slug>.md` | `memory_create` con frontmatter `type:skill` + trigger phrase + step + gotcha |
| Dominio di competenza ricorrente (ri-leggi le stesse 3-5 note in 2+ task) | `{{PROJECT}}/agents/<slug>.md` | `memory_create` con `type:agent` + sezione "Contesto obbligatorio" |
| Bug osservato anche fuori scope del task corrente | `{{PROJECT}}/docs/bugs.md` | `memory_append` come `## BUG-NNN` |
| Domanda aperta senza risposta immediata | `{{PROJECT}}/docs/open-questions.md` | `memory_append` sezione "Aperte" come `### OQ-NNN` |
| Improvement / technical debt identificato | `{{PROJECT}}/docs/improvements.md` | `memory_append` come `## IMP-NNN` |
| Fine task | `{{PROJECT}}/log.md` + `{{PROJECT}}/hot.md` | `memory_append` log, `memory_edit` hot |

### Regola delle ripetizioni

Se nel corso della sessione hai eseguito la stessa sequenza di comandi
(anche con piccole variazioni) **2 o più volte**, fermati, promuovila a
skill **prima** della terza esecuzione. Non aspettare "un giorno in cui
servirà di nuovo" — oggi è quel giorno. Lo stesso vale per gli agent:
se ti ritrovi a ri-leggere gli stessi file di memory per riorientarti
su un'area specifica più volte, quella è una richiesta latente di
agent scritto.

### Regola della cattura immediata

Le discovery laterali — bug osservati, domande senza risposta,
improvement identificati — vanno scritte in
`{{PROJECT}}/docs/{bugs,open-questions,improvements}.md` **nel momento
in cui emergono**, non salvate per l'end-of-task. Costa 30 secondi,
salva settimane di riscoperte. Lasciarle come "side finding" dentro un
plan outcome equivale a perderle.

### Plan: vault vs scratchpad

- **Task small/medium** (1-2 file, fix isolato, niente migration, niente
  API change): scratchpad locale dell'agent, scartato dopo. Annota come
  entry `pattern` in `{{PROJECT}}/log.md`.
- **Task large/architetturale** (3+ file, migration, nuovo sotto-sistema,
  cambio ADR, refactor cross-pacchetto): plan autoritativo in
  `{{PROJECT}}/plans/<YYYYMMDD>-<slug>.md` **prima** di implementare.
  Chiudilo con sezione `Outcome` compilata (hash commit, sorprese,
  side findings).

Status workflow dei plan: `draft` → `in-progress` → `done` | `archived`.
Aggiornare il tag `status:*` nel frontmatter via `memory_edit` quando lo
stato cambia.

### Workflow end-of-task

1. **Skill-check**: hai eseguito una procedura ≥2 volte durante il task?
   Se sì, **prima** di chiudere crea la skill in
   `{{PROJECT}}/skills/<slug>.md`.
2. **Aggiorna `{{PROJECT}}/hot.md`**: new current focus, rimuovi plan
   chiusi, shift recent decisions.
3. **Append a `{{PROJECT}}/log.md`**: entry tipizzata con data ISO
   (`bootstrap`, `plan-closed`, `adr`, `pattern`, `fix`, `discovery`,
   `ops`).
4. **Compila `Outcome`** nel plan se ne esisteva uno.
5. **Aggiorna la memory** se hai scoperto qualcosa di strutturale:
   architecture, decisions, glossary, conventions, environments.

Saltare questo workflow è la via più veloce perché la memoria diventi
inutile alla prossima sessione.

### Vocabolario tag (chiuso)

- `type:{memory,agent,plan,skill,doc,index}` — categoria della nota
- `status:{draft,in-progress,done,archived}` — solo su plan
- `topic:<area>` — dominio (es. `topic:deploy`, `topic:api`)
- `pinned` — sempre in superficie al bootstrap
- `importance: 1..5` nel frontmatter — priorità (complementare a pinned)

<!-- /gosidian:directives -->
