package v1

import (
	"context"

	"github.com/authzed/spicedb/internal/dispatch"
	"github.com/authzed/spicedb/internal/middleware/handwrittenvalidation"
	"github.com/authzed/spicedb/internal/middleware/usagemetrics"
	"github.com/authzed/spicedb/internal/services/shared"
	grpcmw "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcvalidate "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/authzed/authzed-go/proto/authzed/api/v1"
	"github.com/authzed/spicedb/pkg/middleware/consistency"
	corev1 "github.com/authzed/spicedb/pkg/proto/core/v1"
	dispatchv1 "github.com/authzed/spicedb/pkg/proto/dispatch/v1"
	reachv1 "github.com/authzed/spicedb/pkg/proto/reachability/v1"
)

// NewReachabilityServer creates a ReachabilityServiceServer instance.
func NewReachabilityServer(
	dispatch dispatch.Dispatcher,
	defaultDepth uint32,
) reachv1.ReachablityServiceServer {
	return &reachabilityServer{
		dispatch:     dispatch,
		defaultDepth: defaultDepth,
		WithServiceSpecificInterceptors: shared.WithServiceSpecificInterceptors{
			Unary: grpcmw.ChainUnaryServer(
				grpcvalidate.UnaryServerInterceptor(),
				handwrittenvalidation.UnaryServerInterceptor,
				usagemetrics.UnaryServerInterceptor(),
			),
			Stream: grpcmw.ChainStreamServer(
				grpcvalidate.StreamServerInterceptor(),
				handwrittenvalidation.StreamServerInterceptor,
				usagemetrics.StreamServerInterceptor(),
			),
		},
	}
}

type reachabilityServer struct {
	reachv1.UnimplementedReachablityServiceServer
	shared.WithServiceSpecificInterceptors

	dispatch     dispatch.Dispatcher
	defaultDepth uint32
}

func (rs *reachabilityServer) ReachableResources(request *reachv1.ReachableResourcesRequest, stream reachv1.ReachablityService_ReachableResourcesServer) error {
	ctx := stream.Context()
	atRevision, _ := consistency.MustRevisionFromContext(ctx)
	rrRequest := &dispatchv1.DispatchReachableResourcesRequest{
		Metadata: &dispatchv1.ResolverMeta{
			AtRevision:     atRevision.String(),
			DepthRemaining: rs.defaultDepth,
		},
		ResourceRelation: &corev1.RelationReference{
			Namespace: request.TargetObjectType,
			Relation:  "*",
		},
		SubjectIds: []string{request.StartingResource.ObjectId},
		SubjectRelation: &corev1.RelationReference{
			Namespace: request.StartingResource.ObjectType,
			// Relation:  "member",
			Relation: "*",
		},
	}
	reachableResourcesStream := &ReachableResourcesServerResponseAdapter{
		Stream:       stream,
		ResourceType: request.TargetObjectType,
		Ctx:          ctx,
	}

	if err := rs.dispatch.DispatchReachableResources(rrRequest, reachableResourcesStream); err != nil {
		return err
	}
	return nil

}

// Adapts ReachablityService_ReachableResourcesServer to ReachableResourcesStream
type ReachableResourcesServerResponseAdapter struct {
	Stream       reachv1.ReachablityService_ReachableResourcesServer
	ResourceType string
	Ctx          context.Context
}

// Publish publishes the result to the stream.
func (ra *ReachableResourcesServerResponseAdapter) Publish(item *dispatchv1.DispatchReachableResourcesResponse) error {
	for _, resourceId := range item.Resource.ResourceIds {
		response := &reachv1.ReachableResourcesResponse{
			FoundResource: &v1.ObjectReference{
				ObjectType: ra.ResourceType,
				ObjectId:   resourceId,
			},
			DirectPath: true, // TODO
			FoundAt:    nil,  // TODO
		}
		if err := ra.Stream.Send(response); err != nil {
			return status.Errorf(codes.Canceled, "reachableResources canceled by user: %s", err)
		}
	}
	return nil
}

// Context returns the context for the stream.
func (ra *ReachableResourcesServerResponseAdapter) Context() context.Context {
	return ra.Ctx
}
