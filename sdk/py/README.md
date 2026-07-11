# Radiant Norma Python SDK

Official Python client for [Radiant Norma API](https://api.radiantrisk.com) — BACEN CADOC generation and validation.

## Install

```bash
pip install radiant-norma
```

Requires Python 3.9+.

## Quick Start

```python
from radiant import Client

c = Client(
    base_url="https://api.radiantrisk.com/v1",
    auth_token="your-jwt-token",
)

# Health check
health = c.healthz()
print(health.status, health.version)

# List schemas with generation metadata
schemas = c.list_schemas_v2()
for s in schemas.schemas:
    print(s.cadoc_code, s.latest_version)

# Generate a CADOC
resp = c.generate_cadoc(
    cadoc="3040",
    request={
        "cadoc_code": "3040",
        "if_id": "if_demo",
        "cnpj": "12.345.678/0001-90",
        "data_base": "2026-06-30T00:00:00Z",
        "participantes": [
            {"id": "P001", "tipo": "PF", "nome": "Joao Silva",
             "cpf": "123.456.789-00", "rating": "AA"}
        ],
    },
)
print(resp.status, resp.generated.xml_hash if resp.generated else None)

# Validate generated XML
result = c.validate(cadoc="3040", xml=xml_content)
print("passed:", result.passed, "errors:", len(result.errors))

# Cross-doc validation
cross = c.crossdoc_validate(documents=[
    {"cadoc_code": "4111", "xml": xml4111},
    {"cadoc_code": "3040", "xml": xml3040},
])
print("cross-doc passed:", cross.passed, "rules executed:", cross.rules_executed)
```

## Context Manager

```python
with Client(base_url=BASE, auth_token=TOKEN) as c:
    health = c.healthz()
```

## Error Handling

```python
from radiant import Client
from radiant.exceptions import HTTPError

c = Client(base_url=BASE, auth_token=TOKEN)
try:
    c.get_schema("9999")
except HTTPError as e:
    print(e.status_code, e.code, e.message)
```

## OpenAPI

The SDK mirrors [OpenAPI v3.36.2 spec](../../docs/openapi/v1.yaml).
Coverage:

| Group | Methods |
|---|---|
| Health | `healthz`, `readyz` |
| Schema | `list_schemas`, `list_schemas_v2`, `get_schema`, `list_versions` |
| Rules | `list_rules`, `list_rules_by_cadoc` |
| Validation | `validate` |
| Generation | `generate_cadoc`, `list_generate_fields`, `list_source_adapters`, `generate_batch`, `list_generate_history` |
| CrossDoc | `list_crossdoc_rules`, `crossdoc_validate` |
| STA | `sta_submit`, `sta_disponiveis`, `sta_situacao` |
| Radar | `list_radar_alerts`, `get_radar_alert`, `resolve_radar_alert`, `trigger_radar_scan` |
| L4/Audit | `l4_compare`, `list_envios`, `audit_log` |

## Tests

```bash
pip install pytest responses
PYTHONPATH=. pytest tests/ -v
```

All 20 smoke tests use `responses` to mock HTTP — no live server needed.
