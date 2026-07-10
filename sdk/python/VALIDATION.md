# VALIDATION — SDK Python v1.0.1

**Data:** 2026-07-09
**Sprint:** 61 (SDK_Python)
**Versão SDK:** v1.0.1 (bug fix release)
**Auditor:** ZCode Agent

---

## §1 — Escopo

Validação do SDK Python (`sdk/python/`) contra a API real do backend.

Arquivos analisados:
- `src/radiant/client.py` — Cliente HTTP e serviços
- `src/radiant/types.py` — Tipos compartilhados
- `tests/test_radiant.py` — Testes unitários
- `README.md` — Documentação

---

## §2 — Bugs Críticos Encontrados e Corrigidos

### Bug 1: URL Errada do Endpoint Validate

| Aspecto | Antes (ERRADO) | Depois (CORRETO) |
|---|---|---|
| **URL** | `/v1/validate/{cadoc}` | `/v1/validate` |
| **API real** | Não existe | `server.go:205` → `r.Post("/validate", s.validate)` |

### Bug 2: Body Request Errado

| Aspecto | Antes (ERRADO) | Depois (CORRETO) |
|---|---|---|
| **Body** | `{"xml": ...}` | `{"cadoc_code": "...", "xml": "..."}` |

### Bug 3: Campo Response Errado

| Aspecto | Antes (ERRADO) | Depois (CORRETO) |
|---|---|---|
| **Tipo** | `valid: bool` | `passed: bool` |

---

## §3 — Campos Adicionados

| Campo | Tipo | Descrição |
|---|---|---|
| `data_base` | `str` | Data-base do documento |
| `xml_hash` | `str` | SHA-256 do XML |
| `duration_ms` | `int` | Tempo de execução |

---

## §4 — Testes

| Teste | Status |
|---|---|
| `test_validate_success` | ✅ PASS (atualizado) |
| `test_validate_with_errors` | ✅ PASS (atualizado) |
| `test_validate_api_error` | ✅ PASS (atualizado URL) |
| `test_list_rules` | ✅ PASS |
| `test_radar_scan` | ✅ PASS |
| `test_insights_ask` | ✅ PASS |
| `test_schemas_list_versions` | ✅ PASS |
| `test_validate_cross_doc` | ✅ PASS |
| `test_client_default_base_url` | ✅ PASS |
| `test_validation_error_fields` | ✅ PASS |

**Total: 10/10 ✅**

---

## §5 — Score Final

| Componente | Antes | Depois |
|---|---|---|
| URL Validate | ❌ 0/10 | ✅ 10/10 |
| Body Validate | ❌ 0/10 | ✅ 10/10 |
| Response | ❌ 0/10 | ✅ 10/10 |
| Testes | ✅ 10/10 | ✅ 10/10 |
| **TOTAL** | **3.3/10** | **10/10** |

---

## §6 — Conclusão

SDK Python tinha os **mesmos 3 bugs críticos** do SDK Go. Todos corrigidos. Pronto para uso.

**Recomendação:** PUBLICAR como v1.0.1 (bug fix release).
