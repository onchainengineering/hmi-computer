package runner

import "github.com/onchainengineering/hmi-computer/v2/provisionersdk/proto"

func sumDailyCost(resources []*proto.Resource) int {
	var sum int
	for _, r := range resources {
		sum += int(r.DailyCost)
	}
	return sum
}
