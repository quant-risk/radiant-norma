#!/usr/bin/env bash
# Lint check — Sprint 17 — v3.7.0 [S17.6]
#
# Garante que handlers que aceitam if_id/cnpj no payload JSON
# também chamam enforceSameIF() pra prevenir cross-tenant injection.
#
# Heurística (best-effort, conservadora):
#   Flag arquivo SE atender TODOS:
#     1. Tem struct field com `json:"if_id"` ou `json:"cnpj"` (input field)
#     2. Tem json.Unmarshal/decodeJSONStrictly no MESMO ARQUIVO
#     3. NÃO chama enforceSameIF
#
# Limitação conhecida (documentada):
#   - Output structs (auditEventDTO) também têm json tag e podem
#     disparar false positive se arquivo também tem json.Unmarshal
#     pra parsear payload de DB (ex: sprint8c_handlers.go audit events).
#   - Heurística pode gerar false positives; revise manualmente antes
#     de tratar flag como bug.
#
# Uso: ./scripts/lint-enforce-same-if.sh [PATH]
# Exit: 0 = OK, 1 = algum handler sem enforceSameIF.

set -e

TARGET="${1:-internal/api}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(dirname "$SCRIPT_DIR")"

cd "$BACKEND_DIR"

echo "lint-enforce-same-if: scanning $TARGET for handlers missing enforceSameIF"
echo ""

HANDLERS=$(grep -rln 'func.*ResponseWriter.*\*http\.Request' "$TARGET" --include="*.go" 2>/dev/null | grep -v _test.go || true)

if [ -z "$HANDLERS" ]; then
    echo "AVISO: nenhum handler encontrado em $TARGET"
    exit 0
fi

fail=0
for file in $HANDLERS; do
    has_input_field=$(grep -E '`json:"if_id"`|`json:"cnpj"`|`json:"IFID"`|`json:"CNPJ"`' "$file" 2>/dev/null | wc -l | tr -d ' ')
    has_unmarshal=$(grep -E 'json\.Unmarshal|decodeJSONStrictly' "$file" 2>/dev/null | wc -l | tr -d ' ')
    has_enforce=$(grep -E 'enforceSameIF' "$file" 2>/dev/null | wc -l | tr -d ' ')

    # Skip arquivos com marcador explícito de false positive documentado.
    # Use `// lint-enforce-same-if: false-positive — <razão>` no source.
    is_known_fp=$(grep -cE 'lint-enforce-same-if:\s*false-positive' "$file" 2>/dev/null || echo 0)

    if [ "${has_input_field:-0}" -gt 0 ] && [ "${has_unmarshal:-0}" -gt 0 ] && [ "${has_enforce:-0}" -eq 0 ]; then
        if [ "${is_known_fp:-0}" -gt 0 ]; then
            echo "⚠ SKIP: $file — false positive documentado (ver comentário 'lint-enforce-same-if: false-positive')"
        else
            echo "❌ FAIL: $file — aceita if_id/CNPJ do payload mas NÃO chama enforceSameIF"
            grep -nE '`json:"if_id"`|json\.Unmarshal|decodeJSONStrictly' "$file" | head -5
            echo "   (verifique se é false positive — se sim, adicione comentário 'lint-enforce-same-if: false-positive — <razão>')"
            fail=1
        fi
    fi
done


if [ $fail -eq 0 ]; then
    echo "✅ OK: handlers que parseiam if_id/CNPJ do payload chamam enforceSameIF"
    exit 0
else
    echo ""
    echo "Lint falhou. Para cada FAIL: confirme se é bug real antes de adicionar enforceSameIF()."
    echo "Se for false positive (output struct / DB payload), adicione comentário 'lint false positive' explicando."
    exit 1
fi
