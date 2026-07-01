# AuxiTalk Core

> Runtime modular para assistência em conversas.  
> Modular runtime for conversation assistance.

---

## Idiomas / Languages

- [Português (PT-BR)](#português-pt-br)
- [English (EN)](#english-en)

---

# Português (PT-BR)

## Visão geral

AuxiTalk é um runtime modular para auxiliar usuários durante conversas digitais. O objetivo do projeto é observar conversas, construir contexto, sugerir respostas, decidir quando responder ou esperar, e coordenar ações aprovadas pelo usuário por meio de plugins extensíveis.

O core é escrito em Go e foi projetado para ser leve, robusto, testável e fácil de entender por humanos e agentes de IA.

O AuxiTalk **não é um plugin de WhatsApp**, **não é um app de IA específico** e **não é uma interface gráfica fixa**. Ele é a base de orquestração que permite criar integrações para diferentes canais, modelos de IA, memórias, painéis, automações e ferramentas.

## Objetivo do projeto

O objetivo principal é criar um runtime que possa:

- observar conversas em diferentes canais;
- manter sessões e histórico normalizado;
- gerar contexto compacto para IA;
- chamar plugins de IA, memória, UI, ferramentas e integrações;
- sugerir respostas no tom e estilo configurados;
- recomendar quando responder, esperar, ignorar ou pedir mais contexto;
- permitir ações com controle do usuário;
- aprender com feedback, edições e rejeições futuras;
- ser expandido por plugins feitos pela comunidade.

## Princípios de arquitetura

- O core deve ser independente de apps específicos.
- WhatsApp, Telegram, Discord, navegador, OCR, áudio, LLMs e overlays devem ser plugins.
- Plugins podem ser escritos em qualquer linguagem.
- Comunicação inicial via JSON-RPC 2.0 sobre stdio.
- O core deve funcionar sem UI.
- Ações sensíveis devem passar por controle de segurança.
- Documentação deve acompanhar cada módulo e decisão.
- A base deve ser simples o suficiente para agentes de IA entenderem e modificarem.

## Arquitetura geral

```txt
Input Plugin
  -> Event Bus
  -> Session Manager
  -> Context Builder
  -> Action Gate / Policy
  -> Capability Router
  -> AI / Memory / Tool Plugins
  -> Suggestion Event
  -> UI / Output Plugin
  -> User Feedback
  -> Memory Update
```

## Componentes principais

### Core runtime

O core coordena o ciclo de vida do sistema e conecta os módulos internos.

Local principal:

```txt
internal/runtime
```

### Event Bus

Responsável por publicar e distribuir eventos internos.

Recursos atuais:

- pub/sub por tipo de evento;
- wildcard subscriber;
- timeout por handler;
- histórico em memória opcional;
- validação de eventos.

Local:

```txt
internal/events
```

### Configuração

Sistema de configuração com defaults e suporte a arquivo JSON.

Modos disponíveis:

```txt
dev      modo permissivo para testes rápidos
local    modo seguro local com proteção para ações sensíveis
strict   modo restrito com validações mais fortes
```

Local:

```txt
internal/config
configs/auxitalk.example.json
```

### Tipos centrais

Tipos compartilhados entre core e plugins:

- `Event`
- `Session`
- `Message`
- `Suggestion`
- `ActionRequest`
- `PluginManifest`
- `Capability`

Local:

```txt
pkg/types
```

### Protocolo JSON-RPC

Tipos públicos para comunicação com plugins:

- `Request`
- `Response`
- `Error`

Local:

```txt
pkg/protocol
internal/rpc
```

### Plugin Manifest e Registry

Carrega, valida e registra manifests de plugins.

Local:

```txt
internal/plugins
```

### Plugin Supervisor

Responsável pelo ciclo de vida de processos de plugins.

Estado atual:

- start/stop de processo;
- captura de stdin/stdout/stderr;
- base para health checks;
- restart backoff;
- limite de restarts.

Local:

```txt
internal/plugins/supervisor.go
```

### Capability Router

Registra e roteia capabilities fornecidas por plugins.

Exemplos de capabilities:

```txt
conversation.observe
message.send
ai.complete
memory.query
memory.write
ui.suggestion.display
```

Local:

```txt
internal/capabilities
```

### Session Manager e Context Builder

Gerencia conversas normalizadas e constrói contexto textual compacto.

Local:

```txt
internal/sessions
internal/context
```

### Action Gate

Controla ações conforme risco e modo de runtime.

Riscos:

```txt
low
medium
high
```

Comportamento atual:

```txt
dev      permite tudo
local    low=allow, medium=confirm, high=deny
strict   low=confirm, medium/high=deny
```

Local:

```txt
internal/actions
```

## Sistema de plugins

Plugins são processos externos que se comunicam com o core via JSON-RPC sobre stdio.

Isso permite criar plugins em:

- Go;
- TypeScript/Node.js;
- Python;
- Rust;
- qualquer linguagem que leia stdin e escreva stdout.

### Regras básicas

- `stdout` é reservado para mensagens JSON-RPC.
- `stderr` é reservado para logs.
- cada mensagem JSON-RPC deve ocupar uma linha.
- payloads possuem limite configurável.
- chamadas respeitam timeout/cancelamento de contexto.
- ações sensíveis devem usar o fluxo de action request.

### Exemplo de manifesto

```json
{
  "id": "mock-ai",
  "name": "Mock AI",
  "version": "0.1.0",
  "runtime": "node",
  "entry": "index.js",
  "kind": "ai",
  "permissions": [],
  "capabilities": [
    {
      "name": "ai.complete"
    }
  ]
}
```

## Plugins de exemplo

Este repositório contém exemplos mínimos:

```txt
plugins/examples/mock-input
plugins/examples/mock-ai
plugins/examples/console-output
plugins/examples/file-memory
```

Esses plugins servem para validar o fluxo do runtime antes de criar integrações reais.

## Estrutura do repositório

```txt
cmd/                 executáveis: auxitalkd e auxitalkctl
configs/             arquivos de configuração exemplo
docs/                arquitetura, roadmap, decisões e guia de plugins
examples/            exemplos de fluxo
internal/            pacotes internos do core
pkg/                 tipos públicos e protocolo
plugins/             plugins de exemplo
FINAL_STATUS.md      relatório da fundação inicial
```

## Como executar

### Requisitos

- Go instalado.
- Node.js será necessário futuramente para alguns plugins de exemplo.

### Rodar daemon

```sh
go run ./cmd/auxitalkd
```

### Rodar com arquivo de configuração

```sh
go run ./cmd/auxitalkd --config configs/auxitalk.example.json
```

### Rodar CLI placeholder

```sh
go run ./cmd/auxitalkctl
```

### Rodar testes

```sh
go test ./...
```

## Status atual

A fundação inicial do AuxiTalk Core foi concluída.

Já existem módulos para:

- configuração;
- eventos;
- tipos centrais;
- protocolo JSON-RPC;
- manifesto/registry de plugins;
- supervisor de plugins;
- roteamento de capabilities;
- sessões e contexto;
- action gate;
- plugins mock;
- exemplo do primeiro loop completo.

## Próximos passos

As próximas etapas naturais são:

1. integrar o supervisor com chamadas JSON-RPC reais;
2. tornar o loop de exemplo executável via `auxitalkd`;
3. criar o primeiro plugin real de entrada, como WhatsApp Web;
4. criar plugin real de IA, como OpenAI, Anthropic ou modelo local;
5. criar overlay/CLI para exibir sugestões;
6. separar plugins oficiais em repositórios próprios da organização;
7. criar template oficial para plugins da comunidade.

## Organização dos repositórios

A direção planejada é manter este repositório como o core:

```txt
AuxiTalk/auxitalk
```

E criar plugins oficiais em repositórios separados:

```txt
AuxiTalk/plugin-template
AuxiTalk/plugin-whatsapp-web
AuxiTalk/plugin-openai
AuxiTalk/plugin-sqlite-memory
AuxiTalk/plugin-desktop-overlay
AuxiTalk/plugins
```

## Documentação importante

- `docs/architecture/core.md`
- `docs/architecture/core-types.md`
- `docs/architecture/configuration.md`
- `docs/architecture/event-bus.md`
- `docs/architecture/capability-router.md`
- `docs/architecture/session-context.md`
- `docs/architecture/action-gate.md`
- `docs/plugins/authoring-guide.md`
- `docs/plugins/protocol-draft.md`
- `docs/plugins/system.md`
- `docs/roadmap/initial-implementation-plan.md`
- `docs/decisions/0001-go-core-jsonrpc-plugins.md`

---

# English (EN)

## Overview

AuxiTalk is a modular runtime for assisting users during digital conversations. The goal is to observe conversations, build context, suggest replies, decide when to respond or wait, and coordinate user-approved actions through extensible plugins.

The core is written in Go and designed to be lightweight, robust, testable, and easy for both humans and AI agents to understand.

AuxiTalk is **not a WhatsApp plugin**, **not a specific AI app**, and **not a fixed graphical interface**. It is the orchestration foundation that enables integrations for different channels, AI models, memory backends, control panels, automations, and tools.

## Project goal

The main goal is to create a runtime that can:

- observe conversations across different channels;
- keep normalized sessions and message history;
- build compact context for AI;
- call AI, memory, UI, tool, and integration plugins;
- suggest replies using configured tone and style;
- recommend when to respond, wait, ignore, or ask for more context;
- keep user control over sensitive actions;
- learn from future edits, feedback, and rejections;
- be extended by community-built plugins.

## Architecture principles

- The core must stay independent from specific apps.
- WhatsApp, Telegram, Discord, browser, OCR, audio, LLMs, and overlays should be plugins.
- Plugins can be written in any language.
- Initial communication uses JSON-RPC 2.0 over stdio.
- The core should work without a UI.
- Sensitive actions must go through safety controls.
- Documentation must evolve with every module and decision.
- The codebase should be simple enough for AI agents to understand and modify.

## General architecture

```txt
Input Plugin
  -> Event Bus
  -> Session Manager
  -> Context Builder
  -> Action Gate / Policy
  -> Capability Router
  -> AI / Memory / Tool Plugins
  -> Suggestion Event
  -> UI / Output Plugin
  -> User Feedback
  -> Memory Update
```

## Main components

### Core runtime

Coordinates the system lifecycle and connects internal modules.

Location:

```txt
internal/runtime
```

### Event Bus

Publishes and distributes internal events.

Current features:

- typed pub/sub;
- wildcard subscriber;
- per-handler timeout;
- optional in-memory history;
- event validation.

Location:

```txt
internal/events
```

### Configuration

Configuration system with defaults and JSON file support.

Available modes:

```txt
dev      permissive mode for quick tests
local    safer local mode for sensitive actions
strict   restricted mode with stronger validation
```

Location:

```txt
internal/config
configs/auxitalk.example.json
```

### Core types

Shared types between the core and plugins:

- `Event`
- `Session`
- `Message`
- `Suggestion`
- `ActionRequest`
- `PluginManifest`
- `Capability`

Location:

```txt
pkg/types
```

### JSON-RPC protocol

Public message types for plugin communication:

- `Request`
- `Response`
- `Error`

Location:

```txt
pkg/protocol
internal/rpc
```

### Plugin Manifest and Registry

Loads, validates, and registers plugin manifests.

Location:

```txt
internal/plugins
```

### Plugin Supervisor

Manages plugin process lifecycle.

Current state:

- process start/stop;
- stdin/stdout/stderr capture;
- health check foundation;
- restart backoff;
- restart limit.

Location:

```txt
internal/plugins/supervisor.go
```

### Capability Router

Registers and routes capabilities provided by plugins.

Capability examples:

```txt
conversation.observe
message.send
ai.complete
memory.query
memory.write
ui.suggestion.display
```

Location:

```txt
internal/capabilities
```

### Session Manager and Context Builder

Manage normalized conversations and build compact textual context.

Location:

```txt
internal/sessions
internal/context
```

### Action Gate

Controls actions based on runtime mode and risk level.

Risk levels:

```txt
low
medium
high
```

Current behavior:

```txt
dev      allow everything
local    low=allow, medium=confirm, high=deny
strict   low=confirm, medium/high=deny
```

Location:

```txt
internal/actions
```

## Plugin system

Plugins are external processes communicating with the core through JSON-RPC over stdio.

This allows plugins to be written in:

- Go;
- TypeScript/Node.js;
- Python;
- Rust;
- any language that can read stdin and write stdout.

### Basic rules

- `stdout` is reserved for JSON-RPC messages.
- `stderr` is reserved for logs.
- each JSON-RPC message must be one line.
- payload size is configurable and limited.
- calls respect context timeout/cancellation.
- sensitive actions must use the action request flow.

### Manifest example

```json
{
  "id": "mock-ai",
  "name": "Mock AI",
  "version": "0.1.0",
  "runtime": "node",
  "entry": "index.js",
  "kind": "ai",
  "permissions": [],
  "capabilities": [
    {
      "name": "ai.complete"
    }
  ]
}
```

## Example plugins

This repository includes minimal examples:

```txt
plugins/examples/mock-input
plugins/examples/mock-ai
plugins/examples/console-output
plugins/examples/file-memory
```

These plugins are used to validate the runtime flow before real integrations are built.

## Repository structure

```txt
cmd/                 executables: auxitalkd and auxitalkctl
configs/             example configuration files
docs/                architecture, roadmap, decisions, and plugin guide
examples/            flow examples
internal/            private core packages
pkg/                 public protocol and types packages
plugins/             example plugins
FINAL_STATUS.md      initial foundation status report
```

## How to run

### Requirements

- Go installed.
- Node.js will be needed later for some example plugins.

### Run daemon

```sh
go run ./cmd/auxitalkd
```

### Run with config file

```sh
go run ./cmd/auxitalkd --config configs/auxitalk.example.json
```

### Run CLI placeholder

```sh
go run ./cmd/auxitalkctl
```

### Run tests

```sh
go test ./...
```

## Current status

The initial AuxiTalk Core foundation is complete.

Current modules include:

- configuration;
- events;
- core types;
- JSON-RPC protocol;
- plugin manifest/registry;
- plugin supervisor;
- capability routing;
- sessions and context;
- action gate;
- mock plugins;
- first full loop example.

## Next steps

Natural next steps are:

1. integrate the supervisor with real JSON-RPC calls;
2. make the loop example executable through `auxitalkd`;
3. create the first real input plugin, such as WhatsApp Web;
4. create the first real AI plugin, such as OpenAI, Anthropic, or local model;
5. create an overlay/CLI for suggestions;
6. split official plugins into dedicated organization repositories;
7. create an official community plugin template.

## Repository organization

The planned direction is to keep this repository as the core:

```txt
AuxiTalk/auxitalk
```

And create official plugins in separate repositories:

```txt
AuxiTalk/plugin-template
AuxiTalk/plugin-whatsapp-web
AuxiTalk/plugin-openai
AuxiTalk/plugin-sqlite-memory
AuxiTalk/plugin-desktop-overlay
AuxiTalk/plugins
```

## Important documentation

- `docs/architecture/core.md`
- `docs/architecture/core-types.md`
- `docs/architecture/configuration.md`
- `docs/architecture/event-bus.md`
- `docs/architecture/capability-router.md`
- `docs/architecture/session-context.md`
- `docs/architecture/action-gate.md`
- `docs/plugins/authoring-guide.md`
- `docs/plugins/protocol-draft.md`
- `docs/plugins/system.md`
- `docs/roadmap/initial-implementation-plan.md`
- `docs/decisions/0001-go-core-jsonrpc-plugins.md`
