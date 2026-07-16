# Radiant Norma — Backup & Disaster Recovery (DR) Policy

**Versão:** 1.0
**Data:** 2026-07-16
**Classificação:** Confidencial — Uso interno e clientes enterprise

---

## 1. Visão Geral

Este documento estabelece a política de backup e recuperação de desastres
(DR) para o Radiant Norma. O objetivo é garantir **RPO ≤ 1 hora** e
**RTO ≤ 4 horas** para cargas de trabalho de produção.

| Sigla | Definição | Meta |
|---|---|---|
| **RPO** | Recovery Point Objective — perda máxima de dados tolerável | ≤ 1 hora |
| **RTO** | Recovery Time Objective — tempo máximo para restaurar serviço | ≤ 4 horas |
| **RLO** | Recovery Level Objective — ponto mínimo de recuperação funcional | API + DB + STA |

---

## 2. Arquitetura de Dados

### 2.1 Componentes com dado persistente

| Componente | Tecnologia | Dado |
|---|---|---|
| Primary DB | PostgreSQL 16 | Schema registry, audit log, submissions, webhooks, users |
| Write-Ahead Log | PostgreSQL WAL | Replay para point-in-time recovery |
| Object Storage | S3-compatible (ex.: AWS S3, Cloudflare R2) | CADOC XML payloads, exports CSV |
| Redis | Redis (opcional) | Rate limit counters, session cache |

### 2.2 O que NÃO é coberto por backup

- **Cache Redis** — não é fonte de verdade; pode ser reconstruído.
- **Certificates/TLS** — managed via ACM/Let's Encrypt ou secrets manager.
- **Hub SSE em memória** — subscribers não persistidos (reconecta no reconnect).

---

## 3. Estratégia de Backup

### 3.1 PostgreSQL — Camadas de Backup

```
┌─────────────────────────────────────────────────────────────┐
│  Camada 1: WAL contínuo (Write-Ahead Log)                   │
│  → PITR (Point-In-Time Recovery) com RPO = 0 (teórico)     │
│  → WAL segments archivados a cada 5 min para S3             │
├─────────────────────────────────────────────────────────────┤
│  Camada 2: Full backup diário (base backup)                  │
│  → pg_basebackup via cron 01:00 BRT diária                   │
│  → Comprimido com gzip, armazenado em S3                    │
│  → Retenção: 30 dias                                        │
├─────────────────────────────────────────────────────────────┤
│  Camada 3: Snapshot mensal (long-term retention)             │
│  → Armazenado em cold storage (S3 Glacier)                   │
│  → Retenção: 12 meses                                       │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 Object Storage (CADOC payloads)

- **Frequência:** A cada novo submission, o payload XML é written to S3
  antes do commit da transação (写入 antes de commit — garantia de durability).
- **Versioning:** S3 versioning habilitado — cada submissão tem history.
- **Retenção:** 7 anos (prazo de guarda regulatória BACEN + 1 ano buffer).
- **Criptografia:** AES-256 no lado do servidor (SSE-S3).

### 3.3 Backup da Configuração

| Artefato | Onde | Como |
|---|---|---|
| `DATABASE_URL` (connection string) | AWS Secrets Manager / Vault | Sem backup manual — segredo é a fonte |
| JWT public keys | Env var `RADIANT_JWT_PUBLIC_KEY` | Rotação documentada; backup offline |
| STA credentials | Env var `STA_*` | Secrets Manager |
| `RADIANT_NORMA_ADMIN_TOKEN` | Secrets Manager | Backup impresso em cofre físico |
| Go binary (deploy) | Immutable artifact in S3 | Identificado por git SHA |

---

## 4. Procedimentos de Restore

### 4.1 Restore PostgreSQL — Point-In-Time Recovery (PITR)

**Quando usar:** Dados deletados/modificados acidentalmente, até 30 dias.

```bash
# 1. Identificar o timestamp do problema
# Ex.: "CADOC 3040 modificado incorretamente às 14:23:00 BRT 2026-07-15"

# 2. Parar a API
sudo systemctl stop radiant-norma

# 3. Restaurar base backup mais recente ANTES do timestamp do problema
aws s3 cp s3://radiant-backups/postgres/base/2026-07-15/ ./restore/ --recursive
cd restore
tar -xzf base_backup.tar.gz
# (em standby, não na produção)

# 4. Criar recovery.conf para PITR
cat > recovery.conf <<'EOF'
restore_command = 'aws s3 cp s3://radiant-backups/postgres/wal/%f %p'
recovery_target_time = '2026-07-15 14:23:00 BRT'
recovery_target_action = 'promote'
EOF

# 5. Start PostgreSQL em modo recovery
pg_ctl start -D /var/lib/postgresql/radiant

# 6. Verificar integridade após restore
psql -c "SELECT count(*) FROM audit_log;"
psql -c "SELECT count(*) FROM envios;"

# 7. Restart API após validação
sudo systemctl start radiant-norma
```

### 4.2 Restore PostgreSQL — Full Restore (disaster total)

**Quando usar:** Perda total do primary DB (hardware failure, ransomware).

```bash
# 1. Provisionar novo PostgreSQL 16 (mesma versão do source)
# 2. Baixar último full backup
aws s3 cp s3://radiant-backups/postgres/base/$(date +%Y-%m-%d)/latest.tar.gz /tmp/
tar -xzf /tmp/latest.tar.gz -C /var/lib/postgresql/radiant/

# 3. Configurar PostgreSQL
chown -R postgres:postgres /var/lib/postgresql/radiant
chmod 700 /var/lib/postgresql/radiant

# 4. Start PostgreSQL
sudo -u postgres pg_ctl start -D /var/lib/postgresql/radiant

# 5. Executar validações
sudo -u postgres psql -c "SELECT version();"
sudo -u postgres psql -c "SELECT count(*) FROM audit_log;" radiant_norma

# 6. Redeploy API apontando para novo DB
sudo systemctl restart radiant-norma
```

### 4.3 Restore Object Storage (CADOC payloads)

```bash
# Listar versões de um objeto
aws s3api list-object-versions \
  --bucket radiant-cadocs \
  --prefix "envios/2030/2026/07/15/"

# Restaurar versão específica para novo prefixo (não sobrescrever original)
aws s3 cp s3://radiant-cadocs/envios/2030/2026/07/15/bad.xml \
  s3://radiant-cadocs/envios/2030/2026/07/15/bad.xml.restored \
  --version-id "VERSÃO_ID"

# Ou restaurar todo o prefixo para um ponto no tempo (S3 Point-in-Time Restore)
aws s3 sync s3://radiant-cadocs/envios/ \
  s3://radiant-cadocs-restored/envios/ \
  --source-region us-east-1 \
  --exclude "*" \
  --include "envios/2030/2026/07/15/*"
```

---

## 5. Disaster Recovery — Procedimento Completo

### 5.1 ativação de DR (failover para região secundária)

**RTO meta: 4 horas | RPO: 1 hora**

```
┌──────────────────────────────────────────────────────────────────┐
│ INCIDENTE CRÍTICO (SEV-1): Primary Region fora do ar             │
│                                                                  │
│ 1. Declarar DR (líder de on-call): 15 min                       │
│    → Notificar equipe + clientes Enterprise via status page      │
│                                                                  │
│ 2. Promover DR Region: 1-2 horas                                │
│    → Restaurar último full backup + WAL no DR PostgreSQL         │
│    → Provisionar DR API servers com mesma config                 │
│    → Trocar DNS (Route53) para DR Region                         │
│                                                                  │
│ 3. Validação: 30 min                                            │
│    → smoke tests em /healthz, /v1/validate, STA submit          │
│    → Verificar integridade do último submission                  │
│                                                                  │
│ 4. Comunicação: 15 min                                           │
│    → Status page: "Degradado — failover em andamento"            │
│    → Email para contatos técnicos (Enterprise)                    │
│                                                                  │
│ 5. Resume: RTO counted from incident declaration                 │
└──────────────────────────────────────────────────────────────────┘
```

### 5.2 Return to Primary (failback)

Após primary estar restaurado:

1. Sincronizar delta (últimas 1-4h de writes) via pg_dump + restore
2. Fazer switch de volta para primary (DNS flip)
3. Manter DR region em standby por 48h
4. Desligar DR region temporário

---

## 6. Validação de Backups

### 6.1 Testes de Restore

| Frequência | Tipo | Procedimento |
|---|---|---|
| **Semanal** | Restaurar DB em ambiente de staging | Script `scripts/weekly-restore-test.sh` |
| **Mensal** | Simulação de PITR (DR drill) | Time: 2h; executar failover completo |
| **Trimestral** | DR drill completo + comunicação com clientes | Simular outage e validar RTO real |

### 6.2 Smoke Tests Pós-Restore

```bash
#!/bin/bash
# scripts/post-restore-verify.sh

set -e

API_URL="${1:-http://localhost:8080}"

# 1. Health check
curl -sf "$API_URL/healthz" || exit 1

# 2. DB connectivity
psql -c "SELECT count(*) FROM audit_log;" || exit 2

# 3. Schema registry
curl -sf "$API_URL/v1/schemas" | grep -q "3040" || exit 3

# 4. Integrity do audit log (hash chain)
curl -sf "$API_URL/v1/admin/audit/verify" | grep -q '"valid":true' || exit 4

echo "✅ Post-restore verification passed"
```

---

## 7. Monitoring e Alertas

| Verificação | Ferramenta | Frequência | Alerta |
|---|---|---|---|
| Backup completaram | Cron + CloudWatch | A cada backup | PagerDuty SEV-2 se falhar |
| WAL archiving healthy | pg_stat_archiver | Contínuo | PagerDuty SEV-2 |
| Espaço em disco DB | node_exporter + Prometheus | 5 min | SEV-2 |
| S3 integrity (checksum) | AWS Config Rules | Contínuo | SEV-2 |
| Teste de restore semanal | Script automated | Seguindo | Email |

---

## 8. Retenção de Dados

| Dado | Retenção | Justificativa |
|---|---|---|
| Full backups PostgreSQL | 30 dias | Limite operacional; WAL cobre o resto |
| WAL archives | 30 dias | PITR até 30 dias |
| Snapshots mensais | 12 meses | Long-term retention |
| CADOC XML payloads (S3) | 7 anos | BACEN regulamento + buffer |
| Audit logs | 7 anos | LGPD + regulamento BACEN |
| Export CSV | 5 anos | Prático; regenerável se necessário |
| Logs de aplicação | 90 dias | Espaço; logs antigos arquivados |

---

## 9. Responsabilidades

| Papel | Responsabilidade |
|---|---|
| **DevOps / SRE** | Executar backups, validar restore, DR drill |
| **CTO** | Aprovar DR policy, validar DR drill anual |
| **Cliente (On-Premise)** | Executar e validar seus próprios backups |

---

## 10. Revisão

| Versão | Data | Alterações |
|---|---|---|
| 1.0 | 2026-07-16 | Versão inicial |

Revisar **anualmente** ou após qualquer mudança significativa de arquitetura.

---

*Contato de emergência: +55 XX XXXX-XXXX | oncall@radiant.digital*
