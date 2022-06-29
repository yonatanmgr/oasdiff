package diff

import (
	"fmt"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

/*
SchemaListDiff describes the changes between a pair of lists of schema objects: https://swagger.io/specification/#schema-object
The result is a combination of two diffs:
1. Diff of schemas with a $ref: number of added/deleted schemas; modified=diff of schemas with the same $ref
2. Diff of schemas without a $ref (inline schemas): number of added/deleted schemas; modified=only if exactly one schema was added and one deleted, the Modified field will show a diff between them
*/
type SchemaListDiff struct {
	Added    int             `json:"added,omitempty" yaml:"added,omitempty"`
	Deleted  int             `json:"deleted,omitempty" yaml:"deleted,omitempty"`
	Modified ModifiedSchemas `json:"modified,omitempty" yaml:"modified,omitempty"`
}

// Empty indicates whether a change was found in this element
func (diff *SchemaListDiff) Empty() bool {
	if diff == nil {
		return true
	}

	return diff.Added == 0 &&
		diff.Deleted == 0 &&
		len(diff.Modified) == 0
}

func getSchemaListsDiff(config *Config, state *state, schemaRefs1, schemaRefs2 openapi3.SchemaRefs) (*SchemaListDiff, error) {
	diff, err := getSchemaListsDiffInternal(config, state, schemaRefs1, schemaRefs2)
	if err != nil {
		return nil, err
	}

	if diff.Empty() {
		return nil, nil
	}

	return diff, nil
}

type SchemaRefMap map[string]*openapi3.SchemaRef

func toSchemaRefsMap(schemaRefs openapi3.SchemaRefs) SchemaRefMap {
	result := SchemaRefMap{}
	for _, schemaRef := range schemaRefs {
		if !isSchemaInline(schemaRef) {
			result[schemaRef.Ref] = schemaRef
		}
	}
	return result
}

func (diff SchemaListDiff) combine(other SchemaListDiff) (*SchemaListDiff, error) {

	return &SchemaListDiff{
		Added:    diff.Added + other.Added,
		Deleted:  diff.Deleted + other.Deleted,
		Modified: diff.Modified.combine(other.Modified),
	}, nil
}

func getSchemaListsDiffInternal(config *Config, state *state, schemaRefs1, schemaRefs2 openapi3.SchemaRefs) (*SchemaListDiff, error) {

	diffRefs, err := getSchemaListsRefsDiff(config, state, filterSchemaRefs(schemaRefs1, isSchemaRef), filterSchemaRefs(schemaRefs2, isSchemaRef))
	if err != nil {
		return nil, err
	}

	diffInline, err := getSchemaListsInlineDiff(config, state, filterSchemaRefs(schemaRefs1, isSchemaInline), filterSchemaRefs(schemaRefs2, isSchemaInline))
	if err != nil {
		return nil, err
	}

	return diffRefs.combine(diffInline)
}

// getSchemaListsRefsDiff compares schemas by $ref name
func getSchemaListsRefsDiff(config *Config, state *state, schemaRefs1, schemaRefs2 openapi3.SchemaRefs) (SchemaListDiff, error) {
	return getSchemaMapsRefsDiff(config, state, toSchemaRefsMap(schemaRefs1), toSchemaRefsMap(schemaRefs2))
}

func getSchemaMapsRefsDiff(config *Config, state *state, schemaMap1, schemaMap2 SchemaRefMap) (SchemaListDiff, error) {
	deleted := 0
	modified := ModifiedSchemas{}
	for ref, schema1 := range schemaMap1 {
		if schema2, found := schemaMap2[ref]; found {
			if err := modified.addSchemaDiff(config, state, ref, schema1, schema2); err != nil {
				return SchemaListDiff{}, err
			}
		} else {
			deleted++
		}
	}

	added := 0
	for ref := range schemaMap2 {
		if _, found := schemaMap1[ref]; !found {
			added++
		}
	}
	return SchemaListDiff{
		Added:    added,
		Deleted:  deleted,
		Modified: modified,
	}, nil
}

// getSchemaListsRefsDiff compares schemas by their syntax
func getSchemaListsInlineDiff(config *Config, state *state, schemaRefs1, schemaRefs2 openapi3.SchemaRefs) (SchemaListDiff, error) {

	added, err := getGroupDifference(schemaRefs2, schemaRefs1)
	if err != nil {
		return SchemaListDiff{}, err
	}

	deleted, err := getGroupDifference(schemaRefs1, schemaRefs2)
	if err != nil {
		return SchemaListDiff{}, err
	}

	if len(added) == 1 && len(deleted) == 1 {
		d, err := getSchemaDiff(config, state, schemaRefs1[deleted[0]], schemaRefs2[added[0]])
		if err != nil {
			return SchemaListDiff{}, err
		}

		if d.Empty() {
			return SchemaListDiff{}, err
		}

		return SchemaListDiff{
			Modified: ModifiedSchemas{fmt.Sprintf("#%d", 1+deleted[0]): d},
		}, nil
	}

	return SchemaListDiff{
		Added:   len(added),
		Deleted: len(deleted),
	}, nil
}

func getGroupDifference(schemaRefs1, schemaRefs2 openapi3.SchemaRefs) ([]int, error) {

	notContained := []int{}
	matched := map[int]struct{}{}

	for index1, schemaRef1 := range schemaRefs1 {
		if found, index2 := findIndenticalSchema(schemaRef1, schemaRefs2, matched); !found {
			notContained = append(notContained, index1)
		} else {
			matched[index2] = struct{}{}
		}
	}
	return notContained, nil
}

func findIndenticalSchema(schemaRef1 *openapi3.SchemaRef, schemasRefs2 openapi3.SchemaRefs, matched map[int]struct{}) (bool, int) {
	for index2, schemaRef2 := range schemasRefs2 {
		if alreadyMatched(index2, matched) {
			continue
		}

		// compare with DeepEqual rather than SchemaDiff to ensure an exact syntactical match
		if reflect.DeepEqual(schemaRef1, schemaRef2) {
			return true, index2
		}
	}

	return false, 0
}

func alreadyMatched(index int, matched map[int]struct{}) bool {
	_, found := matched[index]
	return found
}

func isSchemaInline(schemaRef *openapi3.SchemaRef) bool {
	if schemaRef == nil {
		return false
	}
	return schemaRef.Ref == ""
}

func isSchemaRef(schemaRef *openapi3.SchemaRef) bool {
	if schemaRef == nil {
		return false
	}
	return schemaRef.Ref != ""
}

type SchemaRefsFilter func(schemaRef *openapi3.SchemaRef) bool

func filterSchemaRefs(schemaRefs openapi3.SchemaRefs, filter SchemaRefsFilter) openapi3.SchemaRefs {
	result := openapi3.SchemaRefs{}
	for _, schemaRef := range schemaRefs {
		if filter(schemaRef) {
			result = append(result, schemaRef)
		}
	}
	return schemaRefs
}
