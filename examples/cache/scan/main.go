//go:build testing

package main

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"gitverse.ru/sbertech/ignite-go-client/examples"
	"sort"
	"strings"
)

// Note - some of the following examples refer to Java classes that must be available in Ignite server node class path
// (e.g. Scan Query Filter implementation). Maven project with all Java sources can be found in
// `internal/testing/java/` (path is relative to the root directory of the current project).
func main() {
	cli, err := ignite.Start(context.Background(), ignite.WithAddresses("localhost"))
	if err != nil {
		panic(fmt.Errorf("failed to start client: %w", err))
	}

	defer func() {
		_ = cli.Close(context.Background())
	}()

	cache, err := cli.GetOrCreateCache(context.Background(), "example-cache")

	if err != nil {
		panic(fmt.Errorf("failed to get cache: %w", err))
	}

	populateCache(cli, cache)

	// Queries all cache entries without filtering. The Page Size option causes the client to request Scan query result
	// rows in batches of corresponding size rather than all at once.
	fmt.Printf(">>> Executing Scan Query [pageSize=2, isLoc=true] for cache `%s`\n", cache.Name())
	pagedScanQueryCursor, err := cache.Scan(context.Background(), ignite.WithScanQueryPageSize(2), ignite.WithScanQueryLocal())

	if err != nil {
		panic(fmt.Errorf("failed to execute Scan Query: %w", err))
	}

	defer func() {
		_ = pagedScanQueryCursor.Close()
	}()

	printScanQueryResult(pagedScanQueryCursor)

	// Queries cache entries that matches the specified filter.
	// Filter is specified by FQN of Java Class implementation that must be present in the Ignite server classpath.
	// Examples of Java filter implementation are located in the current package folder
	// (see PersonByNamePojoFilter/PersonByNameBinaryObjectFilter).
	// If Scan Query filter without KeepBinaryFlag is used, Ignite will try to unmarshall cache entry to the Java class
	// representation before passing it to the Scan Query Filter. So in this case, Ignite server side classpath must
	// contain corresponding Java classes. Note that for this example it is critical that the FQN of the Person class
	// matches the type of the Binary Object created when the cache is populated.
	fmt.Printf(">>> Executing Scan Query [filter = FilterByFieldName[name=Filippe-3]] for cache `%s`\n", cache.Name())
	pojoFilterScanQueryCursor, err := cache.Scan(context.Background(),
		ignite.WithScanQueryFilter(
			"ru.gitverse.sbertech.client.filters.PersonByNamePojoFilter", // Class name of the Scan Query filter.
			// This option allows you to set a value to an arbitrary Scan Query Filter field with the specified name.
			// Take a look at usages of PersonByNamePojoFilter#name  class property.
			ignite.WithClosureField("name", "Filippe-3")))

	if err != nil {
		panic(fmt.Errorf("failed to execute Scan Query: %w", err))
	}
	defer func() {
		_ = pojoFilterScanQueryCursor.Close()
	}()

	printScanQueryResult(pojoFilterScanQueryCursor)

	// The same as previous Scan Query request but Scan Query Filter consumes Binary Object representation
	// of a cache value.
	fmt.Printf(">>> Executing Scan Query [filter=FilterByFieldName[name=Filippe-2], keepBinary=true] for cache `%s`\n", cache.Name())
	binaryObjectFilterScanQueryCursor, err := cache.Scan(context.Background(),
		ignite.WithScanQueryKeepBinary(), // This option tells Ignite not to unmarshal Binary Object before passing it to Scan Query Filter.
		ignite.WithScanQueryFilter(
			"ru.gitverse.sbertech.client.filters.PersonByNameBinaryObjectFilter",
			ignite.WithClosureField("name", "Filippe-2")))

	if err != nil {
		panic(fmt.Errorf("failed to execute Scan Query: %w", err))
	}
	defer func() {
		_ = binaryObjectFilterScanQueryCursor.Close()
	}()

	printScanQueryResult(binaryObjectFilterScanQueryCursor)
}

// Populates cache with objects.
// See binary_object example for more details.
func populateCache(client *ignite.Client, cache *ignite.Cache) {
	for i := 1; i < 6; i++ {
		key := int64(i)
		val, err := client.CreateBinaryObject(context.Background(), "ru.gitverse.sbertech.client.filters.Person",
			ignite.WithField("name", fmt.Sprintf("Filippe-%d", i)),
			ignite.WithField("age", int16(50+i)))
		if err != nil {
			panic(fmt.Errorf("failed to create binary object: %w", err))
		}

		err = cache.Put(context.Background(), key, val)
		if err != nil {
			panic(fmt.Errorf("failed to write entry to the cache: %w", err))
		}
		fmt.Printf(">>> Entry [key=`%d`, val=%s] was written to cache `%s`\n", key, examples.ToString(val), cache.Name())
	}
}

// Iterates over cursor entries and prints them.
func printScanQueryResult(cursor ignite.Cursor) {
	var res []string

	for cursor.Next() {
		var key int64
		var val ignite.BinaryObject
		if err := cursor.Scan(&key, &val); err != nil {
			panic(fmt.Errorf("failed to scan over the query result row: %w", err))
		}
		res = append(res, fmt.Sprintf(">>> Scan Query cursor returned value [key=`%d`, val=%s]", key, examples.ToString(val)))
	}

	sort.Strings(res)
	fmt.Println(strings.Join(res, "\n"))
}
