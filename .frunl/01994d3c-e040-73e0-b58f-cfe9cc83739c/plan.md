# SPEC

Detailed Feature Description
This feature adds an explicit MIT license file to the repository root and makes that license discoverable from the project's README. The goal is to make the legal terms for use, distribution, and contribution explicit and easily accessible to users, contributors, and automated tools that check repository licensing (CI scanners, package registries, license scanners).

Deliverables:
- A file named LICENSE at the repository root containing the canonical MIT license text. The file should include the current year and the repository's chosen copyright holder (a concrete organization or individual). If the owner is not known at implementation time, a clearly-marked placeholder will be used and annotated in the PR for maintainers to replace.
- An update to README.md that contains a clear "License" section and a relative link to ./LICENSE so that the license is discoverable from the project landing page.
- A small, high-level test (Go unit test or shell verification script) that asserts the file exists and that the README links to it. These tests enable automated verification by developers or CI without modifying existing workflows.

Why this exists / Value:
- Legal clarity: Consumers and contributors need to know under what terms they may use or contribute code.
- Discoverability: README is the primary entrypoint for new visitors; linking the license there reduces friction and improves compliance.
- Tooling and automation: Many open-source tools look for a root LICENSE file and README links; providing both ensures compatibility with automated license checks.

User Stories
- As a repository maintainer, I want a canonical MIT license file at the repository root, so that users and contributors can unambiguously understand the project's license.
- As a potential contributor, I want the README to contain a direct link to the license, so that I can quickly confirm contribution and reuse terms before submitting code.
- As a compliance auditor or automation tool, I want the license file to contain the standard MIT text and the SPDX identifier "MIT", so that automated checks and audit processes can validate licensing without manual interpretation.

Cucumber Scenarios

Scenario: Add MIT license file to repository root
Given a repository that does not contain a root-level LICENSE file
When a maintainer adds a file named "LICENSE" at the repository root containing the standard MIT license text including the SPDX identifier "MIT" and a valid year and copyright holder
Then the repository root contains a file named "LICENSE"
And the file contains the phrase "MIT License"
And the file contains the phrase "Permission is hereby granted, free of charge"

Scenario: README links to the LICENSE file
Given a repository with a README.md at the project root
When the README is updated to include a "License" section with a relative link to "./LICENSE"
Then README.md contains a Markdown link referencing "./LICENSE"
And a visitor viewing the README can click the link to reach the LICENSE file in the repository

Scenario: License is machine-verifiable and has required fields
Given the LICENSE file exists
When the LICENSE file is inspected by an automated check
Then the check finds the SPDX identifier "MIT"
And the check finds a four-digit year (e.g., 2025) and a non-empty copyright holder placeholder or name

# DESIGN

Technical Overview
This is a minimal documentation change plus a small verification test. No application code, configuration workflows, or database changes are required. Implementation will:
- Add a new file LICENSE at the repository root containing the canonical MIT license text. The year should default to the current year (2025) and the copyright holder should be the repository owner or a placeholder tagged for review.
- Modify README.md to add a "License" section with a relative link to ./LICENSE (and optionally add an SPDX badge if maintainers want).
- Add a single high-level unit test (Go) or simple verification script to assert the presence and basic validity of LICENSE and the README link so that CI or maintainers can run a fast check.

Constraints
- Do not modify .github/workflows/** (repository permissions do not allow it).
- License text should match canonical MIT wording (no paraphrase) to avoid legal ambiguity.
- Use a file named exactly LICENSE (case-sensitive recommended) placed at repository root to maximize compatibility with tools and registries.
- Any placeholders (owner) must be clearly annotated in the commit/PR for maintainers/legal to approve.

Component Breakdown
- LICENSE (new) — Responsibility: Hold the full canonical MIT license text, include SPDX identifier "MIT", year, and copyright holder.
- README.md (modified) — Responsibility: Add a "License" section that links to ./LICENSE and optionally mentions SPDX ID.
- license_test.go (new) — Responsibility: Provide a high-level unit test (Go) that verifies LICENSE presence and README link. This is intentionally lightweight and only checks presence and a small set of canonical phrases.
- (Optional) scripts/verify-license.sh (new) — Responsibility: Provide a small shell script that performs the same checks for manual/local verification. Optional and complementary to the Go test.

UI/UX Wireframes (README snippet)
A short ASCII representation of README changes:

------------------------------------------------------------
# Project Title

Short description...

## License

This project is licensed under the MIT License — see the [LICENSE](./LICENSE) file for details.
------------------------------------------------------------

ASCII component hierarchy (root)
- README.md (modify: add License section)
- LICENSE (add)
- go.mod
- main.go
- db/
  - data.db

Data Flow Diagrams
User/Tool -> README.md (landing) -> [License link] -> LICENSE (root)
ASCII:
[User or Bot] --> (open README.md) --> [## License link -> ./LICENSE] --> (open LICENSE text)

Logic & Business Rules (mapping to SPEC scenarios)
- Rule: A root-level file named "LICENSE" must exist. (Scenarios: Add MIT license file) — Task 1.1
- Rule: LICENSE must include exact canonical MIT wording (start with "MIT License" header, include "Permission is hereby granted, free of charge") and an SPDX identifier "MIT". (Scenarios: Add MIT license file, machine-verifiable) — Task 1.1, Task 1.3
- Rule: README.md must contain a "License" section with a relative link to "./LICENSE". (Scenarios: README links to the LICENSE) — Task 1.2
- Rule: Tests must pass locally (go test) verifying presence and minimal content of LICENSE and README link; do not add CI workflow changes. (Scenarios: Machine-verifiable) — Task 1.3

API Endpoints
- None required.

Database Schema Changes
- None.

Non-Functional Requirements
- Security/Legal: The MIT text must be the canonical text and must not be modified in a way that changes the legal meaning. If the copyright holder is uncertain, use a clearly-indicated placeholder and require PR reviewers (maintainers or legal) to replace it before merging.
- Performance: Negligible impact.
- Observability: CI status checks (existing) must report passing tests. The added unit test should run quickly (< 1s).
- Feature flags & Rollout: Not needed for a docs change; use branch + PR workflow. Keep the change behind a simple PR to allow code review and legal sign-off.
- Rollout Strategy: Implement on a short-lived branch (e.g., chore/add-license-2025), open PR, request maintainers/legal review, merge when approved.
- SPDX: Include "MIT" in the license file and mention the SPDX identifier in README for automated detection.

Assumptions & Open Questions
- Who should be listed as the copyright holder? (Assumption: use the repository owner or a placeholder "Copyright (c) 2025 <COPYRIGHT HOLDER>" and request that maintainers replace it.)
- Should license headers be added to source files? (Out of scope for this change; could be a follow-up.)
- Does the project prefer a LICENSE file with or without an extension? (We choose "LICENSE" without extension to maximize tool compatibility.)
- Do maintainers want a license badge in the README? (Optional; ask maintainers in PR.)

# TASKS

- [x] **Task 1.1: Add canonical MIT LICENSE file at repository root**
- **Status:** In Progress

Moving task to In Progress: Add canonical MIT LICENSE file at repository root
- **Context:**
 - Create a new file at ./LICENSE containing the canonical MIT license text (including SPDX identifier "MIT").
 - Insert current year (2025) and a COPYRIGHT HOLDER placeholder if the owner is not yet known. Mark the placeholder clearly in the file and PR description.
 - Reference SPEC rules: file name, canonical phrases, SPDX identifier.
- **Dependencies:** None
- **Validation:**
  - [x] Objective file/path checks: ./LICENSE exists at repository root
  - [x] Logic checks: LICENSE contains "MIT License", the phrase "Permission is hereby granted, free of charge", and "SPDX-License-Identifier: MIT" (or "MIT" visible)
  - [x] Passing high-level unit tests: Create/enable TestLicenseFileExists (see Task 1.3) and ensure it detects the file

- [x] **Task 1.2: Update README.md to link to the LICENSE**
- **Status:** Pending

Moving task to In Progress: Update README.md to link to the LICENSE
- **Context:**
 - Modify README.md to add a "## License" section with a relative link to ./LICENSE, e.g.:
   This project is licensed under the MIT License — see the [LICENSE](./LICENSE) file for details.
 - Optionally include "SPDX-License-Identifier: MIT" or a license badge if maintainers want it (annotate as optional in PR).
 - Reference SPEC user story (contributors discovering license) and DESIGN wireframe.
- **Dependencies:** Task 1.1
- **Validation:**
  - [x] Objective file/path checks: README.md contains the Markdown link "[LICENSE](./LICENSE)" or a variant that resolves to ./LICENSE
  - [x] Logic checks: The link text and path are correct and clearly labeled "License"
  - [x] Passing high-level unit tests: Create/enable TestReadmeLinksLicense (see Task 1.3) and ensure it passes

- [ ] **Task 1.3: Add lightweight verification tests and a local verification script**
- **Status:** In Progress

Moving task to In Progress: Add lightweight verification tests and a local verification script
- **Status:** Pending
- **Context:**
 - Add a small Go test file (e.g., license_test.go) at repository root or tests/ that implements:
   - TestLicenseFileExists: checks that ./LICENSE exists and contains the string "Permission is hereby granted"
   - TestReadmeLinksLicense: checks that README.md contains a relative link to ./LICENSE
 - Alternatively (or additionally) add scripts/verify-license.sh that performs the same checks with exit codes for local/manual runs.
 - Purpose: provide automated, quick verification for maintainers and CI (without changing CI workflows).
- **Dependencies:** Task 1.1, Task 1.2
- **Validation:**
  - [x] Objective file/path checks: license_test.go (or equivalent) exists at the planned path
 - [ ] Logic checks: Tests check for canonical MIT phrases and README link (see SPEC scenarios)
 - [ ] Passing high-level unit tests: Running `go test ./...` returns exit code 0 and the two tests pass (or running scripts/verify-license.sh exits 0)

- [ ] **Task 1.4: Commit best-practice, open PR, legal/maintainer review and merge; provide rollback plan**
- **Status:** Pending
- **Context:**
 - Create a branch chore/add-mit-license-2025, add changes (LICENSE, README.md, tests), commit with message:
   "chore: add MIT LICENSE and link from README (2025) — placeholder for copyright holder"
 - Open a PR with a description that explains the placeholder and requests legal/maintainer confirmation of the copyright holder and text.
 - Tag or request review from maintainers and optionally legal.
 - After approval and CI green, merge the PR.
 - Record change in changelog or release notes if the project keeps one.
- **Dependencies:** Task 1.1, Task 1.2, Task 1.3
- **Validation:**
 - [ ] Objective file/path checks: PR contains files ./LICENSE and modified README.md and test file(s)
 - [ ] Logic checks: PR description explicitly documents any placeholders and calls for approval/replacement
 - [ ] Passing high-level unit tests: CI (or local `go test ./...`) passes on PR
 - [ ] Rollback check: Documented rollback steps exist in PR description (e.g., "git revert <commit>" or "revert PR") and have been approved by maintainers

# VERIFICATION

Artifact Check
- Confirm the planning artifact exists:
  - .frunl/01994d3c-e040-73e0-b58f-cfe9cc83739c/plan.md (this file)

Traceability Table (Scenario → Design Element(s) → Task ID(s))
- "Add MIT license file to repository root"
  - Design Elements: LICENSE (new), canonical MIT text, SPDX identifier
  - Task IDs: Task 1.1, Task 1.3, Task 1.4
- "README links to the LICENSE file"
  - Design Elements: README.md modification, "## License" section, relative link ./LICENSE
  - Task IDs: Task 1.2, Task 1.3, Task 1.4
- "License is machine-verifiable and has required fields"
  - Design Elements: LICENSE content rules, tests (license_test.go or verify script)
  - Task IDs: Task 1.1, Task 1.3

Test Strategy (high-level unit tests only)
- Tests to implement (high-level, fast, single-responsibility):
  - TestLicenseFileExists
    - Purpose: Fail fast if LICENSE is missing or missing canonical phrase
    - High-level check: ensure ./LICENSE exists and contains "Permission is hereby granted"
    - Example run: go test -run TestLicenseFileExists ./...
  - TestReadmeLinksLicense
    - Purpose: Ensure README points to the license
    - High-level check: README.md contains "[LICENSE](./LICENSE)" or equivalent relative link
    - Example run: go test -run TestReadmeLinksLicense ./...
- Example local verification commands (for maintainers):
  - Check file exists: test -f LICENSE && echo "LICENSE present" || (echo "LICENSE missing" && exit 1)
  - Check README link: rg -n "\[License\]\(\./LICENSE\)" README.md || (echo "README missing license link" && exit 1)
  - Run tests: go test ./...
- Note: Keep tests minimal and fast; they are intended to surface obvious omissions, not to be a comprehensive legal verification.

Operational Readiness
- Metrics / Observability:
  - CI job status on PR must be green before merge (test pass).
  - PR should be reviewed and approved by at least one maintainer and, if required by policy, by a legal representative.
- Feature flag / Kill-switch:
  - Not applicable as this is a documentation change. Rollback mechanism: revert the PR or use git revert on the merge commit.
- Rollback Plan:
  - If a legal issue is discovered after merge, immediately:
    1. Revert the merge commit (git revert <merge-commit>) and push a revert PR or branch.
    2. Notify maintainers and legal.
    3. If removal of license introduces further actions (rare), coordinate with legal counsel.
- Post-merge housekeeping:
  - Replace placeholder COPYRIGHT HOLDER with final value in a follow-up PR if not resolved in the initial PR.
  - Consider adding license headers to source files as a separate follow-up task if desired.

End of plan.

Location: .frunl/01994d3c-e040-73e0-b58f-cfe9cc83739c/plan.md
