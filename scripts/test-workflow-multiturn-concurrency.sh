#!/usr/bin/env bash
set -euo pipefail

HOST="${HOST:-http://localhost:8081}"
PROFILE="${PROFILE:-}"
CLI="${CLI:-}"
DUPLICATES="${DUPLICATES:-3}"

WORKFLOW1_ID="${WORKFLOW1_ID:-wf-1-$(date +%s)}"
WORKFLOW2_ID="${WORKFLOW2_ID:-wf-2-$(date +%s)}"

RESULT_DIR=".test-results/workflow-multiturn-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULT_DIR"

build_payload() {
  local prompt="$1"
  local workflow_id="$2"

  PROMPT="$prompt" WORKFLOW_ID="$workflow_id" PROFILE="$PROFILE" CLI="$CLI" python - <<'PY'
import json
import os

payload = {
    "prompt": os.environ["PROMPT"],
    "workflow_run_id": os.environ["WORKFLOW_ID"],
}

profile = os.environ.get("PROFILE", "")
if profile:
    payload["profile"] = profile

cli = os.environ.get("CLI", "")
if cli:
    payload["cli"] = cli

print(json.dumps(payload, ensure_ascii=False))
PY
}

parse_response() {
  local resp_file="$1"
  python - "$resp_file" <<'PY'
import json
import sys

path = sys.argv[1]
with open(path, "r", encoding="utf-8") as f:
    data = json.load(f)

answer = data.get("answer", "")
session_id = ""
response_text = ""

if isinstance(answer, str):
    try:
        inner = json.loads(answer)
    except Exception:
        inner = {}
else:
    inner = answer if isinstance(answer, dict) else {}

session_id = inner.get("session_id", "") or ""
print(session_id)
PY
}

call_chat() {
  local dir="$1"
  local label="$2"
  local prompt="$3"
  local workflow_id="$4"

  local req_file="${dir}/${label}_request.json"
  local resp_file="${dir}/${label}_response.json"
  local code_file="${dir}/${label}_http_code.txt"

  build_payload "$prompt" "$workflow_id" > "$req_file"
  http_code="$(curl -s -o "$resp_file" -w "%{http_code}" \
    -H "Content-Type: application/json" \
    -X POST "$HOST/chat" \
    -d @"$req_file")"
  echo "$http_code" > "$code_file"

  if [[ "$http_code" != "200" ]]; then
    echo ""
    return 1
  fi

  local session_id=""
  session_id="$(parse_response "$resp_file" || true)"
  echo "$session_id"
}

run_workflow() {
  local name="$1"
  local workflow_id="$2"
  shift 2
  local prompts=("$@")

  local dir="${RESULT_DIR}/${name}"
  local summary_file="${dir}/summary.txt"
  mkdir -p "$dir"
  touch "$summary_file"

  {
    echo "workflow: $name"
    echo "workflow_run_id: $workflow_id"
    echo "duplicates: $DUPLICATES"
    echo ""
  } >> "$summary_file"

  local dup_ids_file="${dir}/step1_session_ids.txt"
  > "$dup_ids_file"

  for i in $(seq 1 "$DUPLICATES"); do
    (
      session_id="$(call_chat "$dir" "${name}_step1_dup${i}" "${prompts[0]}" "$workflow_id" || true)"
      if [[ -n "$session_id" ]]; then
        echo "$session_id" >> "$dup_ids_file"
      fi
    ) &
  done
  wait

  local total_ids
  local unique_ids
  total_ids="$(wc -l < "$dup_ids_file" | tr -d ' ')"
  unique_ids="$(sort "$dup_ids_file" | uniq | wc -l | tr -d ' ')"

  {
    echo "step1_total_session_ids: $total_ids"
    echo "step1_unique_session_ids: $unique_ids"
  } >> "$summary_file"

  local expected_session_id=""
  if [[ "$total_ids" != "0" ]]; then
    expected_session_id="$(head -n 1 "$dup_ids_file")"
  fi

  if [[ -n "$expected_session_id" ]]; then
    echo "step1_expected_session_id: $expected_session_id" >> "$summary_file"
  fi
  echo "" >> "$summary_file"

  local step_index=2
  while [[ $step_index -le ${#prompts[@]} ]]; do
    local prompt="${prompts[$((step_index - 1))]}"
    local label="${name}_step${step_index}"
    local session_id
    session_id="$(call_chat "$dir" "$label" "$prompt" "$workflow_id" || true)"

    local match="unknown"
    if [[ -n "$expected_session_id" && -n "$session_id" ]]; then
      if [[ "$session_id" == "$expected_session_id" ]]; then
        match="match"
      else
        match="mismatch"
      fi
    fi

    {
      echo "step${step_index}_session_id: $session_id"
      echo "step${step_index}_match: $match"
      echo ""
    } >> "$summary_file"

    step_index=$((step_index + 1))
  done
}

run_workflow "session1" "$WORKFLOW1_ID" \
  "你好，100字介绍一下kafka" \
  "你刚说了什么？" &

run_workflow "session2" "$WORKFLOW2_ID" \
  "你好，100字介绍sql" \
  "介绍sql注入100字" \
  "你刚说了什么？" &

wait

cat "${RESULT_DIR}/session1/summary.txt" "${RESULT_DIR}/session2/summary.txt" > "${RESULT_DIR}/summary.txt"

echo "✅ 测试完成，汇总请查看: ${RESULT_DIR}/summary.txt"
