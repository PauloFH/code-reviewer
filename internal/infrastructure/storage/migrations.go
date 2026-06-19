package storage

// migration001 is the original reviewed_merge_requests table.
const migration001 = `
CREATE TABLE IF NOT EXISTS reviewed_merge_requests (
    id            INTEGER  PRIMARY KEY AUTOINCREMENT,
    client_name   TEXT     NOT NULL,
    project_ref   TEXT     NOT NULL,
    mr_iid        INTEGER  NOT NULL,
    mr_id         INTEGER,
    reviewed_user TEXT     NOT NULL,
    reviewed_at   DATETIME NOT NULL,
    UNIQUE(client_name, project_ref, mr_iid, reviewed_user)
);`

// migration002 adds the clients table and the review_events event log.
const migration002 = `
CREATE TABLE IF NOT EXISTS clients (
    id             INTEGER  PRIMARY KEY AUTOINCREMENT,
    name           TEXT     NOT NULL UNIQUE,
    platform       TEXT     NOT NULL,
    base_url       TEXT     NOT NULL,
    token          TEXT     NOT NULL,
    webhook_secret TEXT     NOT NULL,
    username       TEXT     NOT NULL,
    created_at     DATETIME NOT NULL,
    updated_at     DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS review_events (
    id                INTEGER  PRIMARY KEY AUTOINCREMENT,
    client_name       TEXT     NOT NULL,
    project_ref       TEXT     NOT NULL,
    mr_iid            INTEGER  NOT NULL,
    mr_id             INTEGER,
    mr_title          TEXT     NOT NULL DEFAULT '',
    mr_url            TEXT     NOT NULL DEFAULT '',
    author_username   TEXT     NOT NULL DEFAULT '',
    prompt_tokens     INTEGER  NOT NULL DEFAULT 0,
    completion_tokens INTEGER  NOT NULL DEFAULT 0,
    total_tokens      INTEGER  NOT NULL DEFAULT 0,
    cost_usd          REAL     NOT NULL DEFAULT 0.0,
    duration_ms       INTEGER  NOT NULL DEFAULT 0,
    status            TEXT     NOT NULL DEFAULT 'success',
    error_message     TEXT,
    reviewed_at       DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_review_events_client
    ON review_events(client_name, reviewed_at DESC);

CREATE INDEX IF NOT EXISTS idx_review_events_time
    ON review_events(reviewed_at DESC);`

// migration003 adds the prompts library and scope-based assignments.
const migration003 = `
CREATE TABLE IF NOT EXISTS prompts (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    name            TEXT     NOT NULL UNIQUE,
    system_prompt   TEXT     NOT NULL DEFAULT '',
    output_template TEXT     NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS prompt_assignments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    prompt_id    INTEGER NOT NULL REFERENCES prompts(id) ON DELETE CASCADE,
    scope        TEXT    NOT NULL CHECK(scope IN ('global','client','project')),
    client_name  TEXT    NOT NULL DEFAULT '',
    project_ref  TEXT    NOT NULL DEFAULT '',
    UNIQUE(scope, client_name, project_ref)
);

CREATE INDEX IF NOT EXISTS idx_prompt_assignments_lookup
    ON prompt_assignments(client_name, project_ref);

INSERT INTO prompts(name, system_prompt, output_template, created_at, updated_at)
VALUES(
    'Prompt Padrão',
    '<h2>Papel</h2><p>Você é um revisor sênior de código com ampla experiência em engenharia de software, arquitetura de sistemas e boas práticas de desenvolvimento.</p><h2>Diretrizes</h2><ul><li>Revise APENAS o que está no diff recebido — não invente código que não existe.</li><li>Seja objetivo, direto e em português brasileiro.</li><li>Não faça comentários genéricos ou elogios vazios.</li><li>Identifique bugs, vulnerabilidades de segurança, problemas de performance, violações de SOLID e code smells.</li><li>Sugira melhorias concretas com exemplos de código quando aplicável.</li><li>Se o diff estiver bom e não houver problemas relevantes, diga isso de forma breve.</li><li>Use diagramas Mermaid quando ajudar a explicar fluxos complexos ou relações entre componentes.</li><li>Organize a análise por arquivo quando o diff tocar múltiplos arquivos.</li></ul><h2>Contexto</h2><p>Você receberá o título, descrição e URL do Merge Request, seguido do diff completo. Analise o diff e retorne sua revisão no formato especificado no template de saída.</p>',
    '<h2>Guia do Revisor</h2><p>(Resumo de 2-3 frases explicando o que o MR faz e qual o impacto arquitetural)</p><h3>Diagrama de sequência</h3><p>(Se aplicável — inclua um diagrama Mermaid sequenceDiagram mostrando o fluxo principal introduzido ou modificado)</p><h3>Diagrama de classes/componentes</h3><p>(Se aplicável — inclua um diagrama Mermaid classDiagram mostrando novos tipos, interfaces ou relações entre módulos)</p><h3>Alterações por arquivo</h3><p>(Tabela com resumo das mudanças agrupadas por arquivo)</p><table><tr><th>Mudança</th><th>Detalhes</th><th>Arquivos</th></tr><tr><td>(descrição curta)</td><td><ul><li>detalhe 1</li><li>detalhe 2</li></ul></td><td><code>path/to/file.go</code></td></tr></table><h3>Pontos de atenção</h3><ul><li>(bugs, vulnerabilidades, race conditions, edge cases não tratados)</li></ul><h3>Sugestões de melhoria</h3><ul><li>(refatorações, simplificações, melhor nomenclatura, padrões mais adequados)</li></ul><h3>Riscos</h3><ul><li>(breaking changes, impacto em performance, regressões possíveis)</li></ul><h3>Testes sugeridos</h3><ul><li>(cenários que deveriam ter cobertura de teste)</li></ul><h3>Conclusão</h3><p>(Veredicto: ✅ Aprovado | ⚠️ Aprovado com ressalvas | ❌ Solicitar alterações — com justificativa breve)</p>',
    datetime('now'),
    datetime('now')
);

INSERT INTO prompt_assignments(prompt_id, scope, client_name, project_ref)
VALUES((SELECT id FROM prompts WHERE name='Prompt Padrão'), 'global', '', '');
`

// migration004 adds the users table for multi-user authentication.
const migration004 = `
CREATE TABLE IF NOT EXISTS users (
    id         INTEGER  PRIMARY KEY AUTOINCREMENT,
    username   TEXT     NOT NULL UNIQUE,
    password   TEXT     NOT NULL,
    role       TEXT     NOT NULL CHECK(role IN ('super','gestor','padrao')),
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
`

// migration005 adds the frontend-managed AI provider configs and their
// scope-based assignments. Credential columns store AES-256-GCM ciphertext.
const migration005 = `
CREATE TABLE IF NOT EXISTS ai_configs (
    id                INTEGER  PRIMARY KEY AUTOINCREMENT,
    name              TEXT     NOT NULL UNIQUE,
    provider          TEXT     NOT NULL,
    mode              TEXT     NOT NULL CHECK(mode IN ('api','account')),
    base_url          TEXT     NOT NULL DEFAULT '',
    model             TEXT     NOT NULL DEFAULT '',
    api_key_enc       TEXT     NOT NULL DEFAULT '',
    oauth_access_enc  TEXT     NOT NULL DEFAULT '',
    oauth_refresh_enc TEXT     NOT NULL DEFAULT '',
    oauth_expires_at  DATETIME,
    price_in          REAL     NOT NULL DEFAULT 0.0,
    price_out         REAL     NOT NULL DEFAULT 0.0,
    timeout_seconds   INTEGER  NOT NULL DEFAULT 120,
    created_at        DATETIME NOT NULL,
    updated_at        DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS ai_config_assignments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    config_id    INTEGER NOT NULL REFERENCES ai_configs(id) ON DELETE CASCADE,
    scope        TEXT    NOT NULL CHECK(scope IN ('global','client','project')),
    client_name  TEXT    NOT NULL DEFAULT '',
    project_ref  TEXT    NOT NULL DEFAULT '',
    UNIQUE(scope, client_name, project_ref)
);

CREATE INDEX IF NOT EXISTS idx_ai_config_assignments_lookup
    ON ai_config_assignments(client_name, project_ref);
`

// migration006 adds a configurable max output tokens per AI config.
// 0 means "use the provider/transport default".
const migration006 = `
ALTER TABLE ai_configs ADD COLUMN max_tokens INTEGER NOT NULL DEFAULT 0;
`

// migration007 adds the review profiles library and scope-based assignments,
// mirroring the prompts schema. Seeds a global "Diff" profile (legacy behavior).
const migration007 = `
CREATE TABLE IF NOT EXISTS review_profiles (
    id              INTEGER  PRIMARY KEY AUTOINCREMENT,
    name            TEXT     NOT NULL UNIQUE,
    include_related INTEGER  NOT NULL DEFAULT 0,
    agentic         INTEGER  NOT NULL DEFAULT 0,
    execution       INTEGER  NOT NULL DEFAULT 0,
    max_iterations  INTEGER  NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS review_profile_assignments (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id   INTEGER NOT NULL REFERENCES review_profiles(id) ON DELETE CASCADE,
    scope        TEXT    NOT NULL CHECK(scope IN ('global','client','project')),
    client_name  TEXT    NOT NULL DEFAULT '',
    project_ref  TEXT    NOT NULL DEFAULT '',
    UNIQUE(scope, client_name, project_ref)
);

CREATE INDEX IF NOT EXISTS idx_review_profile_assignments_lookup
    ON review_profile_assignments(client_name, project_ref);

INSERT INTO review_profiles(name, include_related, agentic, execution, max_iterations, created_at, updated_at)
VALUES('Diff', 0, 0, 0, 0, datetime('now'), datetime('now'));

INSERT INTO review_profile_assignments(profile_id, scope, client_name, project_ref)
VALUES((SELECT id FROM review_profiles WHERE name='Diff'), 'global', '', '');
`
