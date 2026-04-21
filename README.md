# 🦋 Morpho: Sistema Agêntico para Desenvolvimento

O **Morpho** é uma CLI em Go para criar, editar e executar agentes de IA voltados a desenvolvimento de software.
<img width="1626" height="862" alt="image" src="https://github.com/user-attachments/assets/be688956-a00f-4250-baba-a3957a85d98a" />

## 🚀 Diferenciais

- **Baixo custo**: usa a API do Gemini para viabilizar um fluxo acessível.
- **Configuração simples**: cada agente pode ser criado e editado via CLI.
- **Agentes prontos**: presets para começar rápido (backend, frontend, review, QA, DevOps).


## 🧠 Competências técnicas demonstradas

- **Engenharia de CLI em Go (Cobra)**: comandos compostos, flags, validação de entrada e UX de terminal.
- **Integração robusta com API externa (Gemini)**: cliente HTTP com tratamento de erros por status code.
- **Resiliência operacional**: fila em memória com execução ordenada, retries e backoff para timeout/rate-limit.
- **Persistência orientada a produto**: specs de agentes em JSON e outputs versionáveis por agente.
- **Segurança de segredos**: API key salva fora do repositório (`os.UserConfigDir`) com permissões restritas.
- **UX premium de terminal**: saída visual com estilo, tabelas, painéis e modo iterativo com navegação por botões.

## 🧱 Como funciona

Cada agente possui uma especificação local em JSON dentro de `.morpho/agents`, com:

- `name`
- `description`
- `system_prompt`
- `model`
- `tags`

Você pode versionar essas especificações no Git e ajustar o comportamento sem alterar código Go.


## Arquitetura
<img width="930" height="747" alt="image" src="https://github.com/user-attachments/assets/46a88688-6f78-439a-89d4-6519e184865c" />

## 📦 Instalação

Pré-requisitos:

- Go 1.18+
- `GEMINI_API_KEY` configurada

Build:

```bash
go build -o morpho
```

Instalador automático do binário local:

```bash
go run . install
# gera: bin/morpho
```

Instalar também em `~/.local/bin` para usar `morpho` diretamente:

```bash
go run . install --link-local-bin
```

Configuração:

```bash
export GEMINI_API_KEY="sua_chave"
```

Ou configure via CLI (fora do repositório):

```bash
./morpho config set-api-key "sua_chave"
./morpho config where
```

## ⌨️ Comandos principais

Ao usar `go run`, sempre execute com `.` (ponto), por exemplo:

```bash
go run . help
go run . model list
go run . agent list
```

Inicializar agentes pré-selecionados:

```bash
./morpho presets init
```

Listar presets disponíveis e status:

```bash
./morpho presets list
```

Inicializar presets com modelo específico:

```bash
./morpho presets init --model gemini-2.5-flash
```

Listar agentes:

```bash
./morpho agent list
```

Iniciar modo iterativo com navegação por botões:

```bash
./morpho interactive
```

Criar agente customizado:

```bash
./morpho agent create arquiteto-go \
  --description "Arquitetura e backend Go" \
  --prompt "Você é um arquiteto Go. Priorize simplicidade, teste e performance." \
  --model "gemini-2.5-flash" \
  --tags "go,backend,arquitetura"
```

Editar agente:

```bash
./morpho agent edit arquiteto-go --prompt "Novo prompt" --tags "go,api"
```

Configurar modelo de um agente via CLI:

```bash
./morpho agent set-model arquiteto-go gemini-2.5-flash
```

Configurar modelo padrão global via CLI:

```bash
./morpho model set gemini-2.5-flash
```

Listar modelos disponíveis do Gemini:

```bash
./morpho model list
```

Executar agente:

```bash
./morpho agent run backend-go "Criar plano para implementar autenticação JWT com testes"
```

Configurar política de edição por agentes (modular):

```bash
./morpho config edit set-mode review
./morpho config edit set-paths "internal,cmd"
./morpho config edit show
```

Executar agente com edição de arquivos:

```bash
# usa política salva em config (review|auto|off)
./morpho agent run backend-go "Implementar validação no comando X" --edit

# override só nesta execução
./morpho agent run backend-go "Ajustar parser" --edit --edit-mode review --edit-paths "internal/agentkit" --edit-max 3

# aprova tudo no modo review
./morpho agent run backend-go "Refatorar função Y" --edit --edit-mode review --yes
```

Cada execução salva um output em:

```text
.morpho/outputs/<agente>/
```

Memória semântica por agente (RAG):

- Banco por agente em `.morpho/memory/<agente>/knowledge.db`
- Ingestão automática após `agent run`
- Recuperação semântica opcional por execução (`--rag`)

Comandos de memória:

```bash
./morpho agent memory status backend-go
./morpho agent memory search backend-go "jwt timeout"
./morpho agent memory reindex backend-go
./morpho agent memory prune backend-go --max-docs 200

# configuração de hardening
./morpho config memory show
./morpho config memory set-read-policy self
./morpho config memory set-ttl-hours 720
```

Execução com RAG:

```bash
./morpho agent run backend-go "resolver timeout no login" --rag --rag-topk 6 --rag-min-score 0.25
```

Listar outputs (todos os agentes):

```bash
./morpho agent output list
```

Listar outputs de um agente específico:

```bash
./morpho agent output list backend-go
```

Visualizar um output específico:

```bash
./morpho agent output show backend-go 20260419-101500-plano-auth-jwt.md
```

Modo offline (mock):

```bash
./morpho agent run backend-go "Refatorar camada de serviço" --mock
```

---

Projeto em evolução com foco em DX, automação e criação fácil de agentes especializados.

## 🛠️ Troubleshooting rápido

Se aparecer erro de `missing go.sum entry`, rode:

```bash
go mod tidy
```

Se você rodar `go run help` ou `go run agent`, vai falhar porque o Go tenta executar um pacote com esse nome.
Use sempre `go run . <comando>`.
