#!/bin/sh
# sdd-session-guard: teto de contexto da sessão (spec 018 FR-4), como hook
# PostToolUse do Claude Code. Lê o JSON do hook na entrada (transcript_path), mede o
# contexto da última chamada (tokens relidos + gravados + de entrada) e, ao passar
# de SDD_SESSION_MAX_TOKENS, devolve ao agente a ordem de gravar WIP e checkpoint e
# terminar: o relé (sdd-relay.sh) abre uma sessão nova a partir do checkpoint.
#
# Uso: sdd-session-guard.sh < entrada-do-hook.json
#   Sem SDD_SESSION_MAX_TOKENS (ou sem transcrição), não faz nada.
# Códigos: sempre 0 (nunca quebra a sessão). Requer jq.
max="${SDD_SESSION_MAX_TOKENS:-}"
case "$max" in '' | *[!0-9]*) exit 0 ;; esac
command -v jq > /dev/null 2>&1 || exit 0
t="$(jq -r '.transcript_path // empty' 2> /dev/null)" || exit 0
[ -n "$t" ] && [ -f "$t" ] || exit 0
ctx="$(tail -n 400 "$t" | jq -s '[.[] | select(.message.usage) | .message.usage
  | (.cache_read_input_tokens // 0) + (.cache_creation_input_tokens // 0) + (.input_tokens // 0)] | last // 0' 2> /dev/null)" || exit 0
[ "${ctx:-0}" -ge "$max" ] || exit 0
jq -nc --arg r "Teto de contexto da sessão atingido ($ctx de $max tokens, SDD_SESSION_MAX_TOKENS). Pare o que está fazendo: faça um commit WIP na branch do ticket, push, grave o checkpoint (sdd-checkpoint.sh save com o feito e o próximo passo exato) e termine a sessão. O relé abre uma sessão nova a partir do checkpoint." \
  '{decision: "block", reason: $r}'
exit 0
