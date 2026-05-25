#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/common.sh
source "${SCRIPT_DIR}/lib/common.sh"

require_cmd docker
require_cmd python3

PARITY_DIR="${PROD_PARITY_DIR:-${REPO_ROOT}/.run/prod-parity}"

parity_default_path() {
  local current="$1"
  local common_default="$2"
  local parity_default="$3"
  if repo_default_path_matches "${current}" "${common_default}"; then
    printf '%s\n' "${parity_default}"
    return
  fi
  printf '%s\n' "${current}"
}

export ENV_TEMPLATE_FILE="${REPO_ROOT}/.env.prod.example"
export ENV_FILE="$(parity_default_path "${ENV_FILE:-}" "${REPO_ROOT}/.env" "${PARITY_DIR}/.env.prod.shared")"
export SECRETS_ENV_FILE="$(parity_default_path "${SECRETS_ENV_FILE:-}" "" "${PARITY_DIR}/.env.prod.secrets.local")"
export GENERATED_ENV_FILE="$(parity_default_path "${GENERATED_ENV_FILE:-}" "${REPO_ROOT}/.env.generated" "${PARITY_DIR}/.env.prod.generated")"
export GENERATED_SECRET_ENV_FILE="$(parity_default_path "${GENERATED_SECRET_ENV_FILE:-}" "${REPO_ROOT}/.env.generated.secrets" "${PARITY_DIR}/.env.prod.generated.secrets")"
export DEPLOY_STATE_DIR="$(parity_default_path "${DEPLOY_STATE_DIR:-}" "${REPO_ROOT}/.deploy" "${PARITY_DIR}/deploy-state")"

load_env

postgres_container="${SHARED_POSTGRES_CONTAINER:-${PROD_PARITY_POSTGRES_CONTAINER:-stuhelper-prod-parity-postgres}}"
stuhelper_db="${STUHELPER_APP_DB_NAME:-${POSTGRES_DB:-stuhelper}}"
app_user="${STUHELPER_APP_DB_USER:-stuhelper_app}"
redis_container="${REDIS_CONTAINER_NAME:-${STACK_NAME:-stuhelper-prod-parity}-redis}"
redis_username="${REDIS_USERNAME:-stuhelper_app}"
evidence_file="${PROD_PARITY_SMOKE_DATA_EVIDENCE_FILE:-${PARITY_DIR}/smoke-data-evidence.json}"
admission_token="${PROD_PARITY_ADMISSION_TOKEN:-PROD-PARITY-ADMIT-LOGIN}"
admission_qq="${PROD_PARITY_ADMISSION_QQ:-990001}"

[[ -n "${STUHELPER_APP_DB_PASSWORD:-}" ]] || die "STUHELPER_APP_DB_PASSWORD is required for prod-parity smoke data"
[[ -n "${REDIS_PASSWORD:-}" ]] || die "REDIS_PASSWORD is required for prod-parity smoke data cache invalidation"
[[ -n "${HMAC_SECRET:-}" ]] || die "HMAC_SECRET is required for prod-parity admission smoke data"

case "${postgres_container}" in
  *prod-parity*) ;;
  *) die "refusing to seed non prod-parity PostgreSQL container: ${postgres_container}" ;;
esac

case "${redis_container}" in
  *prod-parity*) ;;
  *) die "refusing to clear non prod-parity Redis container: ${redis_container}" ;;
esac

docker inspect "${postgres_container}" >/dev/null 2>&1 || die "PostgreSQL container not found: ${postgres_container}"
docker inspect "${redis_container}" >/dev/null 2>&1 || die "Redis container not found: ${redis_container}"

admission_token_hash="$(
  python3 - "${HMAC_SECRET}" "${admission_token}" <<'PY'
import hashlib
import hmac
import sys

key = sys.argv[1].encode()
token = sys.argv[2].strip().encode()
print(hmac.new(key, token, hashlib.sha256).hexdigest())
PY
)"

log "seeding deterministic prod-parity browser smoke data"
docker exec \
  -e PGPASSWORD="${STUHELPER_APP_DB_PASSWORD}" \
  -i "${postgres_container}" \
  psql \
    -v ON_ERROR_STOP=1 \
    -v admission_token="${admission_token}" \
    -v admission_token_hash="${admission_token_hash}" \
    -v admission_qq="${admission_qq}" \
    -h 127.0.0.1 \
    -U "${app_user}" \
    -d "${stuhelper_db}" <<'SQL' >/dev/null
BEGIN;

INSERT INTO public.departments (id, school_id, name, short_name, category, sort_order)
VALUES (900001, 10006, '生产等价学院', '等价', '工科', 900001)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    short_name = EXCLUDED.short_name,
    category = EXCLUDED.category,
    sort_order = EXCLUDED.sort_order;

INSERT INTO public.teachers (id, school_id, name, department_id)
VALUES (900001, 10006, '生产等价教师', 900001)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    department_id = EXCLUDED.department_id;

INSERT INTO public.courses (
    id, school_id, name, code, department_id, credits, category, description, review_count
)
VALUES (
    900001,
    10006,
    '生产等价课程',
    'PARITY9001',
    900001,
    3.0,
    '通识',
    '本课程用于本机生产等价浏览器 smoke，验证真实后端数据驱动的动态课程页面。',
    1
)
ON CONFLICT (id) DO UPDATE
SET name = EXCLUDED.name,
    code = EXCLUDED.code,
    department_id = EXCLUDED.department_id,
    credits = EXCLUDED.credits,
    category = EXCLUDED.category,
    description = EXCLUDED.description,
    review_count = EXCLUDED.review_count,
    updated_at = now();

INSERT INTO public.reviews (
    id, course_id, school_id, teacher_id, term_id, user_hash, title, content, grade,
    ratings, avg_rating, like_count, reply_count, status, created_at, updated_at
)
VALUES (
    '01999999-0001-7000-8000-000000000001',
    900001,
    10006,
    900001,
    '2025-2',
    'prod_parity_user_hash_000000000000000000000000000001',
    '生产等价评课',
    '生产等价评课内容用于验证生产镜像、真实 PostgreSQL、真实 Redis 缓存和前端动态详情页能一起正常显示。',
    'A',
    '{"difficulty":3,"workload":2,"usefulness":5,"teaching":5,"grading":4}'::jsonb,
    3.80,
    2,
    1,
    'published',
    now() - interval '1 day',
    now()
)
ON CONFLICT (id) DO UPDATE
SET title = EXCLUDED.title,
    content = EXCLUDED.content,
    grade = EXCLUDED.grade,
    ratings = EXCLUDED.ratings,
    avg_rating = EXCLUDED.avg_rating,
    like_count = EXCLUDED.like_count,
    reply_count = EXCLUDED.reply_count,
    status = EXCLUDED.status,
    updated_at = now();

INSERT INTO public.review_replies (
    id, review_id, parent_id, user_hash, content, like_count, status, created_at, updated_at
)
VALUES (
    '01999999-0002-7000-8000-000000000002',
    '01999999-0001-7000-8000-000000000001',
    NULL,
    'prod_parity_reply_hash_0000000000000000000000000001',
    '生产等价回复',
    1,
    'published',
    now() - interval '12 hours',
    now()
)
ON CONFLICT (id) DO UPDATE
SET content = EXCLUDED.content,
    like_count = EXCLUDED.like_count,
    status = EXCLUDED.status,
    updated_at = now();

UPDATE public.courses
SET review_count = (
    SELECT count(*)
    FROM public.reviews
    WHERE course_id = 900001 AND status = 'published'
)
WHERE id = 900001;

DELETE FROM public.course_rating_stats WHERE course_id = 900001;
INSERT INTO public.course_rating_stats (
    id, course_id, term_id, dimension_key, avg_rating, rating_count, rating_dist
)
SELECT
    '01999999-0100-7000-8000-' || lpad(row_number() OVER ()::text, 12, '0'),
    900001,
    term_id,
    dimension_key,
    avg_rating,
    rating_count,
    rating_dist
FROM (
    SELECT
        term_id,
        dimension_key,
        round(avg(rating_value)::numeric, 2) AS avg_rating,
        count(*)::int AS rating_count,
        jsonb_object_agg(rating_value::text, rating_count) AS rating_dist
    FROM (
        SELECT
            r.term_id,
            d.key AS dimension_key,
            (r.ratings ->> d.key)::int AS rating_value,
            count(*) AS rating_count
        FROM public.reviews r
        CROSS JOIN public.rating_dimensions d
        WHERE r.course_id = 900001
          AND r.status = 'published'
          AND d.is_active = true
          AND r.ratings ? d.key
        GROUP BY r.term_id, d.key, (r.ratings ->> d.key)::int
    ) per_value
    GROUP BY term_id, dimension_key
    UNION ALL
    SELECT
        NULL::varchar(20) AS term_id,
        dimension_key,
        round(avg(rating_value)::numeric, 2) AS avg_rating,
        count(*)::int AS rating_count,
        jsonb_object_agg(rating_value::text, rating_count) AS rating_dist
    FROM (
        SELECT
            d.key AS dimension_key,
            (r.ratings ->> d.key)::int AS rating_value,
            count(*) AS rating_count
        FROM public.reviews r
        CROSS JOIN public.rating_dimensions d
        WHERE r.course_id = 900001
          AND r.status = 'published'
          AND d.is_active = true
          AND r.ratings ? d.key
        GROUP BY d.key, (r.ratings ->> d.key)::int
    ) per_value
    GROUP BY dimension_key
) stats;

DELETE FROM public.teacher_rating_stats WHERE teacher_id = 900001;
INSERT INTO public.teacher_rating_stats (
    id, teacher_id, term_id, dimension_key, avg_rating, rating_count, rating_dist
)
SELECT
    '01999999-0200-7000-8000-' || lpad(row_number() OVER ()::text, 12, '0'),
    900001,
    term_id,
    dimension_key,
    avg_rating,
    rating_count,
    rating_dist
FROM (
    SELECT
        term_id,
        dimension_key,
        round(avg(rating_value)::numeric, 2) AS avg_rating,
        count(*)::int AS rating_count,
        jsonb_object_agg(rating_value::text, rating_count) AS rating_dist
    FROM (
        SELECT
            r.term_id,
            d.key AS dimension_key,
            (r.ratings ->> d.key)::int AS rating_value,
            count(*) AS rating_count
        FROM public.reviews r
        CROSS JOIN public.rating_dimensions d
        WHERE r.teacher_id = 900001
          AND r.status = 'published'
          AND d.is_active = true
          AND r.ratings ? d.key
        GROUP BY r.term_id, d.key, (r.ratings ->> d.key)::int
    ) per_value
    GROUP BY term_id, dimension_key
    UNION ALL
    SELECT
        NULL::varchar(20) AS term_id,
        dimension_key,
        round(avg(rating_value)::numeric, 2) AS avg_rating,
        count(*)::int AS rating_count,
        jsonb_object_agg(rating_value::text, rating_count) AS rating_dist
    FROM (
        SELECT
            d.key AS dimension_key,
            (r.ratings ->> d.key)::int AS rating_value,
            count(*) AS rating_count
        FROM public.reviews r
        CROSS JOIN public.rating_dimensions d
        WHERE r.teacher_id = 900001
          AND r.status = 'published'
          AND d.is_active = true
          AND r.ratings ? d.key
        GROUP BY d.key, (r.ratings ->> d.key)::int
    ) per_value
    GROUP BY dimension_key
) stats;

REFRESH MATERIALIZED VIEW public.mv_teacher_public_stats;

SELECT setval('public.departments_id_seq', GREATEST((SELECT max(id) FROM public.departments), (SELECT last_value FROM public.departments_id_seq)), true);
SELECT setval('public.teachers_id_seq', GREATEST((SELECT max(id) FROM public.teachers), (SELECT last_value FROM public.teachers_id_seq)), true);
SELECT setval('public.courses_id_seq', GREATEST((SELECT max(id) FROM public.courses), (SELECT last_value FROM public.courses_id_seq)), true);

INSERT INTO public.group_admission_sessions (
    id, platform, bot_self_id, guild_id, channel_id, qq_id, qq_nickname, user_id,
    token_hash, auth_url, token_expires_at, token_consumed_at, status,
    link_wait_deadline_at, submission_wait_deadline_at, manual_review_deadline_at,
    initial_mute_until, verified_at, cancelled_at, last_bot_error, updated_at
)
VALUES (
    'prod-parity-admission-session',
    'qq',
    'prod-parity-bot',
    'prod-parity-guild',
    'prod-parity-channel',
    :'admission_qq',
    '生产等价 QQ',
    NULL,
    :'admission_token_hash',
    format('http://127.0.0.1:28000/admission/a/%s?qq=%s', :'admission_token', :'admission_qq'),
    now() + interval '1 hour',
    NULL,
    'joined_muted',
    now() + interval '1 hour',
    now() + interval '1 day',
    NULL,
    now() + interval '30 days',
    NULL,
    NULL,
    NULL,
    now()
)
ON CONFLICT (id) DO UPDATE
SET qq_id = EXCLUDED.qq_id,
    qq_nickname = EXCLUDED.qq_nickname,
    user_id = NULL,
    token_hash = EXCLUDED.token_hash,
    auth_url = EXCLUDED.auth_url,
    token_expires_at = EXCLUDED.token_expires_at,
    token_consumed_at = NULL,
    status = EXCLUDED.status,
    link_wait_deadline_at = EXCLUDED.link_wait_deadline_at,
    submission_wait_deadline_at = EXCLUDED.submission_wait_deadline_at,
    manual_review_deadline_at = NULL,
    initial_mute_until = EXCLUDED.initial_mute_until,
    verified_at = NULL,
    cancelled_at = NULL,
    last_bot_error = NULL,
    updated_at = now();

COMMIT;
SQL

clear_cache_keys() {
  local key_output
  local keys=()
  local redis_cli=(
    docker exec
    -e "REDISCLI_AUTH=${REDIS_PASSWORD}"
    "${redis_container}"
    redis-cli
    --user "${redis_username}"
  )

  if [[ "${REDIS_TLS_ENABLED:-true}" == "true" ]]; then
    redis_cli+=(--tls --cacert /tls/ca.crt)
  fi

  for pattern in 'course:*' 'review:*' 'cache:version:course*' 'cache:version:review*'; do
    key_output="$("${redis_cli[@]}" --scan --pattern "${pattern}")"
    while IFS= read -r key; do
      [[ -n "${key}" ]] && keys+=("${key}")
    done <<<"${key_output}"
  done

  if (( ${#keys[@]} == 0 )); then
    return
  fi

  "${redis_cli[@]}" DEL "${keys[@]}" >/dev/null
  log "cleared ${#keys[@]} prod-parity course/review cache keys after smoke data seed"
}

clear_cache_keys

query_json="$(
  docker exec \
    -e PGPASSWORD="${STUHELPER_APP_DB_PASSWORD}" \
    -i "${postgres_container}" \
    psql \
      -v ON_ERROR_STOP=1 \
      -v admission_qq="${admission_qq}" \
      -h 127.0.0.1 \
      -U "${app_user}" \
      -d "${stuhelper_db}" \
      -At <<'SQL'
SELECT jsonb_build_object(
  'courseID', 900001,
  'teacherID', 900001,
  'departmentCount', (SELECT count(*) FROM public.departments WHERE id = 900001),
  'courseCount', (SELECT count(*) FROM public.courses WHERE id = 900001),
  'teacherCount', (SELECT count(*) FROM public.teachers WHERE id = 900001),
  'reviewCount', (SELECT count(*) FROM public.reviews WHERE id = '01999999-0001-7000-8000-000000000001' AND status = 'published'),
  'replyCount', (SELECT count(*) FROM public.review_replies WHERE id = '01999999-0002-7000-8000-000000000002' AND status = 'published'),
  'courseRatingStatsCount', (SELECT count(*) FROM public.course_rating_stats WHERE course_id = 900001),
  'teacherRatingStatsCount', (SELECT count(*) FROM public.teacher_rating_stats WHERE teacher_id = 900001),
  'teacherPublicStatsCount', (SELECT count(*) FROM public.mv_teacher_public_stats WHERE teacher_id = 900001 AND review_count > 0),
  'admissionSessionCount', (
      SELECT count(*)
      FROM public.group_admission_sessions
      WHERE id = 'prod-parity-admission-session'
        AND qq_id = :'admission_qq'
        AND status = 'joined_muted'
        AND token_consumed_at IS NULL
        AND token_expires_at > now()
  )
)::text;
SQL
)"

python3 - "${query_json}" "${evidence_file}" <<'PY'
import json
import sys
from datetime import datetime, timezone
from pathlib import Path

payload = json.loads(sys.argv[1])
required = {
    "departmentCount": 1,
    "courseCount": 1,
    "teacherCount": 1,
    "reviewCount": 1,
    "replyCount": 1,
    "courseRatingStatsCount": 10,
    "teacherRatingStatsCount": 10,
    "teacherPublicStatsCount": 1,
    "admissionSessionCount": 1,
}
failures = [
    f"{key} expected {expected}, got {payload.get(key)}"
    for key, expected in required.items()
    if payload.get(key) != expected
]
evidence = {
    "generatedAt": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"),
    "passed": not failures,
    "checks": payload,
    "failures": failures,
}
path = Path(sys.argv[2])
path.parent.mkdir(parents=True, exist_ok=True)
path.write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
if failures:
    print("[prod-parity-smoke-data] failed: " + "; ".join(failures), file=sys.stderr)
    sys.exit(1)
PY

log "prod-parity smoke data ready; evidence: ${evidence_file}"
