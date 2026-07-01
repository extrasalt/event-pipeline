# AI Coding Strategy

## Problem

AI code often lacks proper abstractions — a common AI miss. If code is not layered, any future change ripples through the deepest parts of the codebase. But creating the wrong kind of abstractions prematurely will bite us too.

## Cause

AI tends to produce spaghetti code because it does not know where layer boundaries are, it either generates no abstractions or invents them before a second use case exists, and the first version calcifies without permission to throw it away.

## Solution

### Tooling

- **opencode** — code generation, refactoring, exploration
- **vp** (Vite+) — deterministic scaffolding (`vp create`), build, lint, format

### Guardrails

Ensure the agent uses libraries from npm or shadcn components where appropriate. Do not have it generate code that's available off the shelf. If a popular library exists, use it rather than maintain generated code.

| What | Supervision | Why |
|------|-------------|------|
| Scaffolding (`vp create`, `go mod init`) | None | Deterministic tools |
| UX components (tailwind classes) | Visual review | Deficiencies apparent on render |
| Unit tests | Spot-check | LLMs handle these well; add missing tests after |
| YAML configs (Grafana, Docker Compose) | Spot-check | Safe to generate |
| Pipeline stages (Map/Filter/Reduce) | Design review | Core logic, correctness matters |
| Auth / security | Full review | JWT, cookies, validation |

### Throwaway-then-build (per module)

**Pass 1 — Throwaway.** Build the whole module as a single-file prototype. No interfaces, no abstractions. Run it against real data to discover missing requirements and awkward shapes. Then delete it.

**Pass 2 — Build.** With those lessons, define layer boundaries and interfaces. Generate each layer independently, bottom-up.

### Cake code (layering)

Code quality is maintained by making sure abstractions are created. If code is properly layered, future changes stay localized — "cake code" not "spaghetti code".

- **Bottom-up.** Start with the innermost layer (pipeline), verify it, then the next (API), then the outer (BFF/frontend).
- **One layer per prompt.** Never cross layers in a single prompt.
- **Abstractions must earn their keep.** Write concrete code first. Extract an interface only when a second use case appears.
