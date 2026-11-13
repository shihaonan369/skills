# Issue tracker: GitHub

Issues and specs for this repo live in GitHub Issues for
`shihaonan369/skills`. Use the `gh` CLI for all operations.

## Conventions

Run commands inside this clone, or pass `--repo shihaonan369/skills`
to commands that accept it.

- Create: `gh issue create --title "..." --body-file <file>`.
  Use a temporary file or `--body-file -` with a heredoc for multiline bodies.
- Read: `gh issue view <number> --json number,title,body,labels,comments`.
- List: `gh issue list --state open --json number,title,body,labels`.
  Add appropriate `--label` and `--state` filters.
- Comment: `gh issue comment <number> --body "..."`.
- Apply/remove labels: `gh issue edit <number> --add-label "..."`
  or `--remove-label "..."`.
- Close: `gh issue close <number> --comment "..."`.
- Triage roles map to GitHub labels as defined in `triage-labels.md`.

## Pull requests as a triage surface

**PRs as a request surface: no.**

GitHub shares one number space across issues and pull requests.
If a reference might be a PR, resolve its type before acting.
PRs do not enter the issue triage queue under this configuration.

## When a skill says "publish to the issue tracker"

Create a GitHub issue. Specs are issues too; link implementation
issues to their originating spec.

## When a skill says "fetch the relevant ticket"

Read the referenced GitHub issue, including its comments and labels.

## Existing local material

Existing `.scratch/` files remain as historical or working material.
Do not automatically migrate or delete them. New tracked issues and
specs go to GitHub; explicitly referenced local files can still be read.

## Wayfinding operations

Used by `/wayfinder`. The map is one issue with child issues as tickets.

- Map: an issue labelled `wayfinder:map`, containing Notes,
  Decisions-so-far and Fog.
- Child ticket: link it as a GitHub sub-issue:
  ```bash
  gh api --method POST repos/shihaonan369/skills/issues/<map>/sub_issues \
    -F sub_issue_id=<child-db-id>
  ```
  If unavailable, use a task list in the map and put `Part of #<map>`
  at the top of the child body.
- Type: label children `wayfinder:<type>`, where type is `research`,
  `prototype`, `grilling` or `task`. Ensure needed wayfinding labels
  exist before applying them.
- Blocking: prefer native GitHub issue dependencies:
  ```bash
  gh api --method POST \
    repos/shihaonan369/skills/issues/<child>/dependencies/blocked_by \
    -F issue_id=<blocker-db-id>
  ```
  Obtain database IDs with
  `gh api repos/shihaonan369/skills/issues/<number> --jq .id`;
  these are not issue numbers or node IDs.
  If dependencies are unavailable, put `Blocked by: #N, #N`
  at the top of the child body.
- Frontier: inspect the map's open children in map order. Skip assigned
  tickets and those with an open blocker.
- Claim: `gh issue edit <number> --add-assignee @me` before work.
- Resolve: comment with the answer, close the child, then append a gist
  and link to Decisions-so-far in the map.

Wayfinding labels and assignment are separate from triage roles.
