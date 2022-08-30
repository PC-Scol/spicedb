package namespace

import (
	"context"
	"fmt"

	"github.com/authzed/spicedb/pkg/datastore"
	core "github.com/authzed/spicedb/pkg/proto/core/v1"
)

// ReadNamespaceAndRelation checks that the specified namespace and relation exist in the
// datastore.
//
// Returns ErrNamespaceNotFound if the namespace cannot be found.
// Returns ErrRelationNotFound if the relation was not found in the namespace.
// Returns the direct downstream error for all other unknown error.
func ReadNamespaceAndRelation(
	ctx context.Context,
	namespace string,
	relation string,
	ds datastore.Reader,
) (*core.NamespaceDefinition, *core.Relation, error) {
	config, _, err := ds.ReadNamespace(ctx, namespace)
	if err != nil {
		return nil, nil, err
	}

	for _, rel := range config.Relation {
		if rel.Name == relation {
			return config, rel, nil
		}
	}

	return nil, nil, NewRelationNotFoundErr(namespace, relation)
}

// CheckNamespaceAndRelation checks that the specified namespace and relation exist in the
// datastore.
//
// Returns datastore.ErrNamespaceNotFound if the namespace cannot be found.
// Returns ErrRelationNotFound if the relation was not found in the namespace.
// Returns the direct downstream error for all other unknown error.
func CheckNamespaceAndRelation(
	ctx context.Context,
	namespace string,
	relation string,
	allowEllipsis bool,
	ds datastore.Reader,
) error {
	config, _, err := ds.ReadNamespace(ctx, namespace)
	if err != nil {
		return err
	}

	if allowEllipsis && relation == datastore.Ellipsis {
		return nil
	}

	for _, rel := range config.Relation {
		if rel.Name == relation {
			return nil
		}
	}

	return NewRelationNotFoundErr(namespace, relation)
}

// ReadNamespaceAndTypes reads a namespace definition, version, and type system and returns it if found.
func ReadNamespaceAndTypes(
	ctx context.Context,
	nsName string,
	ds datastore.Reader,
) (*core.NamespaceDefinition, *TypeSystem, error) {
	nsDef, _, err := ds.ReadNamespace(ctx, nsName)
	if err != nil {
		return nil, nil, err
	}

	ts, terr := BuildNamespaceTypeSystemForDatastore(nsDef, ds)
	return nsDef, ts, terr
}

type RelationFilter struct {
	Namespace string
	Relation  string
}

func FindRelations(
	ctx context.Context,
	filter RelationFilter,
	ds datastore.Reader,
) ([]*core.RelationReference, error) {

	relations := []*core.RelationReference{}

	namespaces, err := ds.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	for _, namespace := range namespaces {
		for _, relation := range namespace.Relation {
			if relation.UsersetRewrite != nil {
				switch rw := relation.UsersetRewrite.RewriteOperation.(type) {
				case *core.UsersetRewrite_Union:
					rels, err := lookupSetRewrite(ctx, filter, rw.Union)
					if err != nil {
						return nil, err
					}
					relations = append(relations, rels...)
				case *core.UsersetRewrite_Intersection:
					rels, err := lookupSetRewrite(ctx, filter, rw.Intersection)
					if err != nil {
						return nil, err
					}
					relations = append(relations, rels...)
				case *core.UsersetRewrite_Exclusion:
					rels, err := lookupSetRewrite(ctx, filter, rw.Exclusion)
					if err != nil {
						return nil, err
					}
					relations = append(relations, rels...)
				default:
					return nil, fmt.Errorf("unknown kind of rewrite in relation %s#%s", filter.Namespace, relation.Name)
				}
			} else {
				for _, allowedDirectRelation := range relation.TypeInformation.AllowedDirectRelations {
					if allowedDirectRelation.Namespace == filter.Namespace {
						relations = append(relations, &core.RelationReference{
							Namespace: filter.Namespace,
							Relation:  allowedDirectRelation.GetRelation(),
						})
					}
				}
			}
		}
	}
	return relations, nil
}

func lookupSetRewrite(
	ctx context.Context,
	filter RelationFilter,
	so *core.SetOperation,
) ([]*core.RelationReference, error) {

	results := []*core.RelationReference{}
	for _, childOneof := range so.Child {
		switch child := childOneof.ChildType.(type) {
		/*case *core.SetOperation_Child_XThis:
		log.Ctx(ctx).Trace().Msg("union")
		return cl.lookupSetOperation(ctx, req, rw.Union, newLookupSubjectsUnion(stream))*/
		case *core.SetOperation_Child_ComputedUserset:
			relationsFound, err := lookupComputedUserSet(ctx, filter, child.ComputedUserset)
			if err != nil {
				return nil, err
			}
			results = append(results, relationsFound...)
		case *core.SetOperation_Child_TupleToUserset:
			relationsFound, err := lookupTupleToUserset(ctx, filter, child.TupleToUserset)
			if err != nil {
				return nil, err
			}
			results = append(results, relationsFound...)
		case *core.SetOperation_Child_UsersetRewrite:
			relationsFound, err := lookupUsersetRewrite(ctx, filter, child.UsersetRewrite)
			if err != nil {
				return nil, err
			}
			results = append(results, relationsFound...)
		default:
			return nil, fmt.Errorf("unknown kind of rewrite in lookup subjects")
		}
	}
	return results, nil
}

func lookupComputedUserSet(
	ctx context.Context,
	filter RelationFilter,
	cu *core.ComputedUserset,
) ([]*core.RelationReference, error) {

	// TODO implement
	// Note: not sure, but I believe we are not interested in userRewrite during RR calls
	// so we can leave as is
	return []*core.RelationReference{}, nil
}

func lookupTupleToUserset(
	ctx context.Context,
	filter RelationFilter,
	ttu *core.TupleToUserset,
) ([]*core.RelationReference, error) {

	// TODO implement
	// Note: not sure, but I believe we are not interested in userRewrite during RR calls
	// so we can leave as is
	return []*core.RelationReference{}, nil
}

func lookupUsersetRewrite(
	ctx context.Context,
	filter RelationFilter,
	usr *core.UsersetRewrite,
) ([]*core.RelationReference, error) {

	// TODO implement
	// Note: not sure, but I believe we are not interested in userRewrite during RR calls
	// so we can leave as is
	return []*core.RelationReference{}, nil
}
