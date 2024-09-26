//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"gitverse.ru/sbertech/ignite-go-client/examples"
	"strings"
)

func main() {
	cli, err := ignite.Start(context.Background(), ignite.WithAddresses("localhost"))
	if err != nil {
		panic(fmt.Errorf("failed to start client: %w", err))
	}
	defer func() {
		_ = cli.Close(context.Background())
	}()

	// Creates cache and tells Ignite that cache key and cache value fields should be accessible through SQL queries.
	cache, err := cli.GetOrCreateCacheWithConfiguration(context.Background(), ignite.CreateCacheConfiguration(
		"person-cache",
		ignite.WithSqlSchema("EXAMPLE"), // SQL Schema name.
		ignite.WithQueryEntity("PERSON_KEY", "PERSON", // KeyType and ValueType is used to construct cache entry and write it in the cache during DML SQL queries.
			ignite.WithTableName("PERSON"),                                  // The table name to use in SQL queries.
			ignite.WithQueryField("id", "java.lang.Long", ignite.WithKey()), // WithKey indicates that field is part of composite key.
			ignite.WithQueryField("company", "java.lang.String", ignite.WithKey()),
			ignite.WithQueryField("name", "java.lang.String"), // The name of the cache value field that will be available as a table column. Note, by default the corresponding column names will be in uppercase.
			ignite.WithQueryField("age", "java.lang.Integer"),
			ignite.WithIndex("NAME_IDX", // Tells Ignite to create index in table using specified fields.
				ignite.WithIndexType(ignite.Sorted),
				ignite.WithInlineSize(50),
				ignite.WithIndexField(ignite.IndexField{Name: "name", Asc: true})),
		),
	))
	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}
	fmt.Println(">>> Created table `PERSON_TABLE` [columns=[`ID` LONG, `COMPANY` VARCHAR, `NAME` VARCHAR, `AGE` INT], cacheName=`person-cache`, cacheKeyType=`PERSON_KEY`, cacheValueType=`PERSON`]")

	firstKeyBinObj, err := cli.CreateBinaryObject(context.Background(), "PERSON_KEY", ignite.WithField("id", int64(0)), ignite.WithField("company", "RCM"))
	if err != nil {
		panic(fmt.Errorf("failed to create binary object: %w", err))
	}
	valueBinObj, err := cli.CreateBinaryObject(context.Background(), "PERSON", ignite.WithField("name", "Cuno"), ignite.WithField("age", int32(12)))
	if err != nil {
		panic(fmt.Errorf("failed to create binary object: %w", err))
	}

	err = cache.Put(context.Background(), firstKeyBinObj, valueBinObj)
	if err != nil {
		panic(fmt.Errorf("failed to write entry to the cache: %w", err))
	}
	fmt.Printf(">>> Entry [key=%s, val=%s] was written to cache `%s`\n", examples.ToString(firstKeyBinObj), examples.ToString(valueBinObj), cache.Name())

	// We've just written a record to the cache. Now it's time to access it via SQL.
	fmt.Println(">>> Executing SQL Query [sql=`SELECT * FROM PERSON`, schema=`EXAMPLE`]")
	selectCursor, err := cli.SqlQuery(context.Background(), "SELECT * FROM PERSON", ignite.WithSqlQuerySchema("EXAMPLE"))
	if err != nil {
		panic(fmt.Errorf("failed to execute SQL query: %w", err))
	}
	defer func() {
		_ = selectCursor.Close()
	}()

	fmt.Printf(">>> SQL Query cursor returned column names [%s]\n", strings.Join(selectCursor.Columns(), ", "))

	for selectCursor.Next() {
		var id int64
		var company string
		var name string
		var age int32
		err = selectCursor.Scan(&id, &company, &name, &age)
		if err != nil {
			panic(fmt.Errorf("failed to scan SQL query cursor: %w", err))
		}
		fmt.Printf(">>> SQL Query cursor returned row [id=`%d`, company=`%s`, name=`%s`, age=`%d`]\n", id, company, name, age)
	}

	fmt.Println(">>> Executing SQL Query [sql=`INSERT INTO PERSON(ID, COMPANY, NAME, AGE) VALUES(?, ?, ?, ?)`, schema=`EXAMPLE`, arguments=[`1`, `RCM`, `Kim`, `38`]]")
	insertCursor, err := cli.SqlQuery(context.Background(), "INSERT INTO PERSON(ID, COMPANY, NAME, AGE) VALUES(?, ?, ?, ?)",
		ignite.WithSqlQuerySchema("EXAMPLE"),
		ignite.WithSqlQueryArguments(int64(1), "RCM", "Kim", int32(38)),
	)
	if err != nil {
		panic(fmt.Errorf("failed to execute SQL query: %w", err))
	}
	defer func() {
		_ = insertCursor.Close()
	}()

	secondKeyBinObj, err := cli.CreateBinaryObject(context.Background(), "PERSON_KEY", ignite.WithField("id", int64(1)), ignite.WithField("company", "RCM"))
	if err != nil {
		panic(fmt.Errorf("failed to create binary object: %w", err))
	}

	// The same as before, but now we are accessing cache entry that was created after INSERT SQL query.
	fmt.Printf(">>> Requesting value for key == %v from cache `%s`\n", examples.ToString(secondKeyBinObj), cache.Name())
	val, err := cache.Get(context.Background(), secondKeyBinObj)
	if err != nil {
		panic(fmt.Errorf("failed to get value from the cache: %w", err))
	}

	fmt.Printf(">>> Value == %s\n", examples.ToString(val.(ignite.BinaryObject)))
}
