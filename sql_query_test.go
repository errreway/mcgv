package ignite

import (
	"context"
	"github.com/cockroachdb/apd/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	testing2 "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"strconv"
	"testing"
	"time"
)

type SqlQueryTestSuite struct {
	testing2.IgniteTestSuite
	client *Client
}

var basicQueryFieldsConfiguration = CreateCacheConfiguration(
	cacheName,
	WithSqlSchema("TEST_SCHEMA"),
	WithQueryEntity("java.lang.Integer", "java.lang.String",
		WithTableName("TEST_TABLE"),
		WithKeyFieldName("int_field"),
		WithValueFieldName("string_field"),
		WithQueryField("int_field", "java.lang.Integer"),
		WithQueryField("string_field", "java.lang.String"),
	),
)

var alTypesQueryFieldsConfiguration = CreateCacheConfiguration(
	cacheName,
	WithSqlSchema("TEST_SCHEMA"),
	WithQueryEntity("KEY_TYPE", "VALUE_TYPE",
		WithTableName("TEST_TABLE"),
		WithQueryField("varchar_key_field", "java.lang.String", WithKey()),
		WithQueryField("uuid_key_field", "java.util.UUID", WithKey()),
		WithQueryField("bool_value_field", "java.lang.Boolean"),
		WithQueryField("tinyint_value_field", "java.lang.Byte"),
		WithQueryField("smallint_value_field", "java.lang.Short"),
		WithQueryField("int_value_field", "java.lang.Integer"),
		WithQueryField("bigint_value_field", "java.lang.Long"),
		WithQueryField("real_value_field", "java.lang.Float"),
		WithQueryField("double_value_field", "java.lang.Double"),
		WithQueryField("decimal_value_field", "java.math.BigDecimal"),
		WithQueryField("time_value_field", "java.sql.Time"),
		WithQueryField("timestamp_value_field", "java.sql.Timestamp"),
		WithQueryField("date_value_field", "java.util.Date"),
		WithQueryField("varchar_value_field", "java.lang.String"),
		WithQueryField("uuid_value_field", "java.util.UUID"),
		WithQueryField("binary_value_field", "[B"),
		WithQueryField("binary_object_value_field", "BINARY_OBJECT_FIELD_TYPE"),
		WithQueryField("null_value_field", "BINARY_OBJECT_FIELD_TYPE"),
	),
)

var buildKeyBinaryObject = func(suite *SqlQueryTestSuite, id uuid.UUID) BinaryObject {
	binObj, err := suite.client.CreateBinaryObject(context.Background(), "KEY_TYPE", WithField("varchar_key_field", "0"), WithField("uuid_key_field", id))
	require.NoError(suite.T(), err)

	return binObj
}

var buildAlTypesValueBinaryObject = func(suite *SqlQueryTestSuite, id uuid.UUID, timestamp time.Time) BinaryObject {
	binObjField, err := suite.client.CreateBinaryObject(context.Background(), "BINARY_OBJECT_FIELD_TYPE", WithField("data", "0"))
	require.NoError(suite.T(), err)

	binObj, err := suite.client.CreateBinaryObject(context.Background(), "VALUE_TYPE",
		WithField("bool_value_field", true),
		WithField("tinyint_value_field", int8(0)),
		WithField("smallint_value_field", int16(1)),
		WithField("int_value_field", int32(2)),
		WithField("bigint_value_field", int64(3)),
		WithField("real_value_field", float32(0.4)),
		WithField("double_value_field", float64(0.5)),
		WithField("decimal_value_field", apd.New(6, -1)),
		WithField("time_value_field", NewTime(timestamp)),
		WithField("timestamp_value_field", timestamp),
		WithField("date_value_field", NewDate(timestamp)),
		WithField("varchar_value_field", "0"),
		WithField("uuid_value_field", id),
		WithField("binary_value_field", []byte{1, 0}),
		WithField("binary_object_value_field", binObjField),
		WithField("null_value_field", nil),
	)
	require.NoError(suite.T(), err)

	return binObj
}

func TestSqlQueryTestSuite(t *testing.T) {
	suite.Run(t, new(SqlQueryTestSuite))
}

func (suite *SqlQueryTestSuite) SetupSuite() {
	_, err := suite.StartIgnite()
	if err != nil {
		suite.T().Fatal("Failed to start ignite instance", err)
	}
	suite.client, err = StartTestClient(context.Background(), WithAddresses(defaultAddress))
	if err != nil {
		suite.T().Fatal("failed to start client", err)
	}
}

func (suite *SqlQueryTestSuite) TearDownSuite() {
	suite.KillAllGrids()
	if suite.client != nil {
		_ = suite.client.Close(context.Background())
		suite.client = nil
	}
}

func (suite *SqlQueryTestSuite) TearDownTest() {
	cacheNames, err := suite.client.CacheNames(context.Background())
	require.NoError(suite.T(), err)

	for _, cacheName := range cacheNames {
		err := suite.client.DestroyCache(context.Background(), cacheName)
		if err != nil {
			suite.T().Fatal("failed to destroy caches", err)
		}
	}
}

func (suite *SqlQueryTestSuite) TestPutSelect() {
	suite.runWithParameters(runPutSelectTest)
}

func runPutSelectTest(suite *SqlQueryTestSuite, createTable func() *Cache) {
	id := uuid.New()
	timestamp := time.Now().Truncate(0)
	keyBinObj := buildKeyBinaryObject(suite, id)
	valBinObj := buildAlTypesValueBinaryObject(suite, id, timestamp)

	cache := createTable()

	err := cache.Put(context.Background(), keyBinObj, valBinObj)
	require.NoError(suite.T(), err)

	checkSelect(suite, keyBinObj, valBinObj, id, timestamp)
}

func (suite *SqlQueryTestSuite) TestInsertGetSelect() {
	suite.runWithParameters(runInsertGetSelectTest)
}

func runInsertGetSelectTest(suite *SqlQueryTestSuite, createTable func() *Cache) {
	id := uuid.New()
	timestamp := time.Now().Truncate(0)
	binObjField, err := suite.client.CreateBinaryObject(context.Background(), "BINARY_OBJECT_FIELD_TYPE", WithField("data", "0"))
	require.NoError(suite.T(), err)

	cache := createTable()

	cursor, err := suite.client.SqlQuery(context.Background(),
		"INSERT INTO TEST_SCHEMA.TEST_TABLE("+
			"VARCHAR_KEY_FIELD,"+
			" UUID_KEY_FIELD,"+
			" BOOL_VALUE_FIELD,"+
			" TINYINT_VALUE_FIELD,"+
			" SMALLINT_VALUE_FIELD,"+
			" INT_VALUE_FIELD,"+
			" BIGINT_VALUE_FIELD,"+
			" REAL_VALUE_FIELD,"+
			" DOUBLE_VALUE_FIELD,"+
			" DECIMAL_VALUE_FIELD,"+
			" TIME_VALUE_FIELD,"+
			" TIMESTAMP_VALUE_FIELD,"+
			" DATE_VALUE_FIELD,"+
			" VARCHAR_VALUE_FIELD,"+
			" UUID_VALUE_FIELD,"+
			" BINARY_VALUE_FIELD,"+
			" BINARY_OBJECT_VALUE_FIELD,"+
			" NULL_VALUE_FIELD"+
			") VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		WithSqlQueryArguments(
			"0",
			id,
			true,
			int8(0),
			int16(1),
			int32(2),
			int64(3),
			float32(0.4),
			float64(0.5),
			apd.New(6, -1),
			NewTime(timestamp),
			timestamp,
			NewDate(timestamp),
			"0",
			id,
			[]byte{1, 0},
			binObjField,
			nil,
		),
	)
	require.NoError(suite.T(), err)
	defer func() {
		_ = cursor.Close()
	}()

	keyBinObj := buildKeyBinaryObject(suite, id)
	val, err := cache.Get(context.Background(), keyBinObj)
	require.NoError(suite.T(), err)

	binObj := val.(BinaryObject)

	boolValueField, err := binObj.Field(context.Background(), "bool_value_field")
	require.NoError(suite.T(), err)
	tinyintValueField, err := binObj.Field(context.Background(), "tinyint_value_field")
	require.NoError(suite.T(), err)
	smallintValueField, err := binObj.Field(context.Background(), "smallint_value_field")
	require.NoError(suite.T(), err)
	intValueField, err := binObj.Field(context.Background(), "int_value_field")
	require.NoError(suite.T(), err)
	bigintValueField, err := binObj.Field(context.Background(), "bigint_value_field")
	require.NoError(suite.T(), err)
	realValueField, err := binObj.Field(context.Background(), "real_value_field")
	require.NoError(suite.T(), err)
	doubleValueField, err := binObj.Field(context.Background(), "double_value_field")
	require.NoError(suite.T(), err)
	decimalValueField, err := binObj.Field(context.Background(), "decimal_value_field")
	require.NoError(suite.T(), err)
	timeValueField, err := binObj.Field(context.Background(), "time_value_field")
	require.NoError(suite.T(), err)
	timestampValueField, err := binObj.Field(context.Background(), "timestamp_value_field")
	require.NoError(suite.T(), err)
	dateValueField, err := binObj.Field(context.Background(), "date_value_field")
	require.NoError(suite.T(), err)
	varcharValueField, err := binObj.Field(context.Background(), "varchar_value_field")
	require.NoError(suite.T(), err)
	uuidValueField, err := binObj.Field(context.Background(), "uuid_value_field")
	require.NoError(suite.T(), err)
	binaryValueField, err := binObj.Field(context.Background(), "binary_value_field")
	require.NoError(suite.T(), err)
	binaryObjectValueField, err := binObj.Field(context.Background(), "binary_object_value_field")
	require.NoError(suite.T(), err)
	nullValueField, err := binObj.Field(context.Background(), "null_value_field")
	require.NoError(suite.T(), err)

	require.Equal(suite.T(), true, boolValueField)
	require.Equal(suite.T(), int8(0), tinyintValueField)
	require.Equal(suite.T(), int16(1), smallintValueField)
	require.Equal(suite.T(), int32(2), intValueField)
	require.Equal(suite.T(), int64(3), bigintValueField)
	require.Equal(suite.T(), float32(0.4), realValueField)
	require.Equal(suite.T(), float64(0.5), doubleValueField)
	require.Equal(suite.T(), *apd.New(6, -1), *decimalValueField.(*apd.Decimal))
	require.True(suite.T(), NewTime(timestamp).Time().Equal(timeValueField.(Time).Time()))
	require.Equal(suite.T(), timestamp, timestampValueField)
	require.True(suite.T(), NewDate(timestamp).Time().Equal(dateValueField.(Date).Time()))
	require.Equal(suite.T(), "0", varcharValueField)
	require.Equal(suite.T(), id, uuidValueField)
	require.Equal(suite.T(), []byte{1, 0}, binaryValueField)
	require.Nil(suite.T(), nullValueField)

	innerFieldData, err := binaryObjectValueField.(BinaryObject).Field(context.Background(), "data")
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "0", innerFieldData)

	valBinObj := buildAlTypesValueBinaryObject(suite, id, timestamp)
	checkSelect(suite, keyBinObj, valBinObj, id, timestamp)
}

func (suite *SqlQueryTestSuite) TestCursorColumns() {
	_, err := suite.client.GetOrCreateCacheWithConfiguration(context.Background(), basicQueryFieldsConfiguration)
	require.NoError(suite.T(), err)

	insertCursor, err := suite.client.SqlQuery(context.Background(), "INSERT INTO TEST_SCHEMA.TEST_TABLE(INT_FIELD, STRING_FIELD) VALUES(?, ?)",
		WithSqlQueryArguments(0, "0"),
	)
	require.NoError(suite.T(), err)
	defer func() {
		_ = insertCursor.Close()
	}()

	require.Equal(suite.T(), []string{"UPDATED"}, insertCursor.Columns())

	cursor, err := suite.client.SqlQuery(context.Background(), "SELECT * FROM TEST_SCHEMA.TEST_TABLE")
	require.NoError(suite.T(), err)
	defer func() {
		_ = cursor.Close()
	}()

	require.Equal(suite.T(), []string{"INT_FIELD", "STRING_FIELD"}, cursor.Columns())
}

func (suite *SqlQueryTestSuite) TestQueryExecutionParameters() {
	cache, err := suite.client.GetOrCreateCacheWithConfiguration(context.Background(), basicQueryFieldsConfiguration)
	require.NoError(suite.T(), err)

	for i := 0; i < 2; i++ {
		_, err = suite.client.SqlQuery(context.Background(), "INSERT INTO TEST_SCHEMA.TEST_TABLE(INT_FIELD, STRING_FIELD) VALUES(?, ?)",
			WithSqlQueryArguments(i, strconv.Itoa(i)))

		require.NoError(suite.T(), err)
	}

	val, err := cache.Get(context.Background(), int32(0))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "0", val)

	val, err = cache.Get(context.Background(), int32(1))
	require.NoError(suite.T(), err)
	require.Equal(suite.T(), "1", val)

	selectCursor, err := suite.client.SqlQuery(context.Background(), "SELECT * FROM TEST_TABLE",
		WithSqlQueryTimeout(100*time.Second),
		WithSqlQueryLocal(),
		WithSqlQueryColocated(),
		WithSqlQueryJoinOrderEnforced(),
		WithSqlQueryPartitions(0, 1, 2),
		WithSqlQueryPageSize(1),
		WithSqlQueryUpdateBatchSize(1),
		WithSqlQuerySchema("TEST_SCHEMA"),
	)

	require.NoError(suite.T(), err)
	defer func() {
		_ = selectCursor.Close()
	}()

	selectedValues := make(map[int32]string)

	// Fetch only the first page of the result to keep the cursor open. While the query cursor is open, query execution
	// parameters can be obtained through the SQL_QUERIES system view.
	selectCursor.Next()
	require.NoError(suite.T(), selectCursor.Err())
	scanTo(suite, selectCursor, selectedValues)

	sysViewCursor, err := suite.client.SqlQuery(context.Background(), "SELECT * FROM SYS.SQL_QUERIES")
	require.NoError(suite.T(), err)
	defer func() {
		_ = sysViewCursor.Close()
	}()

	isQueryInfoFound := false
	for sysViewCursor.Next() {
		var queryId interface{}
		var sql interface{}
		var originNodeId interface{}
		var startTime interface{}
		var duration interface{}
		var initiatorNodeId interface{}
		var isLocal interface{}
		var schemaName interface{}
		var subjectId interface{}
		err := sysViewCursor.Scan(&queryId, &sql, &originNodeId, &startTime, &duration, &initiatorNodeId, &isLocal, &schemaName, &subjectId)
		require.NoError(suite.T(), err)
		if sql == "SELECT * FROM TEST_TABLE" {
			require.Equal(suite.T(), true, isLocal)
			require.Equal(suite.T(), "TEST_SCHEMA", schemaName)
			isQueryInfoFound = true
		}
	}
	require.True(suite.T(), isQueryInfoFound)

	selectCursor.Next()
	require.NoError(suite.T(), selectCursor.Err())
	scanTo(suite, selectCursor, selectedValues)

	require.Equal(suite.T(), "0", selectedValues[0])
	require.Equal(suite.T(), "1", selectedValues[1])
	require.False(suite.T(), selectCursor.Next())
}

func checkSelect(suite *SqlQueryTestSuite, keyBinObj BinaryObject, valBinObj BinaryObject, id uuid.UUID, timestamp time.Time) {
	cursor, err := suite.client.SqlQuery(context.Background(), "SELECT "+
		"_KEY"+
		", _VAL"+
		", VARCHAR_KEY_FIELD"+
		", UUID_KEY_FIELD"+
		", BOOL_VALUE_FIELD"+
		", TINYINT_VALUE_FIELD"+
		", SMALLINT_VALUE_FIELD"+
		", INT_VALUE_FIELD"+
		", BIGINT_VALUE_FIELD"+
		", REAL_VALUE_FIELD"+
		", DOUBLE_VALUE_FIELD"+
		", DECIMAL_VALUE_FIELD"+
		", TIME_VALUE_FIELD"+
		", TIMESTAMP_VALUE_FIELD"+
		", DATE_VALUE_FIELD"+
		", VARCHAR_VALUE_FIELD"+
		", UUID_VALUE_FIELD"+
		", BINARY_VALUE_FIELD"+
		", BINARY_OBJECT_VALUE_FIELD"+
		", NULL_VALUE_FIELD"+
		" FROM TEST_SCHEMA.TEST_TABLE")
	require.NoError(suite.T(), err)
	defer func() {
		_ = cursor.Close()
	}()

	var checkedRows int

	for cursor.Next() {
		var key BinaryObject
		var val BinaryObject
		var stringKeyField string
		var uuidKeyField uuid.UUID
		var boolValueField bool
		var tinyintValueField int8
		var smallintValueField int16
		var intValueField int32
		var bigintValueField int64
		var realValueField float32
		var doubleValueField float64
		var decimalValueField *apd.Decimal
		var timeValueField Time
		var timestampValueField time.Time
		var dateValueField time.Time // H2 engine handles Date as Timestamp
		var varcharValueField string
		var uuidValueField uuid.UUID
		var binaryValueField []byte
		var binaryObjectValueField BinaryObject
		var nullValueField BinaryObject
		err := cursor.Scan(
			&key,
			&val,
			&stringKeyField,
			&uuidKeyField,
			&boolValueField,
			&tinyintValueField,
			&smallintValueField,
			&intValueField,
			&bigintValueField,
			&realValueField,
			&doubleValueField,
			&decimalValueField,
			&timeValueField,
			&timestampValueField,
			&dateValueField,
			&varcharValueField,
			&uuidValueField,
			&binaryValueField,
			&binaryObjectValueField,
			&nullValueField,
		)
		require.NoError(suite.T(), err)
		RequireBinaryObjectsEqual(suite.T(), keyBinObj, key)
		RequireBinaryObjectsEqual(suite.T(), valBinObj, val)
		require.Equal(suite.T(), "0", stringKeyField)
		require.Equal(suite.T(), id, uuidKeyField)
		require.Equal(suite.T(), true, boolValueField)
		require.Equal(suite.T(), int8(0), tinyintValueField)
		require.Equal(suite.T(), int16(1), smallintValueField)
		require.Equal(suite.T(), int32(2), intValueField)
		require.Equal(suite.T(), int64(3), bigintValueField)
		require.Equal(suite.T(), float32(0.4), realValueField)
		require.Equal(suite.T(), float64(0.5), doubleValueField)
		require.Equal(suite.T(), *apd.New(6, -1), *decimalValueField)
		require.True(suite.T(), NewTime(timestamp).Time().Equal(timeValueField.Time()))
		require.Equal(suite.T(), timestamp, timestampValueField)
		require.True(suite.T(), NewDate(timestamp).Time().Equal(dateValueField)) // H2 engine handles Date as Timestamp
		require.Equal(suite.T(), "0", varcharValueField)
		require.Equal(suite.T(), id, uuidValueField)
		require.Equal(suite.T(), []byte{1, 0}, binaryValueField)
		require.Nil(suite.T(), nullValueField)

		innerFieldData, err := binaryObjectValueField.Field(context.Background(), "data")
		require.NoError(suite.T(), err)
		require.Equal(suite.T(), "0", innerFieldData)
		checkedRows++
	}

	require.Equal(suite.T(), 1, checkedRows)
}

func (suite *SqlQueryTestSuite) runWithParameters(runTest func(*SqlQueryTestSuite, func() *Cache)) {
	fixtures := []struct {
		name         string
		tableFactory func() *Cache
	}{
		{
			"WithTableCreatedUsingCacheApi",
			func() *Cache {
				cache, err := suite.client.GetOrCreateCacheWithConfiguration(context.Background(), alTypesQueryFieldsConfiguration)
				require.NoError(suite.T(), err)

				return cache
			},
		},
		{
			"WithTableCreatedUsingSqlQuery",
			func() *Cache {
				cursor, err := suite.client.SqlQuery(context.Background(), "CREATE TABLE TEST_SCHEMA.TEST_TABLE ("+
					"VARCHAR_KEY_FIELD VARCHAR"+
					", UUID_KEY_FIELD UUID"+
					", BOOL_VALUE_FIELD BOOL"+
					", TINYINT_VALUE_FIELD TINYINT"+
					", SMALLINT_VALUE_FIELD SMALLINT"+
					", INT_VALUE_FIELD INT"+
					", BIGINT_VALUE_FIELD BIGINT"+
					", REAL_VALUE_FIELD REAL"+
					", DOUBLE_VALUE_FIELD DOUBLE"+
					", DECIMAL_VALUE_FIELD DECIMAL"+
					", TIME_VALUE_FIELD TIME"+
					", TIMESTAMP_VALUE_FIELD TIMESTAMP"+
					", DATE_VALUE_FIELD TIMESTAMP"+
					", VARCHAR_VALUE_FIELD VARCHAR"+
					", UUID_VALUE_FIELD UUID"+
					", BINARY_VALUE_FIELD BINARY"+
					", BINARY_OBJECT_VALUE_FIELD OTHER"+
					", NULL_VALUE_FIELD OTHER"+
					", PRIMARY KEY(VARCHAR_KEY_FIELD, UUID_KEY_FIELD))"+
					" WITH \"KEY_TYPE=KEY_TYPE, VALUE_TYPE=VALUE_TYPE, CACHE_NAME="+cacheName+"\"")
				require.NoError(suite.T(), err)
				defer func() {
					_ = cursor.Close()
				}()

				cache, err := suite.client.GetOrCreateCache(context.Background(), cacheName)
				require.NoError(suite.T(), err)

				return cache
			},
		},
	}

	for _, fixture := range fixtures {
		suite.T().Run(fixture.name, func(t *testing.T) {
			runTest(suite, fixture.tableFactory)
			suite.TearDownTest()
		})
	}
}

func scanTo(suite *SqlQueryTestSuite, cursor Cursor, dest map[int32]string) {
	var intField int32
	var stringField string
	err := cursor.Scan(&intField, &stringField)
	require.NoError(suite.T(), err)
	dest[intField] = stringField
}
