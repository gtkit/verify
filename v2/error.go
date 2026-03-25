package verify

import (
	"errors"
	"fmt"
	"slices"

	"github.com/go-playground/validator/v10"
)

// FieldErr translates a field validation error into a human-readable error.
// field is the display name prepended to the message.
//
//	err := v.Field(p, "required,numeric")
//	if err != nil {
//	    return v.FieldErr("type", err) // → "type 必须是一个有效的数值"
//	}
func (ver *Verifier) FieldErr(field string, err error) error {
	if err == nil {
		return nil
	}
	valErrs, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return err
	}
	if msg, ok := firstSortedMessage(RemoveTopStruct(valErrs.Translate(ver.trans))); ok {
		return fmt.Errorf("%s %s", field, msg)
	}
	return nil
}

// StructErr translates a struct validation error into a human-readable error.
//
//	err := v.Struct(params)
//	if err != nil {
//	    return v.StructErr(err) // → "name长度必须至少为2个字符"
//	}
func (ver *Verifier) StructErr(err error) error {
	if err == nil {
		return nil
	}
	valErrs, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return err
	}
	if msg, ok := firstSortedMessage(RemoveTopStruct(valErrs.Translate(ver.trans))); ok {
		return errors.New(msg)
	}
	return nil
}

// MapErr translates a map validation result into a human-readable error.
// result is the return value of [Verifier.Map].
//
//	result := v.Map(data, rules)
//	if len(result) > 0 {
//	    return v.MapErr(result) // → "name name长度必须至少为8个字符"
//	}
func (ver *Verifier) MapErr(result map[string]any) error {
	if len(result) == 0 {
		return nil
	}
	msgs := ver.AllMapErrors(result)
	if key := firstSortedKey(msgs); key != "" {
		return fmt.Errorf("%s %s", key, msgs[key])
	}
	return nil
}

// AllFieldErrors translates all field validation errors.
// Returns a map of field name → translated message, or nil if err is nil.
//
//	err := v.Struct(params)
//	if err != nil {
//	    for field, msg := range v.AllFieldErrors(err) {
//	        log.Printf("%s: %s", field, msg)
//	    }
//	}
func (ver *Verifier) AllFieldErrors(err error) map[string]string {
	if err == nil {
		return nil
	}
	valErrs, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return nil
	}
	return RemoveTopStruct(valErrs.Translate(ver.trans))
}

// AllMapErrors translates all map validation errors.
// Returns a map of key → translated message, or nil if result is empty.
func (ver *Verifier) AllMapErrors(result map[string]any) map[string]string {
	if len(result) == 0 {
		return nil
	}
	out := make(map[string]string, len(result))
	for key, val := range result {
		valErrs, ok := val.(validator.ValidationErrors)
		if !ok {
			out[key] = fmt.Sprint(val)
			continue
		}
		if msg, ok := firstSortedMessage(valErrs.Translate(ver.trans)); ok {
			out[key] = msg
		}
	}
	return out
}

func firstSortedKey(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys[0]
}
