# `atl store`

Operations over the **durable stores** that installed teams declare.

A durable store is a directory a team declared as holding content that must survive — `capabilities.<name>.store` in its `team.json`, recorded at install time. Core never learns which team owns which path; it honors the declaration. Today the one first-party example is the profile store at `~/.atl/profiles`.

## `atl store version`

Bring every declared store under **local git** and commit whatever changed since the last snapshot.

```bash
atl store version
```

This is the retention floor for stores whose write policy is last-write-wins: a profile field that gets overwritten stays recoverable because the previous value is in a commit.

**It is the same pass `atl session-start` runs automatically** — one implementation with two triggers, rather than two mechanisms versioning the same directory on different terms. That matters because [`/profile-backup`](/teams/profile-team) invokes it: a store that is not yet a repo used to make backup refuse, telling the user that versioning was session-start's job, which is not something they could act on.

### What it declines to do, and why

`atl store version` decides eligibility for itself, so invoking it on demand is exactly as safe as the automatic pass:

- **An absent store is not created.** A missing directory means the owning feature is not in use on this machine; creating it would litter the disk *and* misreport the feature as active.
- **An empty store is not initialised.** Leaving a lone `.git` behind is not harmless — it makes the directory non-empty, and a consumer that tests emptiness to decide whether there is anything to work with then reports on a store holding nothing.
- **A store nested inside another repo is left alone.** Initialising there would shadow the outer repo, and committing would be writing into a repo this pass does not own.

### Output

```
versioned 2 durable store(s)     # something changed and was committed
no-store-versioned               # nothing eligible, or nothing changed
```

It never fails the caller. Unlike the session-start pass — which stays silent unless it did something — this prints either way, because something asked.

## Related

- `atl session-start` — runs the same versioning pass automatically, once per session.
- `/profile-backup` and `/profile-restore` — the off-machine half. Local git makes an overwritten value recoverable; it does nothing about losing the disk.
