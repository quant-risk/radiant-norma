"""Extrai críticas do PDF 2060 DRM via regex no texto já extraído."""

import json
import re
from pathlib import Path
from datetime import datetime

PDF_TXT = Path("/tmp/_drm_criticas.txt")
OUT = Path("/Users/henrique/Downloads/cadocs/_catalogos/drm_criticas_raw.json")

content = PDF_TXT.read_text(encoding="utf-8")
lines = content.split("\n")

# Regex: linha começa com código numérico + 2060 + tipo + descrição + base
pattern = re.compile(
    r"^\s*(\d{4,5})\s+2060\s+(Inconsistência|Erro|Aviso)\s+(.+?)\s{2,}(Cosif|DRM|Ambos)\s*$"
)

criticas = []
for line in lines:
    m = pattern.match(line)
    if not m:
        # Tentar pegar descrição que pode ter quebrado em múltiplas linhas
        m2 = re.match(r"^\s*(\d{4,5})\s+2060\s+(Inconsistência|Erro|Aviso)\s+", line)
        if m2:
            # Coletar até encontrar base confrontada no fim
            codigo = m2.group(1)
            tipo = m2.group(2)
            # A linha inteira é a descrição (até o final da entrada)
            desc = line[m2.end():].strip()
            # Remove espaços múltiplos
            desc = re.sub(r"\s+", " ", desc)
            criticas.append({
                "cadoc": "2060-DRM",
                "codigo": codigo,
                "tipo": tipo,
                "descricao": desc,
                "base_confrontada": None,  # difícil extrair
                "fonte": "Criticas_Pos_Processamento_2060_V2_Jan25.pdf",
            })
            continue
        continue

    codigo, tipo, desc, base = m.groups()
    # Limpa descrição (remove múltiplos espaços)
    desc = re.sub(r"\s+", " ", desc).strip()

    criticas.append({
        "cadoc": "2060-DRM",
        "codigo": codigo,
        "tipo": tipo,
        "descricao": desc,
        "base_confrontada": base,
        "fonte": "Criticas_Pos_Processamento_2060_V2_Jan25.pdf",
    })

# Deduplicação por código (alguns aparecem duplicados)
seen = set()
criticas_unique = []
for c in criticas:
    if c["codigo"] not in seen:
        seen.add(c["codigo"])
        criticas_unique.append(c)

# Adiciona metadata
output = {
    "_metadata": {
        "description": "Críticas do CADOC 2060 (DRM) extraídas do PDF oficial",
        "fonte": "Criticas_Pos_Processamento_2060_V2_Jan25.pdf (BACEN/Desig, Jan/2025)",
        "data_extracao": datetime.now().isoformat(timespec="seconds"),
        "autor": "Mavis · Radiant",
        "uso": "Norma Audit L2 — DRM validation",
        "total_criticas": len(criticas_unique),
    },
    "criticas": criticas_unique,
}

OUT.write_text(json.dumps(output, ensure_ascii=False, indent=2), encoding="utf-8")
print(f"✓ {len(criticas_unique)} críticas extraídas")
print(f"  Codigos: {[c['codigo'] for c in criticas_unique]}")
print(f"  Salvo em: {OUT}")