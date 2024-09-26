package main

import (
	igniteTesting "gitverse.ru/sbertech/ignite-go-client/internal/testing"
	"testing"
)

func TestSqlQueryExample(t *testing.T) {
	igniteTesting.TestExample(t, main,
		">>> Created table `PERSON_TABLE` [columns=[`ID` LONG, `COMPANY` VARCHAR, `NAME` VARCHAR, `AGE` INT], cacheName=`person-cache`, cacheKeyType=`PERSON_KEY`, cacheValueType=`PERSON`]\n"+
			">>> Entry [key=BinaryObject [type=`PERSON_KEY`, id=`0`, company=`RCM`], val=BinaryObject [type=`PERSON`, name=`Cuno`, age=`12`]] was written to cache `person-cache`\n"+
			">>> Executing SQL Query [sql=`SELECT * FROM PERSON`, schema=`EXAMPLE`]\n"+
			">>> SQL Query cursor returned column names [ID, COMPANY, NAME, AGE]\n"+
			">>> SQL Query cursor returned row [id=`0`, company=`RCM`, name=`Cuno`, age=`12`]\n"+
			">>> Executing SQL Query [sql=`INSERT INTO PERSON(ID, COMPANY, NAME, AGE) VALUES(?, ?, ?, ?)`, schema=`EXAMPLE`, arguments=[`1`, `RCM`, `Kim`, `38`]]\n"+
			">>> Requesting value for key == BinaryObject [type=`PERSON_KEY`, id=`1`, company=`RCM`] from cache `person-cache`\n"+
			">>> Value == BinaryObject [type=`PERSON`, name=`Kim`, age=`38`]\n")
}
