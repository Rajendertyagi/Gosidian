<!--
  gosidian_block.tmpl.md — sezione "Memory & workflow (gosidian)" da
  innestare nel file di istruzioni agent-native del progetto
  "{{PROJECT}}". Mantenuto dal tool `memory_init_agent` del server MCP
  gosidian. Placeholder: {{PROJECT}} {{TODAY}} {{LANGUAGE}}
  {{CODE_LANGUAGE}} {{PROJECT_TYPE}} {{STACK}} {{HOT_FILES}}
  {{AGENT_NAME}} {{AGENT_PROFILE}}.
-->

## Memory & workflow (gosidian)

Stai lavorando su **{{PROJECT}}** ({{PROJECT_TYPE}}, stack: {{STACK}}).
La memoria persistente cross-sessione vive nel vault gosidian, nel
progetto `{{PROJECT}}/`, accessibile via i tool MCP `mcp__gosidian__memory_*`.

Il pattern è Karpathy-Wiki-Stack: le note sono la wiki che **tu**
(l'agent) mantieni mentre lavori — non il tuo output primario, ma la
tua memoria compilata.

### Regola zero — Bootstrap di sessione

All'inizio di **ogni** sessione, **prima** di qualsiasi `Grep`/`Read` sul
codice, fai il bootstrap della memoria con una singola call:

```
mcp__gosidian__memory_bootstrap({project: "{{PROJECT}}"})
```

Ritorna in un solo JSON: `hot_md_content`, `readme_content`,
`active_plans[]`, `available_skills[]`, `available_agents[]`,
`recent_notes[]`, `project_stats`, `missing[]`. Sostituisce 4 chiamate
separate. **Controlla sempre `available_skills[]` prima di iniziare una
procedura operativa** (build, deploy, config edit, rotazione,
troubleshoot): riusare una skill è sempre meglio che reinventarla.

Se `memory_bootstrap` non è disponibile, fallback manuale:

1. `mcp__gosidian__memory_get({path: "{{PROJECT}}/hot.md"})` — orientamento rapido
2. `mcp__gosidian__memory_get({path: "{{PROJECT}}/README.md"})` — mappa del progetto
3. `mcp__gosidian__memory_notes_by_tag({tag: "status:in-progress"})` — plan attivi
4. `mcp__gosidian__memory_notes_by_tag({tag: "type:skill"})` — catalogo skill

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

**Hot files** (dominio-specifico): {{HOT_FILES}}

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
| Task non banale che sta per cominciare | `{{PROJECT}}/plans/{{TODAY}}-<slug>.md` | `memory_create` **prima** di toccare il codice |
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
  `{{PROJECT}}/plans/{{TODAY}}-<slug>.md` **prima** di implementare.
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

### Lingua

- UI utente, messaggi, note del vault: **{{LANGUAGE}}**
- Codice, commenti inline, commit message, nomi di variabili/funzioni:
  **{{CODE_LANGUAGE}}**

### Meta

- Questa sezione è stata generata da `memory_init_agent` ({{TODAY}}) per
  il profilo agent **{{AGENT_NAME}}** (`agent_profile={{AGENT_PROFILE}}`).
  Se aggiungi regole o la rigeneri, mantieni il blocco "Memory &
  workflow (gosidian)" come unità riconoscibile.
