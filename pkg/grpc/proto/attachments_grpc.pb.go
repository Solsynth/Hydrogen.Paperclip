// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v5.26.1
// source: attachments.proto

package proto

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Attachments_GetAttachment_FullMethodName         = "/proto.Attachments/GetAttachment"
	Attachments_CheckAttachmentExists_FullMethodName = "/proto.Attachments/CheckAttachmentExists"
)

// AttachmentsClient is the client API for Attachments service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type AttachmentsClient interface {
	GetAttachment(ctx context.Context, in *AttachmentLookupRequest, opts ...grpc.CallOption) (*Attachment, error)
	CheckAttachmentExists(ctx context.Context, in *AttachmentLookupRequest, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type attachmentsClient struct {
	cc grpc.ClientConnInterface
}

func NewAttachmentsClient(cc grpc.ClientConnInterface) AttachmentsClient {
	return &attachmentsClient{cc}
}

func (c *attachmentsClient) GetAttachment(ctx context.Context, in *AttachmentLookupRequest, opts ...grpc.CallOption) (*Attachment, error) {
	out := new(Attachment)
	err := c.cc.Invoke(ctx, Attachments_GetAttachment_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *attachmentsClient) CheckAttachmentExists(ctx context.Context, in *AttachmentLookupRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, Attachments_CheckAttachmentExists_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// AttachmentsServer is the server API for Attachments service.
// All implementations must embed UnimplementedAttachmentsServer
// for forward compatibility
type AttachmentsServer interface {
	GetAttachment(context.Context, *AttachmentLookupRequest) (*Attachment, error)
	CheckAttachmentExists(context.Context, *AttachmentLookupRequest) (*emptypb.Empty, error)
	mustEmbedUnimplementedAttachmentsServer()
}

// UnimplementedAttachmentsServer must be embedded to have forward compatible implementations.
type UnimplementedAttachmentsServer struct {
}

func (UnimplementedAttachmentsServer) GetAttachment(context.Context, *AttachmentLookupRequest) (*Attachment, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAttachment not implemented")
}
func (UnimplementedAttachmentsServer) CheckAttachmentExists(context.Context, *AttachmentLookupRequest) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CheckAttachmentExists not implemented")
}
func (UnimplementedAttachmentsServer) mustEmbedUnimplementedAttachmentsServer() {}

// UnsafeAttachmentsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to AttachmentsServer will
// result in compilation errors.
type UnsafeAttachmentsServer interface {
	mustEmbedUnimplementedAttachmentsServer()
}

func RegisterAttachmentsServer(s grpc.ServiceRegistrar, srv AttachmentsServer) {
	s.RegisterService(&Attachments_ServiceDesc, srv)
}

func _Attachments_GetAttachment_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AttachmentLookupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AttachmentsServer).GetAttachment(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Attachments_GetAttachment_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AttachmentsServer).GetAttachment(ctx, req.(*AttachmentLookupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Attachments_CheckAttachmentExists_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AttachmentLookupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(AttachmentsServer).CheckAttachmentExists(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Attachments_CheckAttachmentExists_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(AttachmentsServer).CheckAttachmentExists(ctx, req.(*AttachmentLookupRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Attachments_ServiceDesc is the grpc.ServiceDesc for Attachments service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Attachments_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.Attachments",
	HandlerType: (*AttachmentsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetAttachment",
			Handler:    _Attachments_GetAttachment_Handler,
		},
		{
			MethodName: "CheckAttachmentExists",
			Handler:    _Attachments_CheckAttachmentExists_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "attachments.proto",
}