# Issue tracker: Local Markdown

Issues and specs for this repo live as Markdown files in `.scratch/`.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`.
- Spec: `.scratch/<feature-slug>/spec.md`.
- Implementation issues: one file per ticket at
  `.scratch/<feature-slug>/issues/<NN>-<slug>.md`, numbered from `01`.
- Triage state: a `Status:` line near the top of each issue file.
  Use the role strings in `triage-labels.md`.
- Comments and conversation history: append under `## Comments`.

## When a skill says "publish to the issue tracker"

Create the appropriate spec or individual issue file using the paths above.
Create directories as needed.

## When a skill says "fetch the relevant ticket"

Read the referenced file. Resolve a bare issue number within its feature
directory; if the feature is ambiguous, ask for the feature or full path.

## Wayfinding operations

Used by `/wayfinder`. The map is a file with one child file per ticket.

- Map: `.scratch/<effort>/map.md`, containing Notes, Decisions-so-far and Fog.
- Child ticket: `.scratch/<effort>/issues/NN-<slug>.md`, numbered from `01`,
  with the question in the body.
- Type: a `Type:` line records `research`, `prototype`, `grilling` or `task`.
- Status: wayfinding tickets use `open`, `claimed` or `resolved`.
  These are separate from the triage vocabulary.
- Blocking: a `Blocked by: NN, NN` line near the top. A ticket is unblocked
  when every listed ticket is `resolved`.
- Frontier: scan for open, unblocked, unclaimed tickets; lowest number wins.
- Claim: set `Status: claimed` and save before work.
- Resolve: append the answer under `## Answer`, set `Status: resolved`,
  then append a gist and link to Decisions-so-far in the map.
