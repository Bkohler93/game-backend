package interfacestruct

import "encoding/json"

func Structify(m map[string]interface{}, v any) error {
	bytes, err := json.Marshal(m)
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &v)
	if err != nil {
		return err
	}
	return nil
}

func Interfacify(s any) (map[string]any, error) {
	m := map[string]any{}
	bytes, err := json.Marshal(s)
	if err != nil {
		return m, err
	}

	err = json.Unmarshal(bytes, &m)
	return m, err
}
