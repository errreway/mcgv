package examples

import (
	"context"
	"fmt"
	"gitverse.ru/sbertech/ignite-go-client"
	"strconv"
	"strings"
)

func ActivateIgniteCluster(cli *ignite.Client) {
	state, err := cli.GetClusterState(context.Background())
	if err != nil {
		panic(fmt.Errorf("failed to request cluster state: %w", err))
	}
	if state != ignite.Active {
		err := cli.ChangeClusterState(context.Background(), ignite.Active)
		if err != nil {
			panic(fmt.Errorf("failed to change cluster state: %w", err))
		}
		fmt.Printf(">>> Ignite cluster state was changed to `%v`\n", ignite.Active)
	}
}

func ToString(binObj ignite.BinaryObject) string {
	builder := strings.Builder{}
	printBinaryObject(&builder, context.Background(), binObj)
	return builder.String()
}

// This example shows how to access the field structure of a Binary Object.
// Iterates over Binary Object fields and prints their values to the specified builder.
func printBinaryObject(builder *strings.Builder, ctx context.Context, binObj ignite.BinaryObject) {
	binObjType, err := binObj.Type(ctx)
	if err != nil {
		panic(fmt.Errorf("failed to get binary object type: %w", err))
	}

	if binObj.IsEnum() {
		builder.WriteString(fmt.Sprintf("BinaryEnum [type=`%s`", binObjType.TypeName()))
		builder.WriteString(", ordinal=")
		builder.WriteString(strconv.Itoa(binObj.EnumOrdinal()))
		builder.WriteString(", name=")
		builder.WriteString(binObj.EnumName())
		builder.WriteString("]")
	} else {
		builder.WriteString(fmt.Sprintf("BinaryObject [type=`%s`", binObjType.TypeName()))

		for _, fieldName := range binObjType.Fields() {
			field, err := binObj.Field(ctx, fieldName)
			if err != nil {
				panic(fmt.Errorf("failed to get binary object field [name=%s]: %w", fieldName, err))
			}

			builder.WriteString(", ")
			builder.WriteString(fieldName)
			builder.WriteString("=")

			if binObjField, ok := field.(ignite.BinaryObject); ok {
				printBinaryObject(builder, ctx, binObjField)
			} else {
				builder.WriteString(fmt.Sprintf("`%v`", field))
			}
		}

		builder.WriteString("]")
	}
}
