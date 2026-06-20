#!/usr/bin/env bash
# needs: none
#
# search — the catalog is searchable by keyword and by name, browsable with no
# query, and a miss reports cleanly. Runs against the injected test index, so it
# never touches the production catalog.
source /e2e/lib.sh

fresh
write_test_index   # seeds ~/.atl/index.json with the atl-e2e-team fixture (keyword: e2e)

# keyword match — the fixture's keyword is "e2e"
out="$(atl search e2e 2>&1)"
echo "$out" | grep -q "agentteamland/atl-e2e-team" && ok "keyword search finds the team" || bad "keyword search missed -- [$out]"
echo "$out" | grep -q "install: atl install agentteamland/atl-e2e-team" && ok "result prints the install command" || bad "no install hint -- [$out]"

# name substring match
out="$(atl search atl-e2e 2>&1)"
echo "$out" | grep -q "agentteamland/atl-e2e-team" && ok "name search finds the team" || bad "name search missed -- [$out]"

# no-arg browse lists the whole catalog
out="$(atl search 2>&1)"
echo "$out" | grep -q "atl-e2e-team" && ok "no-arg browse lists the catalog" || bad "browse missed -- [$out]"

# a miss reports cleanly (and is not a hard error)
out="$(atl search zzz-no-such-team 2>&1)"
echo "$out" | grep -qi "no teams matching" && ok "miss reports cleanly" || bad "miss message wrong -- [$out]"

finish
