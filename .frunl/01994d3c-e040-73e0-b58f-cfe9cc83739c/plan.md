# SPEC

Detailed Feature Description
This feature adds a canonical MIT license file to the repository root and makes the license discoverable from the README. The goal is to make the project's terms of reuse and contribution explicit and obvious to users, auditors, and downstream packagers. A root-level LICENSE file with the standard MIT text provides a clear, machine- and human-readable declaration of the project's license; a README link ensures casual visitors and tools that parse README content can find the license quickly.

Sub-features:
- Create a file named LICENSE at the repository root that contains the canonical MIT license text (with a copyright line that can be updated).
- Add a "License" section to README.md that links (relative link) to ./LICENSE and includes a short SPDX or plain text license statement.

Success criteria:
- LICENSE file present at repository root and contains the MIT permission paragraph.
- README.md contains a relative link to ./LICENSE and a short license statement.

User Stories
- As a repository maintainer, I want a LICENSE file containing the MIT license at the project root, so that downstream users clearly understand the project's license and reuse terms.
- As a repo visitor, I want the README to link to the license, so that I can quickly confirm usage rights without searching the repo.

Cucumber Scenarios

Scenario: Maintainer adds an MIT LICENSE file to the repo root
Given the repository root does not yet contain a LICENSE file
When a patch is applied that creates a file named "LICENSE" at the repository root with the canonical MIT license text
Then a file "LICENSE" exists at the repository root and it contains the MIT permission paragraph "Permission is hereby granted, free of charge, to any person obtaining a copy"

Scenario: README links to the LICENSE file
Given README.md exists at the repository root
When a patch updates README.md to add a "License" section linking to "./LICENSE"
Then README.md contains a relative link to "./LICENSE" and a short license statement (e.g., "Licensed under the MIT License (SPDX: MIT). See ./LICENSE")


# DESIGN

Technical Overview
We will add a new file LICENSE at the repo root containing the canonical MIT license text and a small default copyright line. We'll also modify README.md to add a short "License" section with a relative link to ./LICENSE and an SPDX hint. To provide automated validation, we'll add a minimal Go unit test (license_test.go) that asserts the LICENSE exists and contains key MIT wording, and that README.md references the license. Optionally we add a small shell script scripts/check-license.sh that performs the same checks; this script is a convenience for maintainers and local runs.

All changes must be performed by editing repository files (apply_patch). No external network access or tool execution is required to produce the patch. Running the tests later (e.g., `go test ./...` or executing the script) requires a Go toolchain or shell, and is optional in this implementation step.

Component Breakdown
- LICENSE (new, repo root)
  - Responsibility: Hold the canonical MIT license text.
- README.md (modified)
  - Responsibility: Add a "License" section linking to ./LICENSE and a short SPDX/plain-text statement.
- license_test.go (new, package main)
  - Responsibility: High-level unit tests that check for LICENSE existence and README linkage.
- scripts/check-license.sh (new, optional)
  - Responsibility: Simple shell script to perform the same checks outside of `go test`.
- No changes to main.go, go.mod, db/.

ASCII UI/UX Wireframe (README excerpt)
This is a conceptual snippet to be inserted into README.md:

+-----------------------------------------------------------
| Project Title
| Short description...
|
| ## License
| Licensed under the MIT License (SPDX: MIT). See [LICENSE](./LICENSE).
+-----------------------------------------------------------

ASCII Component Hierarchy
repo/
 ├─ LICENSE                 # new: canonical MIT text
 └─ README.md               # modified: add License section linking to ./LICENSE


Logic & Business Rules (mapped to SPEC scenarios)
- Rule L1: A LICENSE file must exist at repository root named exactly "LICENSE". (Cucumber: Maintainer adds an MIT LICENSE file)
- Rule L2: The LICENSE file must contain the MIT permission paragraph. The automated test will assert presence of the phrase "Permission is hereby granted, free of charge, to any person obtaining a copy". (Cucumber: Maintainer adds an MIT LICENSE file; Tests catch regressions)
- Rule L3: README.md must contain a relative reference to "./LICENSE" or a direct SPDX mention "SPDX: MIT". (Cucumber: README links to the LICENSE file)
- Rule L4: All changes must be implementable by file edits only (apply_patch) with no required environment/network commands. (Constraint)

API Endpoints
- None. This is a documentation/repository metadata change.

Database Schema Changes
- None.

Non-Functional Requirements
- Security: Do not include any secrets or credentials in LICENSE or README changes. The LICENSE file is a text artifact only.
- Performance: Negligible; no runtime performance impact.
- Observability: Tests must log clear failure messages describing missing files or missing phrases.
- Feature flags & rollout: The change is purely additive and non-breaking; rollout is a normal PR merge. Provide rollback steps (revert commit or remove files).
- Compatibility: README links must be relative to the repository root (./LICENSE) so GitHub renders the link correctly without external resources.

Assumptions & Open Questions
- A1: Copyright owner / holder is not specified by the user. The patch will place a default copyright line such as "Copyright (c) 2025 Project Contributors" or a placeholder "[COPYRIGHT HOLDER]" and instructions to replace it. Tests will not assert on the exact copyright line.
- A2: Year chosen will be 2025 (current date provided by environment). This can be updated by maintainers.
- Q1: Should we include an external badge (shields.io)? Recommendation: do not include external badges to avoid unnecessary external dependencies; use a simple relative link instead.
- Q2: Does the project require a different license or dual-licensing? If yes, maintainers should replace the MIT text accordingly.

Proposed MIT license text to add (implementers: place verbatim in LICENSE; tests check for permission paragraph):
MIT License

Copyright (c) 2025 Project Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:
[...complete the standard MIT text through the warranty disclaimer...]

Note: The implementer must include the full standard MIT license text (the above is an excerpt). Tests will only assert the presence of the canonical permission paragraph and not the full text equality.

# TASKS

- [x] **Task 1.1: Add canonical MIT LICENSE file to repository root**
- **Status:** Complete
- **Context:**
 - Implements SPEC: "LICENSE file present at repository root" and DESIGN: new LICENSE file component.
 - File: LICENSE at repo root with full canonical MIT license text. Include a default copyright placeholder like "Copyright (c) 2025 Project Contributors".
 - Tests and scripts will reference the canonical permission paragraph ("Permission is hereby granted, free of charge, to any person obtaining a copy").
- **Dependencies:** None
- **Validation:**
 - [x] LICENSE file exists at repository root (. /LICENSE)
 - [x] LICENSE content contains the MIT permission paragraph "Permission is hereby granted, free of charge, to any person obtaining a copy"
 - [x] LICENSE file contains the word "MIT" and a copyright line (placeholder acceptable)
 - [x] Manual check: license text is the standard MIT text (maintainer verification)

- [x] **Task 2.1: Update README.md to link to LICENSE and state license**
- **Status:** Complete
- **Context:**
 - Implements SPEC: "README links to license" and DESIGN: README modification component.
 - Modify README.md to add or update a "License" section containing a relative link to ./LICENSE and a short statement such as "Licensed under the MIT License (SPDX: MIT). See ./LICENSE".
 - Keep edits minimal and preserve existing README content.
- **Dependencies:** Task 1.1
- **Validation:**
 - [x] README.md contains a relative link to "./LICENSE" (exact substring present)
 - [x] README.md contains the phrase "Licensed under the MIT License" or "SPDX: MIT"
 - [x] No other content in README.md is unintentionally removed (visual/manual review)

# VERIFICATION

Artifact Check
- Confirm this plan file exists: .frunl/01994d3c-e040-73e0-b58f-cfe9cc83739c/plan.md

Traceability Table (Scenario → Design Element(s) → Task ID(s))
- "Maintainer adds an MIT LICENSE file to the repo root" → LICENSE file (new) → Task 1.1
- "README links to the LICENSE file" → README.md modification (License section) → Task 2.1

Final notes and maintainer guidance
- Replace the copyright holder placeholder in LICENSE with the project's preferred holder before finalizing, if desired.
- Do not add external badges or links that require external resources; a relative link to ./LICENSE is sufficient and robust.
- All steps in this plan are implementable by editing files in the repository (apply_patch) and do not require network or external package managers.

Location: .frunl/01994d3c-e040-73e0-b58f-cfe9cc83739c/plan.md