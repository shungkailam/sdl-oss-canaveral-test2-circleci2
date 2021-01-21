package filter_test

import (
	"cloudservices/common/filter"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	DEBUG_FILTER = false
)

type tableAliasFilter struct {
	filter      string
	aliasFilter string
}

func TestFilter(t *testing.T) {
	t.Run("Filter test", func(t *testing.T) {
		t.Log("running filter test")
		aliasFilters := []tableAliasFilter{
			{filter: "a = true", aliasFilter: "test.a = true"},
			{filter: "a != True", aliasFilter: "test.a != true"},
			{filter: "a = FALSE", aliasFilter: "test.a = false"},
			{filter: "a = 3.0", aliasFilter: "test.a = 3.000000"},
			{filter: "a > 1 AND a < 5", aliasFilter: "test.a > 1 AND test.a < 5"},
			{filter: "a <= 1 OR a >= 5", aliasFilter: "test.a <= 1 OR test.a >= 5"},
			{filter: "a like '%suffix'", aliasFilter: "test.a LIKE '%suffix'"},
			{filter: "a like 'prefix%'", aliasFilter: "test.a LIKE 'prefix%'"},
			{filter: "a like 'prefix%suffix'", aliasFilter: "test.a LIKE 'prefix%suffix'"},
			{filter: "a like '%middle%'", aliasFilter: "test.a LIKE '%middle%'"},
			{filter: "a NOT like '%suffix'", aliasFilter: "test.a NOT LIKE '%suffix'"},
			{filter: "a between 1 and 5", aliasFilter: "test.a BETWEEN 1 AND 5"},
			{filter: "a not between 1 and 5", aliasFilter: "test.a NOT BETWEEN 1 AND 5"},
			{filter: "a between 'bar' and 'foo'", aliasFilter: "test.a BETWEEN 'bar' AND 'foo'"},
			{filter: "a between '2019-03-13' AND '2019-03-14'", aliasFilter: "test.a BETWEEN '2019-03-13' AND '2019-03-14'"},
			{filter: "a IN ('a', 'b', 'c')", aliasFilter: "test.a IN ('a', 'b', 'c')"},
			{filter: "a NOT IN ('a', 'b', 'c')", aliasFilter: "test.a NOT IN ('a', 'b', 'c')"},
			{filter: "c=3 and a=1 OR b='foo'", aliasFilter: "test.x = 3 AND test.a = 1 OR test.y = 'foo'"},
			{filter: "c=3 and (a=1 OR b='foo')", aliasFilter: "test.x = 3 AND (test.a = 1 OR test.y = 'foo')"},
			{filter: "a = ''", aliasFilter: "test.a = ''"},
		}
		logicalKeyMap := map[string]string{
			"c": "x",
			"b": "y",
		}
		for _, sf := range aliasFilters {
			f := sf.filter
			t.Logf("Testing filter %s", f)
			fe, err := filter.Parse(f)
			require.NoErrorf(t, err, "Failed to parse filter: %s", f)

			if DEBUG_FILTER {
				t.Logf("Successfully parsed filter: %s => %s", f, *fe)
			} else {
				t.Logf("Successfully parsed filter: %s", f)
			}
			filter := filter.TransformFields(fe, logicalKeyMap, "test", nil)
			if filter != sf.aliasFilter {
				t.Fatalf("Expected [%s], found [%s]", sf.aliasFilter, filter)
			}
			t.Logf("Alias filter %s", filter)
		}

		badFilters := []string{
			"a",
			"a = yes",
			"a != ok",
			"a = no",
			"a = 3.0.2",
			"a >> 1 AND a < 5",
			"a <== 1 OR a >= 5",
			"a likely '%suffix'",
			"a is like 'prefix%'",
			"a looks like 'prefix%suffix'",
			"a isnt like '%middle%'",
			"a NOT really like '%suffix'",
			"a inside 1 and 5",
			"a no between 1 and 5",
			"a between 'bar\" and 'foo'",
			"a INSIDE ('a', 'b', 'c')",
			"a NOT IN ('a'; 'b', 'c')",
			"c=3 and a=1 OR b='foo",
			"c=3 and (a=1 OR b='foo'",
		}
		for _, f := range badFilters {
			_, err := filter.Parse(f)
			require.Errorf(t, err, "Expect parsing of bad filter [%s] to fail!", f)
		}
	})
}
