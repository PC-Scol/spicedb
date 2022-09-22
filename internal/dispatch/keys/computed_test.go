package keys

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/authzed/spicedb/internal/util"

	"github.com/stretchr/testify/require"

	core "github.com/authzed/spicedb/pkg/proto/core/v1"
	v1 "github.com/authzed/spicedb/pkg/proto/dispatch/v1"
	"github.com/authzed/spicedb/pkg/tuple"
)

func TestKeyPrefixOverlap(t *testing.T) {
	encountered := map[string]struct{}{}
	for _, prefix := range cachePrefixes {
		_, ok := encountered[string(prefix)]
		require.False(t, ok)
		encountered[string(prefix)] = struct{}{}
	}
}

var (
	ONR = tuple.ObjectAndRelation
	RR  = tuple.RelationReference
)

func TestStableCacheKeys(t *testing.T) {
	tcs := []struct {
		name      string
		createKey func() DispatchCacheKey
		expected  string
	}{
		{
			"basic check",
			func() DispatchCacheKey {
				return checkRequestToKey(&v1.DispatchCheckRequest{
					ResourceRelation: RR("document", "view"),
					ResourceIds:      []string{"foo", "bar"},
					Subject:          ONR("user", "tom", "..."),
					Metadata: &v1.ResolverMeta{
						AtRevision: "1234",
					},
				}, computeBothHashes)
			},
			"d6a999a7a0a4d39c9f01",
		},
		{
			"different check",
			func() DispatchCacheKey {
				return checkRequestToKey(&v1.DispatchCheckRequest{
					ResourceRelation: RR("document", "edit"),
					ResourceIds:      []string{"foo"},
					Subject:          ONR("user", "sarah", "..."),
					Metadata: &v1.ResolverMeta{
						AtRevision: "123456",
					},
				}, computeBothHashes)
			},
			"d586cee091f9e591c301",
		},
		{
			"canonical check",
			func() DispatchCacheKey {
				return checkRequestToKeyWithCanonical(&v1.DispatchCheckRequest{
					ResourceRelation: RR("document", "view"),
					ResourceIds:      []string{"foo", "bar"},
					Subject:          ONR("user", "tom", "..."),
					Metadata: &v1.ResolverMeta{
						AtRevision: "1234",
					},
				}, "view")
			},
			"be9fff86aa9aebdba801",
		},
		{
			"expand",
			func() DispatchCacheKey {
				return expandRequestToKey(&v1.DispatchExpandRequest{
					ResourceAndRelation: ONR("document", "foo", "view"),
					Metadata: &v1.ResolverMeta{
						AtRevision: "1234",
					},
				}, computeBothHashes)
			},
			"8afff68e91a7cbb3ef01",
		},
		{
			"lookup resources",
			func() DispatchCacheKey {
				return lookupRequestToKey(&v1.DispatchLookupRequest{
					ObjectRelation: RR("document", "view"),
					Subject:        ONR("user", "mariah", "..."),
					Limit:          10,
					Metadata: &v1.ResolverMeta{
						AtRevision: "1234",
					},
				}, computeBothHashes)
			},
			"d4a1bda89c9790f50c",
		},
		{
			"reachable resources",
			func() DispatchCacheKey {
				return reachableResourcesRequestToKey(&v1.DispatchReachableResourcesRequest{
					ResourceRelation: RR("document", "view"),
					SubjectRelation:  RR("user", "..."),
					SubjectIds:       []string{"mariah", "tom"},
					Metadata: &v1.ResolverMeta{
						AtRevision: "1234",
					},
				}, computeBothHashes)
			},
			"e8848b9dd68f93a6c801",
		},
		{
			"lookup subjects",
			func() DispatchCacheKey {
				return lookupSubjectsRequestToKey(&v1.DispatchLookupSubjectsRequest{
					ResourceRelation: RR("document", "view"),
					SubjectRelation:  RR("user", "..."),
					ResourceIds:      []string{"mariah", "tom"},
					Metadata: &v1.ResolverMeta{
						AtRevision: "1234",
					},
				}, computeBothHashes)
			},
			"d699c5b5d3a6dfade601",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			key := tc.createKey()
			require.Equal(t, tc.expected, hex.EncodeToString(key.StableSumAsBytes()))
		})
	}
}

type generatorFunc func(
	resourceIds []string,
	subjectIds []string,
	resourceRelation *core.RelationReference,
	subjectRelation *core.RelationReference,
	revision *v1.ResolverMeta,
) (DispatchCacheKey, []string)

var generatorFuncs = map[string]generatorFunc{
	// Check.
	string(checkViaRelationPrefix): func(
		resourceIds []string,
		subjectIds []string,
		resourceRelation *core.RelationReference,
		subjectRelation *core.RelationReference,
		metadata *v1.ResolverMeta,
	) (DispatchCacheKey, []string) {
		return checkRequestToKey(&v1.DispatchCheckRequest{
				ResourceRelation: resourceRelation,
				ResourceIds:      resourceIds,
				Subject:          ONR(subjectRelation.Namespace, subjectIds[0], subjectRelation.Relation),
				Metadata:         metadata,
			}, computeBothHashes), []string{
				resourceRelation.Namespace,
				resourceRelation.Relation,
				subjectRelation.Namespace,
				subjectIds[0],
				subjectRelation.Relation,
			}
	},

	// Canonical Check.
	string(checkViaCanonicalPrefix): func(
		resourceIds []string,
		subjectIds []string,
		resourceRelation *core.RelationReference,
		subjectRelation *core.RelationReference,
		metadata *v1.ResolverMeta,
	) (DispatchCacheKey, []string) {
		return checkRequestToKeyWithCanonical(&v1.DispatchCheckRequest{
				ResourceRelation: resourceRelation,
				ResourceIds:      resourceIds,
				Subject:          ONR(subjectRelation.Namespace, subjectIds[0], subjectRelation.Relation),
				Metadata:         metadata,
			}, resourceRelation.Relation), append([]string{
				resourceRelation.Namespace,
				resourceRelation.Relation,
				subjectRelation.Namespace,
				subjectIds[0],
				subjectRelation.Relation,
			}, resourceIds...)
	},

	// Lookup Resources.
	string(lookupPrefix): func(
		resourceIds []string,
		subjectIds []string,
		resourceRelation *core.RelationReference,
		subjectRelation *core.RelationReference,
		metadata *v1.ResolverMeta,
	) (DispatchCacheKey, []string) {
		return lookupRequestToKey(&v1.DispatchLookupRequest{
				ObjectRelation: resourceRelation,
				Subject:        ONR(subjectRelation.Namespace, subjectIds[0], subjectRelation.Relation),
				Metadata:       metadata,
			}, computeBothHashes), []string{
				resourceRelation.Namespace,
				resourceRelation.Relation,
				subjectRelation.Namespace,
				subjectIds[0],
				subjectRelation.Relation,
			}
	},

	// Expand.
	string(expandPrefix): func(
		resourceIds []string,
		subjectIds []string,
		resourceRelation *core.RelationReference,
		subjectRelation *core.RelationReference,
		metadata *v1.ResolverMeta,
	) (DispatchCacheKey, []string) {
		return expandRequestToKey(&v1.DispatchExpandRequest{
				ResourceAndRelation: ONR(resourceRelation.Namespace, resourceIds[0], resourceRelation.Relation),
				Metadata:            metadata,
			}, computeBothHashes), []string{
				resourceRelation.Namespace,
				resourceIds[0],
				resourceRelation.Relation,
			}
	},

	// Reachable Resources.
	string(reachableResourcesPrefix): func(
		resourceIds []string,
		subjectIds []string,
		resourceRelation *core.RelationReference,
		subjectRelation *core.RelationReference,
		metadata *v1.ResolverMeta,
	) (DispatchCacheKey, []string) {
		return reachableResourcesRequestToKey(&v1.DispatchReachableResourcesRequest{
				ResourceRelation: resourceRelation,
				SubjectRelation:  subjectRelation,
				SubjectIds:       subjectIds,
				Metadata:         metadata,
			}, computeBothHashes), append([]string{
				resourceRelation.Namespace,
				resourceRelation.Relation,
				subjectRelation.Namespace,
				subjectRelation.Relation,
			}, subjectIds...)
	},

	// Lookup Subjects.
	string(lookupSubjectsPrefix): func(
		resourceIds []string,
		subjectIds []string,
		resourceRelation *core.RelationReference,
		subjectRelation *core.RelationReference,
		metadata *v1.ResolverMeta,
	) (DispatchCacheKey, []string) {
		return lookupSubjectsRequestToKey(&v1.DispatchLookupSubjectsRequest{
				ResourceRelation: resourceRelation,
				SubjectRelation:  subjectRelation,
				ResourceIds:      resourceIds,
				Metadata:         metadata,
			}, computeBothHashes), append([]string{
				resourceRelation.Namespace,
				resourceRelation.Relation,
				subjectRelation.Namespace,
				subjectRelation.Relation,
			}, resourceIds...)
	},
}

func TestCacheKeyNoOverlap(t *testing.T) {
	allResourceIds := [][]string{
		{"1"},
		{"1", "2"},
		{"1", "2", "3"},
		{"hi"},
	}

	allSubjectIds := [][]string{
		{"tom"},
		{"mariah", "tom"},
		{"sarah", "mariah", "tom"},
	}

	resourceRelations := []*core.RelationReference{
		RR("document", "view"),
		RR("document", "viewer"),
		RR("document", "edit"),
		RR("folder", "view"),
	}

	subjectRelations := []*core.RelationReference{
		RR("user", "..."),
		RR("user", "token"),
		RR("folder", "parent"),
		RR("group", "member"),
	}

	revisions := []string{"1234", "4567", "1235"}

	dataCombinationSeen := util.NewSet[string]()
	stableCacheKeysSeen := util.NewSet[string]()
	unstableCacheKeysSeen := util.NewSet[uint64]()

	// Ensure all key functions are generated.
	require.Equal(t, len(generatorFuncs), len(cachePrefixes))

	for _, resourceIds := range allResourceIds {
		t.Run(strings.Join(resourceIds, ","), func(t *testing.T) {
			for _, subjectIds := range allSubjectIds {
				t.Run(strings.Join(subjectIds, ","), func(t *testing.T) {
					for _, resourceRelation := range resourceRelations {
						t.Run(tuple.StringRR(resourceRelation), func(t *testing.T) {
							for _, subjectRelation := range subjectRelations {
								t.Run(tuple.StringRR(subjectRelation), func(t *testing.T) {
									for _, revision := range revisions {
										t.Run(revision, func(t *testing.T) {
											metadata := &v1.ResolverMeta{
												AtRevision: revision,
											}

											for prefix, f := range generatorFuncs {
												t.Run(prefix, func(t *testing.T) {
													generated, usedData := f(resourceIds, subjectIds, resourceRelation, subjectRelation, metadata)
													usedDataString := fmt.Sprintf("%s:%s", prefix, strings.Join(usedData, ","))
													if dataCombinationSeen.Add(usedDataString) {
														require.True(t, stableCacheKeysSeen.Add(hex.EncodeToString((generated.StableSumAsBytes()))))
														require.True(t, unstableCacheKeysSeen.Add(generated.processSpecificSum))
													}
												})
											}
										})
									}
								})
							}
						})
					}
				})
			}
		})
	}
}

func TestComputeOnlyStableHash(t *testing.T) {
	result := checkRequestToKey(&v1.DispatchCheckRequest{
		ResourceRelation: RR("document", "view"),
		ResourceIds:      []string{"foo", "bar"},
		Subject:          ONR("user", "tom", "..."),
		Metadata: &v1.ResolverMeta{
			AtRevision: "1234",
		},
	}, computeOnlyStableHash)

	require.Equal(t, uint64(0), result.processSpecificSum)
}
