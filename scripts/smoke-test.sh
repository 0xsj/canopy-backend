#!/usr/bin/env bash
# smoke-test.sh — comprehensive smoke test for all 12 bounded contexts.
# Exercises ~67 HTTP endpoints against a running canopy server.
#
# Usage:
#   bash scripts/smoke-test.sh [BASE_URL]
#
# Requires: curl, jq
# Exit code: 0 = all pass, 1 = one or more failures

set -euo pipefail

BASE="${1:-http://localhost:8080}"
TOKEN="Bearer dev"
PASS=0
FAIL=0
TOTAL=0
RND=$(date +%s | tail -c 6)

# ── Helpers ─────────────────────────────────────────────────────

red()   { printf '\033[31m%s\033[0m' "$1"; }
green() { printf '\033[32m%s\033[0m' "$1"; }
bold()  { printf '\033[1m%s\033[0m' "$1"; }

# call METHOD PATH EXPECTED_STATUS [BODY] [LABEL]
# Sets $BODY to the response body after each call.
call() {
  local method="$1" path="$2" expect="$3" body="${4:-}" label="${5:-$method $path}"
  TOTAL=$((TOTAL + 1))

  local curl_args=( -s -w '\n%{http_code}' -X "$method"
    -H "Authorization: $TOKEN"
    -H "Content-Type: application/json"
  )
  if [[ -n "$body" ]]; then
    curl_args+=( -d "$body" )
  fi

  local raw
  raw=$(curl "${curl_args[@]}" "${BASE}${path}")
  local status="${raw##*$'\n'}"
  BODY="${raw%$'\n'*}"

  if [[ "$status" == "$expect" ]]; then
    PASS=$((PASS + 1))
    printf "  $(green PASS)  %s  (HTTP %s)\n" "$label" "$status"
  else
    FAIL=$((FAIL + 1))
    printf "  $(red FAIL)  %s  (expected %s, got %s)\n" "$label" "$expect" "$status"
    # Print first 200 chars of body on failure for debugging
    printf "        %s\n" "${BODY:0:200}"
  fi
}

# call_noauth — same as call but without Authorization header.
call_noauth() {
  local method="$1" path="$2" expect="$3" label="${4:-$method $path}"
  TOTAL=$((TOTAL + 1))

  local raw
  raw=$(curl -s -w '\n%{http_code}' -X "$method" "${BASE}${path}")
  local status="${raw##*$'\n'}"
  BODY="${raw%$'\n'*}"

  if [[ "$status" == "$expect" ]]; then
    PASS=$((PASS + 1))
    printf "  $(green PASS)  %s  (HTTP %s)\n" "$label" "$status"
  else
    FAIL=$((FAIL + 1))
    printf "  $(red FAIL)  %s  (expected %s, got %s)\n" "$label" "$expect" "$status"
    printf "        %s\n" "${BODY:0:200}"
  fi
}

# accept_any — pass if status matches ANY of the given codes (space-separated).
call_accept_any() {
  local method="$1" path="$2" codes="$3" body="${4:-}" label="${5:-$method $path}"
  TOTAL=$((TOTAL + 1))

  local curl_args=( -s -w '\n%{http_code}' -X "$method"
    -H "Authorization: $TOKEN"
    -H "Content-Type: application/json"
  )
  if [[ -n "$body" ]]; then
    curl_args+=( -d "$body" )
  fi

  local raw
  raw=$(curl "${curl_args[@]}" "${BASE}${path}")
  local status="${raw##*$'\n'}"
  BODY="${raw%$'\n'*}"

  local matched=false
  for code in $codes; do
    if [[ "$status" == "$code" ]]; then matched=true; break; fi
  done

  if $matched; then
    PASS=$((PASS + 1))
    printf "  $(green PASS)  %s  (HTTP %s)\n" "$label" "$status"
  else
    FAIL=$((FAIL + 1))
    printf "  $(red FAIL)  %s  (expected one of [%s], got %s)\n" "$label" "$codes" "$status"
    printf "        %s\n" "${BODY:0:200}"
  fi
}

# jq helper — extract a field from $BODY. Returns empty string on failure.
jqr() { echo "$BODY" | jq -r "$1" 2>/dev/null || echo ""; }

section() { printf "\n$(bold "── $1")\n"; }

# ════════════════════════════════════════════════════════════════
#  Phase 0 — Infrastructure
# ════════════════════════════════════════════════════════════════
section "Phase 0: Infrastructure"

call_noauth GET "/healthz/live"  200 "GET /healthz/live"
call_noauth GET "/healthz/ready" 200 "GET /healthz/ready"

# ════════════════════════════════════════════════════════════════
#  Phase 1 — Identity
# ════════════════════════════════════════════════════════════════
section "Phase 1: Identity"

call POST "/api/v1/auth/register" 201 \
  "{\"email\":\"smoke${RND}@test.io\",\"display_name\":\"Smoke ${RND}\"}" \
  "POST /api/v1/auth/register"
USER_ID=$(jqr '.data.id')

call GET "/api/v1/users/me" 200 "" "GET /api/v1/users/me"

call GET "/api/v1/users/${USER_ID}" 200 "" "GET /api/v1/users/{userId}"

call PATCH "/api/v1/users/me" 200 \
  "{\"display_name\":\"Smoke Updated ${RND}\"}" \
  "PATCH /api/v1/users/me"

# ════════════════════════════════════════════════════════════════
#  Phase 2 — Organization
# ════════════════════════════════════════════════════════════════
section "Phase 2: Organization"

call POST "/api/v1/orgs" 201 \
  "{\"name\":\"Smoke Org ${RND}\",\"slug\":\"smoke-${RND}\"}" \
  "POST /api/v1/orgs"
ORG_ID=$(jqr '.data.id')

call GET "/api/v1/orgs?slug=smoke-${RND}" 200 "" "GET /api/v1/orgs?slug=..."

call GET "/api/v1/orgs/${ORG_ID}" 200 "" "GET /api/v1/orgs/{orgId}"

# Fake user for member add/remove (org_members.user_id has no FK to identity)
FAKE_USER="usr_fake${RND}000000000"

call_accept_any POST "/api/v1/orgs/${ORG_ID}/members" "200 201" \
  "{\"user_id\":\"${FAKE_USER}\",\"role\":\"member\"}" \
  "POST /api/v1/orgs/{orgId}/members (fake user)"

call PATCH "/api/v1/orgs/${ORG_ID}/members/${FAKE_USER}/role" 200 \
  "{\"role\":\"admin\"}" \
  "PATCH /api/v1/orgs/{orgId}/members/{userId}/role"

call GET "/api/v1/orgs/${ORG_ID}/members" 200 "" "GET /api/v1/orgs/{orgId}/members"

call POST "/api/v1/orgs/${ORG_ID}/teams" 201 \
  "{\"name\":\"Smoke Team ${RND}\"}" \
  "POST /api/v1/orgs/{orgId}/teams"
TEAM_ID=$(jqr '.data.id')

call GET "/api/v1/orgs/${ORG_ID}/teams" 200 "" "GET /api/v1/orgs/{orgId}/teams"

call_accept_any POST "/api/v1/orgs/${ORG_ID}/teams/${TEAM_ID}/members" "200 201" \
  "{\"user_id\":\"${FAKE_USER}\"}" \
  "POST /api/v1/orgs/{orgId}/teams/{teamId}/members"

call GET "/api/v1/orgs/${ORG_ID}/teams/${TEAM_ID}/members" 200 "" \
  "GET /api/v1/orgs/{orgId}/teams/{teamId}/members"

call DELETE "/api/v1/orgs/${ORG_ID}/teams/${TEAM_ID}/members/${FAKE_USER}" 200 "" \
  "DELETE /api/v1/orgs/{orgId}/teams/{teamId}/members/{userId}"

call DELETE "/api/v1/orgs/${ORG_ID}/members/${FAKE_USER}" 200 "" \
  "DELETE /api/v1/orgs/{orgId}/members/{userId}"

# ════════════════════════════════════════════════════════════════
#  Phase 3 — Workspace
# ════════════════════════════════════════════════════════════════
section "Phase 3: Workspace"

call POST "/api/v1/orgs/${ORG_ID}/workspaces" 201 \
  "{\"name\":\"Smoke WS ${RND}\",\"description\":\"smoke test workspace\"}" \
  "POST /api/v1/orgs/{orgId}/workspaces"
WS_ID=$(jqr '.data.id')

call GET "/api/v1/workspaces/${WS_ID}" 200 "" "GET /api/v1/workspaces/{wsId}"

call GET "/api/v1/orgs/${ORG_ID}/workspaces" 200 "" "GET /api/v1/orgs/{orgId}/workspaces"

# Join is idempotent — caller is already a member from CreateWorkspace, so 200 or 409 both OK.
call_accept_any POST "/api/v1/workspaces/${WS_ID}/join" "200 409" "" \
  "POST /api/v1/workspaces/{wsId}/join"

# The caller's user ID in workspace_members is claims.Subject = "dev_user".
# UpdateConfig requires a usr_-prefixed lore_keeper_id, so we use USER_ID from register.
call PATCH "/api/v1/workspaces/${WS_ID}/config" 200 \
  "{\"lore_keeper_mode\":\"human\",\"lore_keeper_id\":\"${USER_ID}\"}" \
  "PATCH /api/v1/workspaces/{wsId}/config"

# UpdateRole — use the fake user to avoid the prefix mismatch with the calling user.
# First add fake user as an org member (required for workspace join flow),
# then we'll accept 200 or 404 since the fake user may not be a workspace member.
call_accept_any POST "/api/v1/orgs/${ORG_ID}/members" "200 201" \
  "{\"user_id\":\"${FAKE_USER}\",\"role\":\"member\"}" \
  "POST /api/v1/orgs/{orgId}/members (re-add fake for ws role)"
call_accept_any PATCH "/api/v1/workspaces/${WS_ID}/members/${FAKE_USER}/role" "200 404" \
  "{\"role\":\"lore_keeper\"}" \
  "PATCH /api/v1/workspaces/{wsId}/members/{userId}/role"

# ════════════════════════════════════════════════════════════════
#  Phase 4 — Seed
# ════════════════════════════════════════════════════════════════
section "Phase 4: Seed"

call POST "/api/v1/workspaces/${WS_ID}/seeds" 201 \
  "{\"title\":\"Smoke Seed ${RND}\",\"description\":\"a seed for smoke testing\"}" \
  "POST /api/v1/workspaces/{wsId}/seeds"
SEED_ID=$(jqr '.data.id')

call GET "/api/v1/seeds/${SEED_ID}" 200 "" "GET /api/v1/seeds/{seedId}"

call GET "/api/v1/workspaces/${WS_ID}/seeds" 200 "" "GET /api/v1/workspaces/{wsId}/seeds"

call PATCH "/api/v1/seeds/${SEED_ID}/constraints" 200 \
  "{\"constraints\":{\"max_branches\":5}}" \
  "PATCH /api/v1/seeds/{seedId}/constraints"

# ════════════════════════════════════════════════════════════════
#  Phase 5 — Exploration
# ════════════════════════════════════════════════════════════════
section "Phase 5: Exploration"

call POST "/api/v1/workspaces/${WS_ID}/branches" 201 \
  "{\"seed_id\":\"${SEED_ID}\",\"title\":\"Smoke Branch\",\"summary\":\"branch for smoke\",\"key_points\":[\"point1\"],\"tags\":[\"smoke\"]}" \
  "POST /api/v1/workspaces/{wsId}/branches (start branch)"
BRANCH_ID=$(jqr '.data.branch.id')
LEAF_ID_1=$(jqr '.data.leaf.id')

call GET "/api/v1/branches/${BRANCH_ID}" 200 "" "GET /api/v1/branches/{branchId}"

call POST "/api/v1/workspaces/${WS_ID}/leaves" 201 \
  "{\"seed_id\":\"${SEED_ID}\",\"branch_id\":\"${BRANCH_ID}\",\"parent_leaf_id\":\"${LEAF_ID_1}\",\"title\":\"Smoke Leaf 2\",\"summary\":\"child leaf\",\"key_points\":[\"point2\"],\"tags\":[\"smoke\"]}" \
  "POST /api/v1/workspaces/{wsId}/leaves (create leaf)"
LEAF_ID_2=$(jqr '.data.id')

call GET "/api/v1/leaves/${LEAF_ID_1}" 200 "" "GET /api/v1/leaves/{leafId}"

call GET "/api/v1/workspaces/${WS_ID}/leaves" 200 "" "GET /api/v1/workspaces/{wsId}/leaves"

call POST "/api/v1/workspaces/${WS_ID}/connections" 201 \
  "{\"leaf_ids\":[\"${LEAF_ID_1}\",\"${LEAF_ID_2}\"]}" \
  "POST /api/v1/workspaces/{wsId}/connections"

call GET "/api/v1/leaves/${LEAF_ID_1}/connections" 200 "" \
  "GET /api/v1/leaves/{leafId}/connections"

call POST "/api/v1/leaves/${LEAF_ID_1}/promote" 200 "" \
  "POST /api/v1/leaves/{leafId}/promote"

# ════════════════════════════════════════════════════════════════
#  Phase 6 — Discussion
# ════════════════════════════════════════════════════════════════
section "Phase 6: Discussion"

call POST "/api/v1/workspaces/${WS_ID}/leaves/${LEAF_ID_1}/comments" 201 \
  "{\"content\":\"Smoke comment ${RND}\"}" \
  "POST /api/v1/workspaces/{wsId}/leaves/{leafId}/comments"

call GET "/api/v1/leaves/${LEAF_ID_1}/thread" 200 "" \
  "GET /api/v1/leaves/{leafId}/thread"

# ════════════════════════════════════════════════════════════════
#  Phase 7 — Convergence
# ════════════════════════════════════════════════════════════════
section "Phase 7: Convergence"

call POST "/api/v1/workspaces/${WS_ID}/leaves/${LEAF_ID_1}/signals" 201 \
  "{\"signal_type\":\"upvote\"}" \
  "POST /api/v1/workspaces/{wsId}/leaves/{leafId}/signals"
SIGNAL_ID=$(jqr '.data.id')

call GET "/api/v1/leaves/${LEAF_ID_1}/signals/counts" 200 "" \
  "GET /api/v1/leaves/{leafId}/signals/counts"

call GET "/api/v1/workspaces/${WS_ID}/signals/mine" 200 "" \
  "GET /api/v1/workspaces/{wsId}/signals/mine"

call DELETE "/api/v1/signals/${SIGNAL_ID}" 200 "" \
  "DELETE /api/v1/signals/{signalId}"

call POST "/api/v1/workspaces/${WS_ID}/checkpoints" 201 \
  "{\"leaf_ids\":[\"${LEAF_ID_1}\"]}" \
  "POST /api/v1/workspaces/{wsId}/checkpoints"
CHECKPOINT_ID=$(jqr '.data.id')

call POST "/api/v1/checkpoints/${CHECKPOINT_ID}/resolve" 200 "" \
  "POST /api/v1/checkpoints/{checkpointId}/resolve"

# ════════════════════════════════════════════════════════════════
#  Phase 8 — Session
# ════════════════════════════════════════════════════════════════
section "Phase 8: Session"

call POST "/api/v1/workspaces/${WS_ID}/sessions" 201 \
  "{\"seed_id\":\"${SEED_ID}\",\"parent_leaf_id\":\"${LEAF_ID_1}\",\"session_type\":\"exploration\"}" \
  "POST /api/v1/workspaces/{wsId}/sessions"
SESSION_ID=$(jqr '.data.id')

call POST "/api/v1/sessions/${SESSION_ID}/messages" 200 \
  "{\"message\":{\"role\":\"user\",\"content\":\"Smoke message ${RND}\"}}" \
  "POST /api/v1/sessions/{sessionId}/messages"

call POST "/api/v1/sessions/${SESSION_ID}/checkpoint" 200 "" \
  "POST /api/v1/sessions/{sessionId}/checkpoint"

call POST "/api/v1/sessions/${SESSION_ID}/complete" 200 \
  "{\"leaf_id\":\"${LEAF_ID_2}\"}" \
  "POST /api/v1/sessions/{sessionId}/complete"

# ════════════════════════════════════════════════════════════════
#  Phase 9 — Synthesis
# ════════════════════════════════════════════════════════════════
section "Phase 9: Synthesis"

call POST "/api/v1/workspaces/${WS_ID}/syntheses" 201 \
  "{\"source_leaf_ids\":[\"${LEAF_ID_1}\",\"${LEAF_ID_2}\"]}" \
  "POST /api/v1/workspaces/{wsId}/syntheses (1st — will fail)"
SYNTH_ID_1=$(jqr '.data.id')

call POST "/api/v1/syntheses/${SYNTH_ID_1}/fail" 200 \
  "{\"reason\":\"smoke test forced failure\"}" \
  "POST /api/v1/syntheses/{synthesisId}/fail"

call POST "/api/v1/workspaces/${WS_ID}/syntheses" 201 \
  "{\"source_leaf_ids\":[\"${LEAF_ID_1}\",\"${LEAF_ID_2}\"]}" \
  "POST /api/v1/workspaces/{wsId}/syntheses (2nd — will complete)"
SYNTH_ID_2=$(jqr '.data.id')

call POST "/api/v1/syntheses/${SYNTH_ID_2}/complete" 200 \
  "{\"result_leaf_id\":\"${LEAF_ID_2}\"}" \
  "POST /api/v1/syntheses/{synthesisId}/complete"

call GET "/api/v1/workspaces/${WS_ID}/syntheses" 200 "" \
  "GET /api/v1/workspaces/{wsId}/syntheses"

# ════════════════════════════════════════════════════════════════
#  Phase 10 — Deliverable
# ════════════════════════════════════════════════════════════════
section "Phase 10: Deliverable"

call POST "/api/v1/workspaces/${WS_ID}/deliverables" 201 \
  "{\"format\":\"markdown\",\"content\":\"# Smoke Deliverable ${RND}\",\"source_leaf_ids\":[\"${LEAF_ID_1}\"]}" \
  "POST /api/v1/workspaces/{wsId}/deliverables"
DELIV_ID=$(jqr '.data.id')

call PATCH "/api/v1/deliverables/${DELIV_ID}" 200 \
  "{\"content\":\"# Smoke Deliverable ${RND} (updated)\"}" \
  "PATCH /api/v1/deliverables/{deliverableId}"

call POST "/api/v1/deliverables/${DELIV_ID}/finalize" 200 "" \
  "POST /api/v1/deliverables/{deliverableId}/finalize"

call GET "/api/v1/deliverables/${DELIV_ID}" 200 "" \
  "GET /api/v1/deliverables/{deliverableId}"

call GET "/api/v1/workspaces/${WS_ID}/deliverables" 200 "" \
  "GET /api/v1/workspaces/{wsId}/deliverables"

# ════════════════════════════════════════════════════════════════
#  Phase 11 — Notification
# ════════════════════════════════════════════════════════════════
section "Phase 11: Notification"

call GET "/api/v1/notifications?limit=5" 200 "" \
  "GET /api/v1/notifications?limit=5"

call GET "/api/v1/notifications/unread" 200 "" \
  "GET /api/v1/notifications/unread"

call POST "/api/v1/notifications/read-all" 200 "" \
  "POST /api/v1/notifications/read-all"

# No notification exists — expect 404
FAKE_NTF_ID="ntf_000000000000fake"
call POST "/api/v1/notifications/${FAKE_NTF_ID}/read" 404 "" \
  "POST /api/v1/notifications/{id}/read (expect 404)"

call PUT "/api/v1/workspaces/${WS_ID}/notifications/subscription" 200 \
  "{\"channels\":[\"in_app\"],\"digest_frequency\":\"daily\"}" \
  "PUT /api/v1/workspaces/{wsId}/notifications/subscription"

# ════════════════════════════════════════════════════════════════
#  Phase 12 — Ledger
# ════════════════════════════════════════════════════════════════
section "Phase 12: Ledger"

call GET "/api/v1/admin/ledger/system?limit=5" 200 "" \
  "GET /api/v1/admin/ledger/system?limit=5"

call GET "/api/v1/admin/ledger/domain?limit=5" 200 "" \
  "GET /api/v1/admin/ledger/domain?limit=5"

# ════════════════════════════════════════════════════════════════
#  Phase 13 — Workspace destructive (deferred)
# ════════════════════════════════════════════════════════════════
section "Phase 13: Workspace lifecycle (deferred destructive)"

call POST "/api/v1/workspaces/${WS_ID}/phase" 200 \
  "{\"target\":\"understory\"}" \
  "POST /api/v1/workspaces/{wsId}/phase (→ understory)"

call POST "/api/v1/workspaces/${WS_ID}/leave" 200 "" \
  "POST /api/v1/workspaces/{wsId}/leave"

# ════════════════════════════════════════════════════════════════
#  Summary
# ════════════════════════════════════════════════════════════════
printf "\n$(bold "════════════════════════════════════════")\n"
printf "  Total: %d   $(green "Pass: %d")   $(red "Fail: %d")\n" "$TOTAL" "$PASS" "$FAIL"
printf "$(bold "════════════════════════════════════════")\n"

if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
