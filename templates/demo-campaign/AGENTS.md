# Demo Campaign

Template project used as a test subject for the capsule campaign pipeline.

## Stack

- Go 1.22
- Standard library only

## Structure

```
src/
  go.mod      - Go module definition
  contact.go  - Entry point with Contact type and feature gaps
```

## Conventions

- All Go source lives under `src/`
- Tests use `_test.go` suffix in the same package
- Validation functions follow the pattern `ValidateX(input string) error`
- Formatting functions follow the pattern `FormatX(c Contact) string`
- The `Contact` struct in `contact.go` is the central data type

## Test Command

```bash
cd src && go test ./...
```

## Feature Gaps

The following functions are referenced but not yet implemented:

### Input Validation (Feature demo-1.1)
- `ValidateEmail(email string) error` - Email format validation
- `ValidatePhone(phone string) error` - Phone format validation

### Contact Formatting (Feature demo-1.2)
- `FormatDisplayName(c Contact) string` - Display name formatting
- `FormatMailingAddress(c Contact) string` - Mailing address formatting

These gaps are intended to be filled by bead tasks during campaign pipeline testing.
