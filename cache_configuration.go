package ignite

import (
	"context"
	"fmt"
	"time"
)

type CacheAtomicityMode = int32

const (
	Transactional CacheAtomicityMode = 0
	Atomic        CacheAtomicityMode = 1
)

type CacheMode = int32

const (
	Replicated  CacheMode = 1
	Partitioned CacheMode = 2
)

type PartitionLossPolicy = int32

const (
	ReadOnlySafe  PartitionLossPolicy = 0
	ReadOnlyAll   PartitionLossPolicy = 1
	ReadWriteSafe PartitionLossPolicy = 2
	ReadWriteAll  PartitionLossPolicy = 3
	Ignore        PartitionLossPolicy = 4
)

type CacheWriteSynchronizationMode = int32

const (
	FullSync    CacheWriteSynchronizationMode = 0
	FullAsync   CacheWriteSynchronizationMode = 1
	PrimarySync CacheWriteSynchronizationMode = 2
)

type CacheRebalanceMode = int32

const (
	Sync  CacheRebalanceMode = 0
	Async CacheRebalanceMode = 1
	None                     = 2
)

type CacheKeyConfig interface {
	TypeName() string
	AffinityKeyFieldName() string
}

type cacheKeyConfig struct {
	typeName      string
	affKeyFldName string
}

func (c *cacheKeyConfig) TypeName() string {
	return c.typeName
}

func (c *cacheKeyConfig) AffinityKeyFieldName() string {
	return c.affKeyFldName
}

type QueryEntity interface {
	KeyType() string
	ValueType() string
	TableName() string
	KeyFieldName() string
	ValueFieldName() string
	Fields() []QueryField
	Aliases() map[string]string
}

type queryEntity struct {
	keyType    string
	valType    string
	tblName    string
	keyFldName string
	valFldName string
	fields     []QueryField
	aliases    map[string]string
}

func (q *queryEntity) KeyType() string {
	return q.keyType
}

func (q *queryEntity) ValueType() string {
	return q.valType
}

func (q *queryEntity) TableName() string {
	return q.tblName
}

func (q *queryEntity) KeyFieldName() string {
	return q.keyFldName
}

func (q *queryEntity) ValueFieldName() string {
	return q.valFldName
}

func (q *queryEntity) Fields() []QueryField {
	return q.fields
}

func (q *queryEntity) Aliases() map[string]string {
	return q.aliases
}

type QueryField interface {
	Name() string
	TypeName() string
	IsKey() bool
	IsNotNull() bool
	DefaultValue() interface{}
	Precision() int
	Scale() int
}

type queryField struct {
	name      string
	typeName  string
	isKey     bool
	isNotNull bool
	precision int
	scale     int
	dfltVal   interface{}
}

func (q *queryField) Name() string {
	return q.name
}

func (q queryField) TypeName() string {
	return q.typeName
}

func (q *queryField) IsKey() bool {
	return q.isKey
}

func (q *queryField) IsNotNull() bool {
	return q.isNotNull
}

func (q *queryField) DefaultValue() interface{} {
	return q.dfltVal
}

func (q *queryField) Precision() int {
	return q.precision
}

func (q *queryField) Scale() int {
	return q.scale
}

type propertyCode = int16

const (
	cacheNameProp              propertyCode = 0
	cacheModeProp              propertyCode = 1
	cacheAtomicityModeProp     propertyCode = 2
	backupsProp                propertyCode = 3
	writeSyncModeProp          propertyCode = 4
	copyOnReadProp             propertyCode = 5
	readFromBackupProp         propertyCode = 6
	dataRegionNameProp         propertyCode = 100
	onheapCacheEnabledProp     propertyCode = 101
	queryEntitiesProp          propertyCode = 200
	queryParallelismProp       propertyCode = 201
	queryDetailsMetricSizeProp propertyCode = 202
	sqlSchemaProp              propertyCode = 203
	sqlIndexMaxInlineSizeProp  propertyCode = 204
	sqlEscapeAllProp           propertyCode = 205
	maxQueryIteratorsProp      propertyCode = 206
	rebalanceModeProp          propertyCode = 300
	rebalanceOrderProp         propertyCode = 305
	groupNameProp              propertyCode = 400
	cacheKeyConfigProp         propertyCode = 401
	maxAsyncOpsProp            propertyCode = 403
	partitionLossPolicyProp    propertyCode = 404
	eagerTtlProp               propertyCode = 405
	statsEnabledProp           propertyCode = 406
	expirePolicyProp           propertyCode = 407
)

type CacheConfiguration struct {
	props map[int16]interface{}
}

func CreateCacheConfiguration(name string, opts ...func(map[int16]interface{})) CacheConfiguration {
	ret := CacheConfiguration{
		props: make(map[int16]interface{}),
	}
	ret.props[cacheNameProp] = name
	for _, opt := range opts {
		opt(ret.props)
	}
	return ret
}

func (config *CacheConfiguration) marshall(marshaller Marshaller, writer BinaryWriter) error {
	protoCtx := marshaller.ProtocolContext()
	origPos := writer.Position()
	writer.WriteInt32(0)
	writer.WriteInt16(int16(len(config.props)))
	for propCode, propValue := range config.props {
		writer.WriteInt16(propCode)
		switch val := propValue.(type) {
		case time.Duration:
			writer.WriteInt64(val.Milliseconds())
		case string:
			marshalString(writer, val)
		case bool:
			writer.WriteBool(val)
		case int:
			writer.WriteInt32(int32(val))
		case int32:
			writer.WriteInt32(val)
		case int64:
			writer.WriteInt64(val)
		case ExpirePolicy:
			{
				if !protoCtx.SupportsExpiryPolicy() {
					return fmt.Errorf("expiry policies are not supported on protocol version %v", protoCtx.Version())
				}
				if val == nil {
					writer.WriteBool(false)
				} else {
					writer.WriteBool(true)
					writer.WriteInt64(durationToMillis(val.Creation()))
					writer.WriteInt64(durationToMillis(val.Update()))
					writer.WriteInt64(durationToMillis(val.Access()))
				}
			}
		default:
			return fmt.Errorf("property is not supported %d, %v", propCode, propValue)
		}
	}
	finalPos := writer.Position()
	writer.SetPosition(origPos)
	writer.WriteInt32(finalPos - origPos - 4)
	writer.SetPosition(finalPos)
	return nil
}

func unmarshall(ctx context.Context, marshaller Marshaller, reader BinaryReader) (CacheConfiguration, error) {
	var err error
	protoCtx := marshaller.ProtocolContext()
	reader.ReadInt32() // Skip unneeded length field
	props := make(map[int16]interface{})
	conf := CacheConfiguration{
		props: props,
	}
	props[cacheAtomicityModeProp] = reader.ReadInt32()
	props[backupsProp] = int(reader.ReadInt32())
	props[cacheModeProp] = reader.ReadInt32()
	props[copyOnReadProp] = reader.ReadBool()
	if err = conf.setStringProperty(reader, dataRegionNameProp); err != nil {
		return conf, err
	}
	props[eagerTtlProp] = reader.ReadBool()
	props[statsEnabledProp] = reader.ReadBool()
	if err = conf.setStringProperty(reader, groupNameProp); err != nil {
		return conf, err
	}
	reader.ReadInt64() // Skip deprecated DefaultLockTimeout
	props[maxAsyncOpsProp] = int(reader.ReadInt32())
	props[maxQueryIteratorsProp] = int(reader.ReadInt32())
	if err = conf.setStringProperty(reader, cacheNameProp); err != nil {
		return conf, err
	}
	props[onheapCacheEnabledProp] = reader.ReadBool()
	props[partitionLossPolicyProp] = reader.ReadInt32()
	props[queryDetailsMetricSizeProp] = int(reader.ReadInt32())
	props[queryParallelismProp] = int(reader.ReadInt32())
	props[readFromBackupProp] = reader.ReadBool()
	reader.ReadInt32() // Skip deprecated RebalanceBatchSize
	reader.ReadInt64() // Skip deprecated RebalanceBatchesPrefetchCount
	reader.ReadInt64() // Skip deprecated RebalanceDelay
	props[rebalanceModeProp] = reader.ReadInt32()
	props[rebalanceOrderProp] = int(reader.ReadInt32())
	reader.ReadInt64() // Skip deprecated RebalanceThrottle
	reader.ReadInt64() // Skip deprecated RebalanceTimeout
	props[sqlEscapeAllProp] = reader.ReadBool()
	props[sqlIndexMaxInlineSizeProp] = int(reader.ReadInt32())
	if err = conf.setStringProperty(reader, sqlSchemaProp); err != nil {
		return conf, err
	}
	props[writeSyncModeProp] = reader.ReadInt32()
	err = setCollectionProperty(&conf, reader, cacheKeyConfigProp, func(reader BinaryReader) (CacheKeyConfig, error) {
		typeName, err0 := unmarshallString(reader, false)
		if err0 != nil {
			return nil, err0
		}
		affKeyFldName, err0 := unmarshallString(reader, false)
		if err0 != nil {
			return nil, err0
		}
		return &cacheKeyConfig{
			typeName:      typeName,
			affKeyFldName: affKeyFldName,
		}, nil
	})
	if err != nil {
		return conf, err
	}
	err = setCollectionProperty(&conf, reader, queryEntitiesProp, func(reader BinaryReader) (QueryEntity, error) {
		keyType, err0 := unmarshallString(reader, false)
		if err0 != nil {
			return nil, nil
		}
		valType, err0 := unmarshallString(reader, false)
		if err0 != nil {
			return nil, nil
		}
		tblName, err0 := unmarshallString(reader, false)
		if err0 != nil {
			return nil, nil
		}
		keyFldName, err0 := unmarshallString(reader, false)
		if err0 != nil {
			return nil, nil
		}
		valFldName, err0 := unmarshallString(reader, false)
		if err0 != nil {
			return nil, nil
		}
		withPrecisionScale := protoCtx.SupportsQueryEntityPrecisionAndScale()
		qryFields, err0 := readCollection(reader, func(reader BinaryReader) (QueryField, error) {
			name, err1 := unmarshallString(reader, false)
			if err1 != nil {
				return nil, err1
			}
			typeName, err1 := unmarshallString(reader, false)
			if err1 != nil {
				return nil, err1
			}
			isKey := reader.ReadBool()
			isNotNull := reader.ReadBool()
			dfltVal, err1 := marshaller.Unmarshall(ctx, reader)
			if err != nil {
				return nil, err
			}
			precision := -1
			if withPrecisionScale {
				precision = int(reader.ReadInt32())
			}
			scale := -1
			if withPrecisionScale {
				scale = int(reader.ReadInt32())
			}
			return &queryField{
				name:      name,
				typeName:  typeName,
				isKey:     isKey,
				isNotNull: isNotNull,
				dfltVal:   dfltVal,
				precision: precision,
				scale:     scale,
			}, nil
		})
		aliasesSz := reader.ReadInt32()
		aliases := make(map[string]string, aliasesSz)
		for i := 0; i < int(aliasesSz); i++ {
			key, err1 := unmarshallString(reader, false)
			if err1 != nil {
				return nil, err1
			}
			val, err1 := unmarshallString(reader, false)
			if err1 != nil {
				return nil, err1
			}
			aliases[key] = val
		}
		return &queryEntity{
			keyType:    keyType,
			valType:    valType,
			tblName:    tblName,
			keyFldName: keyFldName,
			valFldName: valFldName,
			fields:     qryFields,
			aliases:    aliases,
		}, nil
	})
	if err != nil {
		return conf, err
	}
	if protoCtx.SupportsExpiryPolicy() && reader.ReadBool() {
		expPolicy := &expirePolicyImpl{
			creation: millisToDuration(reader.ReadInt64()),
			update:   millisToDuration(reader.ReadInt64()),
			access:   millisToDuration(reader.ReadInt64()),
		}
		props[expirePolicyProp] = expPolicy
	}
	return conf, err
}

func (config *CacheConfiguration) setStringProperty(reader BinaryReader, propCode propertyCode) error {
	var err error
	var val string
	if val, err = unmarshallString(reader, false); err != nil {
		return err
	}
	if len(val) > 0 {
		config.props[propCode] = val
	}
	return nil
}

func (config *CacheConfiguration) Name() string {
	return getProperty(config, 0, "")
}

func (config *CacheConfiguration) Backups() int {
	return getProperty(config, backupsProp, 0)
}

func (config *CacheConfiguration) CacheMode() CacheMode {
	return getProperty[CacheMode](config, cacheModeProp, -1)
}

func (config *CacheConfiguration) CacheAtomicityMode() CacheAtomicityMode {
	return getProperty[CacheAtomicityMode](config, cacheAtomicityModeProp, -1)
}

func (config *CacheConfiguration) IsCopyOnRead() bool {
	return getProperty(config, copyOnReadProp, false)
}

func (config *CacheConfiguration) DataRegionName() string {
	return getProperty(config, dataRegionNameProp, "")
}

func (config *CacheConfiguration) IsEagerTtl() bool {
	return getProperty(config, eagerTtlProp, false)
}

func (config *CacheConfiguration) GroupName() string {
	return getProperty(config, groupNameProp, "")
}

func (config *CacheConfiguration) MaxConcurrentAsyncOperations() int {
	return getProperty(config, maxAsyncOpsProp, 0)
}

func (config *CacheConfiguration) MaxQueryIteratorsCount() int {
	return getProperty(config, maxQueryIteratorsProp, 0)
}

func (config *CacheConfiguration) IsOnHeapCacheEnabled() bool {
	return getProperty(config, onheapCacheEnabledProp, false)
}

func (config *CacheConfiguration) PartitionLossPolicy() PartitionLossPolicy {
	return getProperty[PartitionLossPolicy](config, partitionLossPolicyProp, -1)
}

func (config *CacheConfiguration) QueryDetailsMetricsSize() int {
	return getProperty(config, queryDetailsMetricSizeProp, 0)
}

func (config *CacheConfiguration) QueryParallelism() int {
	return getProperty(config, queryParallelismProp, 0)
}

func (config *CacheConfiguration) IsReadFromBackup() bool {
	return getProperty(config, readFromBackupProp, false)
}

func (config *CacheConfiguration) RebalanceMode() CacheRebalanceMode {
	return getProperty(config, rebalanceModeProp, Sync)
}

func (config *CacheConfiguration) RebalanceOrder() int {
	return getProperty(config, rebalanceOrderProp, 0)
}

func (config *CacheConfiguration) IsSqlEscapeAll() bool {
	return getProperty(config, sqlEscapeAllProp, false)
}

func (config *CacheConfiguration) SqlIndexMaxInlineSize() int {
	return getProperty(config, sqlIndexMaxInlineSizeProp, 0)
}

func (config *CacheConfiguration) SqlSchema() string {
	return getProperty(config, sqlSchemaProp, "")
}

func (config *CacheConfiguration) WriteSynchronizationMode() CacheWriteSynchronizationMode {
	return getProperty[CacheWriteSynchronizationMode](config, writeSyncModeProp, -1)
}

func (config *CacheConfiguration) ExpiryPolicy() ExpirePolicy {
	return getProperty[ExpirePolicy](config, expirePolicyProp, nil)
}

func (config *CacheConfiguration) KeyConfiguration() []CacheKeyConfig {
	return getProperty[[]CacheKeyConfig](config, cacheKeyConfigProp, nil)
}

func (config *CacheConfiguration) QueryEntities() []QueryEntity {
	return getProperty[[]QueryEntity](config, queryEntitiesProp, nil)
}

func getProperty[T any](config *CacheConfiguration, code propertyCode, defaultValue T) T {
	data, hasKey := config.props[code]
	if !hasKey {
		return defaultValue
	}
	ret, ok := data.(T)
	if !ok {
		return defaultValue
	}
	return ret
}

func setCollectionProperty[T any](config *CacheConfiguration, reader BinaryReader, code propertyCode, elemReader func(reader BinaryReader) (T, error)) error {
	coll, err := readCollection[T](reader, elemReader)
	if err != nil {
		return err
	}
	config.props[code] = coll
	return nil
}

func readCollection[T any](reader BinaryReader, elemReader func(reader BinaryReader) (T, error)) ([]T, error) {
	sz := reader.ReadInt32()
	coll := make([]T, sz)
	for i := 0; i < int(sz); i++ {
		el, err := elemReader(reader)
		if err != nil {
			return nil, err
		}
		coll = append(coll, el)
	}
	return coll, nil
}

func WithCacheGroupName(name string) func(map[int16]interface{}) {
	if len(name) == 0 {
		return func(_ map[int16]interface{}) {}
	}
	return func(props map[int16]interface{}) {
		props[groupNameProp] = name
	}
}
func WithBackupsCount(cnt int) func(map[int16]interface{}) {
	return func(props map[int16]interface{}) {
		props[backupsProp] = int32(cnt)
	}
}

func WithCacheMode(mode CacheMode) func(map[int16]interface{}) {
	return func(props map[int16]interface{}) {
		props[cacheModeProp] = mode
	}
}

func WithCacheAtomicityMode(mode CacheAtomicityMode) func(map[int16]interface{}) {
	return func(props map[int16]interface{}) {
		props[cacheAtomicityModeProp] = mode
	}
}

func WithQueryParallelism(parallelism int) func(map[int16]interface{}) {
	return func(props map[int16]interface{}) {
		props[queryParallelismProp] = parallelism
	}
}

func WithExpiryPolicy(expPolicy ExpirePolicy) func(map[int16]interface{}) {
	return func(props map[int16]interface{}) {
		props[expirePolicyProp] = expPolicy
	}
}
