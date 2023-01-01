package request

import (
	"encoding/json"
	"errors"
)

type RawData []byte

// MarshalJSON returns r as the JSON encoding of r.
func (r *RawData) MarshalJSON() ([]byte, error) {
	if r == nil || len(*r) == 0 || string(*r) == "null" {
		return []byte("null"), nil
	}

	data := []byte(*r)
	if len(data) > 6 && string(data[0:6]) == "\"spike" {
		return data, nil
	}

	encoded, err := json.Marshal(&data)
	if err != nil {
		return nil, err
	}
	encoded = append([]byte("\"spike"), encoded[1:]...)
	return encoded, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (r *RawData) UnmarshalJSON(data []byte) error {
	if r == nil {
		return errors.New("RawData: UnmarshalJSON on nil pointer")
	}

	if len(data) > 6 && string(data[0:6]) == "\"spike" {
		marshaled := []byte("\"" + string(data[6:]))
		var rawData []byte
		err := json.Unmarshal(marshaled, &rawData)
		if err != nil {
			return err
		}
		*r = append((*r)[0:0], rawData...)
		return nil
	}

	*r = append((*r)[0:0], data...)
	return nil
}
