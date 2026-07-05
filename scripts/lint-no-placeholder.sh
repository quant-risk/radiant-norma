#!/usr/bin/env bash
# scripts/lint-no-placeholder.sh — detecta placeholders não-preenchidos em SPRINT_*.md
#
# Validação 45 (Sprint 25 follow-up): pattern reincidiu 2 sprints consecutivas
# (v44 F-S23-44-2 + v45 F-S24-45-9) — placeholder "(preencher após push)" ficou
# em SPRINT_RESULTS. Automatiza catching no commit time.
#
# Uso:
#   ./scripts/lint-no-placeholder.sh                 # check (exit 1 se falha)
#
# Exit codes:
#   0  OK (sem placeholder)
#   1  FAIL (placeholder encontrado)
#
# Padrões detectados:
#   - (preencher após X) onde X é qualquer coisa
#   - (fill in ...) — versão inglês
#   - (TODO: ...) em contexto de SPRINT_*.md
#
# Edge cases:
#   - Code blocks (entre ```) são IGNORADOS (legítimo documentar exemplos)
#   - VALIDAÇÃO_*.md são IGNORADOS (podem citar placeholders historicamente)
#   - Comentários inline em code blocks também são ignorados

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PATTERNS=(
    '\(preencher [^)]*\)'
    '\(fill in [^)]*\)'
    '\(TODO: [^)]*\)'
)

fail=0
checked=0

# Encontra todos os SPRINT_*.md no repo root
while IFS= read -r -d '' file; do
    basename="$(basename "$file")"

    # Pula arquivos de validação (que citam placeholders historicamente)
    if [[ "$basename" =~ VALIDATION_.*\.md ]]; then
        continue
    fi

    # Extrai linhas FORA de code blocks (entre ```).
    # awk: track in_code_block (0/1); só emite linhas quando in_code_block == 0.
    content_outside_codeblocks=$(awk '
        /^```/ { in_block = !in_block; next }
        !in_block { print NR ":" $0 }
    ' "$file")

    checked=$((checked + 1))

    for pattern in "${PATTERNS[@]}"; do
        if echo "$content_outside_codeblocks" | grep -E "$pattern" >/dev/null 2>&1; then
            echo "❌ FAIL: $basename contém placeholder não-preenchido:"
            echo "$content_outside_codeblocks" | grep -E "$pattern" | sed 's/^/   /'
            fail=1
        fi
    done
done < <(find "$REPO_ROOT" -maxdepth 1 -name 'SPRINT_*.md' -print0)

if [[ $fail -eq 0 ]]; then
    echo "✅ OK: $checked SPRINT_*.md files limpos (sem placeholders)"
    exit 0
fi

echo ""
echo "💡 Fix: substituir placeholders por valores reais antes de commitar."
echo "   Pattern reincidente v44 + v45 — automatizar catching agora."

exit 1