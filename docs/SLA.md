# Radiant Norma — Service Level Agreement (SLA)

**Versão:** 1.0
**Data de vigência:** 2026-07-16
**Classificação:** Confidencial — Uso interno e clientes enterprise

---

## 1. Escopo do Agreement

Este SLA aplica-se ao **Radiant Norma** — plataforma de inteligência regulatória
para Instituições Financeiras brasileiras (IFs) — compreendendo:

| Componente | Descrição |
|---|---|
| API REST | Endpoints `/v1/*` do Radiant Norma |
| STA Integration | Integração com Sistema de Transferência de Arquivos do BACEN |
| Webhooks | Delivery de eventos outbound (`/v1/webhooks/*`) |
| Schema Registry | Acesso a leiautes CADOC (`/v1/schemas/*`) |
| Dashboard SSE | Stream de eventos real-time (`/v1/events/stream`) |
| Generator API | Geração de CADOCs (`/v1/generate/*`) |
| Insights API | AI Insights (`/v1/insights/*`) |

**Fora do escopo:** infraestrutura de rede do cliente, DNS, provedores de cloud
do cliente, e sistemas externos do BACEN (SCA, STA).

---

## 2. Níveis de Serviço

### 2.1 Disponibilidade por Tier

| Métrica | Starter | Professional | Enterprise |
|---|---|---|---|
| **Uptime mensal garantido** | 99,0% | 99,5% | 99,9% |
| **Downtime máximo mensal** | 7h 18min | 3h 39min | 43min 12seg |
| **MTTR (tempo de recuperação)** | 8h úteis | 4h úteis | 1h útil |
| **Periodicidade de backups** | Diário | A cada 6h | Contínuo (WAL) |
| **Janela de manutenção programada** | Não garantida | Até 4h/mês | Até 2h/mês |
| **Canais de suporte** | Email | Email + Chat | Email + Chat + Telefone |
| **SLA de resposta inicial** | 24h úteis | 8h úteis | 1h útil |

### 2.2 Definições de Uptime

```
Uptime mensal (%) = [(Total minutos no mês − Downtime real) / Total minutos no mês] × 100
```

- **Downtime real** = tempo acumulado em que a API retorna erro 5xx ou está
  completamente indisponível (sem resposta), medido do nosso ponto de presença.
- **Downtime não inclui:** manutenção programada notificada com 72h de antecedência,
  falhas de infraestrutura do cliente, force majeure, ou indisponibilidade do STA/BACEN.
- **Métricas de uptime** são disponibilizadas via página de status pública
  (`status.radiant.digital`) com granularidade mensal.

---

## 3. Tempo de Resposta e Prioridade de Incidentes

### 3.1 Definição de Severidade

| Severidade | Descrição | Exemplo |
|---|---|---|
| **Crítica (SEV-1)** | Sistema indisponível ou perda de dados iminente | API fora do ar, CADOC não consegue ser gerado |
| **Alta (SEV-2)** | Funcionalidade principal degradada com work-around limitado | Validação sempre retorna erro, STA submit falha |
| **Média (SEV-3)** | Funcionalidade secundária afetada | Insights AI fora do ar, webhook não entrega |
| **Baixa (SEV-4)** | Bug cosmético ou melhoria | UI lento, erro de formatação em mensagem |

### 3.2 Tempos de Resposta por Tier

| | Starter | Professional | Enterprise |
|---|---|---|---|
| **SEV-1** — Primeiro Resposta | 24h úteis | 8h úteis | 1h útil |
| **SEV-1** — Resolução | 48h úteis | 16h úteis | 4h úteis |
| **SEV-2** — Primeiro Resposta | 48h úteis | 16h úteis | 4h úteis |
| **SEV-2** — Resolução | 5 dias úteis | 3 dias úteis | 1 dia útil |
| **SEV-3** — Primeiro Resposta | 5 dias úteis | 3 dias úteis | 1 dia útil |
| **SEV-3** — Resolução | 10 dias úteis | 7 dias úteis | 3 dias úteis |
| **SEV-4** — Primeiro Resposta | 10 dias úteis | 5 dias úteis | 2 dias úteis |
| **SEV-4** — Resolução | Próxima release | Próxima release | Próxima release |

---

## 4. Janelas de Manutenção

### 4.1 Manutenção Programada

- **Notificação mínima:** 72 horas de antecedência por email e na página de status.
- **Periodicidade:** No máximo 1 janela por mês, limitada a 4h (Starter/Professional)
  ou 2h (Enterprise).
- **Horário:** Janelas de baixa utilização — domingos 00h-06h BRT ou conforme
  acordado com o cliente.
- **Disponibilidade durante manutenção:** Sistemas em standby são mantidos online;
  funcionalidades em manutenção podem ficar indisponíveis.

### 4.2 Manutenção Emergencial

- Notificação o mais breve possível (tão logo o evento seja identificado).
- Tempo máximo de resolução: 4h para SEV-1, 12h para SEV-2.
- Incidentes de manutenção emergencial não cuentan para downtime do SLA
  quando comunicados em até 1h após o início.

---

## 5. Suporte Técnico

### 5.1 Canais

| Canal | Starter | Professional | Enterprise |
|---|---|---|---|
| Email (`suporte@radiant.digital`) | ✅ | ✅ | ✅ |
| Chat (dashboard) | ❌ | ✅ | ✅ |
| Telefone (horário comercial BRT) | ❌ | ❌ | ✅ |
| Gerente de conta dedicado | ❌ | ❌ | ✅ |
| SLA de primeira resposta | 24h úteis | 8h úteis | 1h útil |
| Horário de atendimento | Dias úteis 09h-18h BRT | Dias úteis 08h-20h BRT | 24/7 para SEV-1 |

### 5.2 Idioma

Suporte disponível em **português brasileiro** (preferencial) e **inglês**.

---

## 6. Exclusões (Downtime não conta para o SLA)

Os seguintes cenários são **excluídos** do cálculo de uptime e não geram
créditos de SLA:

1. **Infraestrutura do cliente** — problemas de rede, DNS, firewall ou
   conectividade do cliente.
2. **Sistemas externos do BACEN** — indisponibilidade do STA, SCA, ou
   qualquer sistema do Banco Central.
3. **Força maior** — desastres naturais, guerras, greves de terceiros,
   pandemias, falhas de energia pública.
4. **Manutenção programada** — notificada com 72h de antecedência.
5. **Manutenção emulada** — falhas causadas por teste de carga não
   coordenado pelo time Radiant.
6. **Uso indevido** — consumo acima dos limites contratados (ex.: rate limit
   excedido propositalmente).
7. **Credenciais comprometidas** — vazamento de chaves API do cliente.
8. **Atualizações de schema BACEN** — quando o BACEN publica novo leiaute
   CADOC que exige atualização do sistema (prazo de adaptação: 30 dias).

---

## 7. Créditos de SLA

### 7.1 Elegibilidade

Se o uptime real ficar **abaixo** do garantido para o tier contratado,
créditos serão aplicados conforme a tabela abaixo. Créditos são limitados
a **30% do valor mensal** da assinatura.

| Uptime real | Crédito (% da mensalidade) |
|---|---|
| 98,0% – 98,9% (Starter) | 10% |
| 97,0% – 97,9% (Starter) | 20% |
| < 97,0% (Starter) | 30% |
| 98,0% – 98,9% (Professional) | 10% |
| 97,0% – 97,9% (Professional) | 20% |
| < 97,0% (Professional) | 30% |
| 99,0% – 99,4% (Enterprise) | 10% |
| 98,5% – 98,9% (Enterprise) | 20% |
| < 98,5% (Enterprise) | 30% |

### 7.2 Processo de Crédito

1. Cliente abre chamado (`suporte@radiant.digital`) em até **7 dias úteis**
   após o incidente, com evidências (timestamps, screenshots, logs).
2. Equipe Radiant verifica os logs internos de uptime em até **3 dias úteis**.
3. Crédito aprovado é aplicado na **próxima fatura** ou compensado
   na renovação do contrato.

---

## 8. Responsabilidades do Cliente

Para que o SLA seja válido, o cliente deve:

1. **Manter credenciais seguras** — chaves API, tokens JWT, senhas.
2. **Configurar rate limits** — não exceder limites contratados
   (`RADIANT_RATE_LIMIT_BACKEND` em produção).
3. **Manter contacto atualizado** — email e telefone do responsável técnico
   para notificações de manutenção.
4. **Reportar incidentes** — abrir chamado em até 7 dias úteis após
   o incidente.
5. **Manter conformidade** — o cliente é responsável pela homologação
   BACEN da sua instituição.

---

## 9. Segurança e Notificações

### 9.1 Violações de Segurança

Incidentes de segurança (acesso não autorizado, vazamento de dados,
comprometimento de credenciais) são tratados como **SEV-1** com tempo de
resposta de 1h para Enterprise. Notificação aos clientes afetados será
realizada em até **72 horas** após confirmação, conforme LGPD Art. 48.

### 9.2 Notificações de Incidente

- **Starter/Professional:** Email para o contato técnico cadastrado.
- **Enterprise:** Email + SMS + ligação telefônica para o gerente de conta.

---

## 10. Limitação de Responsabilidade

Exceto por créditos de SLA explicitados na Seção 7, a Radiant Norma
**não será responsável** por:

- Perdas de lucro cessante, oportunidades de negócio, ou danos indiretos.
- Ações ou omissões do cliente ou de terceiros.
- Dados regulatórios perdidos por falha do cliente em manter backups.
- Consequências de não conformidade regulatória do cliente perante o BACEN.

O SLA não substitui o contrato de prestação de serviços. Em caso de
conflito, os termos do contrato prevalecem.

---

## 11. Revisão e Vigência

| Versão | Data | Alterações |
|---|---|---|
| 1.0 | 2026-07-16 | Versão inicial |

Este SLA é revisado **anualmente** ou em caso de mudança material
de arquitetura. Clientes são notificados por email com 30 dias
de antecedência sobre alterações.

---

*Radiant Norma — Radiant Tecnologia e Sistemas LTDA*
*CNPJ: XX.XXX.XXX/XXXX-XX | contato@radiant.digital*
