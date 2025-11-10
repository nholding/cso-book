package main

import (
	"fmt"
	"github.com/nholding/cso-book/internal/company"
)

func main() {

	c, err := company.NewCompany("test1", "test2", "test3", "test4", "test5", "test6", "")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(c.AuditInfo.CreatedBy)
}
