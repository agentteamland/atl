// store.js — the in-memory work-item store behind the mock azureDevOps MCP server.
//
// This is the "#3 adapter second implementation" the delivery-team test strategy
// (detail-spec #18) calls for: a faithful STATE store the ceremonies' own
// idempotency / velocity / convergence logic can run against, WITHOUT live Azure.
// It is deliberately NOT a code re-implementation of the adapter — the adapter is a
// documented contract (knowledge/azure-adapter.md); this just holds believable state
// so a real ceremony (LLM) drives it over MCP and we assert on the resulting store.
//
// Zero dependencies (pure Node). Persists to a JSON file between processes because
// Claude Code spawns a fresh stdio MCP process per `claude -p` turn, and the
// ceremony chain (kickoff -> refine -> sprint-plan -> ...) spans several turns.

'use strict';
const fs = require('fs');
const path = require('path');

// ---- The Scrum process template (§6 runtime type/state resolution) ------------
// State names + the state->category map a ceremony/worker MUST resolve at runtime
// (never hardcode a literal "Done"). Scrum spells Completed as "Done" but maps
// "Committed"/"In Progress" to the InProgress category and "New"/"Approved" to
// Proposed — so resolving "the Completed-category state" is a real lookup, not a
// string literal. This is what wit_get_work_item_type serves.
const SCRUM_STATES = {
  Epic: ['New', 'In Progress', 'Done', 'Removed'],
  Feature: ['New', 'In Progress', 'Done', 'Removed'],
  'Product Backlog Item': ['New', 'Approved', 'Committed', 'Done', 'Removed'],
  Task: ['To Do', 'In Progress', 'Done', 'Removed'],
  Bug: ['New', 'Approved', 'Committed', 'Done', 'Removed'],
};
// state -> category, per Scrum. Categories: Proposed | InProgress | Resolved |
// Completed | Removed. (Scrum has no Resolved category.)
const SCRUM_CATEGORY = {
  New: 'Proposed', Approved: 'Proposed',
  'To Do': 'Proposed',
  Committed: 'InProgress', 'In Progress': 'InProgress',
  Done: 'Completed',
  Removed: 'Removed',
};

function typeDef(name) {
  const states = SCRUM_STATES[name] || SCRUM_STATES['Product Backlog Item'];
  return {
    name,
    referenceName: 'Microsoft.VSTS.WorkItemTypes.' + name.replace(/\s+/g, ''),
    states: states.map((s) => ({ name: s, category: SCRUM_CATEGORY[s] || 'Proposed' })),
    fields: [
      'System.Title', 'System.Description', 'System.State', 'System.Tags',
      'System.IterationPath', 'System.WorkItemType', 'System.AreaPath',
      'Microsoft.VSTS.Scheduling.StoryPoints', 'Microsoft.VSTS.Common.StackRank',
    ],
  };
}

// ---- The seed fixture ---------------------------------------------------------
// A believable pre-init project: one project + repo (dev/release branches), the
// Scrum template, three CLOSED past iterations carrying Done PBIs with StoryPoints
// (so /sprint-plan velocity = mean over the last N=3 closed has real data), one
// current OPEN iteration, a small backlog of New PBIs to plan, and one wiki.
// Names are generic — no real org/project identifiers.
function seedState() {
  const proj = 'DeliveryTest';
  const iter = (n, closed, offsetDays) => {
    // Dates are fixtures, not "now" (the mock never reads a clock — determinism).
    const base = '2026-01-01';
    return {
      id: 'iter-' + n,
      name: 'Sprint ' + n,
      path: proj + '\\Sprint ' + n,
      attributes: {
        startDate: `2026-0${Math.min(9, n)}-01T00:00:00Z`,
        finishDate: `2026-0${Math.min(9, n)}-14T00:00:00Z`,
        timeFrame: closed ? 'past' : 'current',
      },
    };
  };
  const iterations = [iter(1, true), iter(2, true), iter(3, true), iter(4, false)];

  // Historical Done PBIs for velocity (mean of last 3 closed sprints).
  // Sprint 1: 3+5=8, Sprint 2: 5+8=13, Sprint 3: 8+5+3=16 -> mean ~ 12.3.
  const workItems = {};
  let nextId = 1;
  const addHistoric = (title, points, sprintN) => {
    const id = nextId++;
    workItems[id] = {
      id,
      fields: {
        'System.Id': id,
        'System.Title': title,
        'System.WorkItemType': 'Product Backlog Item',
        'System.State': 'Done',
        'System.IterationPath': proj + '\\Sprint ' + sprintN,
        'System.AreaPath': proj,
        'System.Tags': 'historic',
        'Microsoft.VSTS.Scheduling.StoryPoints': points,
        'Microsoft.VSTS.Common.StackRank': 1000 + id,
      },
      comments: [],
      relations: [],
    };
  };
  addHistoric('Login rate limiting', 3, 1);
  addHistoric('Password reset email', 5, 1);
  addHistoric('Product search filters', 5, 2);
  addHistoric('Cart persistence', 8, 2);
  addHistoric('Checkout address book', 8, 3);
  addHistoric('Order history page', 5, 3);
  addHistoric('Email receipt template', 3, 3);

  return {
    processTemplateName: 'Scrum',
    project: proj,
    projects: [{ id: 'proj-1', name: proj, state: 'wellFormed' }],
    repos: [{
      id: 'repo-1', name: proj, project: proj, defaultBranch: 'refs/heads/dev',
      branches: ['dev', 'release'],
    }],
    workItems,
    nextId,
    iterations,
    capacities: {},
    pullRequests: {},
    nextPrId: 1,
    wikis: [{ id: 'wiki-1', name: proj + '.wiki', type: 'projectWiki' }],
    wikiPages: {},
  };
}

class Store {
  constructor(file) {
    this.file = file;
    if (file && fs.existsSync(file)) {
      this.state = JSON.parse(fs.readFileSync(file, 'utf8'));
    } else {
      this.state = seedState();
      this.save();
    }
  }

  save() {
    if (!this.file) return;
    fs.mkdirSync(path.dirname(this.file), { recursive: true });
    fs.writeFileSync(this.file, JSON.stringify(this.state, null, 2));
  }

  // ---- work-item helpers ------------------------------------------------------
  createWorkItem(type, fieldsInput) {
    const id = this.state.nextId++;
    const fields = { 'System.Id': id, 'System.WorkItemType': type, 'System.State': this._defaultState(type) };
    // fieldsInput may be an array [{name,value}] (real wit_create schema) or a map.
    this._applyFields(fields, fieldsInput);
    if (!fields['System.AreaPath']) fields['System.AreaPath'] = this.state.project;
    if (!fields['System.IterationPath']) fields['System.IterationPath'] = this.state.project;
    const wi = { id, fields, comments: [], relations: [] };
    this.state.workItems[id] = wi;
    return wi;
  }

  _defaultState(type) {
    const states = SCRUM_STATES[type] || SCRUM_STATES['Product Backlog Item'];
    return states[0];
  }

  // Accepts the wit_* field shapes seen in the wild: an array of {name,value},
  // an array of {op,path:"/fields/X",value} (JSON-Patch), or a plain {field:value}.
  _applyFields(fields, input) {
    if (!input) return;
    if (Array.isArray(input)) {
      for (const it of input) {
        if (it && it.path && typeof it.path === 'string' && it.path.startsWith('/fields/')) {
          fields[it.path.slice('/fields/'.length)] = it.value;
        } else if (it && it.name) {
          fields[it.name] = it.value;
        }
      }
    } else if (typeof input === 'object') {
      for (const [k, v] of Object.entries(input)) fields[k] = v;
    }
  }

  updateWorkItem(id, updates) {
    const wi = this.state.workItems[id];
    if (!wi) return null;
    this._applyFields(wi.fields, updates);
    return wi;
  }

  getWorkItem(id) { return this.state.workItems[id] || null; }

  addComment(id, text) {
    const wi = this.state.workItems[id];
    if (!wi) return null;
    const c = { id: wi.comments.length + 1, text: String(text) };
    wi.comments.push(c);
    return c;
  }

  addRelation(id, rel, targetId, extra) {
    const wi = this.state.workItems[id];
    if (!wi) return null;
    wi.relations.push(Object.assign({ rel, targetId }, extra || {}));
    return wi;
  }

  // ---- a small WIQL subset ----------------------------------------------------
  // Handles the query SHAPES the ceremonies emit: tag CONTAINS, field = 'value'
  // (WorkItemType/State/IterationPath), ANDed, with an optional ORDER BY and top.
  // Not a full WIQL parser (that's overkill) — a pragmatic evaluator keyed off the
  // adapter's #5 idempotency probe + velocity/backlog reads.
  queryWiql(wiql, top) {
    const q = String(wiql || '');
    const conds = [];
    const re = /\[([\w.]+)\]\s*(=|<>|CONTAINS|contains|Contains)\s*'([^']*)'/g;
    let m;
    while ((m = re.exec(q)) !== null) {
      conds.push({ field: m[1], op: m[2].toUpperCase(), value: m[3] });
    }
    let items = Object.values(this.state.workItems).filter((wi) =>
      conds.every((c) => {
        const fv = wi.fields[c.field];
        if (c.op === 'CONTAINS') return String(fv || '').includes(c.value);
        if (c.op === '<>') return String(fv) !== c.value;
        return String(fv) === c.value;
      })
    );
    const orderM = /ORDER BY\s+\[([\w.]+)\]\s*(ASC|DESC)?/i.exec(q);
    if (orderM) {
      const f = orderM[1], dir = (orderM[2] || 'ASC').toUpperCase() === 'DESC' ? -1 : 1;
      items = items.slice().sort((a, b) => {
        const av = a.fields[f], bv = b.fields[f];
        if (av === bv) return 0;
        return (av > bv ? 1 : -1) * dir;
      });
    }
    const cap = typeof top === 'number' && top > 0 ? top : 200;
    return items.slice(0, cap).map((wi) => wi.fields['System.Id']);
  }

  // ---- iterations / velocity --------------------------------------------------
  closedIterations() {
    return this.state.iterations.filter((it) => (it.attributes || {}).timeFrame === 'past');
  }

  itemsForIteration(pathOrName) {
    return Object.values(this.state.workItems).filter((wi) => {
      const ip = wi.fields['System.IterationPath'] || '';
      return ip === pathOrName || ip.endsWith('\\' + pathOrName) || ip.endsWith(pathOrName);
    });
  }

  createIteration(name) {
    const it = {
      id: 'iter-' + (this.state.iterations.length + 1),
      name,
      path: this.state.project + '\\' + name,
      attributes: { timeFrame: 'current' },
    };
    this.state.iterations.push(it);
    return it;
  }

  // ---- runtime type resolution (§6) -------------------------------------------
  workItemType(name) { return typeDef(name); }

  // ---- PRs --------------------------------------------------------------------
  createPR(sourceRef, targetRef, title) {
    const id = this.state.nextPrId++;
    const pr = {
      pullRequestId: id, title: title || '', sourceRefName: sourceRef,
      targetRefName: targetRef, status: 'active', threads: [], votes: {},
    };
    this.state.pullRequests[id] = pr;
    return pr;
  }
  getPR(id) { return this.state.pullRequests[id] || null; }

  // ---- wiki (§8 idempotent upsert) --------------------------------------------
  upsertWiki(pagePath, content) {
    this.state.wikiPages[pagePath] = { path: pagePath, content: String(content || '') };
    return this.state.wikiPages[pagePath];
  }
  listWikiPages() { return Object.keys(this.state.wikiPages); }
  getWikiPage(pagePath) { return this.state.wikiPages[pagePath] || null; }
}

module.exports = { Store, typeDef, seedState, SCRUM_STATES, SCRUM_CATEGORY };
