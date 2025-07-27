package store

type Store interface {
	StoreKeyValue(key string, value any) error
	GetAllValuesWithKeys(keypattern string) []any
}
