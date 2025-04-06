// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: proto/dbstream.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	DatabaseChangeStream_StartStream_FullMethodName = "/dbstream.DatabaseChangeStream/StartStream"
	DatabaseChangeStream_StopStream_FullMethodName  = "/dbstream.DatabaseChangeStream/StopStream"
	DatabaseChangeStream_SyncStream_FullMethodName  = "/dbstream.DatabaseChangeStream/SyncStream"
)

// DatabaseChangeStreamClient is the client API for DatabaseChangeStream service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DatabaseChangeStreamClient interface {
	StartStream(ctx context.Context, in *StreamRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[DatabaseEvent], error)
	StopStream(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*StopResponse, error)
	SyncStream(ctx context.Context, in *StreamRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[DatabaseEvent], error)
}

type databaseChangeStreamClient struct {
	cc grpc.ClientConnInterface
}

func NewDatabaseChangeStreamClient(cc grpc.ClientConnInterface) DatabaseChangeStreamClient {
	return &databaseChangeStreamClient{cc}
}

func (c *databaseChangeStreamClient) StartStream(ctx context.Context, in *StreamRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[DatabaseEvent], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &DatabaseChangeStream_ServiceDesc.Streams[0], DatabaseChangeStream_StartStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[StreamRequest, DatabaseEvent]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type DatabaseChangeStream_StartStreamClient = grpc.ServerStreamingClient[DatabaseEvent]

func (c *databaseChangeStreamClient) StopStream(ctx context.Context, in *StopRequest, opts ...grpc.CallOption) (*StopResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(StopResponse)
	err := c.cc.Invoke(ctx, DatabaseChangeStream_StopStream_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *databaseChangeStreamClient) SyncStream(ctx context.Context, in *StreamRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[DatabaseEvent], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &DatabaseChangeStream_ServiceDesc.Streams[1], DatabaseChangeStream_SyncStream_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[StreamRequest, DatabaseEvent]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type DatabaseChangeStream_SyncStreamClient = grpc.ServerStreamingClient[DatabaseEvent]

// DatabaseChangeStreamServer is the server API for DatabaseChangeStream service.
// All implementations must embed UnimplementedDatabaseChangeStreamServer
// for forward compatibility.
type DatabaseChangeStreamServer interface {
	StartStream(*StreamRequest, grpc.ServerStreamingServer[DatabaseEvent]) error
	StopStream(context.Context, *StopRequest) (*StopResponse, error)
	SyncStream(*StreamRequest, grpc.ServerStreamingServer[DatabaseEvent]) error
	mustEmbedUnimplementedDatabaseChangeStreamServer()
}

// UnimplementedDatabaseChangeStreamServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedDatabaseChangeStreamServer struct{}

func (UnimplementedDatabaseChangeStreamServer) StartStream(*StreamRequest, grpc.ServerStreamingServer[DatabaseEvent]) error {
	return status.Errorf(codes.Unimplemented, "method StartStream not implemented")
}
func (UnimplementedDatabaseChangeStreamServer) StopStream(context.Context, *StopRequest) (*StopResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method StopStream not implemented")
}
func (UnimplementedDatabaseChangeStreamServer) SyncStream(*StreamRequest, grpc.ServerStreamingServer[DatabaseEvent]) error {
	return status.Errorf(codes.Unimplemented, "method SyncStream not implemented")
}
func (UnimplementedDatabaseChangeStreamServer) mustEmbedUnimplementedDatabaseChangeStreamServer() {}
func (UnimplementedDatabaseChangeStreamServer) testEmbeddedByValue()                              {}

// UnsafeDatabaseChangeStreamServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DatabaseChangeStreamServer will
// result in compilation errors.
type UnsafeDatabaseChangeStreamServer interface {
	mustEmbedUnimplementedDatabaseChangeStreamServer()
}

func RegisterDatabaseChangeStreamServer(s grpc.ServiceRegistrar, srv DatabaseChangeStreamServer) {
	// If the following call pancis, it indicates UnimplementedDatabaseChangeStreamServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&DatabaseChangeStream_ServiceDesc, srv)
}

func _DatabaseChangeStream_StartStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DatabaseChangeStreamServer).StartStream(m, &grpc.GenericServerStream[StreamRequest, DatabaseEvent]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type DatabaseChangeStream_StartStreamServer = grpc.ServerStreamingServer[DatabaseEvent]

func _DatabaseChangeStream_StopStream_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StopRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DatabaseChangeStreamServer).StopStream(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: DatabaseChangeStream_StopStream_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DatabaseChangeStreamServer).StopStream(ctx, req.(*StopRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _DatabaseChangeStream_SyncStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(DatabaseChangeStreamServer).SyncStream(m, &grpc.GenericServerStream[StreamRequest, DatabaseEvent]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type DatabaseChangeStream_SyncStreamServer = grpc.ServerStreamingServer[DatabaseEvent]

// DatabaseChangeStream_ServiceDesc is the grpc.ServiceDesc for DatabaseChangeStream service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DatabaseChangeStream_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "dbstream.DatabaseChangeStream",
	HandlerType: (*DatabaseChangeStreamServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "StopStream",
			Handler:    _DatabaseChangeStream_StopStream_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StartStream",
			Handler:       _DatabaseChangeStream_StartStream_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "SyncStream",
			Handler:       _DatabaseChangeStream_SyncStream_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/dbstream.proto",
}
