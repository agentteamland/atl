// tools.js — the curated azureDevOps tool surface the mock advertises + handles.
//
// EXACTLY the tools in the delivery-team's curated adapter map (knowledge/
// azure-adapter.md §2). The real @azure-devops/mcp exposes far more (pipelines_*,
// testplan_*, advsec_*, wit_update_work_item_comment, work_list_team_iterations…);
// the mock deliberately OMITS everything off-contract — a caller reaching for a
// real-but-off-contract tool gets "unknown tool", which is the point (it surfaces
// the "real-but-off-contract" defect class the delivery build learned to guard).
//
// Each handler takes (store, args) and returns a plain JS value; server.js wraps it
// as an MCP text-content result. Argument parsing is intentionally lenient (an LLM
// caller varies field shapes) while the STATE effect stays faithful to the contract.

'use strict';

const S = (v) => (v == null ? '' : String(v));
const num = (v, d) => (typeof v === 'number' ? v : (v != null && !isNaN(+v) ? +v : d));

// Pull a work-item id out of the many arg names an LLM might use.
function wid(a) {
  return num(a.id ?? a.workItemId ?? a.workitemId ?? a.workItemID, undefined);
}
// Normalize a wit result to the Azure-ish {id, fields, relations} shape.
function wiOut(wi) {
  return wi ? { id: wi.id, fields: wi.fields, relations: wi.relations, comments: wi.comments.length } : null;
}

const tools = {
  // ---- discovery / core ----
  core_list_projects: {
    description: 'List projects in the organization.',
    inputSchema: { type: 'object', properties: {} },
    handler: (st) => ({ count: st.state.projects.length, value: st.state.projects }),
  },
  core_list_project_teams: {
    description: 'List teams for a project.',
    inputSchema: { type: 'object', properties: { project: { type: 'string' } } },
    handler: (st) => ({ value: [{ id: 'team-1', name: st.state.project + ' Team' }] }),
  },
  core_get_identity_ids: {
    description: 'Resolve identity ids.',
    inputSchema: { type: 'object', properties: { searchFilter: { type: 'string' } } },
    handler: () => ({ value: [{ id: 'id-1', displayName: 'Delivery Bot' }] }),
  },

  // ---- work-items (wit_*) ----
  wit_create_work_item: {
    description: 'Create a work-item (Epic/Feature/PBI/Task/Bug). Set System.Tags in the fields to stamp atl-key/area tags on creation.',
    inputSchema: {
      type: 'object',
      properties: {
        project: { type: 'string' },
        workItemType: { type: 'string' },
        fields: {
          type: 'array',
          description: 'Field values to set. To stamp tags, include a System.Tags field. Example: [{"name":"System.Title","value":"Login"},{"name":"System.Description","value":"..."},{"name":"System.Tags","value":"atl-run:kickoff:1; atl-key:abc123; area:web"}]',
          items: { type: 'object', properties: { name: { type: 'string' }, value: {} }, required: ['name', 'value'] },
        },
      },
      required: ['workItemType', 'fields'],
    },
    handler: (st, a) => wiOut(st.createWorkItem(S(a.workItemType || a.type || 'Product Backlog Item'), a.fields)),
  },
  wit_add_child_work_items: {
    description: 'Create child work-items under a parent. Each item carries its own fields (incl. System.Tags for atl-key/area stamping).',
    inputSchema: {
      type: 'object',
      properties: {
        parentId: { type: 'number' },
        project: { type: 'string' },
        workItemType: { type: 'string' },
        items: {
          type: 'array',
          description: 'Child items. Each: {"fields":[{"name":"System.Title","value":"..."},{"name":"System.Tags","value":"atl-key:...; area:web"}]}',
          items: { type: 'object', properties: { fields: { type: 'array' } } },
        },
      },
      required: ['workItemType', 'items'],
    },
    handler: (st, a) => {
      const parent = num(a.parentId ?? a.parentWorkItemId ?? a.id, undefined);
      const type = S(a.workItemType || 'Product Backlog Item');
      const items = Array.isArray(a.items) ? a.items : (Array.isArray(a.children) ? a.children : []);
      const created = items.map((it) => {
        const wi = st.createWorkItem(type, it.fields || it);
        if (parent) {
          st.addRelation(wi.id, 'System.LinkTypes.Hierarchy-Reverse', parent);
          st.addRelation(parent, 'System.LinkTypes.Hierarchy-Forward', wi.id);
        }
        return wiOut(wi);
      });
      return { created };
    },
  },
  wit_get_work_item: {
    description: 'Read one work-item.',
    inputSchema: { type: 'object', properties: { id: {}, project: {}, expand: {} }, required: ['id'] },
    handler: (st, a) => wiOut(st.getWorkItem(wid(a))) || { error: 'not found' },
  },
  wit_get_work_items_batch_by_ids: {
    description: 'Read a batch of work-items.',
    inputSchema: { type: 'object', properties: { ids: {}, project: {} }, required: ['ids'] },
    handler: (st, a) => {
      const ids = Array.isArray(a.ids) ? a.ids : S(a.ids).split(',').map((x) => +x.trim());
      return { value: ids.map((id) => wiOut(st.getWorkItem(num(id)))).filter(Boolean) };
    },
  },
  wit_get_work_items_for_iteration: {
    description: "Read a sprint's work-items.",
    inputSchema: { type: 'object', properties: { project: {}, team: {}, iterationId: {} } },
    handler: (st, a) => {
      const key = S(a.iterationId ?? a.iterationPath ?? a.iteration ?? a.sprint);
      return { value: st.itemsForIteration(key).map(wiOut) };
    },
  },
  wit_update_work_item: {
    description: 'Update work-item fields (state, IterationPath, tags, StoryPoints…) via JSON-Patch.',
    inputSchema: {
      type: 'object',
      properties: {
        id: { type: 'number' },
        project: { type: 'string' },
        updates: {
          type: 'array',
          description: 'JSON-Patch ops. Example: [{"op":"add","path":"/fields/System.IterationPath","value":"DeliveryTest\\\\Sprint 4"},{"op":"add","path":"/fields/System.Tags","value":"area:web"}]',
          items: { type: 'object', properties: { op: { type: 'string' }, path: { type: 'string' }, value: {} }, required: ['op', 'path', 'value'] },
        },
      },
      required: ['id', 'updates'],
    },
    handler: (st, a) => wiOut(st.updateWorkItem(wid(a), a.updates || a.fields)) || { error: 'not found' },
  },
  wit_update_work_items_batch: {
    description: 'Update a batch of work-items.',
    inputSchema: { type: 'object', properties: { updates: {} } },
    handler: (st, a) => {
      const ups = Array.isArray(a.updates) ? a.updates : [];
      const out = ups.map((u) => wiOut(st.updateWorkItem(num(u.id ?? u.workItemId), u.updates || u.fields))).filter(Boolean);
      return { value: out };
    },
  },
  wit_get_work_item_type: {
    description: "Resolve a type's states/fields at runtime (never hardcode).",
    inputSchema: { type: 'object', properties: { project: {}, workItemType: {} }, required: ['workItemType'] },
    handler: (st, a) => st.workItemType(S(a.workItemType || a.type || 'Product Backlog Item')),
  },
  wit_query_by_wiql: {
    description: 'Run a WIQL query (idempotency check, velocity, selection).',
    inputSchema: { type: 'object', properties: { wiql: {}, query: {}, top: {}, project: {} } },
    handler: (st, a) => {
      const q = a.wiql ?? a.query ?? (a.query && a.query.query) ?? '';
      const ids = st.queryWiql(typeof q === 'string' ? q : (q.query || ''), num(a.top, undefined));
      return { queryType: 'flat', workItems: ids.map((id) => ({ id })) };
    },
  },
  wit_add_work_item_comment: {
    description: 'Add a comment to a work-item.',
    inputSchema: { type: 'object', properties: { workItemId: {}, project: {}, comment: {}, format: {} }, required: ['comment'] },
    handler: (st, a) => {
      const c = st.addComment(wid(a), a.comment ?? a.text);
      return c ? { id: c.id, text: c.text } : { error: 'not found' };
    },
  },
  wit_list_work_item_comments: {
    description: 'List a work-item comments.',
    inputSchema: { type: 'object', properties: { workItemId: {}, project: {} }, required: ['workItemId'] },
    handler: (st, a) => {
      const wi = st.getWorkItem(wid(a));
      return { comments: wi ? wi.comments : [] };
    },
  },
  wit_link_work_item_to_pull_request: {
    description: 'Link a work-item to a PR.',
    inputSchema: { type: 'object', properties: { id: {}, pullRequestId: {}, project: {} } },
    handler: (st, a) => (st.addRelation(wid(a), 'ArtifactLink', num(a.pullRequestId), { name: 'Pull Request' }) ? { ok: true } : { error: 'not found' }),
  },
  wit_work_items_link: {
    description: 'Link work-items (Dependency, Parent…).',
    inputSchema: { type: 'object', properties: { updates: {}, id: {}, linkToId: {}, type: {} } },
    handler: (st, a) => {
      const pairs = Array.isArray(a.updates) ? a.updates : [a];
      let n = 0;
      for (const p of pairs) {
        const from = num(p.id ?? p.sourceId), to = num(p.linkToId ?? p.targetId ?? p.linkedId);
        const type = S(p.type || p.linkType || 'System.LinkTypes.Dependency-Forward');
        if (from && to && st.addRelation(from, type, to)) n++;
      }
      return { linked: n };
    },
  },
  wit_work_item_unlink: {
    description: 'Remove a link between work-items.',
    inputSchema: { type: 'object', properties: { id: {}, targetId: {} } },
    handler: (st, a) => {
      const wi = st.getWorkItem(wid(a));
      if (!wi) return { error: 'not found' };
      const to = num(a.targetId ?? a.linkToId);
      wi.relations = wi.relations.filter((r) => r.targetId !== to);
      return { ok: true };
    },
  },
  wit_list_backlogs: {
    description: 'List backlog levels.',
    inputSchema: { type: 'object', properties: { project: {}, team: {} } },
    handler: () => ({ value: [{ id: 'Microsoft.RequirementCategory', name: 'Backlog items' }, { id: 'Microsoft.EpicCategory', name: 'Epics' }] }),
  },
  wit_list_backlog_work_items: {
    description: 'List work-items on a backlog level.',
    inputSchema: { type: 'object', properties: { project: {}, team: {}, backlogId: {} } },
    handler: (st) => {
      const items = Object.values(st.state.workItems)
        .filter((wi) => wi.fields['System.WorkItemType'] === 'Product Backlog Item')
        .sort((a, b) => (a.fields['Microsoft.VSTS.Common.StackRank'] || 0) - (b.fields['Microsoft.VSTS.Common.StackRank'] || 0));
      return { value: items.map((wi) => ({ id: wi.id })) };
    },
  },
  wit_get_work_item_attachment: {
    description: 'Read an attachment on a work-item.',
    inputSchema: { type: 'object', properties: { id: {}, attachmentId: {} } },
    handler: () => ({ content: '(mock attachment bytes)', note: 'attachment READ is MCP; UPLOAD is the REST carve-out (out of Layer-A scope)' }),
  },

  // ---- iterations / capacity (work_*) ----
  work_list_iterations: {
    description: 'List the team iterations.',
    inputSchema: { type: 'object', properties: { project: {}, team: {} } },
    handler: (st) => ({ value: st.state.iterations }),
  },
  work_create_iterations: {
    description: 'Create iteration(s).',
    inputSchema: { type: 'object', properties: { project: {}, team: {}, iterations: {} } },
    handler: (st, a) => {
      const specs = Array.isArray(a.iterations) ? a.iterations : (a.name ? [{ name: a.name }] : []);
      return { value: specs.map((s) => st.createIteration(S(s.name || s.path || 'Sprint'))) };
    },
  },
  work_assign_iterations: {
    description: 'Assign an iteration to a team.',
    inputSchema: { type: 'object', properties: { project: {}, team: {}, iterationId: {} } },
    handler: () => ({ ok: true }),
  },
  work_get_team_capacity: {
    description: 'Read team capacity for an iteration.',
    inputSchema: { type: 'object', properties: { project: {}, team: {}, iterationId: {} } },
    handler: (st, a) => ({ iterationId: S(a.iterationId), teamMembers: [{ name: 'Delivery Bot', capacityPerDay: 6 }] }),
  },
  work_update_team_capacity: {
    description: 'Write team capacity.',
    inputSchema: { type: 'object', properties: { project: {}, team: {}, iterationId: {}, capacities: {} } },
    handler: (st, a) => { st.state.capacities[S(a.iterationId)] = a.capacities || []; return { ok: true }; },
  },

  // ---- repos / PRs (repo_*) ----
  repo_get_repo_by_name_or_id: {
    description: 'Read a repo.',
    inputSchema: { type: 'object', properties: { project: {}, repositoryNameOrId: {} } },
    handler: (st) => st.state.repos[0],
  },
  repo_get_branch_by_name: {
    description: 'Read a branch.',
    inputSchema: { type: 'object', properties: { repositoryId: {}, name: {}, project: {} } },
    handler: (st, a) => ({ name: S(a.name || 'dev'), objectId: 'deadbeef', repo: st.state.repos[0].name }),
  },
  repo_get_file_content: {
    description: 'Read a file from a branch.',
    inputSchema: { type: 'object', properties: { repositoryId: {}, path: {}, branch: {} } },
    handler: (st, a) => ({ path: S(a.path), content: '// mock file content' }),
  },
  repo_create_pull_request: {
    description: 'Open a pull request.',
    inputSchema: { type: 'object', properties: { repositoryId: {}, sourceRefName: {}, targetRefName: {}, title: {}, description: {} }, required: ['sourceRefName', 'targetRefName'] },
    handler: (st, a) => st.createPR(S(a.sourceRefName || a.source), S(a.targetRefName || a.target || 'refs/heads/dev'), S(a.title)),
  },
  repo_update_pull_request: {
    description: 'Update a PR (status / auto-complete = the only merge mechanism).',
    inputSchema: { type: 'object', properties: { repositoryId: {}, pullRequestId: {}, status: {}, autoCompleteSetBy: {} } },
    handler: (st, a) => {
      const pr = st.getPR(num(a.pullRequestId));
      if (!pr) return { error: 'not found' };
      if (a.status) pr.status = S(a.status);
      if (a.autoCompleteSetBy || a.autoComplete) pr.status = 'completed';
      return pr;
    },
  },
  repo_vote_pull_request: {
    description: 'Vote on a PR.',
    inputSchema: { type: 'object', properties: { repositoryId: {}, pullRequestId: {}, vote: {} } },
    handler: (st, a) => {
      const pr = st.getPR(num(a.pullRequestId));
      if (!pr) return { error: 'not found' };
      pr.votes.reviewer = num(a.vote, 10);
      return { ok: true, vote: pr.votes.reviewer };
    },
  },
  repo_create_pull_request_thread: {
    description: 'Add a review thread to a PR.',
    inputSchema: { type: 'object', properties: { repositoryId: {}, pullRequestId: {}, content: {} } },
    handler: (st, a) => {
      const pr = st.getPR(num(a.pullRequestId));
      if (!pr) return { error: 'not found' };
      const t = { id: pr.threads.length + 1, content: S(a.content), comments: [{ content: S(a.content) }] };
      pr.threads.push(t);
      return t;
    },
  },
  repo_list_pull_request_threads: {
    description: 'List a PR review threads.',
    inputSchema: { type: 'object', properties: { repositoryId: {}, pullRequestId: {} } },
    handler: (st, a) => { const pr = st.getPR(num(a.pullRequestId)); return { value: pr ? pr.threads : [] }; },
  },
  repo_reply_to_comment: {
    description: 'Reply to a PR thread.',
    inputSchema: { type: 'object', properties: { repositoryId: {}, pullRequestId: {}, threadId: {}, content: {} } },
    handler: (st, a) => {
      const pr = st.getPR(num(a.pullRequestId));
      const t = pr && pr.threads.find((x) => x.id === num(a.threadId));
      if (!t) return { error: 'not found' };
      t.comments.push({ content: S(a.content) });
      return { ok: true };
    },
  },

  // ---- wiki (wiki_*) ----
  wiki_list_wikis: {
    description: 'List project wikis.',
    inputSchema: { type: 'object', properties: { project: {} } },
    handler: (st) => ({ value: st.state.wikis }),
  },
  wiki_get_wiki: {
    description: 'Read a wiki.',
    inputSchema: { type: 'object', properties: { project: {}, wikiId: {} } },
    handler: (st) => st.state.wikis[0],
  },
  wiki_list_pages: {
    description: 'List wiki pages (namespace-exists check).',
    inputSchema: { type: 'object', properties: { project: {}, wikiId: {} } },
    handler: (st) => ({ value: st.listWikiPages().map((p) => ({ path: p })) }),
  },
  wiki_get_page_content: {
    description: 'Read a wiki page.',
    inputSchema: { type: 'object', properties: { project: {}, wikiId: {}, path: {} }, required: ['path'] },
    handler: (st, a) => st.getWikiPage(S(a.path)) || { error: 'not found', path: S(a.path) },
  },
  wiki_create_or_update_page: {
    description: 'Idempotent upsert of a wiki page.',
    inputSchema: { type: 'object', properties: { project: {}, wikiId: {}, path: {}, content: {} }, required: ['path'] },
    handler: (st, a) => st.upsertWiki(S(a.path), a.content),
  },

  // ---- discovery search (search_*) ----
  search_workitem: {
    description: 'Search work-items.',
    inputSchema: { type: 'object', properties: { searchText: {}, project: {} } },
    handler: (st, a) => {
      const q = S(a.searchText).toLowerCase();
      const hits = Object.values(st.state.workItems).filter((wi) => S(wi.fields['System.Title']).toLowerCase().includes(q));
      return { count: hits.length, results: hits.map(wiOut) };
    },
  },
  search_wiki: {
    description: 'Search wiki pages.',
    inputSchema: { type: 'object', properties: { searchText: {}, project: {} } },
    handler: (st, a) => {
      const q = S(a.searchText).toLowerCase();
      const hits = st.listWikiPages().filter((p) => p.toLowerCase().includes(q));
      return { count: hits.length, results: hits.map((p) => ({ path: p })) };
    },
  },
  search_code: {
    description: 'Search code.',
    inputSchema: { type: 'object', properties: { searchText: {}, project: {} } },
    handler: () => ({ count: 0, results: [] }),
  },
};

module.exports = { tools };
