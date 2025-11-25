package main

import (
	"fmt"
	"github.com/nholding/cso-book/internal/company"
	"github.com/nholding/cso-book/internal/period"
)

func main() {
	// allPeriods := period.GeneratePeriods(2026, 2026)
	// ps := period.NewPeriodStore(allPeriods)
	// purchaseBreakdowns := CreateTradeBreakdowns(purchase.TradeBase, ps, "user@internal.local")

	c, err := company.NewCompany("test1", "test2", "test3", "test4", "test5", "test6", "")
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(c.AuditInfo.CreatedBy)
}
