# Radiant Norma — On-Premise LLM Hardware Specification

**Data:** 2026-07-09
**Versão:** 1.0
**Status:** Proposta de arquitetura de hardware para implantação on-premise

---

## 1. Contexto

O Radiant Norma é uma plataforma de inteligência regulatória para Instituições Financeiras brasileiras que gera, valida e gerencia documentos CADOCs BACEN.

A proposta de implantação on-premise inclui:
- Plataforma completa (Go API + Next.js + PostgreSQL + Redis)
- LLM para processamento de normativos e assistentes regulatórios
- Parser de documentos (Docling)
- RAG para busca semântica em base de normativos
- Armazenamento cifrado de dados e histórico de CADOCs

### Workflow Principal

```
┌─────────────────────────────────────────────────────────────┐
│  1. Monitoramento (cron)                                    │
│     • Busca novos normativos BACEN                          │
│     • Detecta mudanças via diff/change detection            │
└────────────────────────┬────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  2. Processamento LLM (batch, pode rodar à noite)           │
│     • Parsear documento (Docling)                           │
│     • Extrair pontos-chave, impacto nos CADOCs              │
│     • Classificar tipo: resolução/carta/instrução           │
│     • Gerar resumo executivo em pt-BR                      │
│     • Classificar severidade: BAIXA/MÉDIA/ALTA/CRÍTICA     │
└────────────────────────┬────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  3. Geração Automática de CADOCs                           │
│     • Dados do cliente → preenchimento automático           │
│     • Geração de XML validado                              │
│     • Armazenamento histórico (auditoria)                   │
└────────────────────────┬────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│  4. Ações Automáticas                                       │
│     • Impacto BAIXO/MÉDIO → email para cliente             │
│     • Impacto ALTA/CRÍTICA → email cliente + ticket interno │
│     • Requer DESENVOLVIMENTO → email cliente + ticket dev   │
└─────────────────────────────────────────────────────────────┘
```

### Tarefas do LLM

| Tarefa | Perfil | Precisa de LLM? |
|--------|--------|-----------------|
| Varrer DB, preencher campos fixos | Determinística | Não — Go/SQL |
| Gerar XML CADOC | Structured output | Não — templates Go |
| "Traduzir" dados cliente→campo | Mapeamento | Não — mapping tables |
| Resumir normativos | Extractive summarization | Sim — LLM leve |
| Explicar regras de validação | RAG | Sim — embedding + LLM |
| Buscar cambios em normativos | Information retrieval | Sim — diff semântico |
| Traduzir mudanças | Abstraction | Sim — LLM pequeno |
| Criar tickets automaticamente | Template filling | Não — regras + LLM simples |

---

## 2. Hardware — 3 Opções de Setup (20-30k BRL)

> **Nota:** Preços estimados para julho 2026, considerando mercado brasileiro e importação.

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

**Recursos alocados:**
- Go API + Next.js + PostgreSQL + Redis + ChromaDB: ~12GB RAM
- Reservado para LLM: ~10GB VRAM

**Modelos suportados (em VRAM):**
- Qwen2.5-14B (Q4_K_M)
- Mistral-7B (Q8)
- Phi-3-mini (Q8)

**Limitação:** Não suporta 32B ou 72B com desempenho adequado.

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

**Recursos alocados:**
- Go API + Next.js + PostgreSQL + Redis + ChromaDB: ~15GB RAM
- Reservado para LLM: ~24GB VRAM + 32GB RAM (offload)

**Modelos suportados (em VRAM):**
- Qwen2.5-14B (Q8) — principal
- Qwen2.5-32B (Q4_K_M) — com offload para RAM
- Mistral-7B (Q8)
- Llama 3.1-8B (Q8)

**Offload para RAM:**
- Qwen2.5-72B (Q4) — análise profunda de normativos críticos

**Vantagem:** 128GB RAM permite rodar 32B na VRAM enquanto 14B faz tarefas simples — ou 72B em batch noturno com offload.

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

**Recursos alocados:**
- Go API + Next.js + PostgreSQL + Redis + ChromaDB: ~20GB RAM
- Reservado para LLM: ~32GB RAM offload + 24GB VRAM

**Modelos suportados:**
- Qwen2.5-72B (Q4_K_M)
- Qwen2.5-32B (Q8)
- Qwen2.5-14B (Q8)
- Llama 3.1-70B (Q4)
- Múltiplos modelos simultâneos

**Use case:** Multi-tenant pesado, fine-tuning, múltiplos clientes simultâneos.

---

## 3. Comparativo

| | Setup A | Setup B ⭐ | Setup C |
|------------|---------|---------|---------|
| **Preço** | ~20-22k | ~25-28k | ~28-32k |
| **GPU** | 4080 Super 16GB | 4090 24GB | 4090 24GB |
| **RAM** | 64GB | 128GB | 192GB |
| **Storage** | 6TB | 8TB | 10TB |
| **14B Q8** | VRAM (bom) | VRAM (ótimo) | VRAM (instant) |
| **32B Q4** | ❌ | VRAM+RAM | VRAM+RAM |
| **72B Q4** | ❌ | RAM | RAM |
| **Multi-modelo** | ❌ | 1x | 2x+ |
| **Multi-tenant** | Leve | Moderado | Pesado |

---

## 4. Recomendação

**Setup B (~25-28k)** — sweet spot entre custo e capacidade.

```
RTX 4090 24GB + Ryzen 9 7900X + 128GB RAM + 8TB storage
Preço estimado: ~26.500 BRL
```

**Se orçamento apertado:** Setup A funciona, mas fica limitado a 14B como modelo principal.

---

## 5. Stack de Software

```
┌─────────────────────────────────────────────┐
│  Ubuntu Server 24.04 LTS                     │
├─────────────────────────────────────────────┤
│  ├── radiant-norma (Go API)                 │
│  ├── Next.js (frontend)                    │
│  ├── PostgreSQL 16 (dados + CADOCs)         │
│  ├── Redis (cache + sessions)               │
│  ├── Ollama (LLM runtime)                  │
│  ├── Docling (parseamento PDF/DOCX/XLSX)    │
│  ├── ChromaDB (RAG embeddings)             │
│  └── Cron + scripts (automação)            │
└─────────────────────────────────────────────┘
```

### Modelos Recomendados

| Modelo | Tamanho | Uso | Prioridade |
|--------|---------|-----|------------|
| **Qwen2.5-14B-Instruct** | 14B | Tarefas principais (chat, resumo, explicação) | PRIMARY |
| **Qwen2.5-7B-Instruct** | 7B | Tarefas simples, speed-critical | SECONDARY |
| **BGE-M3-Embedding** | ~560MB | RAG de normativos | ALWAYS |

**Razão Qwen2.5:** Versão em pt-BR muito superior a Llama/Mistral, treinado em código/matemática — relevante para interpretar normativos.

---

## 6. Custos Operacionais

| Item | Estimativa |
|------|------------|
| Energia (750W sistema, 8h/dia) | ~150-250 BRL/mês |
| Manutenção (UPS, replacement) | ~500-1000 BRL/ano |

Custo operacional mínimo. Sem cloud costs, sem licenças.

---

## 7. Futuro — Possíveis Upgrades

- **Fine-tuning** de modelos com dados do cliente (requer Setup C)
- **Multi-tenant** com isolamento de dados (Setup B ou C)
- **GPU adicional** para múltiplos modelos simultâneos (Setup B/C com PSU 1000W+ suporta)
- **NVMe adicional** para base de normativos crescentes

---

## Changelog

| Versão | Data | Descrição |
|--------|------|----------|
| 1.0 | 2026-07-09 | Versão inicial — 3 setups de hardware |
