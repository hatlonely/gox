package main

import (
	"fmt"
	"github.com/hatlonely/gox/rdb/query"
)

func main() {
	termQuery := &query.TermQuery{
		Field: "name",
		Value: "张三",
	}

	sql, args, err := termQuery.ToSQL()
	fmt.Printf("SQL: %s\n", sql)
	fmt.Printf("Args: %v\n", args)
	fmt.Printf("Error: %v\n", err)
}