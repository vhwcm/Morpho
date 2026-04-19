# 🐹 Gopher: Sistema Agêntico para Desenvolvimento

O **Gopher** é uma CLI em Go para criar, editar e executar agentes de IA voltados a desenvolvimento de software.

## 🚀 Diferenciais

- **Baixo custo**: usa a API do Gemini para viabilizar um fluxo acessível.
- **Configuração simples**: cada agente pode ser criado e editado via CLI.
- **Agentes prontos**: presets para começar rápido (backend, frontend, review, QA, DevOps).
- **Mensagens padronizadas**: toda saída inicia com `🐹:`.

## 🧠 Competências técnicas demonstradas

- **Engenharia de CLI em Go (Cobra)**: comandos compostos, flags, validação de entrada e UX de terminal.
- **Integração robusta com API externa (Gemini)**: cliente HTTP com tratamento de erros por status code.
- **Resiliência operacional**: fila em memória com execução ordenada, retries e backoff para timeout/rate-limit.
- **Persistência orientada a produto**: specs de agentes em JSON e outputs versionáveis por agente.
- **Segurança de segredos**: API key salva fora do repositório (`os.UserConfigDir`) com permissões restritas.
- **UX premium de terminal**: saída visual com estilo, tabelas, painéis e modo iterativo com navegação por botões.

## 🧱 Como funciona

Cada agente possui uma especificação local em JSON dentro de `.gopher/agents`, com:

- `name`
- `description`
- `system_prompt`
- `model`
- `tags`

Você pode versionar essas especificações no Git e ajustar o comportamento sem alterar código Go.

## 📦 Instalação

Pré-requisitos:

- Go 1.18+
- `GEMINI_API_KEY` configurada

Build:

```bash
go build -o gopher
```

Instalador automático do binário local:

```bash
go run . install
# gera: bin/gopher
```

Instalar também em `~/.local/bin` para usar `gopher` diretamente:

```bash
go run . install --link-local-bin
```

Configuração:

```bash
export GEMINI_API_KEY="sua_chave"
```

Ou configure via CLI (fora do repositório):

```bash
./gopher config set-api-key "sua_chave"
./gopher config where
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
./gopher presets init
```

Listar agentes:

```bash
./gopher agent list
```

Iniciar modo iterativo com navegação por botões:

```bash
./gopher interactive
```

Criar agente customizado:

```bash
./gopher agent create arquiteto-go \
  --description "Arquitetura e backend Go" \
  --prompt "Você é um arquiteto Go. Priorize simplicidade, teste e performance." \
  --model "gemini-2.0-flash" \
  --tags "go,backend,arquitetura"
```

Editar agente:

```bash
./gopher agent edit arquiteto-go --prompt "Novo prompt" --tags "go,api"
```

Configurar modelo de um agente via CLI:

```bash
./gopher agent set-model arquiteto-go gemini-2.0-flash
```

Configurar modelo padrão global via CLI:

```bash
./gopher model set gemini-2.0-flash
```

Listar modelos disponíveis do Gemini:

```bash
./gopher model list
```

Executar agente:

```bash
./gopher agent run backend-go "Criar plano para implementar autenticação JWT com testes"
```

Cada execução salva um output em:

```text
.gopher/outputs/<agente>/
```

Listar outputs (todos os agentes):

```bash
./gopher agent output list
```

Listar outputs de um agente específico:

```bash
./gopher agent output list backend-go
```

Visualizar um output específico:

```bash
./gopher agent output show backend-go 20260419-101500-plano-auth-jwt.md
```

Modo offline (mock):

```bash
./gopher agent run backend-go "Refatorar camada de serviço" --mock
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
