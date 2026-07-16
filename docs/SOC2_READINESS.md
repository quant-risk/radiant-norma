# SOC 2 Type I — Readiness Package

**Versão:** 1.0
**Data:** 2026-07-16
**Classificação:** Confidencial — Auditors e clientes enterprise

> **Nota:** Este documento constitui o **readiness package** para SOC 2 Type I.
> SOC 2 Type I avalia a *adequação* dos controles em um ponto no tempo
> (moment-in-time),不同于 Type II que observa a *efetividade operativa* ao longo
> de 6-12 meses. Para completar a certificação SOC 2 Type II, um external
> auditor deve conduzir a revisão durante o período de observação.

---

## 1. Trust Service Criteria (TSC) — Mapeamento de Controles

### 1.1 Security (Comum a todos os tipos de SOC 2)

| # | Critério | Controles Implementados | Evidência |
|---|---|---|---|
| CC1.1 | Ambiente de controle | Separação de deveres; políticas de segurança documentadas | Seção 3 deste documento |
| CC1.2 | Comunicação interna | Canais definidos; código review mandatory | GitHub PR reviews; Slack #security |
| CC1.3 | Estrutura organizacional | Org chart; papel do CISO definido | Seção 3.1 |
| CC2.1 | Comunicação de informação | Políticas de comunicação interna/externa | Este documento; BACKUP_DR_POLICY.md |
| CC2.2 | Comunicação interna | tbd | tbd |
| CC2.3 | Comunicação externa | Processos de comunicação de incidentes | SLA.md Seção 9 |
| CC3.1 | Avaliação de riscos | Risk assessment anual | Seção 4 |
| CC3.2 | Identification and analysis of risk | Risk assessment | Seção 4 |
| CC3.3 | Mitigation | Controles documentados; mitigação via controles técnicos | Seção 5 |
| CC3.4 | Monitoring | Revisão trimestral de riscos | Seção 4.3 |
| CC4.1 | Monitoramento | Logs de auditoria; métricas | CC: telemetry via OTel + Sentry |
| CC5.1 | Logical and physical access controls | JWT RS256; RLS Postgres; fail-closed | AUDIT.md; BACKUP_DR_POLICY.md |
| CC5.2 | Authorization | Princípio de menor privilégio; RBAC via JWT claims | AUDIT.md Seção 5 |
| CC5.3 | Physical access controls | AWS physical security (usar se cloud); ou controle físico on-prem | AWS VPC / datacenter SOC 2 (se applicable) |
| CC5.4 | System components access | Credenciais via Secrets Manager; rotação de chaves | AUDIT.md Seção 4 |
| CC6.1 | Logical access security | Rate limiting; CSRF; HMAC webhooks | AUDIT.md; BACKUP_DR_POLICY.md |
| CC6.2 | Provisioning | Processos de provisioning documentados | Runbook de deploy |
| CC6.3 | Removal | Offboarding checklist; revogação imediata | HR process |
| CC6.4 | Prevention | WAF; DDoS protection; encrypted transit | Cloudflare/AWS Shield |
| CC6.5 | Detection | Intrusion detection; anomaly detection via Sentry | OTel + Sentry |
| CC6.6 | Response | Incident response plan | Seção 6 |
| CC7.1 | Change management | CI/CD com approval; git history; immutable deploys | GitHub Actions; terraform state |
| CC7.2 | Environmental changes | Automated deployment pipeline | GitHub Actions |
| CC8.1 | Data integrity | Hash chain audit log; dual-write | AUDIT.md Phase 7 |
| CC8.2 | Processing | Atomic transactions; idempotency | AUDIT.md Phase 4 |
| CC9.1 | Risk mitigation | Controles de segurança; monitoramento | Seção 5 |
| CC9.2 | Vendor risk | Avaliação de vendors críticos (AWS, Cloudflare) | Vendor SOC 2 (AWS, Cloudflare) |

---

## 2. Controles Técnicos Implementados

### 2.1 Segurança de Aplicação

| Controle | Implementação | Referência |
|---|---|---|
| Autenticação | JWT RS256 com rotação por `kid` | `backend/internal/auth/` |
| Autorização multi-tenant | Postgres RLS FORCE mode + `SET LOCAL app.if_id` | Phase 5 (AUDIT.md) |
| Rate limiting | Token bucket (memory) ou sliding window (Redis) | Phase 5 (AUDIT.md) |
| Proteção CSRF | Double-submit cookie pattern | `backend/internal/api/csrf.go` |
| Webhook signatures | HMAC-SHA256 com `X-Radiant-Signature` | Phase 5 (AUDIT.md) |
| Fail-closed production | `RADIANT_ENV=production` bloqueia dev-modes | Phase 5 (AUDIT.md) |
| Slowloris protection | ReadHeaderTimeout 10s; ReadTimeout 30s | Phase 5 (AUDIT.md) |
| Error sanitization | `loggerutil.SafeError` — DSN nunca vaza | Phase 5 (AUDIT.md) |
| Sanitization | HMAC-SHA256; idempotency keys; XML hash deduplication | Phase 4 (AUDIT.md) |

### 2.2 Segurança de Dados

| Controle | Implementação | Referência |
|---|---|---|
| Criptografia em trânsito | TLS 1.2+; HTTP/2; HSTS | Cloudflare/AWS ALB |
| Criptografia em repouso | PostgreSQL encryption at rest (AES-256) | AWS RDS / Cloud SQL |
| Backup encryption | WAL + base backup criptografados em S3 | BACKUP_DR_POLICY.md |
| Audit tamper-evident | Hash chain (SHA-256) + dual-write | Phase 7 (AUDIT.md) |
| Secrets management | AWS Secrets Manager / HashiCorp Vault | Runbook de deploy |
| Data retention | 7 anos para CADOCs; 7 anos audit logs | BACKUP_DR_POLICY.md Seção 8 |

### 2.3 Disponibilidade e Operações

| Controle | Implementação | Referência |
|---|---|---|
| Observabilidade | OTel + Sentry + Prometheus | Phase 5 (AUDIT.md) |
| Graceful shutdown | 10s timeout; drain connections | Phase 5 (AUDIT.md) |
| Disaster recovery | PITR; RTO ≤ 4h; RPO ≤ 1h | BACKUP_DR_POLICY.md |
| Status page | `status.radiant.digital` (uptime monitoring) | tbd |
| Uptime guarantee | 99,9% Enterprise (SLA) | SLA.md |

---

## 3. Políticas de Segurança

### 3.1 Política de Acesso a Sistemas

**Última revisão:** 2026-07-16 | **Próxima revisão:** 2026-07-16 + 1 ano

**Princípios:**
1. **Menor privilégio:** colaboradores recebem apenas as permissões mínimas
   necessárias para suas funções.
2. **Separation of duties:** credenciais de produção exigem MFA; deployment
   requer aprovação de peer review.
3. **Zero trust:** sem confiança implícita; cada request é autenticado e
   autorizado.

**Access levels:**

| Nível | Acesso | Requisitos |
|---|---|---|
| **Desenvolvedor** | Read-only em produção (logs/métricas) | MFA; VPN; justificativa |
| **SRE/DevOps** | Read/write em produção (deploy, config) | MFA; VPN; approval; logging |
| **CTO/Admin** | Full access (incluindo secrets) | MFA; hardware key; offsite backup |
| **Auditor externo** | Read-only audit logs e configurations | NDA; account temporário; 30 dias TTL |

**Revogação:** offboarding procedure executa revogação em ≤ 4h.

### 3.2 Política de Criptografia

1. **Transit:** TLS 1.3 obrigatório para todas as conexões. TLS 1.0/1.1
   bloqueado no load balancer.
2. **At rest:** AES-256 para dados sensíveis (CADOC payloads, credentials).
3. **Keys:** RSA-4096 para JWT signing; rotação a cada 12 meses ou em caso
   de comprometimento (key rotation implementada — AUDIT.md).
4. **Secrets:** AWS Secrets Manager; nunca em código fonte ou variáveis de
   ambiente compartilhadas.

### 3.3 Política de Gestão de Incidentes

(Ver Seção 6 para detalhes completos)

---

## 4. Avaliação de Riscos

### 4.1 Risk Register

| Risco | Probabilidade | Impacto | Controles de Mitigação | Risco Residual |
|---|---|---|---|---|
| Acesso não autorizado a dados de IF | Média | Crítico | JWT RS256; RLS; MFA | Baixo |
| Vazamento de credentials STA | Baixa | Crítico | Secrets Manager; rotação; não logging | Muito Baixo |
| Ransomware (dados regulatórios) | Baixa | Crítico | Immutable backups (S3 Object Lock); PITR | Baixo |
| Indisponibilidade do STA/BACEN | Média | Alto | Retry backoff; DLQ; status page | Médio |
| Falha de backup | Baixa | Alto | WAL + full backup + restore testing semanal | Baixo |
| Insider threat (funcionário) | Muito Baixa | Crítico | RBAC; separation of duties; audit log | Baixo |
| Configuração incorreta de RLS | Baixa | Alto | Migration testing CI (Postgres 16 real) | Baixo |
| Comprometimento de webhook secret | Média | Alto | HMAC-SHA256; idempotency; re-play prevention | Médio |
| DoS/DDoS | Média | Médio | Rate limiting; Cloudflare/AWS Shield; per-tenant caps | Baixo |

### 4.2 Mitigations em Place

| Mitigação | Eficácia | Evidência |
|---|---|---|
| Postgres RLS + FORCE mode | Alta | CI valida em Postgres 16 real (`.github/workflows/test.yml`) |
| Idempotency + deduplicação | Alta | Phase 4 audit; dois níveis de dedup |
| Immutable backups (S3 Object Lock) | Alta | BACKUP_DR_POLICY.md |
| Fail-closed production gates | Alta | Phase 5 audit; `RADIANT_ENV=production` check |
| Hash-chain audit log | Alta | Phase 7 audit; dual-write |
| Webhook HMAC-SHA256 | Alta | Phase 5 audit |
| Rate limiter Redis (sliding window) | Alta | Phase 5 audit; CI tests |

### 4.3 Review Schedule

| Frequência | Escopo | Responsável |
|---|---|---|
| **Trimestral** | Risk register review; verificar se controles continuam eficazes | CTO + CISO |
| **Anual** | Avaliação completa de riscos; atualização de políticas | CTO + Legal |
| **Ad-hoc** | Após incidente significativo, mudança de arquitetura, ou nova regulamentação | Equipe de segurança |

---

## 5. Maturidade de Controles

| Área | Nível de Maturidade | Observações |
|---|---|---|
| Access Control | 4 (Gerenciado) | MFA; RBAC; secrets manager; baseline monitorado |
| Audit Logging | 4 (Gerenciado) | Hash chain; dual-write; tamper-evident; OTel |
| Encryption | 4 (Gerenciado) | TLS 1.3; AES-256; key rotation implementada |
| Change Management | 3 (Definido) | CI/CD; peer review; immutable deploys; staging |
| Incident Response | 3 (Definido) | Playbooks documentados; SEV-1/2/3/4 definidos |
| Backup & Recovery | 3 (Definido) | WAL + full + snapshot; PITR; DR drill planejado |
| Vendor Management | 2 (Reproduzível) | Avaliação de vendors críticos (AWS, Cloudflare) |
| Risk Assessment | 2 (Reproduzível) | Risk register criado; belum audit externo |
| Security Monitoring | 4 (Gerenciado) | OTel + Sentry + Prometheus + PagerDuty |
| Business Continuity | 2 (Reproduzível) | DR policy documentado; belum DR drill |

**Escala:** 1=Ad-hoc, 2=Reproduzível, 3=Definido, 4=Gerenciado, 5=Otimizado

---

## 6. Plano de Incident Response

### 6.1 Classificação de Incidentes

| Classe | Definição | Exemplo |
|---|---|---|
| **P1 — Data Breach** | Acesso não autorizado a dados sensíveis | Dados de IF expostos; credentials vazados |
| **P2 — Service Outage** | Indisponibilidade de sistema | API fora do ar; database inacessível |
| **P3 — Degradation** | Funcionalidade parcialmente degradada | STA submit falhando; webhook não entregue |
| **P4 — Near Miss** | Incidente que quase aconteceu | Configuração incorreta detectada antes de produção |

### 6.2 Processo de Resposta

```
INCIDENTE DETECTADO
        │
        ▼
┌─────────────────┐
│ TRIAGE (15 min) │ → Classificar (P1-P4)
└────────┬────────┘
         │
    ┌────┴────┐
    ▼         ▼
  P1/P2     P3/P4
    │         │
    ▼         ▼
┌──────────┐  ┌──────────────────┐
│ Bridge   │  │ Notificar Lead   │
│war room  │  │ Eng. (async)     │
│+ cliente │  │ Resolver < 5 dias│
└────┬─────┘  └──────────────────┘
     │
     ▼
┌─────────────────┐
│ CONTAINMENT     │ → Isolação;mitigação imediata
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ ERADICATION     │ → Root cause; remoção de vector
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ RECOVERY        │ → Restore;verificação;monitoramento
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ POST-INCIDENT   │ → Retrospectiva blameless;
│ REVIEW (72h)    │ → Playbook atualizado;evidência arquivada
└─────────────────┘
```

### 6.3 Notificações

| Tipo de Incidente | Notificação | Prazo |
|---|---|---|
| P1 (Data Breach) | Clientes afetados + ANPD (LGPD) | 72h (LGPD Art. 48) |
| P1/P2 | Página de status + clientes enterprise | 1h |
| P3 | Documentação interna | Próximo dia útil |

---

## 7. Evidência de Controles

### 7.1 Controles Automatizados (Evidence)

| Controle | Evidência | Como acessar |
|---|---|---|
| CI/CD testing | GitHub Actions run history | `github.com/fortvna/radiant-norma/actions` |
| Postgres RLS validation | `cmd/migrate` + `test.yml` | `backend/.github/workflows/test.yml` |
| Audit log hash chain | Test: `auditlog/log_test.go` | `backend/internal/auditlog/log_test.go` |
| Rate limiter tests | Test: `ratelimit_test.go` | `backend/internal/api/ratelimit_test.go` |
| Webhook HMAC tests | Test: `webhook_handlers_test.go` | `backend/internal/api/webhook_handlers_test.go` |
| Idempotency tests | Test: `sta_handlers_test.go` | `backend/internal/api/sta_range_handlers_test.go` |
| Go vet / static analysis | CI step | `.github/workflows/test.yml` |

### 7.2 Controles Manuais (Evidence)

| Controle | Evidência | Frequência |
|---|---|---|
| Backup restore test | Script output + sign-off | Semanal |
| DR drill | Relatório de drill | Trimestral |
| Access review | Lista de acessos + aprovação | Trimestral |
| Risk assessment | Risk register atualizado | Anual |

---

## 8. Gaps e Plano de Remediação

| Gap | Severidade | Ação | Prazo | Responsável |
|---|---|---|---|---|
| SOC 2 Type II audit externo | Alta | Contratar auditor | 6-12 meses | CTO |
| Pentest externo | Alta | Contratar firma especializada | 3 meses | CISO |
| Business continuity plan (formal) | Média | Documentar BIA | 3 meses | CTO |
| Vulnerability scanning automatizado | Média | Integrar trivy/owasp-zap na CI | 2 meses | DevOps |
| SIEM (Security Information Event Management) | Média | Avaliar e implementar | 6 meses | DevOps |
| Disaster recovery drill | Média | Executar DR drill | 3 meses | SRE |

---

## 9. Próximos Passos

### Para obter SOC 2 Type I (curto prazo)

1. [x] Readiness package completo (este documento)
2. [ ] Pentest externo independente (agendado)
3. [ ] Auditor externo contratados ( RFQ em andamento)
4. [ ] Auditoria SOC 2 Type I (prevista: Q4 2026)

### Para obter SOC 2 Type II (longo prazo)

1. [ ] Operar controles por 6-12 meses (período de observação)
2. [ ] Coleta de evidências contínua
3. [ ] Auditoria SOC 2 Type II (prevista: Q2-Q3 2027)

---

*Para questões sobre este documento, contactar: security@radiant.digital*

*Este documento é confidencial e coberto pelo NDA entre a Radiant e o destinatário.*
