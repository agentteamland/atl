#!/usr/bin/env node
// server.js — a mock `azureDevOps` MCP server (stdio, JSON-RPC 2.0), zero deps.
//
// Claude Code launches this per the blueprint's `.mcp.json` and speaks MCP over
// stdin/stdout (newline-delimited JSON-RPC). It advertises the curated adapter
// tool surface (tools.js) and serves it over the in-memory store (store.js),
// which persists to MOCK_STORE so state survives the fresh process Claude Code
// spawns for each `claude -p` turn in the ceremony chain.
//
// MCP methods handled: initialize, notifications/initialized, ping, tools/list,
// tools/call. Everything else -> JSON-RPC "method not found".

'use strict';
const readline = require('readline');
const path = require('path');
const { Store } = require('./store');
const { tools } = require('./tools');

const storeFile = process.env.MOCK_STORE || path.join(process.cwd(), '.delivery', 'mock-store.json');
const store = new Store(storeFile);

function send(msg) {
  process.stdout.write(JSON.stringify(msg) + '\n');
}

function handle(req) {
  const { id, method, params } = req;

  if (method === 'initialize') {
    send({
      jsonrpc: '2.0', id,
      result: {
        protocolVersion: (params && params.protocolVersion) || '2024-11-05',
        capabilities: { tools: {} },
        serverInfo: { name: 'mock-azure-devops', version: '0.1.0' },
      },
    });
    return;
  }
  // notifications carry no id and get no response
  if (method === 'notifications/initialized' || method === 'initialized') return;
  if (method === 'ping') { send({ jsonrpc: '2.0', id, result: {} }); return; }

  if (method === 'tools/list') {
    const list = Object.entries(tools).map(([name, t]) => ({
      name, description: t.description, inputSchema: t.inputSchema,
    }));
    send({ jsonrpc: '2.0', id, result: { tools: list } });
    return;
  }

  if (method === 'tools/call') {
    const name = params && params.name;
    const args = (params && params.arguments) || {};
    const t = tools[name];
    if (!t) {
      // A caller reaching for a real-but-off-contract tool lands here — the mock
      // exposes ONLY the curated adapter map, so this surfaces the defect instead
      // of silently succeeding.
      send({
        jsonrpc: '2.0', id,
        result: { content: [{ type: 'text', text: `unknown tool: ${name} (not in the curated azureDevOps adapter contract)` }], isError: true },
      });
      return;
    }
    let out;
    try {
      out = t.handler(store, args);
      store.save();
    } catch (e) {
      send({ jsonrpc: '2.0', id, result: { content: [{ type: 'text', text: `mock error: ${e && e.message}` }], isError: true } });
      return;
    }
    send({ jsonrpc: '2.0', id, result: { content: [{ type: 'text', text: JSON.stringify(out) }] } });
    return;
  }

  if (id !== undefined && id !== null) {
    send({ jsonrpc: '2.0', id, error: { code: -32601, message: `method not found: ${method}` } });
  }
}

const rl = readline.createInterface({ input: process.stdin, terminal: false });
rl.on('line', (line) => {
  const s = line.trim();
  if (!s) return;
  let req;
  try { req = JSON.parse(s); } catch { return; }
  try { handle(req); } catch { /* never crash the server on a bad message */ }
});
