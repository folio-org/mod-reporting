package main

import "testing"
import "encoding/json"
import "github.com/stretchr/testify/assert"

type TestCase struct {
	Name     string
	Input    map[string]interface{}
	Order    []string
	Expected string
}

func Test_orderedMap(t *testing.T) {
	testCases := []TestCase{
		TestCase{
			"empty map",
			map[string]interface{}{},
			[]string{},
			`{}`,
		},
		TestCase{
			"single-element map with no keys in order",
			map[string]interface{}{
				"foo": "bar",
			},
			[]string{},
			`{}`,
		},
		TestCase{
			"single-element map with key not included in order",
			map[string]interface{}{
				"foo": "chicken",
			},
			[]string{"bar", "baz"},
			`{}`,
		},
		TestCase{
			"single-element map with key included in order",
			map[string]interface{}{
				"foo": "chicken",
			},
			[]string{"foo"},
			`{"foo":"chicken"}`,
		},
		TestCase{
			"single-element map with numeric value",
			map[string]interface{}{
				"foo": 42,
			},
			[]string{"foo"},
			`{"foo":42}`,
		},
		TestCase{
			"single-element map with boolean value",
			map[string]interface{}{
				"foo": false,
			},
			[]string{"foo"},
			`{"foo":false}`,
		},
		TestCase{
			"single-element map with array value",
			map[string]interface{}{
				"foo": []int{42, 12, 99},
			},
			[]string{"foo"},
			`{"foo":[42,12,99]}`,
		},
		TestCase{
			"two-element map with keys in order",
			map[string]interface{}{
				"foo": "chicken",
				"bar": "badger",
			},
			[]string{"foo", "bar"},
			`{"foo":"chicken","bar":"badger"}`,
		},
		TestCase{
			"two-element map with keys out of order",
			map[string]interface{}{
				"foo": "chicken",
				"bar": "badger",
			},
			[]string{"bar", "foo"},
			`{"bar":"badger","foo":"chicken"}`,
		},
		TestCase{
			"five-element map with keys out of order",
			map[string]interface{}{
				"foo":    "chicken",
				"bar":    "badger",
				"baz":    "ferret",
				"quux":   "stoat",
				"thrick": "herring",
			},
			[]string{"foo", "baz", "thrick", "bar", "quux"},
			`{"foo":"chicken","baz":"ferret","thrick":"herring","bar":"badger","quux":"stoat"}`,
		},
	}

	for _, test := range testCases {
		t.Run(test.Name, func(t *testing.T) {
			m := MapToOrderedMap(test.Input, test.Order)
			bytes, err := json.Marshal(m)
			assert.Nil(t, err)
			s := string(bytes)
			// fmt.Printf("case %d: %+v -> \"%s\": got \"%s\"\n", i, test.Input, test.Expected, s)
			assert.Equal(t, test.Expected, s)
		})
	}
}
