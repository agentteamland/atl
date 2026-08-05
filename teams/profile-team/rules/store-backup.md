# Store backup — act on the no-remote signal

`atl session-start` reports when a declared durable store's history exists only on this
disk. **This rule is what to do about it for the profile store.**

## The signal

Two shapes, both printed by core, both naming no team:

```
atl: the durable store at ~/.atl/profiles has no remote — its history exists only on this
     disk, so losing the disk loses it. Raise this with the user in this session and follow
     the backup rule for that store's owning team.

atl: the durable store at ~/.atl/profiles is N commit(s) ahead of its remote — that much of
     it exists only on this disk. …
```

Core states the condition; it does not name a skill, because it does not know which team
owns which store. Naming the remedy is this rule's job.

## What to do

**Raise it with the user in the session it appears in, and run `/profile-backup`.** For the
*ahead-of-remote* shape the skill finishes on its own. For *no remote* it will stop at the
`no-remote` outcome and ask for a private repo URL — relay that ask and **wait**.

Three things this rule does **not** license, and each of them is the whole point:

- **Never choose the destination.** Do not propose a repo, do not reuse one you happen to
  see in the working tree, do not create one. Where a person's memory is stored is theirs
  to decide, and a plausible-looking guess is the one mistake here that cannot be undone.
- **Never satisfy the visibility gate by inference.** If `/profile-backup` returns
  `visibility-unknown`, the answer comes from the user in words. A URL that looks private,
  an account name, or the absence of evidence that it is public are all inferences.
- **Do not run this in the background and report it done.** The remedy needs an answer from
  a person; a subagent with nobody to ask cannot supply one. Notice autonomously, act with
  the user present.

## Why it is a signal and not an automatic push

The store holds tier-4 financial fields and perception-flagged records about third parties.
Pushing it anywhere is irreversible — git history keeps the content after a delete, and a
remote may already be cloned. So the machine is allowed to **notice** that no copy exists,
and the person is the one who decides where it goes. That boundary is deliberate: an
autonomous mechanism that could pick a destination would be one mistyped URL away from
publishing the most sensitive data on the machine.

## When it stops

The signal goes silent as soon as the history is pushed, and returns if the store moves
ahead of its remote again. Nothing to acknowledge, nothing to dismiss — if you still see
it, the backup still does not exist.
