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

func TestNormalizeMemberDisplayNamesPreservesDuplicatesAndRejectsBlank(t *testing.T) {
	got, err := NormalizeMemberDisplayNames([]string{" Alpha ", "Alpha", "Beta"})
	if err != nil {
		t.Fatalf("NormalizeMemberDisplayNames() error = %v", err)
	}
	if len(got) != 3 || got[0] != "Alpha" || got[1] != "Alpha" || got[2] != "Beta" {
		t.Fatalf("NormalizeMemberDisplayNames() = %#v", got)
	}
	if _, err := NormalizeMemberDisplayNames([]string{"Alpha", " "}); !errors.Is(err, ErrInvalidArguments) {
		t.Fatalf("expected ErrInvalidArguments, got %v", err)
	}
}
