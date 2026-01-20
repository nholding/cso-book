package main

import (
	"context"
	"fmt"
	"log"
	"time"

	//	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nholding/cso-book/internal/period/domain"
	"github.com/nholding/cso-book/internal/period/repository"
	"github.com/nholding/cso-book/internal/period/service"
	"github.com/nholding/cso-book/internal/platform/awsclient"
)

func main() {
	// allPeriods := period.GeneratePeriods(2026, 2026)
	// ps := period.NewPeriodStore(allPeriods)
	// purchaseBreakdowns := CreateTradeBreakdowns(purchase.TradeBase, ps, "user@internal.local")

	fmt.Println("Hello World")

	config := awsclient.Config{
		Profile:      "productionadmin",
		S3BucketName: "terraform-tfstate-production-nh",
		Region:       "eu-central-1",
		DBName:       "postgres",
		DBEndpoint:   "erikkn-test.cluster-ctmmuuqkyfod.eu-central-1.rds.amazonaws.com",
		//DBEndpoint: "erikkn-test-instance-1.ctmmuuqkyfod.eu-central-1.rds.amazonaws.com",
		DBUser: "superadmin",
		DBPort: 5432,
	}

	rdsRepo, err := repository.NewRdsPeriodRepository(&config)
	if err != nil {
		log.Fatalf("error creating RDS client: %v", err)
	}

	periodService := service.NewPeriodService(rdsRepo)

	fy := []domain.FiscalCalendarConfig{{
		StartYear:  2026,
		StartMonth: time.April,
	}}

	if err := periodService.InitializePeriods(context.TODO(), 2026, 2027, fy); err != nil {
		log.Fatalf("error initialising periods: %v", err)
	}

	//oErrs := periodService.ValidateOverlaps()
	//if len(oErrs) > 0 {
	//	fmt.Println("❌ Period overlaps detected! Application cannot continue.")
	//	for _, e := range oErrs {
	//		fmt.Println("   →", e)
	//	}
	//	os.Exit(1)
	//}

	//hErrs := periodService.ValidateHierarchy()
	//if len(hErrs) > 0 {
	//	fmt.Println("❌ Invalid period hierarchy detected! Application cannot continue.")
	//	for _, e := range hErrs {
	//		fmt.Println("   →", e)
	//	}
	//	// Terminate application (fail fast)
	//	os.Exit(1)
	//}

	fmt.Println(periodService.BreakDownTradeRange(domain.PeriodRange{StartPeriodID: "2026-Q1", EndPeriodID: "2027-Q2"}))

}
