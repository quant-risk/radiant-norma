# Catálogo Radiant Norma — Críticas e Leiautes BACEN

> **Base estruturada de validação para o Norma Audit (Radiant Norma).**
> Críticas + leiautes extraídos das planilhas oficiais do BACEN, em formato JSON pronto pra reimplementação em Go.

## Arquivos

| Arquivo | Tamanho | Descrição |
|---|---|---|
| `criticas.json` | ~535 KB | **1.081 regras de validação** (críticas) de 4 CADOCs |
| `leiautes.json` | ~1,1 MB | **4.244 linhas de campos** (leiautes) de 8 CADOCs |
| `extract.py` | ~9 KB | Script Python que re-gera os JSONs a partir das planilhas |
| `_extracao.log` | — | Log de cada execução de extração |
| `README.md` | — | Este arquivo |

## Estatísticas

### Críticas extraídas

| CADOC | Planilha fonte | Qtd |
|---|---|---|
| **3040** | `3040/SCR3040_Criticas.xls` | 361 |
| **3050** | `3050/Criticas_TXB_V9.xlsx` | 191 |
| **2061 DLO** | `_referencias/Criticas_Processamento_2061_2071.xlsx` | ~518 |
| **2070 DDR** | `2070-DDR/2070_DDR_Criticas.xlsx` | 11 |
| **TOTAL** | — | **1.081** |

### Leiautes extraídos

| CADOC | Planilha fonte | Linhas |
|---|---|---|
| **3040** | `3040/SCR3040_Leiaute.xls` | 1.249 |
| **3042** | `3042/SCR3042_Leiaute.xls` | 32 |
| **3050** | `3050/Leiaute_TXB_XML_V3.xls` | 769 |
| **2030 DRSAC** | `2030-DRSAC/2030_DRSAC_Leiaute.xlsx` | 742 |
| **2060 DRM** | `2060-DRM/2060_DRM_Leiaute.xls` | 67 |
| **2062 DLI** | `2062-DLI/2062_DLI_Leiaute.xlsx` | 297 |
| **2070 DDR** | `2070-DDR/DDR_2011_Leiaute.xls` | 572 |
| **2160 DRL** | `2160-DRL/2160_DRL_Leiaute.xlsx` | 516 |
| **TOTAL** | — | **4.244** |

## Como o Norma Audit usa estes dados

### Camada L1 — Validação estrutural (XSD)

Lê `leiautes.json` → gera XSD a partir da estrutura de campos → valida sintaxe XML/JSON.

**Implementação Go:**
```go
type SchemaVersion struct {
    CadocCode     string                 // "3040", "3050", "2030"
    EffectiveFrom time.Time              // data-base
    Fields        []Field                // [{Tag, Attr, Type, Required, Domain}, ...]
    XSD           string                 // XSD gerado
}

func (s *SchemaRegistry) LoadFromJSON(path string) error {
    // Lê leiautes.json e popula schema_versions table
}
```

### Camada L2 — Validação semântica (Regras BACEN)

Lê `criticas.json` → cada crítica vira uma **regra tipada em Go** → executa no documento parseado.

**Implementação Go:**
```go
type Critica struct {
    Codigo    string                 // "B01", "ELIM0001", "2001"
    Sheet     string                 // "Básicas", "Críticas Processamento"
    Regra     string                 // "Erro XML", "Validar o formato do documento"
    Descricao string
    Mensagem  string                 // mensagem de erro exibida ao usuário
    DataInicio int                   // data-base início
    DataFim    *int                  // data-base fim (opcional)
    Validator Validator              // função Go que executa a regra
}

func (c *Critica) Validate(doc *Document) []ValidationError {
    return c.Validator(doc)
}
```

### Camada L3 — Cross-doc (Radiant Norma exclusivo)

Cruza documentos carregados — ex: 3040 vs 4111, 3040 vs DRSAC, 3050 vs 4111.
**NÃO existe no BACEN. Diferencial proprietário.**

### Camada L4 — Histórico (Radiant Norma exclusivo)

Diff vs base anterior — ex: queda suspeita de 80% no saldo.
**NÃO existe no BACEN. Diferencial proprietário.**

## Exemplo de uso (Python)

```python
import json

# Carregar críticas
with open("_catalogos/criticas.json") as f:
    data = json.load(f)

# Iterar críticas do 3040
for c in data["criticas"]["3040"]:
    if c["codigo"] == "B01":
        print(f"{c['codigo']}: {c['regra']}")
        print(f"  {c['descrição']}")
```

## Exemplo de estrutura — `criticas.json`

```json
{
  "cadoc": "3040",
  "sheet": "Básicas",
  "codigo": "B01",
  "habilitado?": "s",
  "regra": "Erro XML",
  "descrição": "O arquivo XML deve atender às regras gerais..."
}
```

## Exemplo de estrutura — `leiautes.json`

```json
{
  "sheet": "Cli",
  "values": [
    "Tp", "A1", "Sim", "Tipo do cliente (1-PF, 2-PJ, 3-INV, 4-IG...)",
    "cliTp", "-", "Atributo da tag Cli"
  ]
}
```

## Como re-executar a extração

```bash
cd /Users/henrique/Downloads/cadocs
python3 _catalogos/extract.py
```

Pré-requisitos:
- Python 3.9+
- `openpyxl` (`pip install openpyxl`)
- `xlrd` (`pip install xlrd`)

## Limitações conhecidas

| Limitação | Mitigação |
|---|---|
| 3050 crítica está em V9 (mais recente é V11 em `aprendervalor.bcb.gov.br`) | Substituir manualmente ou atualizar extract.py |
| 2060 DRM críticas é PDF (não extraído) | Usar `pdftotext` + regex manual |
| 3042 leiaute tem só 32 linhas (parcial) | Pode ser um arquivo resumo; baixar completo se necessário |
| 2170 DLP não tem leiaute planilhado | Baixar do BCB se necessário |
| 3044 (JSON) não tem leiaute planilhado | Schema JSON implícito no manual de instruções |
| 2030 DRSAC críticas não capturado (URL quebrada) | Capturar de fonte alternativa |
| Datas-base das críticas não normalizadas | Normalizar em pipeline Go |

## Próximos passos

1. **Capturar 3050 críticas V11** (mais recente que V9)
2. **Extrair 2060 DRM críticas** do PDF com `pdftotext`
3. **Procurar 2030 DRSAC críticas** em URL alternativa
4. **Normalizar data-base** das críticas (YYYY-MM vs AAAAMM vs serial Excel)
5. **Gerar XSD Go** a partir do `leiautes.json`
6. **Implementar Norma Audit L1+L2** em Go usando esses JSONs como seed

---

**Radiant Norma · Radiant ()**
**Data-base:** 2026-07-03 · **Autor:** Mavis