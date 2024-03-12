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
	None  CacheRebalanceMode = 2
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

type QueryEntity struct {
	keyType    string
	valType    string
	tblName    string
	keyFldName string
	valFldName string
	fields     []QueryField
	aliases    map[string]string
	indexes    []QueryIndex
}

func (q *QueryEntity) KeyType() string {
	return q.keyType
}

func (q *QueryEntity) ValueType() string {
	return q.valType
}

func (q *QueryEntity) TableName() string {
	return q.tblName
}

func (q *QueryEntity) KeyFieldName() string {
	return q.keyFldName
}

func (q *QueryEntity) ValueFieldName() string {
	return q.valFldName
}

func (q *QueryEntity) Fields() []QueryField {
	return q.fields
}

func (q *QueryEntity) Aliases() map[string]string {
	return q.aliases
}

func (q *QueryEntity) Indexes() []QueryIndex {
	return q.indexes
}

func (q *QueryEntity) Copy() QueryEntity {
	ret := QueryEntity{
		keyType:    q.keyType,
		valType:    q.valType,
		tblName:    q.tblName,
		keyFldName: q.keyFldName,
		valFldName: q.valFldName,
	}
	if len(q.fields) > 0 {
		ret.fields = make([]QueryField, len(ret.fields))
		copy(q.fields, ret.fields)
	}
	if len(q.aliases) > 0 {
		ret.aliases = make(map[string]string, len(q.aliases))
		for orig, alias := range q.aliases {
			ret.aliases[orig] = alias
		}
	}
	if len(q.indexes) > 0 {
		szIndexes := len(q.indexes)
		ret.indexes = make([]QueryIndex, szIndexes)
		for i := 0; i < szIndexes; i++ {
			ret.indexes[i] = q.indexes[i].Copy()
		}
	}
	return ret
}

type QueryField struct {
	name      string
	typeName  string
	isKey     bool
	isNotNull bool
	precision int
	scale     int
	dfltVal   interface{}
}

func (q *QueryField) Name() string {
	return q.name
}

func (q *QueryField) TypeName() string {
	return q.typeName
}

func (q *QueryField) IsKey() bool {
	return q.isKey
}

func (q *QueryField) IsNotNull() bool {
	return q.isNotNull
}

func (q *QueryField) DefaultValue() interface{} {
	return q.dfltVal
}

func (q *QueryField) Precision() int {
	return q.precision
}

func (q *QueryField) Scale() int {
	return q.scale
}

type IndexType = int8

const (
	Sorted     IndexType = 0
	FullText   IndexType = 1
	GeoSpatial IndexType = 2
)

type IndexField struct {
	Name string
	Asc  bool
}

type QueryIndex struct {
	name     string
	idxType  IndexType
	inlineSz int
	fields   []IndexField
}

func (q *QueryIndex) Name() string {
	return q.name
}

func (q *QueryIndex) Type() IndexType {
	return q.idxType
}

func (q *QueryIndex) InlineSize() int {
	return q.inlineSz
}

func (q *QueryIndex) Fields() []IndexField {
	return q.fields
}

func (q *QueryIndex) Copy() QueryIndex {
	ret := QueryIndex{
		name:     q.name,
		idxType:  q.idxType,
		inlineSz: q.inlineSz,
	}
	szFlds := len(q.fields)
	if szFlds > 0 {
		ret.fields = make([]IndexField, szFlds)
		copy(q.fields, ret.fields)
	}
	return ret
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
	onHeapCacheEnabledProp     propertyCode = 101
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

func CreateCacheConfiguration(name string, opts ...func(*CacheConfiguration)) CacheConfiguration {
	ret := CacheConfiguration{
		props: make(map[int16]interface{}),
	}
	ret.props[cacheNameProp] = name
	for _, opt := range opts {
		opt(&ret)
	}
	return ret
}

func (config *CacheConfiguration) marshall(ctx context.Context, marshaller marshaller, writer BinaryWriter) error {
	protoCtx := marshaller.protocolContext()
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
		case []CacheKeyConfig:
			{
				err := writeCollection(writer, val, func(output BinaryWriter, keyCfg CacheKeyConfig) error {
					marshalString(output, keyCfg.TypeName())
					marshalString(output, keyCfg.AffinityKeyFieldName())
					return nil
				})
				if err != nil {
					return err
				}
			}
		case []QueryEntity:
			{
				err := writeCollection(writer, val, marshalQueryEntity(ctx, marshaller))
				if err != nil {
					return err
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

func unmarshall(ctx context.Context, marshaller marshaller, reader BinaryReader) (CacheConfiguration, error) {
	var err error
	protoCtx := marshaller.protocolContext()
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
	props[onHeapCacheEnabledProp] = reader.ReadBool()
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
		typeName, err0 := unmarshalString(reader, false)
		if err0 != nil {
			return nil, err0
		}
		affKeyFldName, err0 := unmarshalString(reader, false)
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
	err = setCollectionProperty(&conf, reader, queryEntitiesProp, unmarshalQueryEntity(ctx, marshaller))
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

func marshalQueryEntity(ctx context.Context, marshaller marshaller) func(writer BinaryWriter, entity QueryEntity) error {
	marshalEmptyAsNull := func(writer BinaryWriter, val string) {
		if len(val) == 0 {
			writer.WriteNull()
		} else {
			marshalString(writer, val)
		}
	}
	queryFieldMarshalFunc := marshalQueryField(ctx, marshaller)
	return func(writer BinaryWriter, entity QueryEntity) error {
		marshalString(writer, entity.KeyType())
		marshalString(writer, entity.ValueType())
		marshalEmptyAsNull(writer, entity.TableName())
		marshalEmptyAsNull(writer, entity.KeyFieldName())
		marshalEmptyAsNull(writer, entity.ValueFieldName())
		if err := writeCollection(writer, entity.Fields(), queryFieldMarshalFunc); err != nil {
			return err
		}
		writer.WriteInt32(int32(len(entity.Aliases())))
		if len(entity.Aliases()) > 0 {
			for orig, alias := range entity.Aliases() {
				marshalString(writer, orig)
				marshalString(writer, alias)
			}
		}
		if err := writeCollection(writer, entity.Indexes(), marshalQueryIndex); err != nil {
			return err
		}
		return nil
	}
}

func marshalQueryField(ctx context.Context, marshaller marshaller) func(writer BinaryWriter, field QueryField) error {
	protoCtx := marshaller.protocolContext()
	withPrecisionScale := protoCtx.SupportsQueryEntityPrecisionAndScale()
	return func(writer BinaryWriter, field QueryField) error {
		var err error
		marshalString(writer, field.Name())
		marshalString(writer, field.TypeName())
		writer.WriteBool(field.IsKey())
		writer.WriteBool(field.IsNotNull())
		err = marshaller.marshal(ctx, writer, field.DefaultValue())
		if err != nil {
			return err
		}
		if withPrecisionScale {
			writer.WriteInt32(int32(field.Precision()))
			writer.WriteInt32(int32(field.Scale()))
		}
		return err
	}
}

func marshalQueryIndex(writer BinaryWriter, index QueryIndex) error {
	marshalString(writer, index.Name())
	writer.WriteInt8(index.Type())
	writer.WriteInt32(int32(index.InlineSize()))
	err := writeCollection(writer, index.Fields(), func(writer0 BinaryWriter, field IndexField) error {
		marshalString(writer0, field.Name)
		writer0.WriteBool(field.Asc)
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func unmarshalQueryEntity(ctx context.Context, marshaller marshaller) func(reader BinaryReader) (QueryEntity, error) {
	qryFieldUnmarshalProc := unmarshalQueryField(ctx, marshaller)
	return func(reader BinaryReader) (QueryEntity, error) {
		var err error
		entity := QueryEntity{}
		entity.keyType, err = unmarshalString(reader, false)
		if err != nil {
			return entity, err
		}
		entity.valType, err = unmarshalString(reader, false)
		if err != nil {
			return entity, err
		}
		entity.tblName, err = unmarshalString(reader, false)
		if err != nil {
			return entity, err
		}
		entity.keyFldName, err = unmarshalString(reader, false)
		if err != nil {
			return entity, err
		}
		entity.valFldName, err = unmarshalString(reader, false)
		if err != nil {
			return entity, err
		}
		entity.fields, err = readCollection(reader, qryFieldUnmarshalProc)
		if err != nil {
			return entity, err
		}
		aliasesSz := reader.ReadInt32()
		entity.aliases = make(map[string]string, aliasesSz)
		for i := 0; i < int(aliasesSz); i++ {
			key, err0 := unmarshalString(reader, false)
			if err0 != nil {
				return entity, err0
			}
			val, err0 := unmarshalString(reader, false)
			if err0 != nil {
				return entity, err0
			}
			entity.aliases[key] = val
		}
		entity.indexes, err = readCollection(reader, unmarshallQueryIndex)
		if err != nil {
			return entity, err
		}
		return entity, nil
	}
}

func unmarshalQueryField(ctx context.Context, marshaller marshaller) func(reader BinaryReader) (QueryField, error) {
	protoCtx := marshaller.protocolContext()
	withPrecisionScale := protoCtx.SupportsQueryEntityPrecisionAndScale()
	return func(reader BinaryReader) (QueryField, error) {
		var err error
		fld := QueryField{precision: -1, scale: -1}
		fld.name, err = unmarshalString(reader, false)
		if err != nil {
			return fld, err
		}
		fld.typeName, err = unmarshalString(reader, false)
		if err != nil {
			return fld, err
		}
		fld.isKey = reader.ReadBool()
		fld.isNotNull = reader.ReadBool()
		fld.dfltVal, err = marshaller.unmarshall(ctx, reader)
		if err != nil {
			return fld, err
		}
		if withPrecisionScale {
			fld.precision = int(reader.ReadInt32())
		}
		if withPrecisionScale {
			fld.scale = int(reader.ReadInt32())
		}
		return fld, nil
	}
}

func unmarshallQueryIndex(reader BinaryReader) (QueryIndex, error) {
	idx := QueryIndex{}
	name, err := unmarshalString(reader, false)
	if err != nil {
		return idx, err
	}
	idx.name = name
	idx.idxType = reader.ReadInt8()
	idx.inlineSz = int(reader.ReadInt32())
	idx.fields, err = readCollection(reader, func(reader0 BinaryReader) (IndexField, error) {
		fldName, err0 := unmarshalString(reader0, false)
		if err0 != nil {
			return IndexField{}, err0
		}
		return IndexField{Name: fldName, Asc: reader0.ReadBool()}, nil
	})
	if err != nil {
		return idx, err
	}
	return idx, nil
}

func (config *CacheConfiguration) setStringProperty(reader BinaryReader, propCode propertyCode) error {
	var err error
	var val string
	if val, err = unmarshalString(reader, false); err != nil {
		return err
	}
	if len(val) > 0 {
		config.props[propCode] = val
	}
	return nil
}

func (config *CacheConfiguration) Copy(opts ...func(*CacheConfiguration)) CacheConfiguration {
	newConfig := CacheConfiguration{
		props: make(map[int16]interface{}),
	}
	for code, prop := range config.props {
		switch val := prop.(type) {
		case []QueryEntity:
			{
				szvVal := len(val)
				if szvVal > 0 {
					cpEntities := make([]QueryEntity, szvVal)
					for i := 0; i < szvVal; i++ {
						cpEntities[i] = val[i].Copy()
					}
					newConfig.props[code] = cpEntities
				}
			}
		case []CacheKeyConfig:
			{
				szVal := len(val)
				if szVal > 0 {
					cpKeyCfg := make([]CacheKeyConfig, szVal)
					copy(val, cpKeyCfg)
					newConfig.props[code] = cpKeyCfg
				}
			}
		default:
			newConfig.props[code] = val
		}
	}
	if len(opts) > 0 {
		for _, opt := range opts {
			opt(config)
		}
	}
	return newConfig
}

func (config *CacheConfiguration) Name() string {
	return getProperty(config, cacheNameProp, "")
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

func (config *CacheConfiguration) CacheGroupName() string {
	return getProperty(config, groupNameProp, "")
}

func (config *CacheConfiguration) MaxConcurrentAsyncOperations() int {
	return getProperty(config, maxAsyncOpsProp, 0)
}

func (config *CacheConfiguration) MaxQueryIteratorsCount() int {
	return getProperty(config, maxQueryIteratorsProp, 0)
}

func (config *CacheConfiguration) IsOnHeapCacheEnabled() bool {
	return getProperty(config, onHeapCacheEnabledProp, false)
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

func (config *CacheConfiguration) IsStatsEnabled() bool {
	return getProperty(config, statsEnabledProp, false)
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
		coll[i] = el
	}
	return coll, nil
}

func WithCacheName(name string) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		if len(name) > 0 {
			config.props[cacheNameProp] = name
		}
	}
}

func WithCacheGroupName(name string) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		if len(name) > 0 {
			config.props[groupNameProp] = name
		}
	}
}

func WithBackupsCount(cnt int) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[backupsProp] = cnt
	}
}

func WithWriteSynchronizationMode(mode CacheWriteSynchronizationMode) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[writeSyncModeProp] = mode
	}
}

func WithCopyOnRead(enabled bool) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[copyOnReadProp] = enabled
	}
}

func WithReadFromBackup(enabled bool) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[readFromBackupProp] = enabled
	}
}

func WithDataRegionName(dataRegionName string) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		if len(dataRegionName) > 0 {
			config.props[dataRegionNameProp] = dataRegionName
		}
	}
}

func WithOnHeapCacheEnabled(enabled bool) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[onHeapCacheEnabledProp] = enabled
	}
}

func WithCacheMode(mode CacheMode) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[cacheModeProp] = mode
	}
}

func WithCacheAtomicityMode(mode CacheAtomicityMode) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[cacheAtomicityModeProp] = mode
	}
}

func WithQueryParallelism(parallelism int) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[queryParallelismProp] = parallelism
	}
}

func WithQueryDetailsMetricsSize(size int) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[queryDetailsMetricSizeProp] = size
	}
}

func WithSqlSchema(schema string) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		if len(schema) > 0 {
			config.props[sqlSchemaProp] = schema
		}
	}
}

func WithSqlIndexMaxInlineSize(size int) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[sqlIndexMaxInlineSizeProp] = size
	}
}

func WithSqlEscapeAll(enabled bool) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[sqlEscapeAllProp] = enabled
	}
}

func WithRebalanceMode(mode CacheRebalanceMode) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[rebalanceModeProp] = mode
	}
}

func WithRebalanceOrder(order int) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[rebalanceOrderProp] = order
	}
}

func WithMaxConcurrentAsyncOperations(count int) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[maxAsyncOpsProp] = count
	}
}

func WithPartitionLossPolicy(policy PartitionLossPolicy) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[partitionLossPolicyProp] = policy
	}
}

func WithMaxQueryIteratorsCount(count int) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[maxQueryIteratorsProp] = count
	}
}

func WithEagerTtl(enabled bool) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[eagerTtlProp] = enabled
	}
}

func WithStatsEnabled(enabled bool) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[statsEnabledProp] = enabled
	}
}

func WithExpirePolicy(creation time.Duration, access time.Duration, update time.Duration) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		config.props[expirePolicyProp] = &expirePolicyImpl{creation: creation, access: access, update: update}
	}
}

func WithCacheKeyConfiguration(typeName string, affinityKeyField string) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		var keyConfigs []CacheKeyConfig
		val, found := config.props[cacheKeyConfigProp]
		if !found {
			keyConfigs = make([]CacheKeyConfig, 0)
		} else {
			keyConfigs = val.([]CacheKeyConfig)
		}
		if len(keyConfigs) > 0 {
			for _, keyConfig := range keyConfigs {
				if keyConfig.AffinityKeyFieldName() == affinityKeyField {
					return
				}
			}
		}
		keyConfigs = append(keyConfigs, &cacheKeyConfig{typeName: typeName, affKeyFldName: affinityKeyField})
		config.props[cacheKeyConfigProp] = keyConfigs
	}
}

func WithQueryEntity(keyType string, valueType string, opts ...func(*QueryEntity)) func(*CacheConfiguration) {
	return func(config *CacheConfiguration) {
		var queryEntities []QueryEntity
		val, found := config.props[queryEntitiesProp]
		if !found {
			queryEntities = make([]QueryEntity, 0)
		} else {
			queryEntities = val.([]QueryEntity)
		}
		if len(queryEntities) > 0 {
			for _, ent := range queryEntities {
				if valueType == ent.ValueType() {
					return
				}
			}
		}
		entity := QueryEntity{
			keyType: keyType,
			valType: valueType,
		}
		if len(opts) > 0 {
			for _, opt := range opts {
				opt(&entity)
			}
		}
		queryEntities = append(queryEntities, entity)
		config.props[queryEntitiesProp] = queryEntities
	}
}

func WithTableName(name string) func(*QueryEntity) {
	return func(entity *QueryEntity) {
		if len(name) > 0 {
			entity.tblName = name
		}
	}
}

func WithKeyFieldName(name string) func(*QueryEntity) {
	return func(entity *QueryEntity) {
		if len(name) > 0 {
			entity.keyFldName = name
		}
	}
}

func WithValueFieldName(name string) func(*QueryEntity) {
	return func(entity *QueryEntity) {
		if len(name) > 0 {
			entity.valFldName = name
		}
	}
}

func WithQueryField(name string, typeName string, opts ...func(field *QueryField)) func(entity *QueryEntity) {
	return func(entity *QueryEntity) {
		if entity.fields == nil {
			entity.fields = make([]QueryField, 0)
		}
		field := QueryField{
			name:      name,
			typeName:  typeName,
			isKey:     false,
			isNotNull: false,
			precision: -1,
			scale:     -1,
			dfltVal:   nil,
		}
		if len(opts) > 0 {
			for _, opt := range opts {
				opt(&field)
			}
		}
		entity.fields = append(entity.fields, field)
	}
}

func WithNotNull() func(field *QueryField) {
	return func(field *QueryField) {
		field.isNotNull = true
	}
}

func WithKey() func(field *QueryField) {
	return func(field *QueryField) {
		field.isKey = true
	}
}

func WithDefaultValue(val interface{}) func(field *QueryField) {
	return func(field *QueryField) {
		field.dfltVal = val
	}
}

func WithScale(scale int) func(field *QueryField) {
	return func(field *QueryField) {
		field.scale = scale
	}
}

func WithPrecision(precision int) func(field *QueryField) {
	return func(field *QueryField) {
		field.precision = precision
	}
}

func WithFieldAlias(origName string, alias string) func(entity *QueryEntity) {
	return func(entity *QueryEntity) {
		if entity.aliases == nil {
			entity.aliases = make(map[string]string)
		}
		entity.aliases[origName] = alias
	}
}

func WithIndex(name string, opts ...func(index *QueryIndex)) func(entity *QueryEntity) {
	return func(entity *QueryEntity) {
		if entity.indexes == nil {
			entity.indexes = make([]QueryIndex, 0)
		}
		for _, idx := range entity.indexes {
			if idx.Name() == name {
				return
			}
		}
		idx := QueryIndex{name: name, inlineSz: -1, idxType: Sorted, fields: make([]IndexField, 0)}
		if len(opts) > 0 {
			for _, opt := range opts {
				opt(&idx)
			}
		}
		entity.indexes = append(entity.indexes, idx)
	}
}

func WithIndexType(indexType IndexType) func(index *QueryIndex) {
	return func(index *QueryIndex) {
		index.idxType = indexType
	}
}

func WithInlineSize(inlineSize int) func(index *QueryIndex) {
	return func(index *QueryIndex) {
		index.inlineSz = inlineSize
	}
}

func WithIndexField(field IndexField) func(index *QueryIndex) {
	return func(index *QueryIndex) {
		for _, f := range index.fields {
			if f.Name == field.Name {
				return
			}
		}
		index.fields = append(index.fields, field)
	}
}
