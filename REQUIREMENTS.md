# Engenharia de Requisitos - Morpho

## 1. Requisitos Funcionais (RF)

| ID       | Requisito                              | Descrição                                                                                                                     |
| :------- | :------------------------------------- | :---------------------------------------------------------------------------------------------------------------------------- |
| **RF01** | Interface de Linha de Comando (CLI)    | O sistema deve prover comandos para gerenciar agentes e executar tarefas de desenvolvimento.                                  |
| **RF02** | Criar Agente via CLI                   | O usuário deve conseguir criar um agente com `nome`, `prompt`, `modelo`, `descrição` e `tags`.                                |
| **RF03** | Editar Agente via CLI                  | O usuário deve conseguir editar as especificações de um agente existente sem alterar código-fonte.                            |
| **RF04** | Listar e Exibir Agentes                | O sistema deve permitir listar todos os agentes e visualizar detalhes de um agente específico.                                |
| **RF05** | Persistência de Especificações         | As configurações dos agentes devem ser armazenadas localmente em `.morpho/agents/*.json`.                                     |
| **RF06** | Execução de Agente                     | O usuário deve executar um agente com uma tarefa textual e receber resposta da API Gemini.                                    |
| **RF07** | Modo Offline de Teste                  | O sistema deve oferecer execução `--mock` para testes sem chamada real à API.                                                 |
| **RF08** | Presets de Agentes                     | O sistema deve disponibilizar um comando para criar agentes pré-selecionados prontos para uso.                                |
| **RF09** | Configuração por Variáveis de Ambiente | A autenticação da API Gemini deve usar `GEMINI_API_KEY` e permitir modelo padrão configurável.                                |
| **RF10** | Prefixo de Mensagens do Morpho         | Toda mensagem exibida ao usuário deve iniciar com `🦋:`.                                                                      |
| **RF11** | Fila em Memória para Execução          | As execuções de agentes devem entrar em fila em memória e rodar em ordem de chegada.                                          |
| **RF12** | Retry Automático em Timeout            | Em timeout/rate-limit transitório, o sistema deve aguardar e tentar novamente mantendo a ordem.                               |
| **RF13** | Listagem de Modelos Gemini             | O sistema deve listar via CLI os modelos disponíveis na API Gemini.                                                           |
| **RF14** | Configurar Modelo por Agente via CLI   | O usuário deve definir/alterar o modelo de cada agente por comandos CLI.                                                      |
| **RF15** | Diretório de Output por Agente         | O sistema deve manter a estrutura `.morpho/outputs/<agente>/` para armazenar os outputs de conclusão de cada agente.          |
| **RF16** | Consulta Cruzada de Outputs            | Os outputs salvos por um agente devem poder ser consultados pelos demais agentes como contexto de trabalho.                   |
| **RF17** | Gestão de Artefatos de Output via CLI  | O sistema deve permitir criar/organizar/listar arquivos e pastas de output por agente via CLI.                                |
| **RF20** | Configuração de API Key via CLI        | O sistema deve permitir definir a chave da API Gemini via CLI, armazenando-a fora do repositório em arquivo local do usuário. |
| **RF21** | Modo Iterativo Contínuo                | O sistema deve oferecer um modo iterativo que inicia e permanece em execução até o usuário encerrar.                          |
| **RF22** | Navegação por Botões na CLI            | O modo iterativo deve permitir navegação visual por botões para executar funcionalidades do CLI.                              |
| **RF23** | Instalador Automático do Binário       | O sistema deve disponibilizar um instalador automático que gere o binário `morpho` dentro da pasta `bin/` do projeto.         |
| **RF24** | Edição de Arquivos por Agentes         | O sistema deve permitir que agentes proponham e apliquem alterações em arquivos do workspace com base em uma tarefa textual.  |
| **RF25** | Controle Modular de Edição             | O sistema deve permitir configurar política de edição (`off`, `review`, `auto`) e caminhos permitidos para aplicação segura.  |
| **RF26** | Aprovação por Edição                   | No modo `review`, o sistema deve solicitar aprovação arquivo por arquivo antes de aplicar qualquer modificação.               |

## 2. Requisitos Não Funcionais (RNF)

| ID        | Requisito                    | Descrição                                                                                                                                                          |
| :-------- | :--------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **RNF01** | Baixo Custo Operacional      | O projeto deve priorizar modelos Gemini de baixo custo para manter uso acessível.                                                                                  |
| **RNF02** | Facilidade de Configuração   | Criar e editar agentes deve exigir o mínimo de passos e conhecimento técnico.                                                                                      |
| **RNF03** | Portabilidade                | O sistema deve ser compilável como binário em Go para Linux/Unix.                                                                                                  |
| **RNF04** | Resiliência                  | O sistema deve tratar falhas de API, timeout e arquivos inválidos de forma clara.                                                                                  |
| **RNF05** | Extensibilidade              | Deve ser simples adicionar novos tipos de agentes e novos presets no futuro.                                                                                       |
| **RNF06** | Experiência do Desenvolvedor | A CLI deve apresentar mensagens objetivas, padronizadas e fáceis de interpretar.                                                                                   |
| **RNF07** | Qualidade Visual da CLI      | Todos os outputs da CLI devem ter alto nível de acabamento visual (cores, hierarquia, espaçamento, ícones e estados), em padrão “premium” similar a CLIs modernas. |
