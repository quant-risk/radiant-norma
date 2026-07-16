# Radiant Norma — LGPD Compliance Package

**Versão:** 1.0
**Data:** 2026-07-16
**Classificação:** Confidencial — Clientes e parceiros

---

## Parte I — Política de Privacidade

### 1.1 Identificação do Controlador

**Razão Social:** Radiant Tecnologia e Sistemas LTDA
**CNPJ:** XX.XXX.XXX/XXXX-XX
**Endereço:** [Endereço completo]
**DPO:** [Nome do DPO] — dpo@radiant.digital
**Website:** https://radiant.digital

### 1.2 Dados Pessoais Tratados

O Radiant Norma trata os seguintes dados pessoais no contexto da prestação
de serviços de inteligência regulatória para Instituições Financeiras (IFs):

#### 1.2.1 Dados de nossos clientes (IFs — Controladores)

| Categoria | Dados | Base Legal | Finalidade |
|---|---|---|---|
| **Cadastrais da IF** | CNPJ, nome, segmento, tipo | Art. 7º IX (execução de contrato) | Onboarding; gestão contratual |
| **Técnicos da IF** | Nome, email, telefone de contato técnico | Art. 7º II (consentimento) ou IX | Suporte; comunicação técnica |
| **Operacionais** | Volume de submissões, CADOCs gerados, regras desabilitadas | Art. 7º IX | Prestação do serviço; billing |

#### 1.2.2 Dados dos usuários dos nossos clientes

O Radiant Norma **não trata dados pessoais de usuários finais das IFs**.
Os dados processados são exclusivamente dados regulatórios (CADOCs) que
contêm identificadores de clientes das IFs (ex.: SCR — Cadastro de Crédito).
O **controlador** desses dados é a **IF cliente**, não a Radiant.

#### 1.2.3 Dados dos colaboradores Radiant

| Categoria | Dados | Base Legal | Finalidade |
|---|---|---|---|
| **Cadastrais** | Nome, CPF, endereço, contatos | Art. 7º II (contrato de trabalho) | Gestão de RH; obrigações trabalhistas |
| **Acesso a sistemas** | Email corporativo, logs de acesso | Art. 7º IX (legítimo interesse — segurança) | Controle de acesso; segurança |

### 1.3 Base Legal para Tratamento

| Finalidade | Base Legal (LGPD) | Observações |
|---|---|---|
| Prestação de serviço de automação regulatória | Art. 7º IX — execução de contrato | Dados operacionais da IF |
| Suporte técnico | Art. 7º IX — execução de contrato | Dados de contato técnico |
| Marketing (se aplicável) | Art. 7º I — consentimento | Opt-in; pode ser revogado |
| Obrigações legais (BACEN, Receita Federal) | Art. 7º II — obrigação legal | Regulamentação BACEN |
| Segurança e prevenção de fraudes | Art. 7º IX — legítimo interesse | Logging de acesso; audit trail |
| Melhoria de produto (analytics) | Art. 7º IX — legítimo interesse | Dados agregados e anonimizados |

### 1.4 Compartilhamento de Dados

| Destinatário | Dados Compartilhados | Finalidade | Garantia |
|---|---|---|---|
| **BACEN** | CADOC submissions (XML) | Obrigação regulatória da IF | Envio direto (STA); Radiant é apenas processador |
| **AWS / Provedor de cloud** | Dados de aplicação (DB, backups) | Hospedagem; backup | DPA com AWS; dados criptografados |
| **Autoridades públicas** | Dados regulatórios | Cumprimento de ordem judicial/administrativa | Apenas mediante ordem legal válida |

**A Radiant não vende dados pessoais.**

### 1.5 Transferência Internacional

Dados podem ser transferidos para:

| País/Região | Destinatário | Salvaguarda |
|---|---|---|
| **Estados Unidos** | AWS (us-east-1) | SCCs; AWS Data Processing Agreement |
| **Europa** | OpenAI (se LLM = GPT-4) | Standard Contractual Clauses; adequacy decision |

Aguardando avaliação de impacto para transfers cross-border: **previsto Q3 2026**.

### 1.6 Retenção de Dados

| Categoria | Prazo de Retenção | Destruição |
|---|---|---|
| CADOC submissions (XML) | 7 anos | Conforme BACKUP_DR_POLICY.md |
| Audit logs | 7 anos | Conforme BACKUP_DR_POLICY.md |
| Dados de contato técnico | Durante vigência do contrato + 5 anos | Exclusão segura |
| Logs de acesso | 90 dias (operacional); 7 anos (audit) | Exclusão segura |

### 1.7 Direitos do Titular

Os titulares dos dados (nossos clientes — pessoas jurídicas — e colaboradores)
possuem os seguintes direitos:

1. **Confirmação e acesso** (Art. 18) — confirmar se há tratamento e acessar dados
2. **Correção** (Art. 18) — corrigir dados incompletos ou desatualizados
3. **Anonimização, bloqueio ou eliminação** (Art. 18) — dados desnecessários
4. **Portabilidade** (Art. 18) — receber dados em formato interoperável
5. **Eliminação** (Art. 18) — mediante consentimento, quando aplicável
6. **Informação sobre compartilhamento** (Art. 18) — lista de destinatários
7. **Revogação do consentimento** (Art. 8) — a qualquer momento

**Canal para exercer direitos:** privacidade@radiant.digital

---

## Parte II — Data Processing Agreement (DPA)

### 2.1 Escopo e Definições

Este DPA é incorporado ao contrato de prestação de serviços entre a Radiant
("Operador") e o cliente ("Controlador").

**Definições (conforme LGPD Art. 5):**

- **Dado pessoal:** informações relativas a pessoa identificada ou identificável
- **Tratamento:** qualquer operação realizada com dados pessoais
- **Controlador:** pessoa natural ou jurídica que decide sobre o tratamento
- **Operador:** pessoa natural ou jurídica que trata dados por ordem do controlador
- **Incidente de segurança:** qualquer acesso, aquisição, uso ou vazamento
  não autorizado que resulte em risco aos direitos dos titulares

### 2.2 Objeto do Tratamento

O Operador trata dados pessoais **exclusivamente** para a prestação do
serviço de inteligência regulatória, conforme especificado no contrato
e na documentação técnica do Radiant Norma.

**Operações de tratamento realizadas:**

- Geração de documentos CADOC (XML) — dados regulatórios
- Validação semântica e XSD/L1-L4
- Transmissão ao STA do BACEN
- Armazenamento em base de dados PostgreSQL (criptografado)
- Logging de auditoria (hash chain)
- Webhooks de notificação de eventos

### 2.3 Obrigações do Operador (Radiant)

1. **Tratar dados apenas conforme instruções documentadas** do Controlador,
   exceto quando exigido por lei.
2. **Não compartilhar dados** com terceiros sem consentimento prévio por escrito
   do Controlador, exceto para cumprimento de obrigação legal.
3. **Garantir confidencialidade** de colaboradores com acesso aos dados.
4. **Implementar medidas técnicas** de segurança (criptografia, RLS, audit log,
   rate limiting — ver SOC2_READINESS.md).
5. **Notificar o Controlador** sobre incidentes de segurança em até **72 horas**
   após conhecimento (LGPD Art. 48).
6. **Auxiliar o Controlador** no atendimento a requests de titulares
   (acesso, correção, eliminação) — prazo de **15 dias úteis**.
7. **Eliminar ou devolver dados** ao final do contrato, conforme orientação
   do Controlador, dentro de **30 dias**.
8. **Manter registro das operações** de tratamento (Art. 37 — mandatory).

### 2.4 Obrigações do Controlador (Cliente)

1. Fornecer ao Operador apenas dados necessários para a prestação do serviço.
2. Garantir que a base legal para tratamento existe e é válida.
3. Responder a requests de titulares de dados que são de responsabilidade
   do Controlador.
4. Notificar o Operador imediatamente sobre qualquer incidente de segurança
   que envolva os sistemas do Controlador.
5. Garantir que o Operador tem instruções válidas para tratamento.

### 2.5 Suboperadores

O Operador pode contratar suboperadores para serviços auxiliares:

| Suboperador | Serviço | Localização | Salvaguarda |
|---|---|---|---|
| Amazon Web Services (AWS) | Infraestrutura (EC2, RDS, S3) | us-east-1 (EUA) | AWS DPA + SCCs |
| Cloudflare | CDN, DDoS protection, DNS | Global | Data Processing Agreement |

O Operador notificará o Controlador com **30 dias de antecedência** sobre
qualquer nova contratação de suboperador. O Controlador pode se opor no
prazo de 15 dias.

### 2.6 Medidas de Segurança

(Ver SOC2_READINESS.md — Seção 2 para detalhes completos)

| Medida | Implementação |
|---|---|
| Criptografia em trânsito | TLS 1.3; HTTPS everywhere |
| Criptografia em repouso | AES-256 (PostgreSQL; S3) |
| Controle de acesso | Postgres RLS; JWT RS256; MFA |
| Audit trail | Hash chain; dual-write; tamper-evident |
| Rate limiting | Redis sliding window (DDoS prevention) |
| Secrets management | AWS Secrets Manager |
| Backup | WAL + full + snapshot; RPO ≤ 1h; criptografado |
| Incident response | Playbook; notificação em 72h |

### 2.7 Incidentes de Segurança

```
INCIDENTE DETECTADO POR OPERADOR
        │
        ▼
┌──────────────────────────────────────────┐
│ 1. AVALIAÇÃO (≤ 24h)                     │
│    → Classificar: risco aos direitos?     │
│    → Se SIM → notificar Controller ≤ 72h │
│    → Se NÃO → documentar internamente    │
└────────┬─────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────┐
│ 2. NOTIFICAÇÃO AO CONTROLLER (≤ 72h)     │
│    → Descrição do incidente               │
│    → Dados afetados                       │
│    → Medidas adotadas                     │
│    → Recomendações ao Controller          │
└────────┬─────────────────────────────────┘
         │
         ▼
┌──────────────────────────────────────────┐
│ 3. NOTIFICAÇÃO À ANPD (se aplicável)     │
│    → Se incidente representa risco aos    │
│      direitos dos titulares → notificar  │
│      em até 72h após conhecimento        │
└──────────────────────────────────────────┘
```

### 2.8 Responsabilidade e Indenização

1. O Operador é responsável por danos causados por tratamento em
   desconformidade com a LGPD ou instruções do Controlador.
2. O Controlador é solidariamente responsável se tiver dado causa ao
   incidente ou não tiver fornecido dados adequadamente.
3. Cláusula de limitação de responsabilidade do contrato prevalece.

---

## Parte III — Appointed DPO Documentation

### 3.1 Designação do Encarregado (DPO)

**Nome:** [Nome do DPO]
**Email:** dpo@radiant.digital
**Telefone:** +55 XX XXXX-XXXX
**Qualificações:** [Formação em direito, compliance, ou segurança da informação]

**Atribuições do DPO (Art. 39 LGPD):**

1. Aceitar reportes de titulares e autoridades
2. orientação internal team sobre obrigações LGPD
3. advising on data protection impact assessments
4. monitoramento compliance with LGPD
5. Cooperar with ANPD
6. acting as point of contact for ANPD

### 3.2 Atividades do DPO

| Atividade | Frequência | Status |
|---|---|---|
| Revisão de novos processamentos | Ad-hoc | ✅ Em andamento |
| Treinamento de equipe | Trimestral | ✅ Planejado |
| Revisão de DPA | Anual | ✅ Em andamento |
| Avaliação de impacto (DPIA) | Ad-hoc | ✅ A implementar |
| Relatório para alta direção | Semestral | ✅ A implementar |
| Monitoramento de incidentes | Contínuo | ✅ Em andamento |

---

## Parte IV — Data Subject Rights Procedures

### 4.1 Recebimento de Requests

| Canal | Tempo de Resposta | Status |
|---|---|---|
| Email: privacidade@radiant.digital | 15 dias úteis | ✅ Implementado |
| Portal web (tbd) | 15 dias úteis | 🔲 A implementar |
| Carta física | 15 dias úteis | ✅ Implementado |

### 4.2 Processo de Atendimento a Requests

```
REQUEST RECEBIDO
        │
        ▼
┌────────────────────────┐
│ 1. ACOLHIMENTO (24h)   │ → Confirmar recebimento; registrar protocolo
└────────┬───────────────┘
         │
         ▼
┌────────────────────────┐
│ 2. VERIFICAÇÃO (48h)  │ → Confirmar identidade do requerente
│                       │ → Determinar se request é procedente
└────────┬───────────────┘
         │
    ┌────┴────┐
    ▼         ▼
 SOLICITADO  INDEFERIDO
    │         │
    ▼         ▼
┌──────────────┐  ┌─────────────────────────┐
│ 3. EXECUÇÃO  │  │ Notificar requerente    │
│   (≤ 15 dias │  │ sobre indeferimento     │
│   úteis)     │  │ + motivo + recurso      │
└──────────────┘  └─────────────────────────┘
         │
         ▼
┌────────────────────────┐
│ 4. CONFIRMAÇÃO AO     │
│    REQUERENTE          │
│    Comprovação da      │
│    medida tomada        │
└────────────────────────┘
```

### 4.3 Procedimentos por Tipo de Request

**Acesso (Art. 18 I):**
- Listar todos os dados tratados sobre o titular
- Formato: electronic / PDF
- Prazo: 15 dias úteis

**Correção (Art. 18 II):**
- Identificar dados incorretos
- Corrigir nos sistemas
- Notificar suboperadores se aplicável
- Prazo: 15 dias úteis

**Eliminação (Art. 18 VI):**
- Verificar se dados são realmente desnecessários
- Verificar existência de obrigação legal de retenção
- Se elegível: excluir com certificado
- Se não elegível: notificar requerente com justificativa
- Prazo: 15 dias úteis

**Portabilidade (Art. 18 V):**
- Exportar dados em formato JSON (interoperável)
- Enviar por email seguro ou via portal
- Prazo: 15 dias úteis

---

## Parte V — Data Inventory

### 5.1 Inventário de Dados Pessoais

| Dado | Categoria | Base Legal | Retenção | Responsável |
|---|---|---|---|---|
| CNPJ da IF | Cadastral | Art. 7º IX | Vida do contrato + 5 anos | [Email] |
| Nome da IF | Cadastral | Art. 7º IX | Vida do contrato + 5 anos | [Email] |
| Email de contato técnico | Contato | Art. 7º II | Vida do contrato + 5 anos | [Email] |
| Telefone de contato | Contato | Art. 7º II | Vida do contrato + 5 anos | [Email] |
| CADOC XML (submissions) | Regulatório | Art. 7º II (obrigação legal) | 7 anos | [Email] |
| Audit log entries | Segurança | Art. 7º IX | 7 anos | [Email] |
| Logs de acesso | Segurança | Art. 7º IX | 90 dias operacional | [Email] |
| Nome de colaboradores Radiant | RH | Art. 7º II | Vida do contrato + 5 anos | [Email] |
| CPF de colaboradores Radiant | RH | Art. 7º II | Vida do contrato + 20 anos | [Email] |

### 5.2 Fluxo de Dados

```
┌──────────────┐      HTTPS      ┌──────────────┐
│   IF Client  │ ──────────────► │  Radiant API  │
│  (Browser /  │                │  (Go / chi)   │
│   SDK Go /   │                │               │
│   Python)    │                └──────┬───────┘
└──────────────┘                       │
                                       │ SQL + WAL
                                       ▼
                               ┌──────────────┐
                               │  PostgreSQL  │
                               │  (encrypted) │
                               └──────────────┘
                                       │
                                       │ STA Protocol
                                       ▼
                               ┌──────────────┐
                               │  BACEN STA   │
                               │  (SCA)       │
                               └──────────────┘
```

---

## Parte VI — Adequacy Assessment

### 6.1 Mapeamento de Requisitos LGPD

| Requisito LGPD | Status | Evidência |
|---|---|---|
| Art. 8º — Consentimento (quando aplicável) | ✅ | DPA; consentimento em onboarding |
| Art. 9º — Tratamento de dados sensíveis | N/A | Sistema não trata dados sensíveis |
| Art. 10º — Boas práticas de segurança | ✅ | SOC2_READINESS.md; BACKUP_DR_POLICY.md |
| Art. 11º — Hipóteses de tratamento | ✅ | Base legal mapeada por finalidade |
| Art. 13º — Transferência internacional | ⚠️ | SCCs + DPA AWS; avaliação pendente |
| Art. 14º — Responsabilidade | ✅ | Controles técnicos implementados |
| Art. 15º — Informações públicas | ✅ | Website; política de privacidade |
| Art. 18º — Direitos dos titulares | ✅ | Procedimentos Seção 4 |
| Art. 23º — Tratamento por controlador | ✅ | Contrato; DPA |
| Art. 25º — Encarregado | ✅ | DPO designado |
| Art. 37º — Registro de operações | ✅ | Data inventory (Seção 5) |
| Art. 48º — Notificação de incidentes | ✅ | Playbook + SLA |
| Art. 51º — Relatório de impacto | ⚠️ | A iniciar Q3 2026 |

---

*Documento preparado conforme LGPD (Lei 13.709/2018) e diretrizes da ANPD.*

*Para dúvidas sobre este documento: dpo@radiant.digital | privacidade@radiant.digital*
