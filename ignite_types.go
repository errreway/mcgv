package ignite

import (
	"fmt"
	"time"
)

type Time int64

type Date int64

func NewTime(val time.Time) Time {
	utcTime := val.UTC()
	// Following java contract we return current time as milliseconds since January 1, 1970, 00:00:00 UTC
	return Time(utcTime.Sub(time.Date(utcTime.Year(), utcTime.Month(), utcTime.Day(), 0, 0, 0, 0, time.UTC)).Milliseconds())
}

func (t Time) Time() time.Time {
	locTime := time.UnixMilli(int64(t))
	// The month, day and year components are fixed to January 1, 1970, even after converting the time to the local time zone.
	// This makes the behavior deterministic and predictable, given that the Ignite Go Client can receive time
	// 1. in milliseconds since January 1, 1970, 00:00:00 UTC - in which case, after applying the local time zone offset,
	//    we might get January 1, 1970 or January 2, 1970.
	// 2. as an offset in milliseconds since January 1, 1970, 00:00:00 UTC, which includes the server-side time zone
	//    offset (e.g. time returned by Ignite's H2 and Calcite SQL engines). The offset can even be negative, but after
	//    applying server-side time zone offset it can be converted to the correct time value.
	return time.Date(1970, 1, 1, locTime.Hour(), locTime.Minute(), locTime.Second(), locTime.Nanosecond(), time.Local)
}

func (t Time) String() string {
	return t.Time().Format("15:04:05.000")
}

func NewDate(val time.Time) Date {
	utcVal := val.UTC()
	return Date(utcVal.Unix()*1000 + int64(utcVal.Nanosecond())/int64(time.Millisecond))
}

func (d Date) Time() time.Time {
	return time.Unix(int64(d)/1000, (int64(d)%1000)*int64(time.Millisecond))
}

func (d Date) String() string {
	return d.Time().String()
}

//go:generate go run golang.org/x/tools/cmd/stringer -type=CollectionKind,MapKind

type CollectionKind int8

const (
	UserSet CollectionKind = iota - 1
	UserCollection
	ArrayList
	LinkedList
	HashSet
	LinkedHashSet
	SingletonList
)

type MapKind int8

const (
	UserMap MapKind = iota
	HashMap
	LinkedHashMap
)

type Collection struct {
	kind      CollectionKind
	values    []interface{}
	isNotNull bool
}

func (col *Collection) Kind() CollectionKind {
	return col.kind
}

func (col *Collection) Values() []interface{} {
	return col.values
}

func (col *Collection) Size() int {
	return len(col.values)
}

func (col *Collection) IsNull() bool {
	return !col.isNotNull
}

func NewUserCollection[T any](values ...T) Collection {
	return newIgniteCollection(UserCollection, values...)
}

func NewArrayList[T any](values ...T) Collection {
	return newIgniteCollection(ArrayList, values...)
}

func NewLinkedList[T any](values ...T) Collection {
	return newIgniteCollection(LinkedList, values...)
}

func NewSingletonList[T any](value T) Collection {
	return newIgniteCollection(SingletonList, value)
}

func NewHashSet[T any](values ...T) Collection {
	return newIgniteCollection(HashSet, values...)
}

func NewLinkedHashSet[T any](values ...T) Collection {
	return newIgniteCollection(LinkedHashSet, values...)
}

func ToSlice[T any](coll Collection) (ret []T, err error) {
	if coll.IsNull() {
		return
	}
	ret = make([]T, coll.Size())
	for i, val := range coll.Values() {
		var val0 T
		if err = convertAssign(&val0, val); err != nil {
			err = fmt.Errorf("invalid key type: %w", err)
			return
		}
		ret[i] = val0
	}
	return
}

func newIgniteCollection[T any](kind CollectionKind, values ...T) Collection {
	values0 := make([]interface{}, 0, len(values))
	for _, value := range values {
		values0 = append(values0, value)
	}
	return Collection{
		kind:      kind,
		values:    values0,
		isNotNull: true,
	}
}

type Map struct {
	kind      MapKind
	entries   []KeyValue
	isNotNull bool
}

func (m *Map) Kind() MapKind {
	return m.kind
}

func (m *Map) Entries() []KeyValue {
	return m.entries
}

func (m *Map) Size() int {
	return len(m.entries)
}

func (m *Map) IsNull() bool {
	return !m.isNotNull
}

func NewUserMap(entries ...KeyValue) Map {
	return newIgniteMap(UserMap, entries...)
}

func ToUserMap[K comparable, V any](m map[K]V) Map {
	return toIgniteMap(UserMap, m)
}

func NewHashMap(entries ...KeyValue) Map {
	return newIgniteMap(HashMap, entries...)
}

func ToHashMap[K comparable, V any](m map[K]V) Map {
	return toIgniteMap(HashMap, m)
}

func NewLinkedHashMap(entries ...KeyValue) Map {
	return newIgniteMap(LinkedHashMap, entries...)
}

func ToLinkedHashMap[K comparable, V any](m map[K]V) Map {
	return toIgniteMap(LinkedHashMap, m)
}

func ToMap[K comparable, V any](igniteMap Map) (ret map[K]V, err error) {
	if igniteMap.IsNull() {
		return
	}
	ret = make(map[K]V, igniteMap.Size())
	for _, entry := range igniteMap.entries {
		var key K
		var val V
		if err = convertAssign(&key, entry.Key); err != nil {
			err = fmt.Errorf("invalid key type: %w", err)
			return
		}
		if entry.Value != nil {
			if err = convertAssign(&val, entry.Value); err != nil {
				err = fmt.Errorf("invalid value type: %w", err)
				return
			}
		}
		ret[key] = val
	}
	return
}

func toIgniteMap[K comparable, V any](kind MapKind, m map[K]V) Map {
	if m == nil {
		return Map{
			kind: kind,
		}
	}
	entries := make([]KeyValue, 0, len(m))
	for k, v := range m {
		entries = append(entries, KeyValue{Key: k, Value: v})
	}
	return newIgniteMap(kind, entries...)
}

func newIgniteMap(kind MapKind, entries ...KeyValue) Map {
	return Map{
		kind:      kind,
		entries:   entries,
		isNotNull: true,
	}
}

type igniteType struct {
	typeId   int32
	typeName string
}

func (ignT *igniteType) String() string {
	return fmt.Sprintf("class [typeId=%d, typeName=%s]", ignT.typeId, ignT.typeName)
}

type igniteProxy struct {
	interfaces []*igniteType
}

func (p *igniteProxy) String() string {
	return fmt.Sprintf("proxy interfaces %v", p.interfaces)
}

type OptimizedMarshallerObject struct {
	payload []byte
}

func (o *OptimizedMarshallerObject) Payload() []byte {
	return o.payload
}
