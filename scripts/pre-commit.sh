#!/usr/bin/env bash
# scripts/pre-commit.sh — pre-commit hook que roda lint + gofmt + go vet.
#
# Sprint 27: fecha gap operacional do Sprint 25 (lint-no-placeholder.sh
# rodava manual, agora roda automaticamente antes de cada commit).
#
# Uso (auto-instalado em .git/hooks/pre-commit):
#   git commit -m "msg"  # hook roda automaticamente
#   git commit --no-verify -m "msg"  # bypass (use só em emergências)
#
# O que roda:
#   1. scripts/lint-no-placeholder.sh — detecta (preencher X) em SPRINT_*.md
#   2. gofmt -l backend/ — detecta drift de formatação Go
#   3. go vet ./... — detecta constructs suspeitos
#
# Exit codes:
#   0  todos os checks passaram (commit prossegue)
#   1  algum check falhou (commit bloqueado)
#
# Instalação:
#   ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit
#   # ou rodar manualmente: ./scripts/pre-commit.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

failed=0

echo "==> [1/3] Lint: scripts/lint-no-placeholder.sh"
if ! ./scripts/lint-no-placeholder.sh; then
    failed=1
    echo "❌ FAIL: lint-no-placeholder.sh encontrou placeholders."
fi

echo ""
echo "==> [2/3] gofmt: drift check (backend/)"
drift=$(gofmt -l ./backend 2>&1 || true)
if [[ -n "$drift" ]]; then
    failed=1
    echo "❌ FAIL: drift de formatação Go detectado:"
    echo "$drift" | sed 's/^/   /'
    echo "   Fix: rodar 'gofmt -w ./backend'"
fi

echo ""
echo "==> [3/3] go vet: ./..."
if ! (cd backend && go vet ./... 2>&1); then
    failed=1
    echo "❌ FAIL: go vet encontrou problemas."
fi

echo ""
if [[ $failed -eq 0 ]]; then
    echo "✅ Todos os checks passaram. Commit prossegue."
    exit 0
fi

echo "❌ Algum check falhou. Commit bloqueado."
echo ""
echo "💡 Para bypass temporário (apenas emergências):"
echo "   git commit --no-verify -m 'msg'"
echo ""
echo "💡 Para entender cada check:"
echo "   - lint-no-placeholder.sh: ./scripts/lint-no-placeholder.sh --help"
echo "   - gofmt: https://pkg.go.dev/cmd/gofmt"
echo "   - go vet: https://pkg.go.dev/cmd/vet"

exit 1