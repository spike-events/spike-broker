package socket

import (
	"fmt"
	"strings"

	"github.com/spike-events/spike-broker/v2/pkg/rids"
	spikeutils "github.com/spike-events/spike-broker/v2/pkg/spike-utils"
)

func PatternFromEndpoint(patterns []rids.Pattern, specificEndpoint string) rids.Pattern {
	for _, pattern := range patterns {
		patternParts := strings.Split(pattern.EndpointName(), ".")
		queryParts := strings.Split(specificEndpoint, "?")
		endpointParts := strings.Split(queryParts[0], ".")

		if len(patternParts) != len(endpointParts) {
			continue
		}

		matches := true
		params := make(map[string]fmt.Stringer, 0)
		for i := range patternParts {
			if strings.HasPrefix(patternParts[i], "$") {
				key := patternParts[i][1:]
				params[key] = spikeutils.Stringer(endpointParts[i])
				continue
			}
			if patternParts[i] != endpointParts[i] {
				matches = false
				break
			}
		}

		if matches {
			var patternClone rids.Pattern
			patternClone = pattern.Clone()
			patternClone.SetParams(params)
			if len(queryParts) > 1 {
				patternClone.Query(queryParts[1])
			}
			return patternClone
		}
	}
	return nil
}
