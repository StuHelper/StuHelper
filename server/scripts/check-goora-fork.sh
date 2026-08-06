#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "$0")/.." && pwd)"
fork_dir="$root_dir/internal/thirdparty/goora"
upstream_module="github.com/sijms/go-ora/v2"
upstream_version="v2.9.0"

fail() {
  printf '[goora-fork-check][error] %s\n' "$*" >&2
  exit 1
}

approved_hash() {
  case "$1" in
    auth_object.go) printf '%s\n' '4db55d4b019201067a52902014a85f7ae1385da51a241be74d5fcc6766987dc5' ;;
    bulk_copy.go) printf '%s\n' 'fb2283a3f057cd93493f6d93158cab935cf190596201eb2af47d230d6c0767df' ;;
    command.go) printf '%s\n' 'e8467f0b9f8043744d83f859adb449fda95362d39c25017989cdf6ffda88d598' ;;
    connection.go) printf '%s\n' '9a6853591ad6d7b0788f124135ba38619ffded8b684ad45c3ef444d08baeddf8' ;;
    connection_string.go) printf '%s\n' 'c95ec5cfaef84dd99813fb18d02181ad419c82f954623a874e7b136d827255ee' ;;
    driver.go) printf '%s\n' 'c41f97fa3ab7fe1f1863fc9693146802a7cfc2a9e668216f9213142442a6a33a' ;;
    transaction.go) printf '%s\n' 'a82857fd0e73f1b62b6394ad71d2e9daf5247ec77bfa7d0dbe1782324c121b00' ;;
  esac
}

read -r resolved_version upstream_dir < <(
  cd "$root_dir"
  go list -m -f '{{.Version}} {{.Dir}}' "$upstream_module"
)
[[ "$resolved_version" == "$upstream_version" ]] ||
  fail "expected $upstream_module $upstream_version, got $resolved_version"
[[ -d "$upstream_dir" ]] || fail "resolved upstream module directory is missing"
[[ -d "$fork_dir" ]] || fail "StuHelper go-ora fork directory is missing"

if ! diff -u \
  <(find "$upstream_dir" -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' -printf '%f\n' | LC_ALL=C sort) \
  <(find "$fork_dir" -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' ! -name 'stuhelper_readonly_policy.go' -printf '%f\n' | LC_ALL=C sort); then
  fail "forked root package source set differs from upstream $upstream_version"
fi

while IFS= read -r file; do
  expected="$(approved_hash "$file")"
  if [[ -n "$expected" ]]; then
    actual="$(sha256sum "$fork_dir/$file" | awk '{print $1}')"
    [[ "$actual" == "$expected" ]] ||
      fail "$file differs from its reviewed StuHelper policy fork hash"
    continue
  fi
  cmp -s "$upstream_dir/$file" "$fork_dir/$file" ||
    fail "$file has an unreviewed difference from upstream $upstream_version"
done < <(find "$upstream_dir" -maxdepth 1 -type f -name '*.go' ! -name '*_test.go' -printf '%f\n' | LC_ALL=C sort)

cmp -s "$upstream_dir/LICENSE" "$fork_dir/LICENSE" ||
  fail "upstream license does not match $upstream_version"
[[ -f "$fork_dir/stuhelper_readonly_policy.go" ]] ||
  fail "StuHelper SELECT-only runtime policy is missing"
[[ -f "$fork_dir/stuhelper_policy_test.go" ]] ||
  fail "StuHelper policy regression test is missing"

printf '[goora-fork-check] pinned upstream source and reviewed policy fork hashes passed\n'
