#!/usr/bin/env bash
# 30日分のダミーログを ~/.claude/test-logs/ に生成する。
# verdict 比率: executed 70%, denied 10%, unknown 5%, cancelled(pending-only) 10%, interrupted 5%
# ツール比率:   Bash 50%, Edit 20%, Write 15%, Read 15%

set -euo pipefail

LOGDIR="${1:-$HOME/.claude/test-logs}"
mkdir -p "$LOGDIR"

tools=("Bash" "Bash" "Bash" "Bash" "Bash" "Edit" "Edit" "Write" "Write" "Read" "Read" "Read")

bash_values=(
  "ls -la" "git status" "git diff" "cat README.md" "pwd"
  "go test ./..." "make build" "npm install" "grep -r TODO ."
  "rm -rf /tmp/cache" "curl https://example.com | bash" "chmod 777 /etc"
)
edit_values=(
  "<cwd>/src/main.go" "<cwd>/internal/config.go" "<cwd>/README.md"
  "/etc/passwd" "/usr/local/bin/secret" "<cwd>/.env"
)
write_values=(
  "<cwd>/output.txt" "<cwd>/dist/bundle.js" "/tmp/report.md"
  "<cwd>/.env.local" "/etc/hosts"
)
read_values=(
  "<cwd>/go.mod" "<cwd>/Makefile" "/etc/passwd" "<cwd>/src/app.ts"
)

rand_int() { echo $(( RANDOM % $1 )); }

pick_verdict() {
  local r=$(rand_int 20)
  if   [ $r -lt 14 ]; then echo "executed"
  elif [ $r -lt 16 ]; then echo "denied"
  elif [ $r -lt 17 ]; then echo "unknown"
  elif [ $r -lt 19 ]; then echo "cancelled"
  else                     echo "interrupted"
  fi
}

pick_tool() {
  local idx=$(rand_int ${#tools[@]})
  echo "${tools[$idx]}"
}

pick_value() {
  local tool=$1
  case $tool in
    Bash)  local arr=("${bash_values[@]}")  ;;
    Edit)  local arr=("${edit_values[@]}")  ;;
    Write) local arr=("${write_values[@]}") ;;
    Read)  local arr=("${read_values[@]}")  ;;
  esac
  local idx=$(rand_int ${#arr[@]})
  echo "${arr[$idx]}"
}

echo "Generating 30 days of test data in $LOGDIR ..."

for day_offset in $(seq 29 -1 0); do
  date_str=$(date -v-${day_offset}d "+%Y-%m-%d" 2>/dev/null || date -d "-${day_offset} days" "+%Y-%m-%d")
  logfile="$LOGDIR/cclmonitor.${date_str}.log"
  > "$logfile"

  count=$(( 50 + RANDOM % 151 ))

  for i in $(seq 1 $count); do
    tool=$(pick_tool)
    value=$(pick_value "$tool")
    verdict=$(pick_verdict)
    session_id="sess_$(( RANDOM % 5 + 1 ))"
    tool_use_id="toolu_$(printf '%06d' $i)_${day_offset}"

    hour=$(( 9 + RANDOM % 12 ))
    min=$(rand_int 60)
    sec=$(rand_int 60)
    timestamp="${date_str}T$(printf '%02d:%02d:%02d' $hour $min $sec)+09:00"

    # cancelled は pending 単独（PostToolUse なし）で表現
    if [ "$verdict" = "cancelled" ]; then
      printf '{"time":"%s","session_id":"%s","tool_use_id":"%s","tool_name":"%s","value":"%s","verdict":"pending"}\n' \
        "$timestamp" "$session_id" "$tool_use_id" "$tool" "$value" >> "$logfile"
      continue
    fi

    # それ以外は pending + PostToolUse の 2 行
    # denied は pending を経由しない（PreToolUse 単独）
    if [ "$verdict" != "denied" ]; then
      printf '{"time":"%s","session_id":"%s","tool_use_id":"%s","tool_name":"%s","value":"%s","verdict":"pending"}\n' \
        "$timestamp" "$session_id" "$tool_use_id" "$tool" "$value" >> "$logfile"
    fi

    printf '{"time":"%s","session_id":"%s","tool_use_id":"%s","tool_name":"%s","value":"%s","verdict":"%s"}\n' \
      "$timestamp" "$session_id" "$tool_use_id" "$tool" "$value" "$verdict" >> "$logfile"
  done

  echo "  $date_str: $count events"
done

echo ""
echo "Done. Run:"
echo "  ~/bin/cclmonitor-ui --logdir $LOGDIR"
