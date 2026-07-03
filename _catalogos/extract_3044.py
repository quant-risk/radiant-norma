"""Gera JSON schema para CADOC 3044 (JSON, IN BCB 530/2024) a partir do manual."""

import json
from datetime import datetime
from pathlib import Path

OUT = Path("/Users/henrique/Downloads/cadocs/_catalogos/leiautes_3044.json")

schema = {
    "_metadata": {
        "description": "Schema JSON do CADOC 3044 (eventos de operações de crédito)",
        "fonte": "Manual SCR_InstrucoesDePreenchimento_Doc3044.pdf (BACEN, jul/2025)",
        "data_extracao": datetime.now().isoformat(timespec="seconds"),
        "autor": "Mavis · Radiant Risk Solutions",
        "uso": "Sentinel Audit L1 — validação estrutural do JSON 3044",
        "normativo": "IN BCB 530/2024 (vigência 11/2025)",
        "formato_transporte": "JSON (não XML)",
    },
    "documento": {
        "descricao": "Documento raiz 3044",
        "campos": [
            {
                "nome": "cnpjIF",
                "tipo": "string",
                "tamanho": 8,
                "obrigatorio": True,
                "descricao": "CNPJ da entidade supervisionada remetente",
                "exemplo": "12345678",
            },
            {
                "nome": "dataHoraRemessa",
                "tipo": "string",
                "formato": "AAAA-MM-DD HH:mm:ss",
                "obrigatorio": True,
                "descricao": "Data e hora local de geração da remessa",
                "exemplo": "2026-07-03 14:30:00",
            },
            {
                "nome": "envia3050",
                "tipo": "enum",
                "valores": ["S", "N"],
                "obrigatorio": True,
                "descricao": "S se IF envia 3050, N caso contrário",
                "exemplo": "S",
            },
            {
                "nome": "operacoes",
                "tipo": "array",
                "obrigatorio": True,
                "descricao": "Lista de operações (objetos do tipo Operacao)",
                "items": "Operacao",
            },
        ],
    },
    "objetos": {
        "Operacao": {
            "campos": [
                {"nome": "acao", "tipo": "enum", "valores": [1, 2], "obrigatorio": True,
                 "descricao": "1=Incluir/alterar IPOC, 2=Excluir IPOC"},
                {"nome": "ipoc", "tipo": "string", "obrigatorio": True,
                 "descricao": "Identificação Padronizada da Operação de Crédito"},
                {"nome": "class3050", "tipo": "string", "obrigatorio": "condicional",
                 "condicao": "obrigatório se envia3050='S'; vedado se envia3050='N'",
                 "exemplo": "112212101"},
                {"nome": "saldoDevedor", "tipo": "number", "obrigatorio": True,
                 "descricao": "Saldo devedor atualizado (mesma forma do 3040 vértices)"},
                {"nome": "dataSaldoDevedor", "tipo": "string", "formato": "AAAA-MM-DD",
                 "obrigatorio": True, "descricao": "Data de apuração do saldo"},
                {"nome": "atraso", "tipo": "enum", "valores": ["S", "N"], "obrigatorio": True,
                 "descricao": "S=Operação com atraso ≥15 dias, N=Sem atraso"},
                {"nome": "pagamentos", "tipo": "array", "items": "Pagamento"},
                {"nome": "concessoes", "tipo": "array", "items": "Concessao"},
                {"nome": "cessoes", "tipo": "array", "items": "Cessao"},
                {"nome": "aquisicoes", "tipo": "array", "items": "Aquisicao"},
            ],
        },
        "Pagamento": {
            "campos": [
                {"nome": "acao", "tipo": "enum", "valores": [1, 2, 3], "obrigatorio": True,
                 "descricao": "1=Incluir, 2=Excluir (cancelar/estornar), 3=Alterar"},
                {"nome": "tpMotivo", "tipo": "string", "obrigatorio": False,
                 "valores": ["1", "2"],
                 "descricao": "1=Portabilidade, 2=Assunção de dívida"},
                {"nome": "data", "tipo": "string", "formato": "AAAA-MM-DD", "obrigatorio": True},
                {"nome": "valor", "tipo": "number", "obrigatorio": True,
                 "descricao": "Até 11 dígitos + 2 decimais"},
            ],
        },
        "Concessao": {
            "campos": [
                {"nome": "acao", "tipo": "enum", "valores": [1, 2, 3], "obrigatorio": True},
                {"nome": "tpMotivo", "tipo": "string", "obrigatorio": False,
                 "valores": ["1", "2"], "descricao": "1=Portabilidade, 2=Assunção"},
                {"nome": "data", "tipo": "string", "formato": "AAAA-MM-DD", "obrigatorio": True},
                {"nome": "valor", "tipo": "number", "obrigatorio": True},
            ],
        },
        "Cessao": {
            "campos": [
                {"nome": "acao", "tipo": "enum", "valores": [1, 2, 3], "obrigatorio": True},
                {"nome": "data", "tipo": "string", "formato": "AAAA-MM-DD", "obrigatorio": True},
                {"nome": "cdCessionario", "tipo": "string", "tamanho": 8, "obrigatorio": True,
                 "descricao": "CNPJ do cessionário (8 chars), ou 'XXXXXXXX' para PF"},
                {"nome": "valor", "tipo": "number", "obrigatorio": True,
                 "descricao": "Valor com taxa de cessão"},
            ],
        },
        "Aquisicao": {
            "campos": [
                {"nome": "acao", "tipo": "enum", "valores": [1, 2, 3], "obrigatorio": True},
                {"nome": "data", "tipo": "string", "formato": "AAAA-MM-DD", "obrigatorio": True},
                {"nome": "cdCedente", "tipo": "string", "tamanho": 8, "obrigatorio": True,
                 "descricao": "CNPJ do cedente (8 chars)"},
                {"nome": "valor", "tipo": "number", "obrigatorio": True},
            ],
        },
    },
    "regras_validacao": [
        {"codigo": "T01", "descricao": "Rejeitar se dataHoraRemessa < dataSaldoDevedor"},
        {"codigo": "T02", "descricao": "Rejeitar se houver pagamento com data > dataSaldoDevedor"},
        {"codigo": "T03", "descricao": "Rejeitar se houver concessão com data > dataSaldoDevedor"},
        {"codigo": "T04", "descricao": "Rejeitar se dataHoraRemessa é futura OU anterior a 21 dias"},
        {"codigo": "T05", "descricao": "Rejeitar se houver mais de um pagamento para o mesmo IPOC na mesma data"},
        {"codigo": "T06", "descricao": "Rejeitar se houver mais de uma concessão para o mesmo IPOC na mesma data"},
        {"codigo": "T07", "descricao": "Rejeitar se houver class3050 quando envia3050='N'"},
        {"codigo": "T08", "descricao": "Rejeitar se envia3050='S' e class3050 fora do domínio"},
        {"codigo": "T11", "descricao": "Rejeitar se data do pagamento fora dos últimos 6 meses"},
        {"codigo": "T12", "descricao": "Rejeitar se data da concessão fora dos últimos 6 meses"},
        {"codigo": "T13", "descricao": "Rejeitar se data da cessão fora dos últimos 6 meses"},
        {"codigo": "T14", "descricao": "Rejeitar se data da aquisição fora dos últimos 6 meses"},
        {"codigo": "T15", "descricao": "Rejeitar se valor negativo"},
        {"codigo": "T16", "descricao": "Rejeitar se saldoDevedor negativo (exceto anuidade/cashback)"},
        {"codigo": "T17", "descricao": "Rejeitar se IPOC duplicado no mesmo documento"},
        {"codigo": "T18", "descricao": "Rejeitar se acao=2 e IPOC não existe na base"},
        {"codigo": "T19", "descricao": "Rejeitar se acao=3 e IPOC não existe na base"},
    ],
    "exemplo_minimo": {
        "cnpjIF": "12345678",
        "dataHoraRemessa": "2026-07-03 14:30:00",
        "envia3050": "S",
        "operacoes": [
            {
                "acao": 1,
                "ipoc": "876543210216210020716C1234",
                "class3050": "112212101",
                "saldoDevedor": 5000.00,
                "dataSaldoDevedor": "2026-07-03",
                "atraso": "N",
                "pagamentos": [],
                "concessoes": [],
                "cessoes": [],
                "aquisicoes": [],
            },
        ],
    },
}

OUT.write_text(json.dumps(schema, ensure_ascii=False, indent=2), encoding="utf-8")
print(f"✓ Schema 3044 salvo em {OUT}")
print(f"  - {len(schema['documento']['campos'])} campos no documento raiz")
print(f"  - {len(schema['objetos'])} objetos: {', '.join(schema['objetos'].keys())}")
print(f"  - {len(schema['regras_validacao'])} regras de validação")