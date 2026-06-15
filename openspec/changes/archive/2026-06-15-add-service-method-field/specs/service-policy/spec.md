# Spec Delta: Service Policy

## ADDED Requirements

### Requirement: Service Method Field
WHEN a service is created or updated, the system SHALL accept an optional `methods` field containing uppercase HTTP methods.

#### Scenario: Method Field Stored
GIVEN an admin submits a service with `methods: ["GET", "HEAD"]`
WHEN the service is saved
THEN the stored service includes `methods: ["GET", "HEAD"]`
AND service list output returns those methods.

#### Scenario: Lowercase Methods Normalized
GIVEN an admin submits a service with `methods: ["get", "post"]`
WHEN the service is validated
THEN the system stores `methods: ["GET", "POST"]`.

#### Scenario: Invalid Methods Rejected
GIVEN an admin submits a service with duplicate, empty, or invalid method tokens, or with `*` mixed alongside other methods
WHEN the service is validated
THEN the system rejects the service with a validation error
AND the previous service config remains unchanged.

#### Scenario: Omitted Methods Mean Any Method
GIVEN a service is submitted without a `methods` field
WHEN the service is saved
THEN the stored service has no method restriction
AND the field is treated as allowing any method.

#### Scenario: Wildcard Token Means Any Method
GIVEN an admin submits a service with `methods: ["*"]`
WHEN the service is validated
THEN the system accepts it as unrestricted
AND normalizes it to an empty (no-restriction) method set.

#### Scenario: Proposal Methods Survive Apply
GIVEN an agent raises a proposal for a service with `methods: ["GET"]`
WHEN a human applies the proposal
THEN the merged broker service includes `methods: ["GET"]`
AND a later proposal that omits `methods` clears the restriction back to any method.

