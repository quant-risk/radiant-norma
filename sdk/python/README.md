# Radiant Norma Python SDK

Cliente Python oficial para a [Radiant Norma API](https://radiantnorma.com).

## Instalação

```bash
pip install radiant-norma
```

## Uso rápido

```python
from radiant import Client

c = Client(
    api_key="sk-...",
    if_id="00000",
)

# Validar um documento CADOC
xml_data = b"<Doc3040>...</Doc3040>"
result = c.cadocs.validate("3040", xml_data)
print(f"Valid: {result.valid}")
for e in result.errors:
    print(f"  [{e.code}] {e.message}")

# Radar scan
scan = c.radar.scan("3040")
print(f"Radar status: {scan.status}")

# Listar regras
rules = c.audit.list_rules("3040")
for r in rules:
    print(f"  [{r.code}] {r.message}")
```

## Configuração

```python
c = Client(
    api_key="sk-...",           # obrigatório
    if_id="00000",              # IF-ID do tenant
    base_url="https://api.radiantnorma.com",  # opcional
    timeout=30,                 # segundos, opcional
)
```

## Serviços

| Serviço | Métodos |
|---|---|
| `cadocs` | `validate`, `validate_cross_doc` |
| `audit` | `list_rules` |
| `radar` | `scan` |
| `insights` | `ask` |
| `schemas` | `list_versions`, `get_changelog` |

## Validação de documento

```python
result = c.cadocs.validate("3040", xml_data)
if not result.valid:
    for e in result.errors:
        print(f"ERRO [{e.code}]: {e.message}")
    for w in result.warnings:
        print(f"AVISO [{w.code}]: {w.message}")
```

## Validação cross-document

```python
result = c.cadocs.validate_cross_doc({
    "3040": xml3040,
    "4111": xml4111,
})
print(f"Passed: {result.passed}")
```

## Radar scan

```python
scan = c.radar.scan("3040")
if scan.status == "changed":
    for ch in scan.changes:
        print(f"  {ch.kind} {ch.tag}.{ch.attr}: {ch.old_value} → {ch.new_value}")
```

## Schema registry

```python
versions = c.schemas.list_versions("3040")
for v in versions:
    print(f"v{v.id} effective={v.effective_from}")

changelog = c.schemas.get_changelog("3040")
for e in changelog:
    print(f"  {e.effective_from}: {e.changelog}")
```

## Roadmap

Consulte [ROADMAP.md](../../ROADMAP.md) para sprints futuros.

## License

Proprietary — Radiant Financial Technology
