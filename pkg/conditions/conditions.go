package conditions

import (
	"slices"

	corev1 "k8s.io/api/core/v1"
)

// Condition contains details for the current condition of Object
type Condition[T ~string] interface {
	GetType() T
	GetStatus() corev1.ConditionStatus
}

// Upsertable describes conditions that can be used upserted
type Upsertable[T ~string, C any] interface {
	Condition[T]
	Match(o C) bool
	Touch() C
}

// Index returns then index of first occurence of condition with given tpe, or -1 if not present
func Index[C Condition[T], T ~string](conditions []C, tpe T) int {
	for i, cond := range conditions {
		if cond.GetType() == tpe {
			return i
		}
	}

	return -1
}

func StatusTrue[C Condition[T], T ~string](conditions []C, tpe T) bool {
	return StatusEqual(conditions, tpe, corev1.ConditionTrue)
}

func StatusEqual[C Condition[T], T ~string](conditions []C, tpe T, status corev1.ConditionStatus) bool {
	for _, cond := range conditions {
		if cond.GetType() == tpe && cond.GetStatus() == status {
			return true
		}
	}

	return false
}

func Get[C Condition[T], T ~string](conditions []C, tpe T) (C, bool) {
	for _, cond := range conditions {
		if cond.GetType() == tpe {
			return cond, true
		}
	}

	var zero C
	return zero, false
}

// Upsert inserts or updates exisiting condition with given value, returns true if modified
func Upsert[T ~string, C Upsertable[T, C]](conditions *[]C, cond C) bool {
	dst := *conditions

	i := Index(dst, cond.GetType())
	switch {
	// not found - insert
	case i == -1:
		dst = append(dst, cond.Touch())
	// not equal - update
	case !cond.Match(dst[i]):
		dst[i] = cond.Touch()
	// found and equal - noop
	default:
		return false
	}

	*conditions = dst
	return true
}

// Clear clears condition of given tpe, returns true if modified
func Clear[T ~string, C Condition[T]](conditions *[]C, types ...T) bool {
	dst := (*conditions)[:0]

	for _, cond := range *conditions {
		if !slices.Contains(types, cond.GetType()) {
			dst = append(dst, cond)
		}
	}

	*conditions = dst
	return true
}
