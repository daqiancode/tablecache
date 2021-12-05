package tablecache

import "encoding/json"

type Marshaller interface {
	Marshal(value interface{}) (string, error)
	Unmarshal(valueRef interface{}, bs string) error
}

type JSONMarshaller struct {
}

func (s *JSONMarshaller) Marshal(value interface{}) (string, error) {
	bs, err := json.Marshal(value)
	return string(bs), err
}

func (s *JSONMarshaller) Unmarshal(valueRef interface{}, bs string) error {
	return json.Unmarshal([]byte(bs), valueRef)
}
