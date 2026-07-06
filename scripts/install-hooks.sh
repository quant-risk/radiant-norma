#!/usr/bin/env bash
# scripts/install-hooks.sh — instala git hooks do repo (pre-commit).
#
# Sprint 27: setup automatizado para pre-commit hook (lint + gofmt + vet).
# Idempotente — pode rodar múltiplas vezes.
#
# Uso:
#   ./scripts/install-hooks.sh

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_DIR="$REPO_ROOT/.git/hooks"
PRE_COMMIT_HOOK="$HOOKS_DIR/pre-commit"

if [[ ! -d "$HOOKS_DIR" ]]; then
    echo "❌ ERRO: $HOOKS_DIR não existe (não é um repo git?)."
    exit 1
fi

if [[ -f "$PRE_COMMIT_HOOK" && ! -L "$PRE_COMMIT_HOOK" ]]; then
    echo "⚠️  ATENÇÃO: $PRE_COMMIT_HOOK já existe como arquivo (não symlink)."
    echo "   Pre-commit hook customizado vai ser sobrescrito."
    echo "   Backup automático: ${PRE_COMMIT_HOOK}.bak"
    cp "$PRE_COMMIT_HOOK" "${PRE_COMMIT_HOOK}.bak"
fi

# Cria symlink (relativo para portabilidade)
ln -sf "../../scripts/pre-commit.sh" "$PRE_COMMIT_HOOK"
chmod +x "$PRE_COMMIT_HOOK"

echo "✅ Pre-commit hook instalado em $PRE_COMMIT_HOOK"
echo ""
echo "Hook roda automaticamente antes de cada commit:"
echo "  1. scripts/lint-no-placeholder.sh — detecta placeholders em SPRINT_*.md"
echo "  2. gofmt -l backend/ — detecta drift de formatação Go"
echo "  3. go vet ./... — detecta constructs suspeitos"
echo ""
echo "Bypass (apenas emergências): git commit --no-verify -m 'msg'"
echo "Reinstalar: ./scripts/install-hooks.sh"