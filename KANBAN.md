# Kanban Gopher

## 🔴 Backlog / Todo

- [ ] **RF18**: Exportar/importar agentes via pacote (`.zip` ou `.yaml`).
- [ ] **RF19**: Suporte a parâmetros de contexto por agente (stack, repo, idioma).
- [ ] **RNF06**: Adicionar testes automatizados para comandos da CLI.

## 🟡 In Progress

- [ ] **RF06**: Evoluir execução de agente para contexto multi-arquivo.
- [ ] **RF08**: Expandir presets com mais perfis especializados.

## 🟢 Done

- [x] Pivot do produto para sistema agêntico de desenvolvimento.
- [x] **RF01**: CLI base para gerenciamento de agentes.
- [x] **RF02**: Criação de agente via CLI.
- [x] **RF03**: Edição de agente via CLI.
- [x] **RF04**: Listagem e visualização de agentes.
- [x] **RF05**: Persistência local em `.gopher/agents/*.json`.
- [x] **RF07**: Execução em modo `--mock`.
- [x] **RF08**: Comando de presets iniciais.
- [x] **RF09**: Integração base com Gemini via `GEMINI_API_KEY`.
- [x] **RF10**: Toda mensagem do Gopher inicia com o prefixo `🐹:`.
- [x] **RF11**: Fila em memória para execução ordenada de agentes.
- [x] **RF12**: Retry automático em timeout/rate-limit sem quebrar a ordem da fila.
- [x] **RF13**: Listagem de modelos disponíveis do Gemini via CLI.
- [x] **RF14**: Configuração de modelo por agente via CLI.
- [x] **RF15**: Diretório de output por agente em `.gopher/outputs/<agente>/`.
- [x] **RF16**: Consulta de outputs entre agentes para contexto compartilhado.
- [x] **RF17**: Gestão de outputs via CLI (`agent output list/show`).
- [x] **RF20**: Configuração da API key via CLI com armazenamento fora do repositório.
- [x] **RF21**: Modo iterativo contínuo (`gopher interactive`).
- [x] **RF22**: Navegação por botões no modo iterativo.
- [x] **RF23**: Instalador automático (`gopher install`) para gerar `bin/gopher`.
- [x] **RNF04**: Tratamento inicial de timeout/rate-limit com backoff e retries.
- [x] **RNF07**: Output visual premium na CLI (tema, componentes e estados).
