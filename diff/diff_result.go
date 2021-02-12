package diff

import (
	"regexp"

	"github.com/getkin/kin-openapi/openapi3"
	log "github.com/sirupsen/logrus"
)

type DiffResult struct {
	AddedEndpoints    []string          `json:"addedEndpoints,omitempty"`
	DeletedEndpoints  []string          `json:"deletedEndpoints,omitempty"`
	ModifiedEndpoints ModifiedEndpoints `json:"modifiedEndpoints,omitempty"`
}

func (diffResult *DiffResult) empty() bool {
	return len(diffResult.AddedEndpoints) == 0 &&
		len(diffResult.DeletedEndpoints) == 0 &&
		len(diffResult.ModifiedEndpoints) == 0
}

func newDiffResult() *DiffResult {
	return &DiffResult{
		AddedEndpoints:    []string{},
		DeletedEndpoints:  []string{},
		ModifiedEndpoints: ModifiedEndpoints{},
	}
}

func (diffResult *DiffResult) addAddedEndpoint(endpoint string) {
	diffResult.AddedEndpoints = append(diffResult.AddedEndpoints, endpoint)
}

func (diffResult *DiffResult) addDeletedEndpoint(endpoint string) {
	diffResult.DeletedEndpoints = append(diffResult.DeletedEndpoints, endpoint)
}

func (diffResult *DiffResult) addModifiedEndpoint(entrypoint1 string, pathItem1 *openapi3.PathItem, pathItem2 *openapi3.PathItem) {
	diffResult.ModifiedEndpoints.addEndpointDiff(entrypoint1, pathItem1, pathItem2)
}

func (diffResult *DiffResult) FilterByRegex(filter string) {
	r, err := regexp.Compile(filter)
	if err != nil {
		log.Errorf("Failed to compile filter regex '%s' with '%v'", filter, err)
		return
	}

	diffResult.AddedEndpoints = filterEndpoints(diffResult.AddedEndpoints, r)
	diffResult.DeletedEndpoints = filterEndpoints(diffResult.DeletedEndpoints, r)
	diffResult.ModifiedEndpoints = filterModifiedEndpoints(diffResult.ModifiedEndpoints, r)
}

func filterEndpoints(endpoints []string, r *regexp.Regexp) []string {
	result := []string{}
	for _, endpoint := range endpoints {
		if r.MatchString(endpoint) {
			result = append(result, endpoint)
		}
	}

	return result
}

func filterModifiedEndpoints(modifiedEndpoints ModifiedEndpoints, r *regexp.Regexp) ModifiedEndpoints {
	result := ModifiedEndpoints{}

	for endpoint, endpointDiff := range modifiedEndpoints {
		if r.MatchString(endpoint) {
			result[endpoint] = endpointDiff
		}
	}

	return result
}

func (diffResult *DiffResult) GetSummary() *DiffSummary {
	return getDiffSummary(diffResult)
}
