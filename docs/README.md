# docs/

The VitePress docs site (EN canonical → TR mirror) for the ATL platform.

**Status:** ported from v1 `agentteamland/docs` (content-port part 6, source-prep). The cross-repo docs-sync machinery is retired under v2 (concept inventory item 12); only the EN→TR site flow survives.

```bash
npm install
npm run docs:dev     # local preview
npm run docs:build   # static build
```

**Pending (deploy step):** the VitePress `base` (`/docs/`) and the repo/edit links still point at the v1 topology; they're re-pointed to the monorepo when v2 docs deployment is wired (alongside distribution). Content is current; only the deploy-time config needs the v2 sweep.
