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
| Report/dashboard/widget HTML self-contained | nota `.html` nel folder pertinente (es. `{{PROJECT}}/docs/`) | `memory_create` con path `.html` (richiede `capabilities.html_notes`) |
| Dati tabellari lunghi (audit, export CSV) | table note linkata dal report | `memory_create_table_note` (richiede `capabilities.table_notes`) — non incollare la tabella nel body |
| File binario (screenshot, PDF, zip) | attachment del vault | `memory_upload_attachment` / `memory_upload_resource` — per file grandi POST all'endpoint `/upload`, **mai** base64 nel contesto |
| Fine task | `{{PROJECT}}/log.md` + `{{PROJECT}}/hot.md` | `memory_append` log, `memory_edit` hot |

### Formati di nota e allegati

Il payload di bootstrap include un blocco `capabilities` con i formati attivi
su **questa** istanza (`html_notes`, `media_notes`, limiti/estensioni degli
allegati): consultalo prima di scegliere il formato.

- **Markdown è il default.** Contenuto semplice, piccolo, diff-abile,
  ricercabile: nel dubbio la nota è una `.md`. Gli altri formati sono
  eccezioni motivate, non alternative equivalenti.
- **Note `.html` native** (se `capabilities.html_notes`): un path `.html` in
  `memory_create` crea una nota HTML first-class — frontmatter in un commento
  HTML di testa, indicizzata in ricerca/grafo/backlink come una `.md`,
  renderizzata nella web UI in iframe sandbox. Usale **solo** per contenuto
  intrinsecamente HTML (report generati, dashboard/viz self-contained con JS
  inline); gli asset esterni sono bloccati by-design — inline tutto.
- **Allegati** (screenshot, PDF, CSV, dataset, zip): usa
  `memory_upload_attachment` (allegato per una nota, ritorna l'embed) o
  `memory_upload_resource` (handle indipendente). Per file oltre pochi KB
  **non** passare base64 nel contesto: POST multipart (field `file`, bearer
  token) all'endpoint `/upload` — il tuo URL MCP con `/sse` sostituito da
  `/upload`. Limite e estensioni ammesse in `capabilities.attachments`.
- **Media notes** (se `capabilities.media_notes`): un'immagine diventa nota
  first-class con `memory_create_media_note` — nota `.md` con `type: image` +
  puntatore `media:` all'attachment; la caption nel body è ciò che entra in
  ricerca, scrivila sempre.
- **Table notes** (se `capabilities.table_notes`): dati tabellari lunghi
  (audit report, export) diventano nota first-class con
  `memory_create_table_note` — nota `.md` con `type: table` + puntatore
  `media:` a un attachment `.csv`, resa come tabella paginata nella web UI e
  linkabile dal report con un wikilink. Header di colonna e numero righe
  vengono inlinati nel body (ricercabili); i **valori** delle celle no —
  scrivi una caption che dica cosa contiene la tabella. Non incollare tabelle
  lunghe nel body markdown.

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

- `type:{memory,agent,plan,skill,doc,index,handoff}` — categoria della nota
- `status:{draft,in-progress,done,archived}` — solo su plan
- `status:{pending,claimed,done,rejected}` — solo su handoff (vedi sotto)
- `topic:<area>` — dominio (es. `topic:deploy`, `topic:api`)
- `pinned` — sempre in superficie al bootstrap
- `importance: 1..5` nel frontmatter — priorità (complementare a pinned)

### Handoff fra agenti (lifecycle)

Un handoff (`memory_create_handoff`) passa contesto da un agente a un altro
come nota in `{{PROJECT}}/handoffs/`. Ciclo di vita: `pending → claimed →
done | rejected`. Se ricevi lavoro via handoff: scoprilo con
`memory_pending_handoffs`, **prendilo in carico con `memory_claim_handoff`
prima di iniziare** (il claim è atomico: fra più agenti concorrenti ne vince
uno solo) e chiudilo con `memory_complete_handoff` (outcome `done` o
`rejected`, con nota di esito opzionale). `created_by`/`claimed_by`/
`completed_by` sono stampati dal server dall'identità del token — non
falsificabili; `from_agent`/`to_agent` restano slug di ruolo dichiarativi.
Non editare a mano il frontmatter di lifecycle.

### Economia dei token (bootstrap ripetuti e letture bulk)

Dal secondo bootstrap in poi risparmia contesto: passa
`known_directives_version` (se coincide, queste direttive vengono omesse),
`known_etags` con gli etag di hot/README/agent_md dell'ultimo bootstrap
(i file invariati tornano `unchanged:true` senza body) e `mode: "lite"`
(frontmatter+outline di hot.md invece del body; le sezioni servono via
`memory_get_section`). Per letture bulk usa `memory_batch_get` con
`mode: outline|frontmatter` o `max_bytes_per_note`. Se `memory_lint`
segnala `hot-oversize`, compatta hot.md invece di lasciarlo crescere.

<!-- /gosidian:directives -->
