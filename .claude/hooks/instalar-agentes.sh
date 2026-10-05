#!/bin/bash
# Instala el plugin agentes en sesiones de Claude Code en la nube (web/móvil).
# Ahí no aparece el diálogo para confiar en el marketplace declarado en
# .claude/settings.json, así que el plugin nunca se instalaría solo.
# Los plugins se cargan antes de este hook: tras instalarlo hay que correr
# /reload-plugins para usarlo en la misma sesión.
set -u

if [ "${CLAUDE_CODE_REMOTE:-}" != "true" ]; then
  exit 0
fi

if claude plugin list 2>/dev/null | grep -q "agentes@alejandro-agentes"; then
  exit 0
fi

claude plugin marketplace add ronin410/agentes >&2 || exit 0
claude plugin install agentes@alejandro-agentes >&2 || exit 0

cat <<'JSON'
{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"El plugin agentes se acaba de instalar en esta sesión, pero aún no está cargado. En tu primera respuesta pide al usuario que escriba /reload-plugins antes de usar /agentes:* o los agentes agentes:arquitecto, agentes:desarrollador y agentes:qa-seguridad."}}
JSON
