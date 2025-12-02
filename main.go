package main

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/nholding/cso-book/internal/repository"
)

func main() {
	// allPeriods := period.GeneratePeriods(2026, 2026)
	// ps := period.NewPeriodStore(allPeriods)
	// purchaseBreakdowns := CreateTradeBreakdowns(purchase.TradeBase, ps, "user@internal.local")

	fmt.Println("Hello World")

	config := repository.Config{
		Profile:      "productionadmin",
		S3BucketName: "terraform-tfstate-production-nh",
		Region:       "eu-central-1",
		DBName:       "postgres",
		DBEndpoint:   "erikkn-test.cluster-ctmmuuqkyfod.eu-central-1.rds.amazonaws.com",
		//DBEndpoint: "erikkn-test-instance-1.ctmmuuqkyfod.eu-central-1.rds.amazonaws.com",
		DBUser: "superadmin",
		DBPort: 5432,
	}

	client, err := repository.NewAWSClients(&config)
	if err != nil {
		fmt.Println(err)
	}

	output, _ := client.S3.Client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: &client.S3.BucketName,
	})

	fmt.Println(output)

	fmt.Println(client)

}
