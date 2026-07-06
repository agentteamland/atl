'use strict';
// Auth-free unit tests for the mock azureDevOps MCP server — the always-on backbone
// that proves the mock's behavior without a Claude token. Run: `node --test`.
//
// Covers the adapter-contract behaviors Layer-A relies on: idempotency round-trip
// (§5), a WIQL subset, runtime type/state resolution (§6), the curated-map boundary
// (off-contract tools are absent), cross-process persistence, and the MCP protocol
// itself (a spawned server answers initialize/tools/list/tools/call).

const { test } = require('node:test');
const assert = require('node:assert');
const os = require('os');
const fs = require('fs');
const path = require('path');
const { spawnSync } = require('child_process');
const { Store } = require('./store');
const { tools } = require('./tools');

function tmpStore() {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'mockstore-'));
  return { st: new Store(path.join(dir, 'store.json')), dir };
}
const call = (st, name, args) => tools[name].handler(st, args || {});

test('seed: a believable pre-init project with velocity history', () => {
  const { st } = tmpStore();
  assert.equal(st.state.processTemplateName, 'Scrum');
  assert.ok(st.state.projects.length >= 1);
  // three CLOSED iterations carry Done PBIs with StoryPoints for velocity math
  const closed = st.closedIterations();
  assert.equal(closed.length, 3);
  const doneWithPoints = Object.values(st.state.workItems)
    .filter((wi) => wi.fields['System.State'] === 'Done' && wi.fields['Microsoft.VSTS.Scheduling.StoryPoints'] > 0);
  assert.ok(doneWithPoints.length >= 6, 'seed has Done PBIs with points across closed sprints');
});

test('create -> get -> update -> comment -> link converge in the store', () => {
  const { st } = tmpStore();
  const epic = call(st, 'wit_create_work_item', { workItemType: 'Epic', fields: [{ name: 'System.Title', value: 'Checkout revamp' }] });
  assert.ok(epic.id > 0);
  const got = call(st, 'wit_get_work_item', { id: epic.id });
  assert.equal(got.fields['System.Title'], 'Checkout revamp');
  assert.equal(got.fields['System.WorkItemType'], 'Epic');

  call(st, 'wit_update_work_item', { id: epic.id, updates: [{ op: 'add', path: '/fields/System.State', value: 'In Progress' }] });
  assert.equal(call(st, 'wit_get_work_item', { id: epic.id }).fields['System.State'], 'In Progress');

  const c = call(st, 'wit_add_work_item_comment', { workItemId: epic.id, comment: '**[Technical Analysis]**\n## Approach\nmock' });
  assert.match(c.text, /\[Technical Analysis\]/);
  const comments = call(st, 'wit_list_work_item_comments', { workItemId: epic.id });
  assert.equal(comments.comments.length, 1);

  const child = call(st, 'wit_add_child_work_items', { parentId: epic.id, workItemType: 'Feature', items: [{ fields: [{ name: 'System.Title', value: 'Address book' }] }] });
  assert.equal(child.created.length, 1);
  const parent = call(st, 'wit_get_work_item', { id: epic.id });
  assert.ok(parent.relations.some((r) => r.rel.includes('Hierarchy-Forward')), 'parent has a child link');
});

test('idempotency round-trip (§5): a stamped atl-key tag is found by a check-first WIQL', () => {
  const { st } = tmpStore();
  // create-then-stamp
  const wi = call(st, 'wit_create_work_item', {
    workItemType: 'Product Backlog Item',
    fields: [{ name: 'System.Title', value: 'Login' }, { name: 'System.Tags', value: 'atl-run:kickoff:s1; atl-key:abc123' }],
  });
  // check-first WIQL on the same key must FIND it (the single most important behavior)
  const found = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.Tags] CONTAINS 'atl-key:abc123'" });
  assert.equal(found.workItems.length, 1);
  assert.equal(found.workItems[0].id, wi.id);
  // a different key finds nothing -> the caller would create
  const none = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.Tags] CONTAINS 'atl-key:zzz'" });
  assert.equal(none.workItems.length, 0);
});

test('WIQL subset: field equality + ORDER BY + top', () => {
  const { st } = tmpStore();
  // Done PBIs exist in the seed; query by state
  const done = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Done' AND [System.WorkItemType] = 'Product Backlog Item'" });
  assert.ok(done.workItems.length >= 6);
  // top caps the result
  const capped = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Done'", top: 2 });
  assert.equal(capped.workItems.length, 2);
});

test('runtime type/state resolution (§6): states carry a non-literal category map', () => {
  const { st } = tmpStore();
  const t = call(st, 'wit_get_work_item_type', { workItemType: 'Product Backlog Item' });
  const byName = Object.fromEntries(t.states.map((s) => [s.name, s.category]));
  // "Done" resolves to the Completed CATEGORY (not a literal), "Committed" to InProgress
  assert.equal(byName['Done'], 'Completed');
  assert.equal(byName['Committed'], 'InProgress');
  assert.equal(byName['New'], 'Proposed');
  // a caller resolving "the Completed-category state" gets "Done" by lookup, not a hardcode
  const completed = t.states.find((s) => s.category === 'Completed');
  assert.equal(completed.name, 'Done');
});

test('velocity read: closed-iteration Done points are summable', () => {
  const { st } = tmpStore();
  const s3 = call(st, 'wit_get_work_items_for_iteration', { iterationId: 'Sprint 3' });
  const pts = s3.value.reduce((n, wi) => n + (wi.fields['Microsoft.VSTS.Scheduling.StoryPoints'] || 0), 0);
  assert.ok(pts > 0, 'Sprint 3 has summable story points');
});

test('wit_get_work_items_for_iteration resolves an iteration by ITS ID, not just name', () => {
  const { st } = tmpStore();
  // the real flow: work_list_iterations hands the LLM the id, which it passes back
  const closed = call(st, 'work_list_iterations', {}).value.find((i) => (i.attributes || {}).timeFrame === 'past');
  const byId = call(st, 'wit_get_work_items_for_iteration', { iterationId: closed.id });
  assert.ok(byId.value.length >= 1, 'resolves the closed sprint by id (velocity would be 0 otherwise)');
  const pts = byId.value.reduce((n, wi) => n + (wi.fields['Microsoft.VSTS.Scheduling.StoryPoints'] || 0), 0);
  assert.ok(pts > 0, 'story points summable via the id path');
});

test('WIQL UNDER scopes to a path; IN matches a value set; an unknown operator narrows to empty', () => {
  const { st } = tmpStore();
  const allDone = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'Done'" });
  const under = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.IterationPath] UNDER 'DeliveryTest\\Sprint 1' AND [System.State] = 'Done'" });
  assert.ok(under.workItems.length >= 1);
  assert.ok(under.workItems.length < allDone.workItems.length, 'UNDER narrows to one sprint, not the whole store');
  call(st, 'wit_create_work_item', { workItemType: 'Product Backlog Item', fields: [{ name: 'System.Title', value: 'fresh' }, { name: 'System.State', value: 'New' }] });
  const inq = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.State] IN ('New','Approved')" });
  assert.equal(inq.workItems.length, 1, "IN matches the value set");
  const unknown = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.Title] LIKE 'foo'" });
  assert.equal(unknown.workItems.length, 0, 'unsupported operator narrows to empty, never a full-store scan');
});

test('wit_list_backlog_work_items excludes Completed (Done) items — active backlog only', () => {
  const { st } = tmpStore();
  assert.equal(call(st, 'wit_list_backlog_work_items', {}).value.length, 0, 'the seed is all Done -> active backlog empty');
  call(st, 'wit_create_work_item', { workItemType: 'Product Backlog Item', fields: [{ name: 'System.Title', value: 'new work' }] });
  assert.equal(call(st, 'wit_list_backlog_work_items', {}).value.length, 1, 'a New PBI is an active backlog candidate');
});

test('WIQL ORDER BY places a missing-sort-field item last (stable)', () => {
  const { st } = tmpStore();
  const a = call(st, 'wit_create_work_item', { workItemType: 'Product Backlog Item', fields: [{ name: 'System.Title', value: 'A' }, { name: 'System.State', value: 'New' }, { name: 'Microsoft.VSTS.Common.StackRank', value: 2 }] });
  const b = call(st, 'wit_create_work_item', { workItemType: 'Product Backlog Item', fields: [{ name: 'System.Title', value: 'B' }, { name: 'System.State', value: 'New' }] });
  const c = call(st, 'wit_create_work_item', { workItemType: 'Product Backlog Item', fields: [{ name: 'System.Title', value: 'C' }, { name: 'System.State', value: 'New' }, { name: 'Microsoft.VSTS.Common.StackRank', value: 1 }] });
  const r = call(st, 'wit_query_by_wiql', { wiql: "SELECT [System.Id] FROM WorkItems WHERE [System.State] = 'New' ORDER BY [Microsoft.VSTS.Common.StackRank] ASC" });
  assert.deepEqual(r.workItems.map((w) => w.id), [c.id, a.id, b.id], 'ranked first (C,A), unranked (B) last');
});

test('wiki upsert is idempotent (§8)', () => {
  const { st } = tmpStore();
  call(st, 'wiki_create_or_update_page', { wikiId: 'wiki-1', path: 'Sprints/Sprint-4-Review', content: 'v1' });
  call(st, 'wiki_create_or_update_page', { wikiId: 'wiki-1', path: 'Sprints/Sprint-4-Review', content: 'v2' });
  const pages = call(st, 'wiki_list_pages', {});
  assert.equal(pages.value.filter((p) => p.path === 'Sprints/Sprint-4-Review').length, 1, 'upsert, not append');
  assert.equal(call(st, 'wiki_get_page_content', { path: 'Sprints/Sprint-4-Review' }).content, 'v2');
});

test('PR lifecycle: create -> vote -> auto-complete is the only merge mechanism', () => {
  const { st } = tmpStore();
  const pr = call(st, 'repo_create_pull_request', { sourceRefName: 'refs/heads/feature/x', targetRefName: 'refs/heads/dev', title: 'x' });
  assert.equal(pr.status, 'active');
  call(st, 'repo_vote_pull_request', { pullRequestId: pr.pullRequestId, vote: 10 });
  const completed = call(st, 'repo_update_pull_request', { pullRequestId: pr.pullRequestId, autoCompleteSetBy: 'reviewer' });
  assert.equal(completed.status, 'completed');
});

test('curated-map boundary: off-contract tools are absent', () => {
  // these are REAL azureDevOps MCP tools but deliberately NOT in the delivery adapter map
  assert.equal(tools['wit_update_work_item_comment'], undefined);
  assert.equal(tools['work_list_team_iterations'], undefined);
  assert.equal(tools['work_get_iteration_capacities'], undefined);
  assert.equal(tools['pipelines_run_pipeline'], undefined);
  assert.equal(tools['testplan_create_test_plan'], undefined);
  // the sanctioned equivalents ARE present
  assert.ok(tools['wit_add_work_item_comment']);
  assert.ok(tools['work_list_iterations']);
  assert.ok(tools['work_get_team_capacity']);
});

test('persistence: a fresh Store process reloads the prior mutations', () => {
  const { st, dir } = tmpStore();
  call(st, 'wit_create_work_item', { workItemType: 'Epic', fields: [{ name: 'System.Title', value: 'Persist me' }] });
  st.save();
  const reloaded = new Store(path.join(dir, 'store.json'));
  const titles = Object.values(reloaded.state.workItems).map((wi) => wi.fields['System.Title']);
  assert.ok(titles.includes('Persist me'), 'mutation survived a fresh process');
});

test('MCP protocol: a spawned server answers initialize / tools/list / tools/call', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'mockproto-'));
  const input = [
    JSON.stringify({ jsonrpc: '2.0', id: 1, method: 'initialize', params: { protocolVersion: '2024-11-05' } }),
    JSON.stringify({ jsonrpc: '2.0', method: 'notifications/initialized' }),
    JSON.stringify({ jsonrpc: '2.0', id: 2, method: 'tools/list' }),
    JSON.stringify({ jsonrpc: '2.0', id: 3, method: 'tools/call', params: { name: 'core_list_projects', arguments: {} } }),
    JSON.stringify({ jsonrpc: '2.0', id: 4, method: 'tools/call', params: { name: 'nope_off_contract', arguments: {} } }),
  ].join('\n') + '\n';

  const res = spawnSync('node', [path.join(__dirname, 'server.js')], {
    input, encoding: 'utf8',
    env: Object.assign({}, process.env, { MOCK_STORE: path.join(dir, 'store.json') }),
  });
  assert.equal(res.status, 0, res.stderr);
  const msgs = res.stdout.trim().split('\n').filter(Boolean).map((l) => JSON.parse(l));
  const byId = Object.fromEntries(msgs.filter((m) => m.id != null).map((m) => [m.id, m]));

  assert.equal(byId[1].result.serverInfo.name, 'mock-azure-devops');
  assert.ok(byId[2].result.tools.length >= 30, 'advertises the curated surface');
  assert.ok(byId[2].result.tools.some((t) => t.name === 'wit_create_work_item'));
  const projects = JSON.parse(byId[3].result.content[0].text);
  assert.ok(projects.value.length >= 1);
  assert.equal(byId[4].result.isError, true, 'off-contract tool call is an error, not a silent success');
});
