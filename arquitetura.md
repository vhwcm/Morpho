# Arquitetura do Sistema — Morpho

Este documento descreve a arquitetura de pastas/arquivos e o fluxo principal do sistema.

## Visão geral

O Morpho é uma CLI em Go orientada a agentes de desenvolvimento.

- Entrada principal via comandos Cobra
- Camada de domínio para agentes e execução
- Integração com Gemini (real e mock)
- Persistência local em `.morpho/` e configuração em diretório do usuário
- Interface de terminal com UX aprimorada

## Estrutura de pastas

```text
.
├── main.go
├── go.mod
├── cmd/
│   ├── root.go
│   ├── diagnostic.go
│   ├── interactive.go
│   └── install.go
├── internal/
│   ├── agentkit/
│   │   ├── types.go
│   │   ├── storage.go
│   │   ├── presets.go
│   │   ├── runner.go
│   │   ├── queue.go
│   │   ├── outputs.go
│   │   └── editing.go
│   ├── agents/
│   │   ├── orchestrator.go
│   │   ├── plan.go
│   │   ├── log.go
│   │   ├── metrics.go
│   │   └── solution.go
│   ├── gemini/
│   │   ├── client.go
│   │   └── mock.go
│   ├── config/
│   │   ├── env.go
│   │   ├── store.go
│   │   └── editing.go
│   ├── memory/
│   │   ├── store.go
│   │   ├── retriever.go
│   │   ├── ingest.go
│   │   └── extract.go
│   ├── ui/
│       ├── help.go
│       ├── messages.go
│       └── morphoLogo.txt
├── .morpho/
│   ├── agents/
│   ├── outputs/
│   └── backups/
└── docs operacionais
    ├── README.md
    ├── MANUAL.md
    ├── REQUIREMENTS.md
    └── KANBAN.md
```

## Responsabilidades por camada

### 1) CLI (`cmd/`)

- Define comandos, subcomandos e flags
- Orquestra chamadas para `agentkit`, `config` e `gemini`
- Apresenta mensagens no terminal via `ui`

### 2) Núcleo de agentes (`internal/agentkit/`)

- `storage.go`: CRUD das specs dos agentes em `.morpho/agents/*.json`
- `presets.go`: catálogo de agentes predefinidos
- `runner.go`: montagem de prompt e execução
- `queue.go`: fila e retries para robustez
- `outputs.go`: grava/leitura de outputs em `.morpho/outputs/`
- `editing.go`: plano de edição, validação de path e aplicação com backup

### 2.1) Memória semântica (`internal/memory/`)

- Banco SQLite por agente em `.morpho/memory/<agente>/knowledge.db`
- Ingestão automática de conhecimento após execução
- Retenção real por TTL (`ttl_hours`) com expiração de documentos
- Recuperação híbrida (score semântico + lexical)
- Política explícita de leitura: `self` (somente o próprio agente) ou `shared` (pode ler memória de outros agentes)
- Fallback lexical quando embeddings não estiverem disponíveis

### 3) Diagnóstico multiagente (`internal/agents/`)

- Pipeline especializado (plan/log/metrics/solution)
- Execução concorrente com `errgroup`
- Consolidação de relatório de diagnóstico

### 4) Configuração (`internal/config/`)

- `store.go`: persistência de configuração local (`config.json` no diretório do usuário)
- `env.go`: resolução de precedência (env var > arquivo > default)
- `editing.go`: política modular de edição por agentes
  - modos: `off`, `review`, `auto`
  - allowlist de caminhos permitidos

### 5) IA (`internal/gemini/`)

- `client.go`: cliente HTTP para Gemini
- `mock.go`: cliente mock para execução offline/testes

### 6) UI de terminal (`internal/ui/`)

- componentes de saída (header, painel, tabela, alertas)
- ajuda customizada e visual da CLI

## Fluxo principal (execução de agente)

1. Usuário chama `agent run`.
2. CLI carrega `Spec` do agente.
3. Resolve configuração (`API key`, modelo, política de edição).
4. Recupera contexto RAG da memória do agente (quando habilitado).
5. Executa tarefa via fila com retry (`agentkit.RunQueued`).
6. Salva output em `.morpho/outputs/<agente>/`.
7. Ingere task+resultado na memória do agente para próximas execuções.
8. Opcionalmente, gera plano de edição e aplica arquivos conforme política:
   - `off`: não aplica
   - `review`: aprova arquivo por arquivo
   - `auto`: aplica automaticamente
9. Em arquivos alterados, cria backup em `.morpho/backups/<timestamp>/`.

## Armazenamento e segurança

- **Specs dos agentes**: versionáveis em `.morpho/agents/`
- **Outputs de execução**: `.morpho/outputs/`
- **Backups de edição**: `.morpho/backups/`
- **Config sensível** (ex.: API key): fora do repositório, no diretório de configuração do usuário

## Princípios arquiteturais aplicados

- Separação clara entre CLI, domínio e infraestrutura
- Composição modular para facilitar extensão
- Modo seguro por padrão (`edit mode = off`)
- Observabilidade via outputs persistidos
- Testabilidade com mock de IA e cobertura de componentes críticos
