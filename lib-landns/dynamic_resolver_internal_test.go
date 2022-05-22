package landns

import (
	"testing"
)

func TestDynamicRecord_unmarshalAnnotation(t *testing.T) {
	t.Parallel()

	I := func(i int) *int {
		return &i
	}
	tests := []struct {
		Text     string
		ID       *int
		Volatile bool
		Error    error
	}{
		{"ID:1", I(1), false, nil},
		{" ID:2 ", I(2), false, nil},
		{"\tID:3 \t ", I(3), false, nil},
		{"volatile", nil, true, nil},
		{"VoLaTiLE", nil, true, nil},
		{"VOLATILE", nil, true, nil},
		{"hello id:4 volatile world", I(4), true, nil},
		{"VOLATILE ID:5", I(5), true, nil},
		{" ", nil, false, nil},
		{"", nil, false, nil},
		{"id", nil, false, ErrInvalidDynamicRecordFormat},
		{"volatile:1", nil, false, ErrInvalidDynamicRecordFormat},
	}

	r := new(DynamicRecord)

	for _, tt := range tests {
		if err := r.unmarshalAnnotation([]byte(tt.Text)); err != tt.Error {
			if tt.Error == nil {
				t.Errorf("failed to unmarshal annotation: %s", err)
			} else {
				t.Errorf("failed to unmarshal annotation: expected error %#v but got %#v", tt.Error, err)
			}
			continue
		}

		if tt.ID != nil && r.ID != nil {
			if *r.ID != *tt.ID {
				t.Errorf("failed to unmarshal ID: expected %d but got %d", tt.ID, r.ID)
			}
		} else {
			if tt.ID == nil && r.ID != nil {
				t.Errorf("failed to unmarshal ID: expected <nil> but got %d", r.ID)
			} else if tt.ID != nil && r.ID == nil {
				t.Errorf("failed to unmarshal ID: expected %d but got <nil>", tt.ID)
			}
		}

		if r.Volatile != tt.Volatile {
			t.Errorf("failed to unmarshal Volatile: expected %v but got %v", tt.Volatile, r.Volatile)
		}
	}
}
