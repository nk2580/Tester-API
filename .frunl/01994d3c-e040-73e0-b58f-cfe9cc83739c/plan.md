# SPEC

Detailed Feature Description
This feature adds a canonical MIT license file to the repository root and ensures the repository README links to it. A repository license is a small but high-value change: it gives legal clarity to downstream consumers, enables automated tooling (license scanners, package managers and compliance tools) to detect reuse permissions, and removes friction for contributors and integrators.

Primary goals:
- Add a plain text file named LICENSE in the repository root containing the standard MIT license text with a minimal, editable copyright line.
- Make the README.md explicitly reference and link to the LICENSE file so humans visiting the repo immediately see the license and automated crawlers find the reference inside the common README file.

Secondary goals:
- Provide a small, high-level unit test to make it easy for maintainers to confirm the presence of the license and README link using the local Go toolchain.
- Keep the change minimal and safe: no workflows or CI configuration changes, no binary or dependency installation, and only repository file edits.

Why this exists
- Without a license file, code reuse is legally ambiguous and many organizations will avoid using the project.
- A root-level LICENSE file in a known format (MIT) is the most widely recognized pattern; linking it from README.md improves discoverability for human readers and some automation.

User Stories
- As a repository consumer, I want a clear license file at the project root, so that I know how I may reuse and redistribute the code.
- As a maintainer, I want the README to link to the LICENSE file, so that human visitors find license terms immediately when viewing the project.
- As an automated scanner, I want the repository to expose a root LICENSE file with recognizable MIT wording, so the scanner can pick up the licensing information reliably.

Cucumber Scenarios

Scenario: License file present at repository root (consumer)
Given I am viewing the repository root
When I look for a license file named "LICENSE"
Then a file named "LICENSE" exists at the repository root and its contents include the phrase "MIT License" and the standard MIT permission clause "Permission is hereby granted, free of charge, to any person obtaining a copy of this software"

Scenario: README links to the LICENSE file (maintainer/user)
Given I have opened README.md at the repository root
When I read the README
Then README.md contains a relative link to the license file (for example: "[MIT License](LICENSE)" or "(LICENSE)" ) and a short sentence stating the project uses the MIT License

Scenario: Automated scanner detects MIT text (automation)
Given a license scanning tool examines the repository root
When it reads LICENSE
Then it recognizes the standard MIT permission and warranty disclaimer clauses and classifies the license as MIT

# DESIGN

Technical Overview
- Create a new file LICENSE in the repo root with the canonical MIT license text and a single copyright line ready to be customized.
- Modify README.md to include a "License" section (or add the link near the end of README) containing a relative link to the LICENSE file, using an explicit phrase such as "This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details."
- Add a small high-level test file (Go) that validates the license file exists and README links to it. This test is optional in environments without Go, but useful for maintainers running go test locally.

What will change:
- New file: LICENSE
- Modified file: README.md (append or add License section)
- New file: license_test.go (unit tests in Go to assert presence and basic content)

What can be reused:
- Existing README.md contents will be preserved; we will add a short License section at the end to avoid disrupting README structure.
- The project's Go toolchain (go.mod) indicates where tests can be placed; tests will be simple and self-contained.

Component Breakdown
- LICENSE (new)
  - Responsibility: Provide canonical MIT license text.
  - Path: /LICENSE
  - Format: Plain text (no extension), standard MIT content with "Copyright (c) 2025 Tester-API contributors" placeholder.

- README.md (modified)
  - Responsibility: Reference the license so humans and many crawlers see it immediately.
  - Change: Add "## License" section with a relative link to LICENSE and a one-line summary.

- license_test.go (new)
  - Responsibility: Unit tests that verify LICENSE exists and is clearly an MIT license and that README.md links to LICENSE.
  - Location: repo root, package main (to match main.go), or package "licensecheck" if preferred; tests are small, high-level, and runnable with `go test ./...`.

UI/UX Wireframes (README snippet)
A small ASCII preview of README.md addition (append this snippet near the end):

## License

This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.

ASCII component hierarchy
- repo root/
  - README.md
    - title, description, usage...
    - ## License -> link to LICENSE
  - LICENSE (new)
  - main.go, go.mod, db/, ...

Data Flow Diagram (simple)
User or automation -> open repository root -> README.md (follow link) -> LICENSE
ASCII:
[User/Scanner] --> [README.md] --> [LICENSE]

Logic & Business Rules (mapped to SPEC scenarios)
- Rule: LICENSE must be named "LICENSE" at repository root.
  - Maps to: "License file present" scenario.
- Rule: LICENSE must contain canonical MIT permission and warranty disclaimer clauses.
  - Maps to: "Automated scanner detects MIT text" scenario.
- Rule: README.md must contain a relative link to LICENSE (either "(LICENSE)" or "[MIT License](LICENSE)").
  - Maps to: "README links to LICENSE" scenario.

API Endpoints
- None required.

Database Schema Changes
- None required.

Non-Functional Requirements
- Security: The LICENSE file is plain text only; no executable content. Confirm license text does not contain binary content or injected scripts.
- Performance: No runtime impact.
- Observability: Add a simple unit test that can be used by maintainers/CI to detect regressions.
- Feature flags & rollout: Not applicable; this is a documentation/legal change. Rollout is safe to merge to main. If the repository uses protected branches, create a PR. No GitHub Actions/workflow edits are required.
- Compliance: Recommend maintainers replace the copyright holder placeholder with an organization or person name before publishing.

Assumptions & Open Questions
- Which copyright holder string should be used? The plan uses a clear placeholder "2025 Tester-API contributors". Maintainers should replace with the preferred name (individual or organization).
- Should the file be named LICENSE or LICENSE.md? The plan uses LICENSE (no extension) as the most common, machine-recognized pattern.
- No CI workflow edits will be made in this plan because modifying workflows is not permitted by the execution environment.

# TASKS

- [ ] **Task 1.1: Add canonical MIT license file at repository root**
- **Status:** Pending
- **Context:**
 - Implements SPEC feature "Add MIT license file".
 - Create file at path: LICENSE (repository root).
 - File contents must include the standard MIT text, including the permission clause and warranty disclaimer, and an editable copyright line.
 - Reference: DESIGN component "LICENSE".
- **Dependencies:** None
- **Validation:**
 - [ ] LICENSE exists at repository root (./LICENSE)
 - [ ] LICENSE contains the header "MIT License"
 - [ ] LICENSE includes the permission clause starting "Permission is hereby granted, free of charge, to any person obtaining a copy..."
 - [ ] LICENSE contains a copyright line (e.g., "Copyright (c) 2025 Tester-API contributors")

- [ ] **Task 2.1: Update README.md to link to LICENSE**
- **Status:** Pending
- **Context:**
 - Implements SPEC story "README links to the LICENSE file".
 - Modify README.md to add a "## License" section (append or insert near end) with a relative link: "This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details."
 - Reference: DESIGN component "README.md".
- **Dependencies:** Task 1.1
- **Validation:**
 - [ ] README.md contains a "License" section or equivalent sentence linking to LICENSE
 - [ ] README.md contains a relative link pattern "(LICENSE)" or "](LICENSE)"
 - [ ] Changes preserve existing README content (no deletion of unrelated sections)

- [ ] **Task 3.1: Add high-level unit tests to assert LICENSE presence and README link**
- **Status:** Pending
- **Context:**
 - Implements the test artifact to validate SPEC scenarios programmatically.
 - Add file license_test.go (at repository root) with two tests:
   - TestLicenseFileExists: verifies README and LICENSE file presence and checks for "MIT License" and the permission clause.
   - TestReadmeLinksLicense: verifies README.md contains a relative link to LICENSE.
 - The tests are simple and runnable with: go test ./... (if Go toolchain is available).
 - Reference: DESIGN component "license_test.go".
- **Dependencies:** Task 1.1, Task 2.1
- **Validation:**
 - [ ] license_test.go exists at repository root
 - [ ] Tests compile and pass when run in an environment with Go toolchain: go test ./...
 - [ ] Tests check for both LICENSE content and README link as described in SPEC
 - [ ] Documentation (README or PR description) explains how to run the tests locally: `go test ./...`

# VERIFICATION

Artifact Check
- The deliverable plan must be committed to: .frunl/01994d3c-e040-73e0-b58f-cfe9cc83739c/plan.md (this file).
- After implementation, verify these files/edits exist in the repository root: LICENSE, README.md (modified), license_test.go.

Traceability Table (Scenario → Design Element(s) → Task ID(s))
- "License file present at repository root" → LICENSE (Design) → Task 1.1
- "README links to the LICENSE file" → README.md (Design) → Task 2.1
- "Automated scanner detects MIT text" → LICENSE text and license_test.go → Task 1.1, Task 3.1
- "Unit test asserts license and link" → license_test.go → Task 3.1

Test Strategy (high-level unit tests only)
- Primary tests to add (high-level):
 - TestLicenseFileExists (license_test.go)
   - Asserts ./LICENSE exists
   - Asserts file contains "MIT License"
   - Asserts contains the permission clause "Permission is hereby granted, free of charge, to any person obtaining a copy"
 - TestReadmeLinksLicense (license_test.go)
   - Asserts README.md exists
   - Asserts README contains "(LICENSE)" or "](LICENSE)"
- How to run (maintainer/dev machine with Go installed):
 - go test ./...
 - Example local smoke checks (POSIX):
   - grep -n "MIT License" LICENSE
   - grep -n "\[.*License.*\](LICENSE)" README.md || grep -n "(LICENSE)" README.md
- Note: The execution environment used to build this plan is restricted (no network and no external package managers); the tests are provided for maintainers to run in their development environment that has the Go toolchain.

Operational Readiness
- Metrics / logs / traces:
 - No runtime service; operational readiness is limited to repository compliance checks.
 - Monitor PR feedback and automated scanning results (if repository CI includes license scanning outside the scope of this plan).
- Feature flag / kill-switch:
 - Not applicable in runtime sense; rollback procedure is:
   1. Revert the commit that introduced LICENSE and README change (or create a new commit that deletes LICENSE and removes README license paragraph).
   2. If changes were merged to main, open a new PR that reverts the files and reference the merge commit.
- Rollback plan:
 - To undo quickly, remove the LICENSE file and restore README.md from the pre-change state (use git revert or restore).
 - If CI gating is used, revert via PR and wait for pipeline to succeed.
- Post-deployment checklist:
 - Confirm license appears rendered in Git hosting UI (GitHub/GitLab display “MIT License”).
 - Run `go test ./...` locally and confirm tests return success.
 - Optionally run a local license scanner (eg. `licensee` or similar) in developer environment; not required by this plan.

Notes for implementer
- Replace the placeholder copyright holder text in LICENSE with the preferred holder before publishing (example: a company or individual).
- Do not modify workflow files; this plan avoids any .github/workflows edits per repository permission constraints.

Location: .frunl/01994d3c-e040-73e0-b58f-cfe9cc83739c/plan.md
