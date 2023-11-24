package serdes

const (
	NAME                             = 0
	CACHE_MODE                       = 1
	ATOMICITY_MODE                   = 2
	BACKUPS                          = 3
	WRITE_SYNC_MODE                  = 4
	READ_FROM_BACKUP                 = 6
	EAGER_TTL                        = 405
	GROUP_NAME                       = 400
	DEFAULT_LOCK_TIMEOUT             = 402
	PART_LOSS_POLICY                 = 404
	REBALANCE_BATCH_SIZE             = 303
	REBALANCE_BATCHES_PREFETCH_COUNT = 304
	REBALANCE_DELAY                  = 301
	REBALANCE_MODE                   = 300
	REBALANCE_ORDER                  = 305
	REBALANCE_THROTTLE               = 306
	REBALANCE_TIMEOUT                = 302
	COPY_ON_READ                     = 5
	DATA_REGION_NAME                 = 100
	STATS_ENABLED                    = 406
	MAX_ASYNC_OPS                    = 403
	MAX_QUERY_ITERATORS              = 206
	ONHEAP_CACHE_ENABLED             = 101
	QUERY_METRIC_SIZE                = 202
	QUERY_PARALLELISM                = 201
	SQL_ESCAPE_ALL                   = 205
	SQL_IDX_MAX_INLINE_SIZE          = 204
	SQL_SCHEMA                       = 203
	KEY_CONFIGS                      = 401
	QUERY_ENTITIES                   = 200
)

type cacheConfigurationWriter struct {
	propCnt int16

	buf *IgniteBuffer
}

func (writer *cacheConfigurationWriter) WriteField(cfgItem int16, cfgWriter func(buf *IgniteBuffer)) {
	writer.buf.WriteInt16(cfgItem)
	cfgWriter(writer.buf)
	writer.propCnt += 1
}

func (buf *IgniteBuffer) WriteCacheConfiguration(req CacheConfiguration) {
	origPos := buf.CurrentPosition()

	writer := cacheConfigurationWriter{0, buf}

	buf.WriteInt32(0) // configuration length is to be assigned in the end
	buf.WriteInt16(0) // properties count is to be assigned in the end

	writer.WriteField(NAME, func(buf *IgniteBuffer) { buf.WriteString(req.Name) })
	writer.WriteField(CACHE_MODE, func(buf *IgniteBuffer) { buf.WriteInt32(req.CacheMode) })
	writer.WriteField(ATOMICITY_MODE, func(buf *IgniteBuffer) { buf.WriteInt32(req.AtomicityMode) })
	writer.WriteField(BACKUPS, func(buf *IgniteBuffer) { buf.WriteInt32(req.Backups) })
	writer.WriteField(WRITE_SYNC_MODE, func(buf *IgniteBuffer) { buf.WriteInt32(req.WriteSynchronizationMode) })
	writer.WriteField(READ_FROM_BACKUP, func(buf *IgniteBuffer) { buf.WriteBool(req.ReadFromBackup) })
	writer.WriteField(EAGER_TTL, func(buf *IgniteBuffer) { buf.WriteBool(req.EagerTTL) })
	writer.WriteField(GROUP_NAME, func(buf *IgniteBuffer) { buf.WriteString(req.GroupName) })
	writer.WriteField(DEFAULT_LOCK_TIMEOUT, func(buf *IgniteBuffer) { buf.WriteInt64(req.DefaultLockTimeout) })
	writer.WriteField(PART_LOSS_POLICY, func(buf *IgniteBuffer) { buf.WriteInt32(req.PartitionLossPolicy) })
	writer.WriteField(REBALANCE_BATCH_SIZE, func(buf *IgniteBuffer) { buf.WriteInt32(req.RebalanceBatchSize) })
	writer.WriteField(REBALANCE_BATCHES_PREFETCH_COUNT, func(buf *IgniteBuffer) { buf.WriteInt64(req.RebalanceBatchesPrefetchCount) })
	writer.WriteField(REBALANCE_DELAY, func(buf *IgniteBuffer) { buf.WriteInt64(req.RebalanceDelay) })
	writer.WriteField(REBALANCE_MODE, func(buf *IgniteBuffer) { buf.WriteInt32(req.RebalanceMode) })
	writer.WriteField(REBALANCE_ORDER, func(buf *IgniteBuffer) { buf.WriteInt32(req.RebalanceOrder) })
	writer.WriteField(REBALANCE_THROTTLE, func(buf *IgniteBuffer) { buf.WriteInt64(req.RebalanceThrottle) })
	writer.WriteField(REBALANCE_TIMEOUT, func(buf *IgniteBuffer) { buf.WriteInt64(req.RebalanceTimeout) })
	writer.WriteField(COPY_ON_READ, func(buf *IgniteBuffer) { buf.WriteBool(req.CopyOnRead) })
	writer.WriteField(DATA_REGION_NAME, func(buf *IgniteBuffer) { buf.WriteString(req.DataRegionName) })
	writer.WriteField(STATS_ENABLED, func(buf *IgniteBuffer) { buf.WriteBool(req.StatisticsEnabled) })
	writer.WriteField(MAX_ASYNC_OPS, func(buf *IgniteBuffer) { buf.WriteInt32(req.MaxConcurrentAsyncOperations) })
	writer.WriteField(MAX_QUERY_ITERATORS, func(buf *IgniteBuffer) { buf.WriteInt32(req.MaxQueryIterators) })
	writer.WriteField(ONHEAP_CACHE_ENABLED, func(buf *IgniteBuffer) { buf.WriteBool(req.IsOnheapCacheEnabled) })
	writer.WriteField(QUERY_METRIC_SIZE, func(buf *IgniteBuffer) { buf.WriteInt32(req.QueryDetailMetricsSize) })
	writer.WriteField(QUERY_PARALLELISM, func(buf *IgniteBuffer) { buf.WriteInt32(req.QueryParallelism) })
	writer.WriteField(SQL_ESCAPE_ALL, func(buf *IgniteBuffer) { buf.WriteBool(req.SqlEscapeAll) })
	writer.WriteField(SQL_IDX_MAX_INLINE_SIZE, func(buf *IgniteBuffer) { buf.WriteInt32(req.SqlIndexInlineMaxSize) })
	writer.WriteField(SQL_SCHEMA, func(buf *IgniteBuffer) { buf.WriteString(req.SqlSchema) })
	writer.WriteField(KEY_CONFIGS, func(buf *IgniteBuffer) { buf.WriteCacheKeyConfigurationArrayWithoutType(req.CacheKeyConfigurations) })
	writer.WriteField(QUERY_ENTITIES, func(buf *IgniteBuffer) { buf.WriteQueryEntityArrayWithoutType(req.QueryEntities) })

	curPos := buf.CurrentPosition()
	l := int32(curPos - origPos)

	buf.Position(origPos)
	buf.WriteInt32(l)
	buf.WriteInt16(writer.propCnt)
	buf.Position(curPos)
}
