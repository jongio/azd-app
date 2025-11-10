## [0.5.2] - 2025-11-10

- This PR refactors package manager detection for Node.js and Python to eliminate code duplication and provide better traceability of how package managers are detected. (#50) (396d4d5)
- Align documentation with implementation (#45) (1dee2d9)
- Improve process lifecycle management with context-based coordination and graceful shutdown (#40) (bc1e67f)
- Fix release notes showing full changelog instead of version-specific commits (#33) (127c069)
- Redesign console output for better UX and readability (#42) (74c6e7b)
- Achieve 98.52% test coverage for dashboard with comprehensive unit and e2e tests (#36) (71d4da2)
- chore: update registry for v0.5.1 (fb0fdd5)

## [0.5.1] - 2025-11-08

- Add packageManager field support for Node.js package manager detection (#34) (bdbb25e)
- chore: update registry for v0.5.0 (1c76011)
- chore: bump version to 0.5.0 (c19e7cd)
- Enhance port management and environment variable support in azure.yaml (#32) (24d0aac)
- chore: update registry for v0.4.0 (77aafc6)
- chore: bump version to 0.4.0 (013f4cc)
- chore: update registry for v0.3.7 (a776022)
- chore: bump version to 0.3.7 (cb0fe96)
- Fix: update registry path in publish step to use GITHUB_WORKSPACE (8212e95)
- chore: bump version to 0.3.6 (5128532)

## [0.5.0] - 2025-11-07

- Enhance port management and environment variable support in azure.yaml (#32) (24d0aac)
- chore: update registry for v0.4.0 (77aafc6)
- chore: bump version to 0.4.0 (013f4cc)
- chore: update registry for v0.3.7 (a776022)
- chore: bump version to 0.3.7 (cb0fe96)
- Fix: update registry path in publish step to use GITHUB_WORKSPACE (8212e95)
- chore: bump version to 0.3.6 (5128532)
- ci: refactor CI workflows to consolidate checks and improve structure (dbadfee)
- chore: add git pull --rebase before version bump in release workflow (2f7b676)
- Fix: streamline build and packaging steps in release workflow (f8038f7)

## [0.4.0] - 2025-11-07

- chore: update registry for v0.3.7 (a776022)
- chore: bump version to 0.3.7 (cb0fe96)
- Fix: update registry path in publish step to use GITHUB_WORKSPACE (8212e95)
- chore: bump version to 0.3.6 (5128532)
- ci: refactor CI workflows to consolidate checks and improve structure (dbadfee)
- chore: add git pull --rebase before version bump in release workflow (2f7b676)
- Fix: streamline build and packaging steps in release workflow (f8038f7)
- chore: bump version to 0.3.5 (9aabd2a)
- ci: refactor CI workflows to use shared checks and streamline build process (a6e4e4a)
- chore: bump version to 0.3.4 (55d8898)

## [0.3.7] - 2025-11-07

- Fix: update registry path in publish step to use GITHUB_WORKSPACE (8212e95)
- chore: bump version to 0.3.6 (5128532)
- ci: refactor CI workflows to consolidate checks and improve structure (dbadfee)
- chore: add git pull --rebase before version bump in release workflow (2f7b676)
- Fix: streamline build and packaging steps in release workflow (f8038f7)
- chore: bump version to 0.3.5 (9aabd2a)
- ci: refactor CI workflows to use shared checks and streamline build process (a6e4e4a)
- chore: bump version to 0.3.4 (55d8898)
- Fix: Python services use venv instead of system Python (#28) (66268c9)
- ci: reorganize release workflow — run tool installs & preflight earlier, extract release notes, and remove dashboard build step (ecf77d5)

## [0.3.6] - 2025-11-07

- ci: refactor CI workflows to consolidate checks and improve structure (dbadfee)
- chore: add git pull --rebase before version bump in release workflow (2f7b676)
- Fix: streamline build and packaging steps in release workflow (f8038f7)
- chore: bump version to 0.3.5 (9aabd2a)
- ci: refactor CI workflows to use shared checks and streamline build process (a6e4e4a)
- chore: bump version to 0.3.4 (55d8898)
- Fix: Python services use venv instead of system Python (#28) (66268c9)
- ci: reorganize release workflow — run tool installs & preflight earlier, extract release notes, and remove dashboard build step (ecf77d5)
- chore: update registry for v0.3.3 (55ca088)
- chore: bump version to 0.3.3 (53fdcc5)

## [0.3.5] - 2025-11-07

- ci: refactor CI workflows to use shared checks and streamline build process (a6e4e4a)
- chore: bump version to 0.3.4 (55d8898)
- Fix: Python services use venv instead of system Python (#28) (66268c9)
- ci: reorganize release workflow — run tool installs & preflight earlier, extract release notes, and remove dashboard build step (ecf77d5)
- chore: update registry for v0.3.3 (55ca088)
- chore: bump version to 0.3.3 (53fdcc5)
- feat: improve dependency installer robustness, add tests, and document npm TAR_ENTRY_ERROR (#26) (968fb3c)
- refactor: improve type safety, fix tests, and optimize build system (#25) (820359f)
- chore: add Release workflow badge to README (af6fba5)
- chore: update registry for v0.3.2 (c789016)

## [0.3.4] - 2025-11-07

- Fix: Python services use venv instead of system Python (#28) (66268c9)
- ci: reorganize release workflow — run tool installs & preflight earlier, extract release notes, and remove dashboard build step (ecf77d5)
- chore: update registry for v0.3.3 (55ca088)
- chore: bump version to 0.3.3 (53fdcc5)
- feat: improve dependency installer robustness, add tests, and document npm TAR_ENTRY_ERROR (#26) (968fb3c)
- refactor: improve type safety, fix tests, and optimize build system (#25) (820359f)
- chore: add Release workflow badge to README (af6fba5)
- chore: update registry for v0.3.2 (c789016)
- chore: bump version to 0.3.2 (9e67028)
- fix: add --repo flag to azd x publish for remote mode (ed2522c)

## [0.3.3] - 2025-11-07

- feat: improve dependency installer robustness, add tests, and document npm TAR_ENTRY_ERROR (#26) (968fb3c)
- refactor: improve type safety, fix tests, and optimize build system (#25) (820359f)
- chore: add Release workflow badge to README (af6fba5)
- chore: update registry for v0.3.2 (c789016)
- chore: bump version to 0.3.2 (9e67028)
- fix: add --repo flag to azd x publish for remote mode (ed2522c)
- chore: update registry for v0.3.1 (f44bb9a)
- chore: bump version to 0.3.1 (0a1d584)
- chore: revert to simple direct push now that branch protection is removed (cc4754f)
- chore: update release workflow to create PR instead of direct push (ebf909c)

## [0.3.2] - 2025-11-06

- fix: add --repo flag to azd x publish for remote mode (ed2522c)
- chore: update registry for v0.3.1 (f44bb9a)
- chore: bump version to 0.3.1 (0a1d584)
- chore: revert to simple direct push now that branch protection is removed (cc4754f)
- chore: update release workflow to create PR instead of direct push (ebf909c)
- chore: add azd-app.sln to .gitignore (f5143c2)
- chore: reset version to 0.3.0 (36b9ebf)
- Test ci 2 (#23) (6d99c4f)
- Prepare release v0.4.3 (#22) (01670c2)
- docs: append additional line to testreleaseprocess.md (#21) (06d6981)

## [0.3.1] - 2025-11-06

- chore: revert to simple direct push now that branch protection is removed (cc4754f)
- chore: update release workflow to create PR instead of direct push (ebf909c)
- chore: add azd-app.sln to .gitignore (f5143c2)
- chore: reset version to 0.3.0 (36b9ebf)
- Test ci 2 (#23) (6d99c4f)
- Prepare release v0.4.3 (#22) (01670c2)
- docs: append additional line to testreleaseprocess.md (#21) (06d6981)
- Test ci 2 (#19) (bca6166)
- Prepare release 0.4.2 (#18) (77c954f)
- Test ci 2 (#17) (62947ac)

## [0.4.3] - 2025-11-06

- docs: append additional line to testreleaseprocess.md (#21) (06d6981)

## [0.4.2] - 2025-11-06

- Test ci 2 (#17) (62947ac)

## [0.4.1] - 2025-11-06

- Update testreleaseprocess.md with additional information (#15) (e333643)

## [0.4.0] - 2025-11-06

- Testing process implementation (#13) (c3e86b9)

## [0.3.0] - 2025-11-06

- docs: document complete release workflow (7c3e375)

# Changelog

All notable changes to the App Extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.1](https://github.com/jongio/azd-app/compare/v0.2.0...v0.2.1) (2025-11-05)


### Bug Fixes

* force new release-please PR with CI checks ([87e3a84](https://github.com/jongio/azd-app/commit/87e3a84f530c88e28af4a14dcb9c2f0e71e19e5e))
* make build work without azd installed for CI ([117aa68](https://github.com/jongio/azd-app/commit/117aa6869e73031c036431d2c8088d68dbb5c9bb))
* skip integration tests in coverage run ([7123ef2](https://github.com/jongio/azd-app/commit/7123ef29dd762cc25533bd882421f324b0f4f312))
* wait for service options before selecting in test ([9d8b027](https://github.com/jongio/azd-app/commit/9d8b02751ed1ae992000974998f13c5864e8a139))
* wait for service options to populate in test ([a917be0](https://github.com/jongio/azd-app/commit/a917be02bcec43397323e6aaac370d248678bf87))

## [0.2.0](https://github.com/jongio/azd-app/compare/v0.1.2...v0.2.0) (2025-11-05)


### ⚠ BREAKING CHANGES

* **reqs/docs:** azure.yaml reqs items now use "name" instead of "id". Run `azd app reqs --generate` to update existing projects.

### Features

* Enhance `azd app reqs --generate` command to auto-create missing `reqs` section in `azure.yaml` ([56cb705](https://github.com/jongio/azd-app/commit/56cb705bd23d6ab3ebc6f8a962064df4ea0c5d87))
* enhance requirement merging in azure.yaml and update version to 0.1.5 ([20fbfda](https://github.com/jongio/azd-app/commit/20fbfda37667fd455de42b9bde840ed4007c483a))
* Implement installation and uninstallation scripts for azd app extension with file watching capabilities ([f35d77a](https://github.com/jongio/azd-app/commit/f35d77a1f86f50c6660dae970a7532d6195cec71))
* Initial release of azd-app CLI ([eef9202](https://github.com/jongio/azd-app/commit/eef92021c1b1eb75877dcf54ad6c4c12e43f8cfe))
* **reqs/docs:** rename reqs item key 'id' to 'name' across CLI, tests, and docs ([dbaeb56](https://github.com/jongio/azd-app/commit/dbaeb56dbf78f54ab56de9c84ef0f74c5dbdfd92))
* **testing/docs:** add extensive tests, mocks, vitest setup, docs, and Go unit tests ([be238cf](https://github.com/jongio/azd-app/commit/be238cf9c6bb11caf7f7f2617df23047f28be2a5))


### Bug Fixes

* correct last array line detection in YAML processing and bump version to 0.1.32 ([0172d10](https://github.com/jongio/azd-app/commit/0172d10f0529f678cc77562b50a0fe833c11aa31))
* enhance changelog update logic to prevent duplicate entries and improve release note extraction ([09e4c53](https://github.com/jongio/azd-app/commit/09e4c53526269ed3ea0c95240bdd6260e0b169c4))
* update artifact URLs and extension commands in workflows and README ([6bcd1fc](https://github.com/jongio/azd-app/commit/6bcd1fc1a718f1f87c176943a5020f029d63b251))
* update changelog format to remove redundant Unreleased section ([e25de6b](https://github.com/jongio/azd-app/commit/e25de6b951a7eea1614cfc6eb3f18a18ee24cfeb))
* update changelog section to reflect version 0.1.1 and add function to extract unreleased changelog content ([5036332](https://github.com/jongio/azd-app/commit/5036332d0fbbe105d6a4cefeaeb7be3ae8d31d35))
* update CHANGELOG.md to reflect versioning changes and improve release process ([0e4269d](https://github.com/jongio/azd-app/commit/0e4269d9a79c53931fa46382d213d976ac97323f))

## [0.1.2] - 2025-11-04

### Added
- Dashboard test coverage with React component tests
- Command documentation for all CLI commands

### Fixed
- Windows process health checks now work correctly
- Test cleanup properly releases log file handles on Windows
- Dashboard tests pass without React warnings

### Changed
- Improved overall code coverage with comprehensive test suites
- **BREAKING CHANGE**: Renamed `reqs.req.id` to `reqs.req.name` in azure.yaml. Run `azd app reqs --generate` to update your project to work with `name` instead of `id`

## [0.1.1] - 2025-11-04

### Fixed
- `azd app reqs --generate` now automatically creates the `reqs` section in `azure.yaml` if it doesn't exist, instead of failing with "section 'reqs' not found in YAML" error

## [0.1.0] - 2025-11-04

Initial release of azd-app CLI extension.

### Added

#### Core Commands
- `azd app reqs` - Check prerequisites and validate installed tools
- `azd app deps` - Install dependencies across all projects automatically  
- `azd app run` - Start development environment with live dashboard
- `azd app info` - Display information about running services
- `azd app logs` - View and stream logs from running services

#### Language & Framework Support
- Node.js (npm, pnpm, yarn)
- Python (pip, poetry, uv)
- .NET (.csproj, .sln)
- .NET Aspire (AppHost orchestration)

#### Live Dashboard
- Real-time service monitoring with React/TypeScript/Vite
- Log streaming with filtering
- Local and Azure endpoint management
- Service health indicators
- WebSocket-based updates

#### Features
- Automatic package manager detection
- Azure environment integration
- Secure command execution with validation
- Cross-platform support (Windows, Linux, macOS on AMD64 and ARM64)
- 80%+ code coverage with comprehensive testing

### Security
- Input validation and path traversal protection
- Secure random number generation (crypto/rand)
- HTTP timeout configurations
- Regular security scanning with gosec

[Unreleased]: https://github.com/jongio/azd-app/compare/azd-app-cli-v0.1.0...HEAD
[0.1.0]: https://github.com/jongio/azd-app/releases/tag/azd-app-cli-v0.1.0
