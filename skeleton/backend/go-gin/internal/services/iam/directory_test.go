package iam

import (
	"errors"
	"testing"
)

func TestNormalizeMemberUUIDsRejectsDuplicateOrBlankUUID(t *testing.T) {
	for _, input := range [][]string{{"member-a", "member-a"}, {"member-a", " "}} {
		_, err := NormalizeMemberUUIDs(input)
		if !errors.Is(err, ErrInvalidArguments) {
			t.Fatalf("NormalizeMemberUUIDs(%#v) error = %v, want ErrInvalidArguments", input, err)
		}
	}
}

func TestNormalizeMemberUUIDsPreservesCallerOrder(t *testing.T) {
	got, err := NormalizeMemberUUIDs([]string{"member-b", "member-a"})
	if err != nil {
		t.Fatalf("NormalizeMemberUUIDs() error = %v", err)
	}
	if len(got) != 2 || got[0] != "member-b" || got[1] != "member-a" {
		t.Fatalf("NormalizeMemberUUIDs() = %#v", got)
	}
}
