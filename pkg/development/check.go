package development

import (
	core "github.com/authzed/spicedb/pkg/proto/core/v1"

	v1 "github.com/authzed/spicedb/pkg/proto/dispatch/v1"
)

// RunCheck performs a check against the data in the development context.
//
// Note that it is up to the caller to call DistinguishGraphError on the error
// if they want to distinguish between user errors and internal errors.
func RunCheck(devContext *DevContext, resource *core.ObjectAndRelation, subject *core.ObjectAndRelation) (v1.DispatchCheckResponse_Membership, error) {
	ctx := devContext.Ctx
	cr, err := devContext.Dispatcher.DispatchCheck(ctx, &v1.DispatchCheckRequest{
		ResourceRelation: &core.RelationReference{
			Namespace: resource.Namespace,
			Relation:  resource.Relation,
		},
		ResourceIds:    []string{resource.ObjectId},
		ResultsSetting: v1.DispatchCheckRequest_ALLOW_SINGLE_RESULT,
		Subject:        subject,
		Metadata: &v1.ResolverMeta{
			AtRevision:     devContext.Revision.String(),
			DepthRemaining: maxDispatchDepth,
		},
	})
	if err != nil {
		return v1.DispatchCheckResponse_NOT_MEMBER, err
	}

	if found, ok := cr.ResultsByResourceId[resource.ObjectId]; ok {
		return found.Membership, nil
	}

	return v1.DispatchCheckResponse_NOT_MEMBER, nil
}
