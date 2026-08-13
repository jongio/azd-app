# Vally Skill Evaluations

Evaluation suites for azd-app Copilot skills, powered by [@microsoft/vally](https://aka.ms/vally).

## Structure

```
cli/evals/
├── .vally.yaml              # Root config, suite definitions
├── package.json             # Dev dependency + npm scripts
├── .gitignore               # Ignores results/ and node_modules/
└── azd-app-onboard/
    └── eval.yaml            # Onboarding skill eval suite
```

## Running Locally

```bash
cd cli/evals

# Install Vally CLI
npm install

# Run smoke suite (fast: routing checks only)
npm run eval:smoke

# Run a specific skill's eval
npm run eval:skill -- azd-app-onboard/eval.yaml --output-dir ./results

# Run full suite (all stimuli, LLM-backed)
npm run eval:full

# Re-grade an existing trajectory
npm run grade -- azd-app-onboard/eval.yaml < results/results.jsonl
```

## Suites

| Suite | Filter | Use Case |
|-------|--------|----------|
| `smoke` | `tier: smoke` | PR gate, fast routing checks |
| `pr` | `tier: smoke` | Same as smoke (alias for clarity) |
| `routing` | `type: routing` | All routing/trigger evals |
| `integration` | `type: integration` | Full behavior tests (LLM-backed) |
| `full` | (none) | All evals, nightly CI |

## CI Integration

- **On PR** (skill/eval changes): Runs `smoke` suite automatically
- **Nightly** (2am UTC): Runs `full` suite
- **Manual**: Dispatch with any suite via workflow_dispatch

Results are uploaded as artifacts and summarized in the PR check.

## Adding Evals for a New Skill

1. Create `cli/evals/<skill-name>/eval.yaml`
2. Add stimuli with appropriate `tags.tier` (`smoke` for PR gate, `full` for nightly)
3. Use graders: `skill-invocation`, `output-matches`, `output-not-matches`
4. Test locally with `npm run eval:skill -- <skill-name>/eval.yaml`

## Authentication

- **Locally**: Uses your `gh` CLI session automatically
- **CI**: Uses `COPILOT_GITHUB_TOKEN` (set from `secrets.GITHUB_TOKEN`)
