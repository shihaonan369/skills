# Domain Docs

How engineering skills consume this repo's domain documentation.

## Before exploring, read these

This repo uses a single-context layout:

- `GLOSSARY.md` at the repo root.
- Relevant ADRs under root `docs/adr/`.

If these files do not exist, proceed silently. Do not flag their absence
or suggest creating them upfront. `/domain-modeling` creates them lazily
when terms or decisions are resolved.

## File structure

- `/GLOSSARY.md`: shared domain vocabulary.
- `/docs/adr/NNNN-<decision-slug>.md`: architectural decisions.

## Use the glossary's vocabulary

When naming a domain concept in an issue title, refactor proposal,
hypothesis or test name, use the glossary's defined term rather than
synonyms it explicitly avoids.

If a concept is missing, reconsider whether it belongs to the project's
language. Note genuine gaps for `/domain-modeling`.

## Flag ADR conflicts

If a proposal contradicts an existing ADR, surface the conflict explicitly:

> Contradicts ADR-0007 (<decision>), but worth reopening because…
