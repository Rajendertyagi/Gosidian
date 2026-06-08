#!/bin/sh
# Seed the test directory with users alice/bob. Run once after `up`.
# Idempotent-ish: re-running reports "Already exists" for present entries.
set -e
dir="$(dirname "$0")"
docker exec -i gosidian-ldap-test ldapadd -c -x -H ldap://localhost:389 \
  -D "cn=admin,dc=example,dc=org" -w adminpassword <"$dir/bootstrap/01-users.ldif" || true
echo "seeded: alice / alicepass, bob / bobpass"
