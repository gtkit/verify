// Package verify provides Chinese validation error translation helpers for
// Gin's shared binding validator.
//
// It configures Gin's shared *validator.Validate with the binding tag, JSON
// field names, and default Chinese translations. Applications should call [New]
// once during startup, before any Gin ShouldBind* call or verify validation.
//
// Custom field validations, translations, struct-level validations, and
// validator options should be registered through [New] with [Option] values
// during startup. validator/v10 registration methods are not safe to run
// concurrently with validation, so runtime registration in handlers is outside
// this package's safety contract.
package verify
