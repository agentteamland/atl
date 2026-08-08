---
knowledge-base-summary: "What counts as evidence, and which cheap-looking checks produce a green that is not one. The tiering: unit tests on the rule, in-process HTTP tests against the **composed** app (the only gate that catches an unregistered endpoint), and a real dependency — a throwaway container — for anything whose behaviour lives in the broker, the cache or the database rather than in C#. The rule that generalizes: a fresh environment cannot prove a fix for a defect that only exists where data already is, so exercise BOTH states. A method with zero callers is unverified code, not working code."
---

# Testing and verification

A .NET API unit is verified entirely on the **code surface** — there is no browser and no emulator to
drive. That makes the code-surface gate *the whole gate*, so it has to be trustworthy on its own. This
page is what makes it trustworthy: which tier proves what, which cheap-looking check produces a green
that is not one, and what gets attached as evidence.

Everything below exists because some faster check passed over a real defect.

## The three tiers, and what only each one can prove

### Tier 1 — unit tests on the rule (the wide base)

A handler with its dependencies expressed as interfaces tests with fakes: no host, no database, no
broker, milliseconds, in parallel. **Every business rule, boundary and error path belongs here** —
including the ones that are annoying to reach through HTTP, which is exactly why they end up untested
otherwise.

Validators test the same way and are worth testing directly: a validator is a pure function from a
command to a failure set, and its per-field failures are a contract the client binds to.

**Push logic down.** Proving a rule through an HTTP round trip is slower, couples the rule to the
transport, and tests the rule and the wiring at once — so when it goes red you learn less.

### Tier 2 — in-process HTTP against the **composed** app (the thin band)

This tier exists to prove what a unit test structurally cannot see: **that the code is wired in.**

```csharp
public class ReportEndpointTests : IClassFixture<WebApplicationFactory<Program>>
{
    private readonly HttpClient _client;
    public ReportEndpointTests(WebApplicationFactory<Program> factory) => _client = factory.CreateClient();

    [Fact]
    public async Task Post_creates_and_returns_201()
    {
        var res = await _client.PostAsJsonAsync("/api/reports", new { title = "q3", range = "month" });
        Assert.Equal(HttpStatusCode.Created, res.StatusCode);
    }

    [Fact]
    public async Task Post_with_invalid_body_returns_400_with_the_field_dictionary()
    {
        var res = await _client.PostAsJsonAsync("/api/reports", new { title = "" });
        Assert.Equal(HttpStatusCode.BadRequest, res.StatusCode);
        var problem = await res.Content.ReadFromJsonAsync<ValidationProblemDetails>();
        Assert.True(problem!.Errors.ContainsKey("Title"));   // the contract key, not display text
    }
}
```

`WebApplicationFactory<TEntryPoint>` boots **the real host** — the real `Program.cs`, the real DI
registrations, the real middleware order, the real endpoint mapping — in-process, with no port and no
network. That is the whole point: it is the **only** gate in this stack that catches

- an endpoint whose group was never mapped in the composition root,
- a hosted service or a dependency never registered,
- middleware in the wrong position,
- a route sitting under the wrong prefix.

Each of those compiles, unit-tests green, and is dead at runtime.

Two mechanics to get right, both of which cost an hour the first time:

- **`Program` must be reachable from the test assembly.** With top-level statements the generated class
  is internal, so add `public partial class Program { }` at the bottom of `Program.cs` (or expose it via
  `InternalsVisibleTo`). Without it the factory has no entry point to boot.
- **Substitute only what genuinely cannot run in a test.** Override registrations in
  `ConfigureWebHost`/`WithWebHostBuilder` for the outbound edges (a payment gateway, an SMTP client), and
  leave the pipeline, the routing and the error mapping real. Every registration you replace is a piece
  of the composed app you are no longer testing.

**Test the failure contract, not just the happy path.** The boundary cases — the validation reject with
its field-keyed dictionary, the business reject with its status and its problem shape — are the evidence
a reviewer actually needs. A suite that proves only the success path proves the least interesting half,
and it is the error path that catches middleware registered in the wrong order.

### Tier 3 — a real dependency, in a throwaway container

Some behaviour does not live in C# at all. It lives in the database engine, the broker, or the cache, and
no amount of mocking will produce it:

| Behaviour | Where it actually lives |
|---|---|
| a unique / partial-unique index rejecting a duplicate | the database |
| a migration applying to a table that already has rows | the database |
| transaction rollback, isolation, a concurrency-token conflict | the database |
| SQL translation of a projection or a query filter | the provider |
| a queue argument mismatch, DLX routing, a redelivery | the broker |
| key expiry, an atomic set-if-not-exists | the cache |

Stand one up per run — Testcontainers, or a plain `docker run` you tear down — apply migrations to it
fresh, and reset between runs so tests stay independent and order-free.

**The in-memory provider is not a database.** It does not enforce unique indexes, foreign keys or check
constraints, it does not run relational SQL translation, and its transaction semantics are not the
engine's. So a test on the in-memory provider **passes over exactly the constraint the code relies on** —
the partial unique index that backs an idempotent fan-out, for instance, does not exist there, and the
duplicate the test was written to reject is happily inserted. Use it, if at all, for something whose
behaviour is genuinely provider-independent, and never as the proof that a database-level guarantee holds.

**Anything you create is shared state and a cleanup debt.** If tests run against a shared instance rather
than a per-run container, a leftover row becomes the next person's phantom bug. Record what you created
and reverse it — or use the throwaway container, which makes the question moot.

## The rule that generalizes: a fresh environment cannot prove a data-dependent fix

The sharpest failures in this stack exist **only where data already is**:

- a queue argument change fails only on a broker that already holds the queue;
- a destructive migration (a `NOT NULL` on a populated table, a type narrowing, a positional
  column rename) applies cleanly to an empty schema and destroys a populated one;
- a batch operation that is quadratic in the number of tracked entities is instant at test volumes;
- a backfill covers only rows that existed when it ran, so anything created afterwards has no row;
- a cached query plan frozen with the wrong captured value needs a *second* caller to expose it.

Every one of these is green on a developer machine, green in CI, green on a clean install — and red on
the environments that matter. **So the bar is both states, before merge:** the fresh path *and* a state
seeded to look like the one the code exists for. Testing only the fresh path proves nothing about the
case the code was written for; it is the one path that cannot fail.

This is also why a fresh-environment run is not a regression test for this class of defect. State the
condition you seeded, not just that you ran it.

## Reflexes

- **A method with zero callers is unverified code, not working code.** Before building on an existing
  infrastructure method, check it has a caller. If it does not, exercise it end to end first — it has
  never run, and "it compiles and looks right" is how it got there.
- **Ask of every assertion: what would this report if the thing under test simply had not happened?** If
  the answer is "pass", the assertion is vacuous — worse than absent, because it counts as coverage.
  Assert the *event* first (a row appeared, a status changed, a message was consumed), correctness second.
- **Take a positive control before calling anything absent, dead or broken.** An absence is a claim about
  your instrument as much as about the system. Run the identical probe somewhere a positive result is
  known to exist. A run of "dead endpoint" findings is usually the harness.
- **Assert the fixture before believing the measurement.** When a probe depends on a consumable — a
  one-shot token, a seeded row, a quota — a depleted consumable and a real defect are indistinguishable
  from the status code alone.
- **Never hang `&& echo passed` off a pipeline.** In a pipeline the exit status is the *last* command's,
  so `dotnet test | tail -5 && echo "clean"` prints green when the test host never ran at all. Redirect to
  a file, capture the status from the bare command, print the code you got. **A verification whose failure
  mode is indistinguishable from its success mode is not a verification.**
- **Say which level you actually reached.** "Builds" is not "tests pass" is not "observed running". Label
  a source-only check honestly — *"source-verified, not runtime-verified"* — rather than implying a green
  nobody observed. A claim is never re-checked when the standard is later raised.
- **Prove a new gate bites by injecting a deliberate error.** A gate you never watched fail is a gate you
  have not tested. This is not theoretical: a check that reports clean *always* looks exactly like a check
  that reports clean *correctly*.

## Runtime traps that read as code failures

- **Restart the process before trusting a live call after a signature change.** Hot Reload cannot apply a
  rude edit — a changed method signature, a changed type — to a running process. What it emits instead is
  an edit-and-continue diagnostic and a file-not-found style exception that reads exactly like a broken
  build. Body-only edits reload; signature and type changes need a restart.
- **Code that is not the code being served is not under test.** If the running process was started from a
  different checkout, a different image, or a stale build output, then querying it measures somebody
  else's code and reports itself verified. Establish *which* build the process is running before drawing
  any conclusion from it — and if you cannot, say the check was build-level only.
- **A red that is an environment artefact is not a code failure.** Report it as what it is, and do not
  "fix" the code to make it go away.

## The gate, and the evidence

The code-surface gate is a conjunction, all green:

```bash
dotnet build -c Release          # a suite passing over code that does not compile is a false green
dotnet test                      # unit + in-process HTTP + real-dependency suites
```

Add whatever static analysis the project actually has configured. **Check that it exists before running
it**: invoking an analyzer through a package runner against a project with no configuration downloads a
tool and points it at nothing, and a clean report from a tool with no rules is not a clean codebase.

**Coverage — produced, never claimed.** The bar is a conjunction: **diff coverage ≥ 90% of the lines
this change added or modified**, *and* **at least one test that goes RED when the change is reverted**.
The first half is mechanical; the second is what makes it mean anything, because a test that calls the
code and asserts nothing scores 100%. Confirming it costs a minute: revert the change, run the test,
see red, restore.

Generate the report with the project's configured runner — for most .NET projects
`dotnet test --collect:"XPlat Code Coverage"` produces a Cobertura report. Where the project has **no**
coverage tooling at all, that is a project-level finding to record, not a reason to skip the test:
*no test, no green*, regardless of whether the project can measure. Where 90% is genuinely unreachable,
write the exception down — which lines and why — never a silent pass.

**Attach the run's output, not a pass count.** For this unit type the decisive evidence is the
demonstrated contract: the success status and shape, plus the boundary rejections with their problem
shape. A raw "42 passed" says nothing about which 42.

State plainly in the hand-off **which checks you observed and which you assumed** — most of all whether
the HTTP tests boot the real host, since that single line is the difference between a proven wiring and a
fictional one. An unverified claim is worse than an admitted gap: the tester's independent pass gates the
green, and it can only check what it is told about.

## Checklist

- [ ] Rules, boundaries and error paths covered by unit tests against fakes — no host, no database.
- [ ] The HTTP contract covered by in-process tests booting the **real** host: success status and shape,
      the validation reject with its field-keyed dictionary, the business reject with its status.
- [ ] `Program` reachable from the test assembly; only outbound edges substituted, pipeline left real.
- [ ] Anything living in the database, the broker or the cache exercised against a **real** one in a
      throwaway container — not the in-memory provider.
- [ ] Any data-dependent change exercised in **both** states: fresh, and seeded to look like production.
- [ ] Every new method has a caller; every new assertion fails when the thing under test does not happen.
- [ ] `dotnet build` + `dotnet test` green, output captured, and the coverage conjunction satisfied or its
      exception recorded on the work-item.
- [ ] The hand-off says which level each claim was verified at.
