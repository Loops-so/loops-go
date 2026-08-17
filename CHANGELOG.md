## v0.4.0 - Jul 24, 2026

OpenAPI 1.21.2:

- Added workflow write APIs: `CreateWorkflow`, `UpdateWorkflow`, `ChangeWorkflowMailingList`, `CreateWorkflowNode`, `UpdateWorkflowNode`, `AddWorkflowBranch`, `DeleteWorkflowNode`, and `DeleteWorkflowNodeRecursive`.
- Added `ListEventPatterns`, `GetEventPattern`, and `GetEventPatternByName`.
- Added `CreateAudienceSegment`.
- Added `CreateTheme` / `UpdateTheme` and `CreateComponent` / `UpdateComponent`.
- Added `GetEmailMessageGuardian` and extra email message update fields.
- Fixed workflow node fields to match the OpenAPI spec.

## v0.3.1 - Jun 24, 2026

- Added workflow read APIs: `ListWorkflows`, `GetWorkflow`, and `GetWorkflowNode`.
- Added `Client.Do` as a low-level escape hatch for endpoints without dedicated methods.

## v0.3.0 - Jun 18, 2026

- Added campaign and transactional updates from OpenAPI 1.12.0.
- Added dedicated sending IP endpoints.
- SDK version is now resolved from build info when imported as a dependency.

## v0.2.0 - Jun 9, 2026

Added support for transactional email management (`/api/v1/transactional-emails`).

## v0.1.5 - May 26, 2026

Added Go package documentation.

## v0.1.4 - May 26, 2026

Added support for `/uploads`, including the `Upload` helper.

## v0.1.3 - May 5, 2026

Added themes API.

## v0.1.2 - May 5, 2026

Added components API.

## v0.1.1 - Apr 30, 2026

Installed govulncheck via mise to keep the SDK's dependencies clean.

## v0.1.0 - Apr 30, 2026

Initial release.
