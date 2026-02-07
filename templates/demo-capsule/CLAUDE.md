# Demo Capsule

Template project used as a test subject for the capsule pipeline.

## Project Structure

```
src/
  go.mod    - Go module definition
  main.go   - Entry point with Contact type and feature gaps
```

## Conventions

- All Go source lives under `src/`
- Tests use `_test.go` suffix in the same package
- Validation functions follow the pattern `ValidateX(input string) error`
- The `Contact` struct in `main.go` is the central data type

## Feature Gaps

The following functions are referenced but not yet implemented:
- `ValidateEmail(email string) error` - Email format validation
- `ValidatePhone(phone string) error` - Phone format validation

These gaps are intended to be filled by bead tasks during pipeline testing.
