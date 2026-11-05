package custfind_test

import (
	"context"
	"fmt"
	"log"

	custfind "github.com/parevo/custfind-go"
)

func Example() {
	ctx := context.Background()
	cf, err := custfind.NewClient() // reads CUSTFIND_API_KEY
	if err != nil {
		log.Fatal(err)
	}
	created, err := cf.Searches.Create(ctx, custfind.CreateSearchParams{
		Sector:             "Dental clinic",
		Location:           "Kadıköy, Istanbul",
		Limit:              25,        // caps the credits this search reserves
		ExternalCustomerID: "acme-42", // optional: bill your own customer
	})
	if err != nil {
		log.Fatal(err)
	}
	if _, err := cf.Searches.WaitForCompletion(ctx, created.JobID, custfind.WaitOptions{}); err != nil {
		log.Fatal(err)
	}
	for lead, err := range cf.Searches.AllLeads(ctx, created.JobID, custfind.LeadListParams{}) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(lead.Name, lead.Email)
	}
}
