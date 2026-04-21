# Manual de Uso — Morpho CLI

## 1) Visão geral

O Morpho é uma CLI para criar, configurar e executar agentes de IA para tarefas de desenvolvimento.

Principais recursos:

- Gerenciamento de agentes (`create`, `edit`, `list`, `show`, `set-model`)
- Execução com fila em memória e retry automático
- Presets prontos para uso
- Consulta de modelos Gemini
- Outputs por agente em `.morpho/outputs/<agente>/`
- Modo iterativo com navegação visual
- Configuração local segura da API key fora do repositório

---

## 2) Pré-requisitos

- Go instalado
- Chave da API Gemini

Compilar binário local:

```bash
go run . install
```

Isso gera o binário em `bin/morpho`.

---

## 3) Formas de execução

### Rodando sem binário (desenvolvimento)

```bash
go run . help
```

### Rodando com binário

```bash
./bin/morpho help
```

> Dica: com `go run`, use sempre `go run . <comando>`.

---

## 4) Configuração da API key

### Opção A: variável de ambiente

```bash
export GEMINI_API_KEY="SUA_CHAVE"
```

### Opção B: salvar via CLI (recomendado)

```bash
./bin/morpho config set-api-key "SUA_CHAVE"
./bin/morpho config where
```

A chave fica em arquivo local do usuário (fora do repositório).

---

## 5) Fluxo rápido (começar em 1 minuto)

```bash
./bin/morpho presets init
./bin/morpho agent list
./bin/morpho model list
./bin/morpho agent run backend-go "Crie um plano para autenticação JWT"
```

Para visualizar os presets predefinidos:

```bash
./bin/morpho presets list
```

Para aplicar os presets com um modelo específico:

```bash
./bin/morpho presets init --model gemini-2.5-flash
```

---

## 6) Comandos de agentes

### Listar agentes

```bash
./bin/morpho agent list
```

### Ver detalhes de um agente

```bash
./bin/morpho agent show backend-go
```

### Criar agente

```bash
./bin/morpho agent create arquiteto-go \
  --description "Arquitetura backend" \
  --prompt "Você é um arquiteto Go. Seja objetivo." \
  --model "gemini-2.5-flash" \
  --tags "go,backend,arquitetura"
```

### Editar agente

```bash
./bin/morpho agent edit arquiteto-go --prompt "Novo prompt" --tags "go,api"
```

### Trocar modelo do agente

```bash
./bin/morpho agent set-model arquiteto-go gemini-2.5-flash
```

---

## 7) Execução de agentes

### Execução normal

```bash
./bin/morpho agent run backend-go "Implemente um plano de testes para API"
```

### Execução com edição de arquivos por agente

1. Configure política modular:

```bash
./bin/morpho config edit set-mode review
./bin/morpho config edit set-paths "internal,cmd"
./bin/morpho config edit show
```

2. Execute com edição habilitada:

```bash
./bin/morpho agent run backend-go "Implementar feature Z" --edit
```

3. Overrides por execução:

```bash
./bin/morpho agent run backend-go "Refatorar parser" \
  --edit \
  --edit-mode review \
  --edit-paths "internal/agentkit" \
  --edit-max 4
```

No modo `review`, cada edição é aprovada arquivo por arquivo.
No modo `auto`, o agente aplica todas as edições válidas diretamente.
Backups são salvos em `.morpho/backups/<timestamp>/` quando um arquivo existente é alterado.

### Execução offline (mock)

```bash
./bin/morpho agent run backend-go "Refatorar camada de serviço" --mock
```

### Flags úteis de execução

- `--timeout`: timeout por tentativa
- `--queue-retries`: retries para timeout/rate-limit
- `--queue-delay`: delay base entre retries
- `--context-entries`: quantos outputs de outros agentes usar como contexto
- `--context-chars`: limite de caracteres do contexto compartilhado
- `--no-shared-context`: desativa contexto de outros agentes
- `--rag`: ativa recuperação semântica (RAG) nesta execução
- `--no-rag`: desativa RAG nesta execução
- `--rag-topk`: override da quantidade de memórias recuperadas
- `--rag-min-score`: override da similaridade mínima da recuperação

Exemplo:

```bash
./bin/morpho agent run backend-go "Planejar migração" --queue-retries 5 --queue-delay 3s
```

---

## 8) Outputs por agente

Cada execução salva um arquivo Markdown em:

```text
.morpho/outputs/<agente>/
```

### Listar outputs

```bash
./bin/morpho agent output list
./bin/morpho agent output list backend-go
```

### Visualizar output específico

```bash
./bin/morpho agent output show backend-go 20260419-101500-task.md
```

Esses outputs podem ser usados como contexto por outros agentes nas próximas execuções.

### Memória semântica por agente

Cada agente mantém uma base própria em:

```text
.morpho/memory/<agente>/knowledge.db
```

Comandos:

```bash
./bin/morpho agent memory status backend-go
./bin/morpho agent memory search backend-go "erro 500 login"
./bin/morpho agent memory reindex backend-go
./bin/morpho agent memory prune backend-go --max-docs 200

# hardening
./bin/morpho config memory show
./bin/morpho config memory set-read-policy self
./bin/morpho config memory set-ttl-hours 720
```

---

## 9) Modelos Gemini

### Listar modelos

```bash
./bin/morpho model list
```

### Listar todos (incluindo sem `generateContent`)

```bash
./bin/morpho model list --all
```

---

## 10) Modo iterativo (TUI)

Iniciar:

```bash
./bin/morpho interactive
```

Controles:

- `↑/↓`: navegar
- `Enter`: executar ação
- `Esc`: voltar
- `b`: voltar ao menu (na tela de resultado)
- `r`: reexecutar último comando
- `q`: sair

---

## 11) Troubleshooting

### Erro: `package help is not in GOROOT`

Você executou `go run help`.

Use:

```bash
go run . help
```

### Erro: `missing go.sum entry`

Rode:

```bash
go mod tidy
```

### Erro de API key ausente

Configure com:

```bash
./bin/morpho config set-api-key "SUA_CHAVE"
```

ou exporte `GEMINI_API_KEY`.

---

## 12) Comando de instalação

Para gerar/atualizar o binário local:

```bash
go run . install
```

Saída padrão: `bin/morpho`
