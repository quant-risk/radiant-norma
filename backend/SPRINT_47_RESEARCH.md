# Sprint 47 — RESEARCH.md

## DRSACResearch — Solicitação Formal BACEN

**Sprint:** 47
**Tema:** DRSACResearch — Solicitar formalmente ao BACEN as especificações oficiais (XSD + críticas)
**Período:** 2026-07-07
**Versão alvo:** v3.34.28

---

## 1. O que é DRSAC

**Documento:** CADOC 2030 — Documento de Riscos Social, Ambiental e Climático
**Sistema:** STA (Sistema de Transmissão de Arquivos do BACEN)
**Base legal:** Resolução CMN nº 4.945/21 (PRSAC) e Resolução CMN nº 4.557/17 (GIR)

**Finalidade:** Captura dados sobre riscos sociais, ambientais e climáticos das exposições de instituições financeiras em operações de crédito e TVM (Títulos e Valores Mobiliários) e seus respectivos devedores.

**Instituições obrigadas:** S1 a S4 (IPs inclusos). S5 isento.
**Frequência:** Semestral (bases de junho e dezembro)
**Prazo:** 10º dia útil do segundo mês subsequente à base
**Modo de envio:** Um arquivo por conglomerado prudencial, enviado pela instituição líder

---

## 2. Estrutura do Documento XML

### Tags principais

```xml
<DocumentoDRSAC cnpj="XXXXXXXX" dataBase="AAAA-MM" codigoDocumento="2030" tipoEnvio="I">
  <Contato nome="..." fone="..." email="..."/>
  <Clientes>
    <Cliente ident="..." tipo="..." CNAE="..." versaoCNAE="...">
      <ExpAtivos>
        <ExpOperCred IPOC="..." Sicor="S|N" saldo="...">
          <RiscSoc tipo=".." av=".."/>
          <RiscAmb tipo=".." av=".."/>
          <RiscClimFis tipo=".." av=".."/>
          <RiscClimTrans tipo=".." av=".."/>
          <ContribPositiva enquad=".."/>
          <MitRiscClimFis exist=".."/>
          <HistAbsorEmissGEE tipo=".." sit=".." valor=".."/>
          <CompEmissGEE tipo=".." sit=".." valor=".."/>
          <LocalizCoord>...</LocalizCoord> | <LocalizCEP CEP="..."/> | <LocalizMun codMun="..."/> | <LocalizPais codPais="..."/>
        </ExpOperCred>
        <ExpTVM sisReg="..." tipo="..." ident="..." valor="...">
          <!-- mesmo child tags que ExpOperCred -->
        </ExpTVM>
      </ExpAtivos>
      <ExpCliente>
        <RiscSoc tipo=".." av=".."/>
        <RiscAmb tipo=".." av=".."/>
        <RiscClimFis tipo=".." av=".."/>
        <RiscClimTrans tipo=".." av=".."/>
        <DetContribPositiva enquad=".." saldoCred=".." saldoTVM=".."/>
        <HistAbsorEmissGEE tipo=".." sit=".." valor=".."/>
        <ExpAbsorEmissGEE tipo=".." sit=".." valor=".."/>
        <CompEmissGEE tipo=".." sit=".." valor=".."/>
        <AgrMit tipo=".." sit=".."/>
      </ExpCliente>
    </Cliente>
  </Clientes>
  <Setores>
    <ExpSetor CNAE="..." versaoCNAE="...">
      <RiscSoc tipo=".." av=".."/>
      <RiscAmb tipo=".." av=".."/>
      <RiscClimFis tipo=".." av=".."/>
      <RiscClimTrans tipo=".." av=".."/>
    </ExpSetor>
  </Setores>
  <SetoresRestritos>
    <SetorRestrito CNAE="..." versaoCNAE="..." restricao=".."/>
  </SetoresRestritos>
</DocumentoDRSAC>
```

### Três Níveis de Análise

| Nível | Descrição | Tag |
|---|---|---|
| **Setor** | Classificação econômica por CNAE — apenas para pessoas jurídicas | `<ExpSetor>` |
| **Cliente** | Avaliação individual de devedores | `<ExpCliente>` |
| **Operação** | Operações de crédito específicas ou TVM | `<ExpOperCred>` ou `<ExpTVM>` |

### Dimensões de Risco

| Dimensão | Tag | Anexo (Tipos) | Anexo (Avaliação) |
|---|---|---|---|
| Risco Social | `<RiscSoc>` | Anexo 06 | Anexo 09 |
| Risco Ambiental | `<RiscAmb>` | Anexo 07 | Anexo 09 |
| Risco Climático Físico | `<RiscClimFis>` | Anexo 08 | Anexo 09 |
| Risco Climático de Transição | `<RiscClimTrans>` | Anexo 18 | Anexo 09 |

### Anexos do DRSAC

| Anexo | Conteúdo |
|---|---|
| 01 | Tipo de envio: I=Inclusão, S=Substituição |
| 02 | Sistema de registro TVM: B3, CERC, CSDBR, Outro |
| 03 | Tipo de TVM: CPR, CDCA, CRA, DEB, Outro |
| 04 | Registro Sicor: S/N |
| 05 | Tipo de cliente: 1=PF, 2=PJ, 3-6=especiais |
| 06 | Tipos de Risco Social (01-05, 99) |
| 07 | Tipos de Risco Ambiental (01-09, 99) |
| 08 | Tipos de Risco Climático Físico (01-03, 99) |
| 09 | Avaliação de Risco: 01=Alto, 02=Médio, 03=Baixo, 04=Irrelevante, 98=Não avaliado, 99=Fora do escopo |
| 10 | Classificação Contribuição Positiva: 01, 02, 03, 98, 99 |
| 11 | Mitigador Risco Climático Físico: 01=Existe, 02=Não existe, 98, 99 |
| 12 | Tipos de Histórico de Absorção/Emissão GEE |
| 13 | Tipos de Expectativa de Absorção/Emissão GEE |
| 14 | Tipos de Compensação de Emissão GEE |
| 15 | Status Info GEE: 01=Absorção, 02=Emissão, 98, 99 |
| 16 | Tipos de Fatores Agravantes/Mitigadores (01-10) |
| 17 | Status Aggravante/Mitigador: 01=Existe, 02=Não existe, 98, 99 |
| 18 | Tipos de Risco Climático de Transição (01-04, 99) |
| 19 | Códigos de País (ISO 3166-1 numérico) |
| 20 | Tipos de Restrição Econômica: 01=Social, 02=Ambiental |

---

## 3. Comparação com Outros CADOCs

| Aspecto | DRSAC (2030) | SCR (3040) | 3050 (TXB) |
|---|---|---|---|
| **Propósito** | Riscos Social/Ambiental/Climático | Registro de operações de crédito | Operações de tesouraria |
| **Nível de análise** | 3 níveis (Setor/Cliente/Operação) | Somente operação | Somente operação |
| **Foco** | Avaliação de riscos, emissões GEE, mitigadores | Detalhes completos da operação | Transações de tesouraria |
| **Avaliação de risco** | Qualitativa (escala 01-04) + 98/99 | N/A | N/A |
| **Dados GEE** | Sim (Histórico, Esperado, Compensado) | Não | Não |
| **Dados de localização** | Coordenadas, CEP, Município, País | Limitado | N/A |
| **Cobertura TVM** | Sim (CPR, CDCA, CRA, DEB) | Não | Sim |
| **Frequência** | Semestral | Mensal | Mensal |
| **XSD disponível** | **NÃO** (não está no repositório) | Sim (SCR3045.xsd) | Sim |
| **Regras de validação** | **NÃO** (precisa solicitar ao BACEN) | Manual_Validador_SCR3040.pdf | 3050_Criticas_TXB_V11.xlsx |

---

## 4. O que existe no Repositório

| Arquivo | Status |
|---|---|
| `2030-DRSAC/Leiaute_DRSAC.xlsx` | ✅ Layout detalhado com campos |
| `2030-DRSAC/2030_DRSAC_Leiaute.xlsx` | ✅ Mesmo conteúdo |
| `2030-DRSAC/Instrucoes_Preenchimento_DRSAC.pdf` | ✅ Instruções de preenchimento |
| `2030-DRSAC/Perguntas_Respostas_DRSAC.pdf` | ✅ FAQ do BACEN |
| `_normativos/IN_BCB_222_DRSAC.pdf` | ⚠️ Arquivo corrompido (página HTML) |
| `_normativos/IN_BCB_328_DRSAC.pdf` | ⚠️ Arquivo corrompido (página HTML) |
| **XSD Schema** | ❌ **NÃO EXISTE** — precisa solicitar |
| **Críticas/Validações** | ❌ **NÃO EXISTE** — precisa solicitar |

---

## 5. Lacunas Críticas para o Radiant Norma

### 5.1 XSD Schema — FALTANDO
O arquivo XSD de validação estrutural do DRSAC **não existe** no repositório.
Todos os outros CADOCs (3040, 3050, 4111) têm XSD oficial.
O BACEN disponibiliza em: https://www.bcb.gov.br/estabilidadefinanceira/leiautedocumentoscrd

**Ação Sprint 47:** Solicitar formalmente ao BACEN (via STA ou portal) o XSD mais recente do DRSAC.

### 5.2 Regras de Validação (Críticas) — FALTANDO
Diferente do SCR (SCR3040_Criticas.xls) e 3050 (3050_Criticas_TXB_V11.xlsx), não há documento de críticas para DRSAC no repositório.
As instruções mencionam: "Ver documento sobre críticas e validações disponível na página de leiautes do DRSAC".

**Ação Sprint 47:** Incluir request do documento de críticas junto com o XSD.

### 5.3 Atualização da IN_BCB
O "Histórico de Alterações" no Excel mostra ajustes pendentes de IN BCB de 2023, incluindo:
- `<Clientes>` e `<ExpCliente>` tornaram-se opcionais (a partir de dez/23)
- Inclusão do atributo `versaoCNAE`
- Redução do ID do cliente de 40 para 14 caracteres

### 5.4 Arquivos IN_BCB Corrompidos
Os arquivos `_normativos/IN_BCB_222_DRSAC.pdf` e `IN_BCB_328_DRSAC.pdf` são páginas HTML (erro 404), não PDFs reais.

---

## 6. Plano de Ação Sprint 47

### 6.1 Solicitar XSD e Críticas ao BACEN

**Canais oficiais:**
1. Portal do BACEN: https://www.bcb.gov.br/estabilidadefinanceira/leiautedocumentoscrd
2. STA: https://sta.bcb.gov.br/sta/dologin
3. CRD: https://www3.bcb.gov.br/crd
4. Validador XML: https://www.bcb.gov.br/estabilidadefinanceira/validador_xml_info

**Contenido da solicitação:**
- XSD mais recente do Leiaute DRSAC (documento 2030)
- Documento de críticas e validações (análogo ao SCR3040_Criticas e 3050_Criticas_TXB)
- Último histórico de alterações (IN_BCB atualizada)

### 6.2 Estrutura de Código Proposta (para Sprint 49+)

```
internal/drsac/
  parser.go        — Parse XML DRSAC → struct
  validator.go     — Validações de domínio e regras de negócio
  rules.go         — Regras de validação DRSAC (anexos 06-20)
  drsac.go         — Serviço principal (Integrate com engine de validação)
  drsac_test.go
```

### 6.3 Domainos de Validação (pré-pesquisa)

**Estrutural (XSD):**
- XML bem formado, encoding correto
- Tags obrigatórias presentes
- Tipos de atributos válidos

**Domínio (anexos):**
- `tipoEnvio`: I | S
- `Sicor`: S | N
- `av` (avaliação): 01-04, 98, 99 (Anexo 09)
- `tipo` (risco): valores por anexo (06, 07, 08, 18)
- `CNAE`: 7 dígitos numéricos
- `codPais`: ISO 3166-1 numérico

**Regras cross-field:**
1. **Consistência 98/99**: Se código "99" usado em qualquer fator, deve ser usado para esse mesmo fator em TODOS os registros
2. **Consistência 98**: Se código "98" usado, o mesmo fator deve ter valor diferente de "98" ou "99" em pelo menos um registro
3. **GEE condicional**: Valores GEE só obrigatórios quando situação (Anexo 15) = 01 ou 02
4. **CNAE obrigatório para PJ**: CNAE e versaoCNAE obrigatórios quando tipo cliente = 02
5. **Coordenadas**: Latitude (-34° a +06°), Longitude (-074° a -030°), Altitude (-100m a 3000m)
6. **IPOC**: Deve ser idêntico ao IPOC reportado no Documento 3040 (SCR)

---

## 7. Dependências para Sprints Futuros

| Sprint | Dependência |
|---|---|
| **49** (DRSAC_v1) | XSD + críticas do BACEN |
| **52** (CrossDoc_DRSAC) | Parser DRSAC + parser 3040 + parser 4111 |

---

## 8. Riscos

| Risco | Mitigação |
|---|---|
| BACEN não responde solicitação em tempo hábil | Buscar XSD em versões anteriores disponíveis publicamente |
| XSD muda entre now e Sprint 49 | Versionar XSD no schema_versions assim que obtido |
| IN_BCB corrompida impede参考 | Baixar diretamente do portal BACEN |
