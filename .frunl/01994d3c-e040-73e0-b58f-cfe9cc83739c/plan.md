# SPEC

Detailed Feature Description

This feature will add an MIT license file to the root of the repository and make sure the README links to it. The purpose is legal clarity for users and contributors, to enable automatic license detection by hosting platforms (GitHub/GitLab), and to reduce friction when others reuse or contribute to the code. The change is small and low-risk but important for open-source publishing and compliance.

Sub-features
- Add a canonical LICENSE file at repository root named exactly LICENSE (no extension) containing the canonical MIT license text. The file should include the correct copyright year and copyright holder (see Assumptions & Open Questions).
- Add a "License" section to README.md that contains a short statement and a link to ./LICENSE so visitors and tooling can easily find the license. The README section should be simple, human-readable, and machine-detectable.

User Stories

- As a repository maintainer, I want an MIT LICENSE file in the repo root, so that the legal grant of permission to use, copy, modify, and distribute the code is explicit.
- As a contributor, I want to find the license from README, so that I can quickly confirm the project's contribution/licensing terms before contributing.
- As a consumer/integrator, I want to verify the license text is canonical MIT, so that I can confidently reuse the code under known terms.
- As a CI engineer, I want an automated check that verifies the license file exists and contains canonical text, so that accidental removal or modification is detected in PRs.

Cucumber Scenarios

Scenario: Add MIT LICENSE file to repository root
Given / a repository without a LICENSE file at its root
When / a maintainer creates a file named "LICENSE" at the repository root containing the canonical MIT license text with year and copyright holder
Then / the repository root contains a file named "LICENSE" and the file contains the phrase "Permission is hereby granted" and "MIT License"

Scenario: README links to LICENSE
Given / the repository has a README.md file
When / a maintainer adds or updates a "License" section in README.md with a link that points to "./LICENSE" or "LICENSE"
Then / README.md contains a Markdown link to the repository LICENSE file and the link target resolves to the LICENSE file path

Scenario: Consumers can verify canonical MIT text
Given / the repository contains a LICENSE file
When / an automated check or reviewer examines the LICENSE file
Then / the LICENSE file must include canonical MIT phrases such as "Permission is hereby granted, free of charge" and the copyright line must include the chosen year and holder

Scenario: CI prevents PRs that remove/replace license incorrectly
Given / a pull request modifies or removes the LICENSE file or README license link
When / CI runs the license verification job on the PR
Then / the job fails when the LICENSE file is missing or when the canonical license text is not present, preventing merge until fixed

# DESIGN

Technical Overview
- Add a new file named LICENSE at the repository root containing the canonical MIT License text. Replace YEAR and COPYRIGHT HOLDER placeholders with accurate values (see Assumptions).
- Update README.md to include a short "License" section with a Markdown link to the LICENSE file at the repository root.
- Add a simple verification script (scripts/verify-license.sh) and a CI job (GitHub Actions or existing CI pipeline) that runs the script on PRs to ensure the LICENSE file exists and contains canonical text.
- Optional: add a minimal unit test that asserts the LICENSE file is present and contains key canonical phrases. No database or application code changes are required.

Component Breakdown
- LICENSE (new)
  - Responsibility: Provide canonical MIT license text. File name: LICENSE (no extension).
- README.md (modified)
  - Responsibility: Provide human-visible link and short summary of licensing terms.
- scripts/verify-license.sh (new, optional but recommended)
  - Responsibility: Exit 0 when LICENSE exists and matches basic canonical checks; exit non-zero otherwise.
- .github/workflows/verify-license.yml (new, optional)
  - Responsibility: Run the verify-license script on PRs and main branch merges.
- CI configuration (modified)
  - Responsibility: Ensure license verification is required for PR merges.
- Tests/verification (new)
  - Responsibility: Automated checks used by CI and local developers to validate presence and content.

UI/UX Wireframes (README snippet ASCII)
+------------------------------------------------------------+
| # Project Title                                            |
| ...                                                        |
|                                                          |
| ## License                                                 |
| This project is licensed under the MIT License - see the   |
| [License](./LICENSE) file for details.                     |
+------------------------------------------------------------+

ASCII component hierarchy (repo root)
.
├── LICENSE                       # new file, canonical MIT text
├── README.md                     # modified: add License section/link
├── scripts/
│   └── verify-license.sh         # new, optional verification script
├── .github/
│   └── workflows/
│       └── verify-license.yml    # new, optional CI job
└── (existing project files)

Data Flow Diagram (ascii)
Developer edits files
   |
   v
Create branch -> commit LICENSE + README change
   |
   v
Push -> PR
   |
   v
CI runs verify-license.sh (checks presence & phrases)
   |
   v
If pass -> reviewers approve -> merge -> LICENSE visible in main
   |
   v
GitHub license detection + downstream consumers can fetch LICENSE

Logic & Business Rules
- Filename rule: The license file must be named exactly "LICENSE" (case-sensitive recommended for portability).
- Content rule: The file must be the canonical MIT license template. A minimal automatic check is to assert the presence of canonical phrases:
  - "MIT License"
  - "Permission is hereby granted, free of charge"
  - "THE SOFTWARE IS PROVIDED 'AS IS', WITHOUT WARRANTY OF ANY KIND"
- Copyright rule: Year must be updated to the current year, and the copyright holder should be a person or organization as determined in Preflight.
- README link rule: README.md must contain a Markdown link to the license file (./LICENSE or LICENSE), case-insensitive for the anchor text but target must point to the path that resolves in the repository root.
- CI rule: A verification script or job must run on PRs to prevent removal or corruption of the LICENSE file.

Mapping to SPEC scenarios: these rules map to the cucumber scenarios described in SPEC.

API Endpoints
- No application API endpoints are required for this feature.
- Optional: repository maintainers may query GitHub's license API to assert repository license detection post-merge, but no new endpoints will be added to the project itself.

Database Schema Changes
- None.

Non-Functional Requirements
- Security: The license file is public; ensure no sensitive data is accidentally placed into the LICENSE file. CI should block merges that modify LICENSE unexpectedly.
- Performance: No measurable performance impact.
- Observability: CI job results (pass/fail) for the license check should be visible in CI logs. Add a one-line summary in CI output.
- Feature flag / kill-switch: CI job should be guarded by an environment variable (VERIFY_LICENSE=true by default). If CI needs to be bypassed for emergency merges, set VERIFY_LICENSE=false temporarily (use sparingly with documented approval).
- Rollout strategy: Create a small branch and PR, require at least one reviewer and green CI. Merge to main after approvals. Monitor the CI job for failures on follow-up PRs for 24-48 hours.
- Backwards compatibility: Adding LICENSE and README link is additive; safe to merge.

Assumptions & Open Questions
- Who should be listed as the copyright holder? (Suggested default: repository owner or organization.)
- Which year should be used? (Suggested default: current year at time of commit.)
- Should we name the file "LICENSE" or "LICENSE.md"? (We choose "LICENSE" as the canonical filename used by GitHub.)
- Is adding a CI job authorized in this repository, or should we only add the license and README link? (If the repository maintains strict CI rules, adding a workflow may require additional approvals.)

# TASKS

### Preflight & Decisions
- [ ] **Task 0.1: Determine copyright holder and year**
- **Status:** Pending
- **Context:**
 - Required by SPEC to populate the LICENSE header.
 - Affects LICENSE file content.
- **Dependencies:** None
- **Validation:**
 - [ ] File/path check: decision recorded in an issue or commit message (e.g., CHANGELOG or PR description).
 - [ ] Logic check: chosen holder/year aligns with repository owner or org.
 - [ ] Passing high-level unit tests: N/A (decision step).

- [ ] **Task 0.2: Confirm no DB migrations are required**
- **Status:** Pending
- **Context:**
 - Confirm with SPEC/DESIGN that no schema changes are needed.
- **Dependencies:** None
- **Validation:**
 - [ ] File/path check: N/A
 - [ ] Logic check: confirm no references to DB schema changes in plan.
 - [ ] Passing high-level unit tests: N/A

### Feature: Add LICENSE file
- [ ] **Task 1.1: Create canonical MIT license content (draft)**
- **Status:** Pending
- **Context:**
 - SPEC: canonical license text required.
 - DESIGN: populate YEAR and COPYRIGHT HOLDER.
- **Dependencies:** Task 0.1
- **Validation:**
 - [ ] File/path check: draft text prepared (in local editor or branch).
 - [ ] Logic check: draft contains canonical phrases: "Permission is hereby granted", "THE SOFTWARE IS PROVIDED 'AS IS'".
 - [ ] Passing high-level unit tests: N/A (preparation).

- [ ] **Task 1.2: Add file LICENSE to repository root containing the prepared MIT text**
- **Status:** Pending
- **Context:**
 - COMPONENT: LICENSE file (new) at repo root.
 - DESIGN rules: exact filename "LICENSE", canonical content.
- **Dependencies:** Task 1.1
- **Validation:**
 - [ ] Objective file/path checks: file exists at ./LICENSE in the branch.
 - [ ] Logic check: file contains canonical phrases ("Permission is hereby granted", "MIT License", "THE SOFTWARE IS PROVIDED 'AS IS'").
 - [ ] Passing high-level unit tests tied to SPEC scenarios: running verify-license script returns success (see Task 3.2).

- [ ] **Task 1.3: Add commit with clear message "Add MIT LICENSE"**
- **Status:** Pending
- **Context:**
 - Maintain commit hygiene; message should reference SPEC and PR.
- **Dependencies:** Task 1.2
- **Validation:**
 - [ ] Objective file/path checks: commit exists on branch with LICENSE present.
 - [ ] Logic check: commit message contains "Add MIT" or similar.
 - [ ] Passing high-level unit tests: N/A

### Feature: Link LICENSE from README
- [ ] **Task 2.1: Update README.md to add a "License" section linking to ./LICENSE**
- **Status:** Pending
- **Context:**
 - SPEC: README must link to license.
 - DESIGN: Use Markdown link [License](./LICENSE).
- **Dependencies:** Task 1.2
- **Validation:**
 - [ ] Objective file/path checks: README.md contains a line like "## License" and a link "[License](./LICENSE)".
 - [ ] Logic check: link target resolves to LICENSE file path and matches filename case.
 - [ ] Passing high-level unit tests: README-link verification script returns success (see Task 3.2).

- [ ] **Task 2.2: Add brief one-line license summary in README (human-friendly)**
- **Status:** Pending
- **Context:**
 - Provide context for casual visitors (e.g., "This project is licensed under the MIT License. See [License](./LICENSE).")
- **Dependencies:** Task 2.1
- **Validation:**
 - [ ] Objective file/path checks: README includes the summary.
 - [ ] Logic check: content does not include sensitive information.
 - [ ] Passing high-level unit tests: README-link verification returns success.

### CI & Verification
- [ ] **Task 3.1: Create scripts/verify-license.sh to validate presence and canonical text**
- **Status:** Pending
- **Context:**
 - DESIGN: script should check for existence and presence of canonical phrases.
 - Use for local dev and CI.
- **Dependencies:** Task 1.2, Task 2.1
- **Validation:**
 - [ ] Objective file/path checks: scripts/verify-license.sh exists and is executable.
 - [ ] Logic check: script returns exit 0 on valid LICENSE and non-zero on missing/invalid.
 - [ ] Passing high-level unit tests: run script locally: exit code 0 when LICENSE is present.

- [ ] **Task 3.2: Add CI job (e.g., .github/workflows/verify-license.yml) to run verification script on PRs**
- **Status:** Pending
- **Context:**
 - DESIGN: CI job ensures PRs do not remove or corrupt license.
 - Respect feature flag VERIFY_LICENSE env var.
- **Dependencies:** Task 3.1
- **Validation:**
 - [ ] Objective file/path checks: workflow file exists under .github/workflows.
 - [ ] Logic check: job runs verify-license.sh and respects VERIFY_LICENSE env var.
 - [ ] Passing high-level unit tests: CI job passes on branch with correct LICENSE; fails on branch without.

- [ ] **Task 3.3: Add a small unit test (optional language-appropriate) that asserts LICENSE presence**
- **Status:** Pending
- **Context:**
 - Adds automated verification in repo test suite: e.g., a tiny Go/Python/bash test that asserts LICENSE contains canonical phrases.
- **Dependencies:** Task 1.2
- **Validation:**
 - [ ] Objective file/path checks: test file exists under tests/ or pkg/.
 - [ ] Logic check: test asserts expected canonical phrases.
 - [ ] Passing high-level unit tests: running the repo test command (e.g., go test ./... or bash test script) returns success.

### PR, Review & Rollout
- [ ] **Task 4.1: Push branch and create PR asking for at least one reviewer**
- **Status:** Pending
- **Context:**
 - Standard workflow for small repository change.
- **Dependencies:** Task 1.3, Task 2.1, Task 3.2
- **Validation:**
 - [ ] Objective file/path checks: PR exists and shows diffs for LICENSE and README.md.
 - [ ] Logic check: PR description references license choice and preflight decision.
 - [ ] Passing high-level unit tests: CI passes on PR.

- [ ] **Task 4.2: Merge PR after CI success and reviewer approval**
- **Status:** Pending
- **Context:**
 - Finalize rollout to main.
- **Dependencies:** Task 4.1
- **Validation:**
 - [ ] Objective file/path checks: LICENSE and README changes appear on main branch.
 - [ ] Logic check: verify-license job passes on main.
 - [ ] Passing high-level unit tests: smoke checks (see VERIFICATION) pass.

- [ ] **Task 4.3: Post-merge validation and monitoring for 48 hours**
- **Status:** Pending
- **Context:**
 - Monitor CI runs and incoming PRs for license check failures.
- **Dependencies:** Task 4.2
- **Validation:**
 - [ ] Objective file/path checks: main branch contains LICENSE and README.
 - [ ] Logic check: no regressions introduced.
 - [ ] Passing high-level unit tests: CI license job passes for subsequent builds.

- [ ] **Task 4.4: Rollback plan (documented)**
- **Status:** Pending
- **Context:**
 - Prepare steps to revert license change if an issue is discovered.
- **Dependencies:** Task 4.2
- **Validation:**
 - [ ] Objective file/path checks: documented revert steps present in PR description or a small doc/issue.
 - [ ] Logic check: ability to revert by creating a revert PR or using git revert.
 - [ ] Passing high-level unit tests: revert PR passes CI.

# VERIFICATION

Artifact Check
- Confirm that `.frunl/01994d3c-e040-73e0-b58f-cfe9cc83739c/plan.md` exists (this artifact).
- After implementation, confirm the following files exist in the repository root:
  - ./LICENSE
  - ./README.md (updated)
  - scripts/verify-license.sh (optional but recommended)
  - .github/workflows/verify-license.yml (optional)

Traceability Table (Scenario → Design Element(s) → Task ID(s))

- "Add MIT LICENSE file to repository root"
  - Design Elements: LICENSE file (canonical text), Content rule.
  - Tasks: Task 1.1, Task 1.2, Task 1.3

- "README links to LICENSE"
  - Design Elements: README.md modification, README link rule.
  - Tasks: Task 2.1, Task 2.2

- "Consumers can verify canonical MIT text"
  - Design Elements: scripts/verify-license.sh, content rule, unit test.
  - Tasks: Task 3.1, Task 3.3

- "CI prevents PRs that remove/replace license incorrectly"
  - Design Elements: .github/workflows/verify-license.yml, CI rule, VERIFY_LICENSE env var.
  - Tasks: Task 3.2, Task 4.1

Test Strategy (high-level unit tests only)
- Purpose: Provide a small set of focused, high-level tests that validate the presence and basic correctness of the LICENSE file and the README link. These tests are not exhaustive license validators but serve as quick guards.

High-level tests (names + example commands)
- verify_license_presence
 - Description: Check that LICENSE exists and contains canonical MIT phrases.
 - Example command: bash scripts/verify-license.sh
 - Minimal internal checks:
   - File exists: ./LICENSE
   - Contains both: "Permission is hereby granted" and "THE SOFTWARE IS PROVIDED 'AS IS'"
- verify_readme_link
 - Description: Check README.md contains a link to the license file.
 - Example command: grep -i -E '\[license\]\(.*LICENSE' README.md || echo "README link missing"
- ci_smoke (CI job)
 - Description: Run verify_license_presence and verify_readme_link in CI; set VERIFY_LICENSE=true to enable.
 - Example CI step: run: bash scripts/verify-license.sh && grep -i -E '\[license\]\(.*LICENSE' README.md

Operational Readiness
- Metrics & Logs:
 - CI job pass/fail counts for the verify-license workflow.
 - One-line CI logs with outcome: "LICENSE OK" or "LICENSE FAIL: <reason>".
- Alerting:
 - Failures in the license CI job should block merges. For active monitoring, maintainers should watch CI for repeated failures in first 48 hours post-merge.
- Feature Flag / Kill-switch:
 - Environment variable VERIFY_LICENSE (default true). Setting VERIFY_LICENSE=false in CI temporarily skips the license check. Use only with documented approval.
- Rollback Plan:
 - If an incorrect license is merged:
   1. Immediately open a revert PR using git revert <commit-hash> that added or modified LICENSE and README.
   2. If urgent removal is required, remove LICENSE in a controlled PR that documents reason and next steps.
   3. Notify maintainers and legal/compliance (if applicable).
   4. Re-run CI and verify post-revert state.
- Responsible parties:
 - Repository maintainers: create PR, review, and merge.
 - CI owners: ensure verify-license workflow runs with required permissions.
 - Legal/Org contact: confirm copyright holder if required.

End of plan.

Location: .frunl/01994d3c-e040-73e0-b58f-cfe9cc83739c/plan.md
