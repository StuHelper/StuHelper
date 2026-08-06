package go_ora

import (
	"database/sql"
	"reflect"
	"testing"
)

func mustStuHelperNumber(t *testing.T, input string) *Number {
	t.Helper()
	number, err := NewNumberFromString(input)
	if err != nil {
		t.Fatalf("create Oracle number %q: %v", input, err)
	}
	return number
}

func TestSetNumberRejectsIntegerOverflow(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target reflect.Type
	}{
		{name: "plain int8", input: "128", target: reflect.TypeOf(int8(0))},
		{name: "plain uint8 negative", input: "-1", target: reflect.TypeOf(uint8(0))},
		{name: "plain uint8 high", input: "256", target: reflect.TypeOf(uint8(0))},
		{name: "null byte negative", input: "-1", target: reflect.TypeOf(sql.NullByte{})},
		{name: "null byte high", input: "256", target: reflect.TypeOf(sql.NullByte{})},
		{name: "null int16 low", input: "-32769", target: reflect.TypeOf(sql.NullInt16{})},
		{name: "null int16 high", input: "32768", target: reflect.TypeOf(sql.NullInt16{})},
		{name: "null int32 low", input: "-2147483649", target: reflect.TypeOf(sql.NullInt32{})},
		{name: "null int32 high", input: "2147483648", target: reflect.TypeOf(sql.NullInt32{})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := reflect.New(test.target).Elem()
			if err := setNumber(target, mustStuHelperNumber(t, test.input)); err == nil {
				t.Fatalf("setNumber accepted overflowing %s for %v", test.input, test.target)
			}
		})
	}
}

func TestSetStringRejectsIntegerOverflow(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target reflect.Type
	}{
		{name: "plain int8", input: "128", target: reflect.TypeOf(int8(0))},
		{name: "plain uint8 negative", input: "-1", target: reflect.TypeOf(uint8(0))},
		{name: "plain uint8 high", input: "256", target: reflect.TypeOf(uint8(0))},
		{name: "null byte negative", input: "-1", target: reflect.TypeOf(sql.NullByte{})},
		{name: "null byte high", input: "256", target: reflect.TypeOf(sql.NullByte{})},
		{name: "null int16 low", input: "-32769", target: reflect.TypeOf(sql.NullInt16{})},
		{name: "null int16 high", input: "32768", target: reflect.TypeOf(sql.NullInt16{})},
		{name: "null int32 low", input: "-2147483649", target: reflect.TypeOf(sql.NullInt32{})},
		{name: "null int32 high", input: "2147483648", target: reflect.TypeOf(sql.NullInt32{})},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := reflect.New(test.target).Elem()
			if err := setString(target, test.input); err == nil {
				t.Fatalf("setString accepted overflowing %s for %v", test.input, test.target)
			}
		})
	}
}

func TestIntegerAssignmentAcceptsBoundaryValues(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		target reflect.Type
	}{
		{name: "plain int8 minimum", input: "-128", target: reflect.TypeOf(int8(0))},
		{name: "plain int8 maximum", input: "127", target: reflect.TypeOf(int8(0))},
		{name: "plain uint8 maximum", input: "255", target: reflect.TypeOf(uint8(0))},
		{name: "null byte maximum", input: "255", target: reflect.TypeOf(sql.NullByte{})},
		{name: "null int16 minimum", input: "-32768", target: reflect.TypeOf(sql.NullInt16{})},
		{name: "null int16 maximum", input: "32767", target: reflect.TypeOf(sql.NullInt16{})},
		{name: "null int32 minimum", input: "-2147483648", target: reflect.TypeOf(sql.NullInt32{})},
		{name: "null int32 maximum", input: "2147483647", target: reflect.TypeOf(sql.NullInt32{})},
	}

	for _, test := range tests {
		t.Run(test.name+" from number", func(t *testing.T) {
			target := reflect.New(test.target).Elem()
			if err := setNumber(target, mustStuHelperNumber(t, test.input)); err != nil {
				t.Fatalf("setNumber rejected boundary %s for %v: %v", test.input, test.target, err)
			}
		})
		t.Run(test.name+" from string", func(t *testing.T) {
			target := reflect.New(test.target).Elem()
			if err := setString(target, test.input); err != nil {
				t.Fatalf("setString rejected boundary %s for %v: %v", test.input, test.target, err)
			}
		})
	}
}
