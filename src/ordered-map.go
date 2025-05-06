package main

import "encoding/json"

// OrderedMap is a slice of key/value pairs with custom JSON marshalling
type OrderedMapPair struct {
	Key   string
	Value interface{}
}

type OrderedMap []OrderedMapPair

// MapToOrderedMap constructs an OrderedMap from a map and a desired
// key order. Keys in the order slice that don't exist in the map are
// skipped. Map keys not in the order slice are ignored.
func MapToOrderedMap(m map[string]interface{}, order []string) OrderedMap {
	// We can't initialize om as a slice of length len(order),
	// because then if some of the keys are omitted from `m` they
	// turn up in the result as empty entries.
	var om OrderedMap
	for _, key := range order {
		if val, ok := m[key]; ok {
			om = append(om, OrderedMapPair{Key: key, Value: val})
		}
	}
	return om
}

func (om OrderedMap) MarshalJSON() ([]byte, error) {
	// Convert to a regular map so we can look up the keys
	m := make(map[string]interface{}, len(om))
	for _, pair := range om {
		m[pair.Key] = pair.Value
	}

	buf := []byte{'{'}
	for i, entry := range om {
		k := entry.Key
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		valJSON, err := json.Marshal(m[k])
		if err != nil {
			return nil, err
		}
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, keyJSON...)
		buf = append(buf, ':')
		buf = append(buf, valJSON...)
	}
	buf = append(buf, '}')
	return buf, nil
}
