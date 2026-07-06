#!/usr/bin/env bash
# needs: none
#
# mock-unit — the auth-free backbone for delivery-team stone #9 ②. Runs the mock
# azureDevOps MCP server's OWN unit tests (node --test) in-container: store CRUD +
# idempotency round-trip (§5), the WIQL subset, runtime type/state resolution (§6),
# the curated-map boundary (off-contract tools absent), persistence, and the MCP
# stdio protocol. No Claude token, no Azure — this is the always-on half of stone #9
# (the token-gated ceremony chain lives in delivery-loop.sh).
source /e2e/lib.sh

if node --test /e2e/mock-mcp/mock.test.js >/tmp/mock-test.log 2>&1; then
  ok "mock MCP unit tests pass ($(sed -n 's/^# pass //p' /tmp/mock-test.log | head -1) subtests)"
else
  bad "mock MCP unit tests failed"
  tail -40 /tmp/mock-test.log
fi

finish
