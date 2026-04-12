# shellcheck shell=bash

# In some CI environments (notably GitHub Actions container jobs), the checkout may not
# include full VCS metadata or git may refuse to operate due to "dubious ownership".
# Go's default VCS stamping can then fail with:
#   error obtaining VCS status: exit status 128
# This disables VCS stamping for Go commands during CI test workflows only.

append_goflags() {
  local flag="$1"
  case " ${GOFLAGS:-} " in
    *" ${flag} "*) ;;
    *) export GOFLAGS="${GOFLAGS:-} ${flag}" ;;
  esac
}

if [ "${CI:-}" = "true" ] || [ -n "${GITHUB_ACTIONS:-}" ]; then
  append_goflags "-buildvcs=false"
fi

