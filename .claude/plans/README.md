# GLX Planning Documents

This directory contains the active planning and design documents for the GLX (Genealogical Ledger eXchange) project.

## Current Plan

### Active: v0.0.0-beta.2 Completion Plan
**File**: [glx-v0.0.0-beta.2-completion-plan.md](./glx-v0.0.0-beta.2-completion-plan.md)
**Status**: ACTIVE - Implementation in progress
**Scope**: Complete the import/split/join functionality for v0.0.0-beta.2 release

This plan addresses the final ~15% of work needed to complete the GEDCOM import and GLX serializer features:
- Fix multi-file vocabulary loading
- Fix validation test failures
- Add round-trip tests
- Update documentation
- Optional: Create property vocabularies

## Archived Plans

Historical planning documents have been moved to [`./old/`](./old/) for reference. These include:
- Original GEDCOM import plans (9,000+ lines)
- GLX serializer design documents
- Implementation task breakdowns
- Gap analysis documents

See [`./old/README.md`](./old/README.md) for details on archived plans.

## Implementation Status

As of 2025-11-18:

| Feature | Status | Notes |
|---------|--------|-------|
| GEDCOM Import | ✅ 100% Complete | Production ready, all tests passing |
| GLX Serializer | ⚠️ 95% Complete | Missing vocab loading in multi-file |
| CLI Commands | ✅ 100% Complete | import, split, join all working |
| E2E Tests | ✅ Passing | 10/10 test suites passing |
| Documentation | ⚠️ Needs Update | Structure changes not reflected |

## Quick Links

### Implementation Code
- Library: `/workspaces/spec/glx/lib/`
- CLI: `/workspaces/spec/glx/`
- Tests: `/workspaces/spec/glx/lib/*_test.go`

### Documentation
- User Guide: `/workspaces/spec/docs/`
- Project Guide: `/workspaces/spec/CLAUDE.md`
- Changelog: `/workspaces/spec/CHANGELOG.md`

### Test Data
- GEDCOM Files: `/workspaces/spec/glx/testdata/gedcom/`
- Example Archives: `/workspaces/spec/docs/examples/`

## How to Use These Plans

1. **For current work**: Refer to the active completion plan
2. **For context**: Check archived plans if you need historical decisions
3. **For updates**: Update the completion plan as tasks are completed
4. **For new features**: Create new plan documents as needed

---

*Last Updated: 2025-11-18*