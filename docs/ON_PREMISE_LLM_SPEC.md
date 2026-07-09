# Radiant Norma — On-Premise LLM Hardware Specification

**Data:** 2026-07-09
**Versão:** 1.1
**Status:** Proposta de arquitetura de hardware para implantação on-premise
**Autor:** ZCode (validado contra código real do projeto em 2026-07-09)

---

## 1. Contexto

O Radiant Norma é uma plataforma de inteligência regulatória para Instituições Financeiras brasileiras que gera, valida e gerencia documentos CADOCs BACEN. O código atual (v3.36.0) já inclui módulos que cobrem grande parte do workflow aqui proposto:

| Módulo existente | Papel no workflow LLM |
|------------------|----------------------|
| `internal/radar` | Monitor de URLs BACEN com SHA-256 diff (já detecta mudanças) |
| `internal/ingest` | Conectores: Manual, File, API, DB, **MCP** |
| `internal/generator` | Geração de XML dos CADOCs a partir de CanonicalDocument |
| `internal/sta` | Submissão ao BACEN via STA |
| `internal/audit` | Trilha tamper-evident para LGPD/SOC2 |
| `cmd/worker` | Processamento assíncrono (batch) |

A proposta de implantação on-premise adiciona:
- **LLM local** (Ollama) para processamento de normativos e assistentes regulatórios
- **Parser de documentos** (Docling) para XLSX/PDF/DOCX
- **RAG** (ChromaDB + embeddings BGE-M3) para busca semântica em base de normativos
- **Criptografia at-rest** para dados sensíveis e histórico de CADOCs

### Workflow Principal (proposto)

```
┌─────────────────────────────────────────────────────────────┐
│  1. Monitoramento — já existe em internal/radar             │
│     • Cron em cmd/radar faz fetch de URLs BACEN             │
│     • SHA-256 diff → tabela radar_alerts                    │
│     • Estende-se com: scraping de novas resoluções/cartas   │
└────────────────────────┬────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  2. Processamento LLM (batch, pode rodar à noite)           │
│     • Docling parseia PDF/DOCX/XLSX                        │
│     • LLM extrai pontos-chave, impacto nos CADOCs          │
│     • Classifica tipo: resolução/carta/instrução            │
│     • Gera resumo executivo em pt-BR                       │
│     • Classifica severidade: BAIXA/MÉDIA/ALTA/CRÍTICA      │
└────────────────────────┬────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  3. Geração Automática de CADOCs                           │
│     • Conector ingest (File/DB/MCP) → CanonicalDocument     │
│     • internal/generator produz XML validado                │
│     • internal/audit registra hash para auditoria          │
└────────────────────────┬────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  4. Ações Automáticas                                       │
│     • Impacto BAIXO/MÉDIO → email para cliente             │
│     • Impacto ALTA/CRÍTICA → email cliente + ticket interno │
│     • Requer DESENVOLVIMENTO → email cliente + ticket dev   │
└─────────────────────────────────────────────────────────────┘
```

### O que NÃO precisa de LLM (regras determinísticas em Go)

| Tarefa | Implementação nativa |
|--------|----------------------|
| Varrer DB do cliente e extrair dados | `internal/ingest` (adapter DB) |
| Preencher campos fixos de CADOC | `internal/generator/gen3040` etc. |
| Gerar XML validado | Templates Go + `internal/audit` |
| Submeter ao BACEN | `internal/sta` |
| Disparar cron de monitoramento | `cmd/radar` (já existe) |
| Enviar email + ticket | Worker + SMTP/IMAP + sistema de tickets |

### O que o LLM faz (componentes que só IA resolve)

| Tarefa | Modelo/Abordagem |
|--------|------------------|
| Resumir normativo denso em linguagem simples | Qwen2.5-14B (extractive summarization) |
| Explicar regra de validação específica | RAG: BGE-M3 + Qwen2.5-7B |
| Traduzir mudança regulatória para o cliente | Qwen2.5-14B (abstraction) |
| Sugerir campos de CADOC a partir de dado não-estruturado | Qwen2.5-14B + few-shot examples |
| Classificar severidade do impacto | Qwen2.5-7B (classification fine-tuned) |

---

## 2. Hardware — 3 Opções de Setup (20-30k BRL)

> **Nota:** Preços estimados para julho 2026, considerando mercado brasileiro e importação. Hardware empresarial brasileiro inclui impostos (ICMS, IPI) sobre componentes importados.

### Setup A: "Competente" — ~20-22 mil BRL

| Componente | Sugestão | Preço estimado |
|------------|----------|----------------|
| GPU | RTX 4080 Super 16GB | ~7.500 |
| CPU | AMD Ryzen 7 7700 (8c/16t) | ~2.200 |
| RAM | 64GB DDR5 | ~1.800 |
| SSD | 2TB NVMe (sistema + DB) + 4TB NVMe (dados + modelos) | ~2.200 |
| Motherboard | B650 + case ATX + PSU 850W | ~2.000 |
| Cooler | Air cooler premium | ~500 |
| **Total** | | **~16.200** |

**Recursos alocados (16GB VRAM / 64GB RAM):**
- Plataforma (Go API + Next.js + worker + radar + Postgres + Redis): ~12GB RAM
- LLM runtime (Ollama + KV cache + overhead): ~3GB VRAM
- **Reservado para o modelo: ~13GB VRAM**

**Modelos suportados (em VRAM):**
- Qwen2.5-14B-Instruct (Q4_K_M, ~10GB) — modelo principal
- Mistral-7B-Instruct (Q8, ~8GB) — para tarefas simples
- Phi-3-mini-4k (Q8, ~4GB) — para classificação de severidade

**Limitação:** 14B é o teto. Modelos 32B+ não cabem com qualidade aceitável.

---

### Setup B: "Recomendado" — ~25-28 mil BRL ⭐

| Componente | Sugestão | Preço estimado |
|------------|----------|----------------|
| GPU | **RTX 4090 24GB** | ~13.000 |
| CPU | AMD Ryzen 9 7900X (12c/24t) | ~3.000 |
| RAM | 128GB DDR5 | ~3.500 |
| SSD | 2TB NVMe (sistema) + 4TB NVMe (dados) + 2TB NVMe (modelos) | ~3.000 |
| Motherboard | X670E + case ATX + PSU 1000W | ~2.800 |
| Cooler | AIO 360mm | ~1.200 |
| **Total** | | **~26.500** |

**Recursos alocados (24GB VRAM / 128GB RAM):**
- Plataforma: ~15GB RAM
- LLM runtime (Ollama + overhead): ~4GB VRAM
- **Reservado para o modelo: ~20GB VRAM + 32GB RAM (offload)**

**Modelos suportados (em VRAM):**
- Qwen2.5-14B-Instruct (Q8, ~16GB) — qualidade alta
- Qwen2.5-7B-Instruct (Q8, ~8GB) — para classificação/rotulagem
- Mistral-7B-Instruct (Q8)
- Llama 3.1-8B-Instruct (Q8)
- Qwen2.5-32B-Instruct (Q4_K_M, ~22GB) — com offload parcial

**Com offload CPU→RAM:**
- Qwen2.5-72B-Instruct (Q4_K_M, ~45GB) — análise profunda noturna de normativos críticos

**Vantagem:** 128GB RAM permite rodar 32B com offload leve em VRAM, ou 72B em batch noturno (mais lento, mas possível). CPU 12-core acelera o pipeline de parseamento Docling em paralelo.

---

### Setup C: "Workstation" — ~28-32 mil BRL

| Componente | Sugestão | Preço estimado |
|------------|----------|----------------|
| GPU | **RTX 4090 24GB** | ~13.000 |
| CPU | AMD Ryzen 9 7950X (16c/32t) | ~4.000 |
| RAM | 192GB DDR5 | ~5.000 |
| SSD | 2x2TB NVMe RAID0 (sistema) + 4TB NVMe (dados) + 2TB NVMe (modelos) | ~4.000 |
| Motherboard | X670E E-ATX + full tower + PSU 1200W | ~3.500 |
| Cooler | AIO 420mm | ~1.500 |
| **Total** | | **~31.000** |

**Recursos alocados (24GB VRAM / 192GB RAM):**
- Plataforma: ~20GB RAM
- LLM runtime: ~4GB VRAM
- **Reservado para o modelo: ~32GB RAM offload + 20GB VRAM**

**Modelos suportados:**
- Qwen2.5-72B-Instruct (Q4_K_M) — análise regulatória profunda
- Qwen2.5-32B-Instruct (Q8) — rápido
- Qwen2.5-14B-Instruct (Q8) — instantaneous
- Llama 3.1-70B-Instruct (Q4)
- **Múltiplos modelos simultâneos** (7B + 14B em paralelo)

**Use case:** Multi-tenant pesado, fine-tuning LoRA de modelos com dados do cliente (QLoRA com 24GB VRAM + 192GB RAM é factível), múltiplos clientes simultâneos.

---

## 3. Comparativo

| | Setup A | Setup B ⭐ | Setup C |
|------------|---------|---------|---------|
| **Preço** | ~16-22k | ~25-28k | ~28-32k |
| **GPU** | 4080 Super 16GB | 4090 24GB | 4090 24GB |
| **RAM** | 64GB | 128GB | 192GB |
| **Storage** | 6TB | 8TB | 10TB |
| **14B Q8** | ❌ (só Q4) | ✅ VRAM (ótimo) | ✅ VRAM (instant) |
| **14B Q4** | ✅ VRAM (bom) | ✅ | ✅ |
| **32B Q4** | ❌ | ✅ VRAM+RAM | ✅ VRAM+RAM |
| **72B Q4** | ❌ | ✅ RAM (lento) | ✅ RAM (aceitável) |
| **Multi-modelo** | ❌ | 1x (com swap) | 2x+ |
| **Multi-tenant** | Leve | Moderado | Pesado |
| **Fine-tuning LoRA** | ❌ | ⚠️ Marginal | ✅ QLoRA factível |

---

## 4. Recomendação

**Setup B (~25-28k)** — sweet spot entre custo e capacidade.

```
RTX 4090 24GB + Ryzen 9 7900X + 128GB RAM + 8TB storage
Preço estimado: ~26.500 BRL
```

**Por quê 4090 e não 4080 Super?** A diferença de preço (~+5.500 BRL) compra 50% mais VRAM (24GB vs 16GB) e 50% mais bandwidth de memória. Para LLM inference, bandwidth > FLOPS. Isso significa rodar 14B em Q8 (não Q4), com qualidade visivelmente melhor.

**Por quê 128GB RAM?** Permite offload parcial de 32B ou 72B em batch noturno, sem precisar trocar modelo durante o dia. Também acomoda múltiplas instâncias de ChromaDB e embeddings para RAG.

**Se orçamento apertado:** Setup A funciona, mas o LLM fica limitado a 14B em Q4 — qualidade marginal em análise de normativos complexos.

**Se for multi-tenant ou multi-cliente:** Setup C é o caminho. Sem ele, fine-tuning de modelos com dados do cliente não é viável.

---

## 5. Stack de Software (alinhado com o código real)

```
┌─────────────────────────────────────────────────────────────┐
│  Ubuntu Server 24.04 LTS (ou Rocky Linux 9)                 │
├─────────────────────────────────────────────────────────────┤
│  Containers / Processes:                                     │
│                                                             │
│  ├── cmd/api              (Go 1.25 — chi router)            │
│  ├── cmd/worker           (Go — processamento assíncrono)   │
│  ├── cmd/radar            (Go — monitor BACEN, cron)        │
│  ├── cmd/generator-server (Go — serviço de geração XML)     │
│  ├── Next.js 14           (App Router + Tailwind + shadcn)  │
│  ├── PostgreSQL 16        (opcional — SQLite por padrão)    │
│  ├── Redis                (cache + sessões)                 │
│  ├── Ollama               (LLM runtime — v0.5+)             │
│  ├── Docling               (parseamento PDF/DOCX/XLSX)       │
│  ├── ChromaDB             (RAG — embeddings)                │
│  └── Cron + scripts       (automação)                       │
└─────────────────────────────────────────────────────────────┘
```

### Modelo de deployment Ollama ↔ Backend Go

```
┌──────────────────┐         ┌─────────────────────┐
│  radiant-norma   │  HTTP   │  Ollama             │
│  (Go backend)    │ ◄─────► │  (LLM runtime)      │
│                  │         │                     │
│  internal/llm    │         │  :11434             │
│  (novo package)  │         │  - qwen2.5:14b      │
└──────────────────┘         │  - bge-m3           │
                             └─────────────────────┘
                                      │
                                      ▼
                             ┌─────────────────────┐
                             │  CUDA / ROCm / CPU  │
                             │  (RTX 4090)         │
                             └─────────────────────┘
```

O backend Go se comunica com Ollama via HTTP/REST local. Vantagens:
- Ollama gerencia o modelo em VRAM (carregamento, KV cache, batching)
- Go não precisa de bindings C++ para llama.cpp
- Trocar de modelo é uma chamada de API

### Modelos Recomendados

| Modelo | Tamanho | Uso | Quando carregar |
|--------|---------|-----|-----------------|
| **qwen2.5:14b-instruct-q8_0** | ~16GB VRAM | Resumo, explicação, abstração | Sempre em VRAM (Setup B/C) |
| **qwen2.5:7b-instruct-q8_0** | ~8GB VRAM | Classificação, extração, validação | Carregado sob demanda |
| **bge-m3** | ~620MB VRAM | Embeddings para RAG | Sempre (residente) |
| **qwen2.5:32b-instruct-q4_K_M** | ~22GB VRAM + 16GB RAM | Análise profunda de normativos críticos | Batch noturno |

**Por que Qwen2.5 e não Llama/Mistral:** Benchmarks independentes (MLPerf, lmsys arena 2025) mostram que Qwen2.5 tem a melhor performance em pt-BR entre modelos open-weight ≤ 32B, especialmente em tarefas estruturadas (JSON output, listas, classificações) — exatamente o que precisamos para extrair impacto regulatório.

---

## 6. Segurança (on-premise implica em regulação bancária)

### 6.1 Criptografia

| Camada | Tecnologia | Onde |
|--------|-----------|------|
| Discos | LUKS (Linux Unified Key Setup) | SSDs inteiros |
| Volumes | dm-crypt com AES-256-XTS | `/data`, `/var/log/radiant` |
| Backup | age ou gpg cifrado | Off-site (S3 compatível, opcional) |
| TLS interno | mTLS com self-signed CA | API ↔ Ollama ↔ Postgres |

### 6.2 Rede

- **Air-gap por padrão** — máquina on-premise não tem rota default para internet
- **Whitelist egress** — apenas BACEN (www.bcb.gov.br, sta.bcb.gov.br), SMTP do cliente, registry de updates
- **Firewall**: `ufw` ou `nftables` com política default deny
- **SSH**: apenas chave pública, fail2ban, porta não-padrão

### 6.3 Auditoria

- `internal/audit` já implementa log tamper-evident (chain hash)
- Logs centralizados em `/var/log/radiant` (volume cifrado)
- Retenção mínima: 5 anos (Resolução BCB 4.658/2018 — Segurança Cibernética)

### 6.4 LGPD / SOC2

- Dados pessoais cifrados at-rest
- Anonimização de CPF/CNPJ em logs de LLM
- Direito ao esquecimento: comando `radiant-cli forget --cpf <hash>`

---

## 7. Custos Operacionais

| Item | Estimativa |
|------|------------|
| Energia (750W sistema, 24x7) | ~250-400 BRL/mês |
| Energia (750W sistema, 8h/dia) | ~100-150 BRL/mês |
| Manutenção (UPS, replacement parts) | ~500-1000 BRL/ano |
| Suplemento de equipe (sysadmin part-time) | ~3.000-5.000 BRL/mês |

**Custo operacional mínimo** — sem cloud costs, sem licenças de LLM (open-weight), sem assinatura de API.

**Comparação com cloud:**
- AWS Bedrock (Claude 3.5 Sonnet): ~R$ 0,08/1k tokens input — inviável para processamento de normativos densos (50+ páginas = 100k+ tokens)
- Self-hosted Qwen2.5-14B: ~R$ 0,001/1k tokens (custo de energia amortizado)
- **Economia estimada: 50-200x em workload de normativos**

---

## 8. Limitações Conhecidas e Mitigações

| Limitação | Impacto | Mitigação |
|-----------|---------|-----------|
| Qwen2.5-14B em pt-BR | Bom, mas não excelente em jargão jurídico específico BACEN | RAG com base de normativos (ChromaDB + BGE-M3) fornece contexto |
| Latência de inferência CPU-only | 5-30s para 14B em CPU puro | GPU dedicada mitiga (1-3s) — Setup B/C |
| Atualização de modelos | LLM não atualiza sozinho | Pipeline de fine-tuning LoRA (Setup C) ou refresh mensal |
| Rate limit de fetching BACEN | throttling | Cron com backoff exponencial em `cmd/radar` (já implementado) |
| Falta de GPU AMD/Intel Arc | Drivers CUDA only | Setup A/B/C usam NVIDIA exclusivamente |

---

## 9. Roadmap de Implementação

### Fase 1 (Sprint 58-60): Fundação
- [ ] Adicionar `internal/llm` package com cliente Ollama
- [ ] Implementar Docling como serviço (sidecar Python)
- [ ] Configurar ChromaDB + BGE-M3 para RAG
- [ ] Estender `internal/radar` para detectar novos tipos de normativo (resoluções, cartas, instruções)

### Fase 2 (Sprint 61-63): Processamento
- [ ] Worker de normativos: parse → LLM → classificar → email
- [ ] Templates de email para cliente + interno
- [ ] Sistema de tickets interno (CLI + web)

### Fase 3 (Sprint 64-66): Produção
- [ ] Fine-tuning LoRA com dados do cliente (opcional, Setup C)
- [ ] Dashboard de monitoramento de inferência
- [ ] Alertas de saúde (VRAM, temperatura, latência)

---

## 10. Changelog

| Versão | Data | Descrição |
|--------|------|----------|
| 1.0 | 2026-07-09 | Versão inicial — 3 setups de hardware |
| 1.1 | 2026-07-09 | Validação contra código real: adicionado `internal/radar` (já existe), `internal/ingest` (5 adapters incluindo MCP), `cmd/worker` e `cmd/generator-server`; corrigido Setup A (Q4 vs Q8); adicionada seção de Segurança, Limitações e Roadmap; adicionada arquitetura Ollama ↔ Go backend |
