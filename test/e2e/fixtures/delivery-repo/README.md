# atl-e2e-delivery — GitHub-backend delivery-team e2e fixture

A minimal, buildable repository the `github-delivery-loop` e2e blueprint drives the
delivery-team's autonomous loop against (the GitHub-backend **Layer-B / T-point** proof).
It is force-restored to this baseline at the start of every run by `reset_delivery_repo`,
so the run is repeatable.

The "app" is deliberately trivial — a developer worker only needs a real file to change and
a real test to pass so a PR can open, merge to `dev`, and close its issue.

- `app.js` — the trivial app (a single pure function).
- `app.test.js` — a `node --test` unit test the developer/tester exercises.
- `docs/` — the in-repo durable-knowledge store (`docs/domain/`, `docs/architecture/`, …)
  the ceremonies seed and read (GitHub backend, adapter §9).
