// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v2/services/user_list_service.proto

package services

import (
	context "context"
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	resources "google.golang.org/genproto/googleapis/ads/googleads/v2/resources"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	status "google.golang.org/genproto/googleapis/rpc/status"
	field_mask "google.golang.org/genproto/protobuf/field_mask"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status1 "google.golang.org/grpc/status"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

// Request message for [UserListService.GetUserList][google.ads.googleads.v2.services.UserListService.GetUserList].
type GetUserListRequest struct {
	// Required. The resource name of the user list to fetch.
	ResourceName         string   `protobuf:"bytes,1,opt,name=resource_name,json=resourceName,proto3" json:"resource_name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *GetUserListRequest) Reset()         { *m = GetUserListRequest{} }
func (m *GetUserListRequest) String() string { return proto.CompactTextString(m) }
func (*GetUserListRequest) ProtoMessage()    {}
func (*GetUserListRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_0075e5e715d4f105, []int{0}
}

func (m *GetUserListRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_GetUserListRequest.Unmarshal(m, b)
}
func (m *GetUserListRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_GetUserListRequest.Marshal(b, m, deterministic)
}
func (m *GetUserListRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_GetUserListRequest.Merge(m, src)
}
func (m *GetUserListRequest) XXX_Size() int {
	return xxx_messageInfo_GetUserListRequest.Size(m)
}
func (m *GetUserListRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_GetUserListRequest.DiscardUnknown(m)
}

var xxx_messageInfo_GetUserListRequest proto.InternalMessageInfo

func (m *GetUserListRequest) GetResourceName() string {
	if m != nil {
		return m.ResourceName
	}
	return ""
}

// Request message for [UserListService.MutateUserLists][google.ads.googleads.v2.services.UserListService.MutateUserLists].
type MutateUserListsRequest struct {
	// Required. The ID of the customer whose user lists are being modified.
	CustomerId string `protobuf:"bytes,1,opt,name=customer_id,json=customerId,proto3" json:"customer_id,omitempty"`
	// Required. The list of operations to perform on individual user lists.
	Operations []*UserListOperation `protobuf:"bytes,2,rep,name=operations,proto3" json:"operations,omitempty"`
	// If true, successful operations will be carried out and invalid
	// operations will return errors. If false, all operations will be carried
	// out in one transaction if and only if they are all valid.
	// Default is false.
	PartialFailure bool `protobuf:"varint,3,opt,name=partial_failure,json=partialFailure,proto3" json:"partial_failure,omitempty"`
	// If true, the request is validated but not executed. Only errors are
	// returned, not results.
	ValidateOnly         bool     `protobuf:"varint,4,opt,name=validate_only,json=validateOnly,proto3" json:"validate_only,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MutateUserListsRequest) Reset()         { *m = MutateUserListsRequest{} }
func (m *MutateUserListsRequest) String() string { return proto.CompactTextString(m) }
func (*MutateUserListsRequest) ProtoMessage()    {}
func (*MutateUserListsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_0075e5e715d4f105, []int{1}
}

func (m *MutateUserListsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MutateUserListsRequest.Unmarshal(m, b)
}
func (m *MutateUserListsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MutateUserListsRequest.Marshal(b, m, deterministic)
}
func (m *MutateUserListsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MutateUserListsRequest.Merge(m, src)
}
func (m *MutateUserListsRequest) XXX_Size() int {
	return xxx_messageInfo_MutateUserListsRequest.Size(m)
}
func (m *MutateUserListsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_MutateUserListsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_MutateUserListsRequest proto.InternalMessageInfo

func (m *MutateUserListsRequest) GetCustomerId() string {
	if m != nil {
		return m.CustomerId
	}
	return ""
}

func (m *MutateUserListsRequest) GetOperations() []*UserListOperation {
	if m != nil {
		return m.Operations
	}
	return nil
}

func (m *MutateUserListsRequest) GetPartialFailure() bool {
	if m != nil {
		return m.PartialFailure
	}
	return false
}

func (m *MutateUserListsRequest) GetValidateOnly() bool {
	if m != nil {
		return m.ValidateOnly
	}
	return false
}

// A single operation (create, update) on a user list.
type UserListOperation struct {
	// FieldMask that determines which resource fields are modified in an update.
	UpdateMask *field_mask.FieldMask `protobuf:"bytes,4,opt,name=update_mask,json=updateMask,proto3" json:"update_mask,omitempty"`
	// The mutate operation.
	//
	// Types that are valid to be assigned to Operation:
	//	*UserListOperation_Create
	//	*UserListOperation_Update
	//	*UserListOperation_Remove
	Operation            isUserListOperation_Operation `protobuf_oneof:"operation"`
	XXX_NoUnkeyedLiteral struct{}                      `json:"-"`
	XXX_unrecognized     []byte                        `json:"-"`
	XXX_sizecache        int32                         `json:"-"`
}

func (m *UserListOperation) Reset()         { *m = UserListOperation{} }
func (m *UserListOperation) String() string { return proto.CompactTextString(m) }
func (*UserListOperation) ProtoMessage()    {}
func (*UserListOperation) Descriptor() ([]byte, []int) {
	return fileDescriptor_0075e5e715d4f105, []int{2}
}

func (m *UserListOperation) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_UserListOperation.Unmarshal(m, b)
}
func (m *UserListOperation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_UserListOperation.Marshal(b, m, deterministic)
}
func (m *UserListOperation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_UserListOperation.Merge(m, src)
}
func (m *UserListOperation) XXX_Size() int {
	return xxx_messageInfo_UserListOperation.Size(m)
}
func (m *UserListOperation) XXX_DiscardUnknown() {
	xxx_messageInfo_UserListOperation.DiscardUnknown(m)
}

var xxx_messageInfo_UserListOperation proto.InternalMessageInfo

func (m *UserListOperation) GetUpdateMask() *field_mask.FieldMask {
	if m != nil {
		return m.UpdateMask
	}
	return nil
}

type isUserListOperation_Operation interface {
	isUserListOperation_Operation()
}

type UserListOperation_Create struct {
	Create *resources.UserList `protobuf:"bytes,1,opt,name=create,proto3,oneof"`
}

type UserListOperation_Update struct {
	Update *resources.UserList `protobuf:"bytes,2,opt,name=update,proto3,oneof"`
}

type UserListOperation_Remove struct {
	Remove string `protobuf:"bytes,3,opt,name=remove,proto3,oneof"`
}

func (*UserListOperation_Create) isUserListOperation_Operation() {}

func (*UserListOperation_Update) isUserListOperation_Operation() {}

func (*UserListOperation_Remove) isUserListOperation_Operation() {}

func (m *UserListOperation) GetOperation() isUserListOperation_Operation {
	if m != nil {
		return m.Operation
	}
	return nil
}

func (m *UserListOperation) GetCreate() *resources.UserList {
	if x, ok := m.GetOperation().(*UserListOperation_Create); ok {
		return x.Create
	}
	return nil
}

func (m *UserListOperation) GetUpdate() *resources.UserList {
	if x, ok := m.GetOperation().(*UserListOperation_Update); ok {
		return x.Update
	}
	return nil
}

func (m *UserListOperation) GetRemove() string {
	if x, ok := m.GetOperation().(*UserListOperation_Remove); ok {
		return x.Remove
	}
	return ""
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*UserListOperation) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*UserListOperation_Create)(nil),
		(*UserListOperation_Update)(nil),
		(*UserListOperation_Remove)(nil),
	}
}

// Response message for user list mutate.
type MutateUserListsResponse struct {
	// Errors that pertain to operation failures in the partial failure mode.
	// Returned only when partial_failure = true and all errors occur inside the
	// operations. If any errors occur outside the operations (e.g. auth errors),
	// we return an RPC level error.
	PartialFailureError *status.Status `protobuf:"bytes,3,opt,name=partial_failure_error,json=partialFailureError,proto3" json:"partial_failure_error,omitempty"`
	// All results for the mutate.
	Results              []*MutateUserListResult `protobuf:"bytes,2,rep,name=results,proto3" json:"results,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                `json:"-"`
	XXX_unrecognized     []byte                  `json:"-"`
	XXX_sizecache        int32                   `json:"-"`
}

func (m *MutateUserListsResponse) Reset()         { *m = MutateUserListsResponse{} }
func (m *MutateUserListsResponse) String() string { return proto.CompactTextString(m) }
func (*MutateUserListsResponse) ProtoMessage()    {}
func (*MutateUserListsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_0075e5e715d4f105, []int{3}
}

func (m *MutateUserListsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MutateUserListsResponse.Unmarshal(m, b)
}
func (m *MutateUserListsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MutateUserListsResponse.Marshal(b, m, deterministic)
}
func (m *MutateUserListsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MutateUserListsResponse.Merge(m, src)
}
func (m *MutateUserListsResponse) XXX_Size() int {
	return xxx_messageInfo_MutateUserListsResponse.Size(m)
}
func (m *MutateUserListsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_MutateUserListsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_MutateUserListsResponse proto.InternalMessageInfo

func (m *MutateUserListsResponse) GetPartialFailureError() *status.Status {
	if m != nil {
		return m.PartialFailureError
	}
	return nil
}

func (m *MutateUserListsResponse) GetResults() []*MutateUserListResult {
	if m != nil {
		return m.Results
	}
	return nil
}

// The result for the user list mutate.
type MutateUserListResult struct {
	// Returned for successful operations.
	ResourceName         string   `protobuf:"bytes,1,opt,name=resource_name,json=resourceName,proto3" json:"resource_name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *MutateUserListResult) Reset()         { *m = MutateUserListResult{} }
func (m *MutateUserListResult) String() string { return proto.CompactTextString(m) }
func (*MutateUserListResult) ProtoMessage()    {}
func (*MutateUserListResult) Descriptor() ([]byte, []int) {
	return fileDescriptor_0075e5e715d4f105, []int{4}
}

func (m *MutateUserListResult) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_MutateUserListResult.Unmarshal(m, b)
}
func (m *MutateUserListResult) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_MutateUserListResult.Marshal(b, m, deterministic)
}
func (m *MutateUserListResult) XXX_Merge(src proto.Message) {
	xxx_messageInfo_MutateUserListResult.Merge(m, src)
}
func (m *MutateUserListResult) XXX_Size() int {
	return xxx_messageInfo_MutateUserListResult.Size(m)
}
func (m *MutateUserListResult) XXX_DiscardUnknown() {
	xxx_messageInfo_MutateUserListResult.DiscardUnknown(m)
}

var xxx_messageInfo_MutateUserListResult proto.InternalMessageInfo

func (m *MutateUserListResult) GetResourceName() string {
	if m != nil {
		return m.ResourceName
	}
	return ""
}

func init() {
	proto.RegisterType((*GetUserListRequest)(nil), "google.ads.googleads.v2.services.GetUserListRequest")
	proto.RegisterType((*MutateUserListsRequest)(nil), "google.ads.googleads.v2.services.MutateUserListsRequest")
	proto.RegisterType((*UserListOperation)(nil), "google.ads.googleads.v2.services.UserListOperation")
	proto.RegisterType((*MutateUserListsResponse)(nil), "google.ads.googleads.v2.services.MutateUserListsResponse")
	proto.RegisterType((*MutateUserListResult)(nil), "google.ads.googleads.v2.services.MutateUserListResult")
}

func init() {
	proto.RegisterFile("google/ads/googleads/v2/services/user_list_service.proto", fileDescriptor_0075e5e715d4f105)
}

var fileDescriptor_0075e5e715d4f105 = []byte{
	// 782 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x55, 0x4d, 0x4f, 0xdc, 0x46,
	0x18, 0xae, 0xbd, 0x15, 0x2d, 0x63, 0x28, 0xea, 0x94, 0xc2, 0x76, 0x5b, 0xa9, 0x5b, 0x83, 0x54,
	0xba, 0xb4, 0xb6, 0x6a, 0xaa, 0x8a, 0x18, 0x71, 0xf0, 0x46, 0x7c, 0x44, 0x0a, 0x1f, 0x32, 0x0a,
	0x8a, 0xa2, 0x95, 0xac, 0xc1, 0x1e, 0x36, 0x16, 0xb6, 0xc7, 0x99, 0x19, 0xaf, 0x84, 0x10, 0x97,
	0x48, 0xf9, 0x05, 0xf9, 0x07, 0x39, 0xe6, 0x1a, 0xe5, 0x17, 0xe4, 0xc6, 0x35, 0x37, 0x4e, 0x1c,
	0xa2, 0x1c, 0xf2, 0x13, 0x92, 0x4b, 0x64, 0x8f, 0x67, 0xbf, 0x60, 0xb5, 0x82, 0xdb, 0xeb, 0x99,
	0xe7, 0x79, 0xde, 0xf7, 0x9d, 0x67, 0xde, 0x31, 0x58, 0x6d, 0x13, 0xd2, 0x8e, 0xb0, 0x89, 0x02,
	0x66, 0x8a, 0x30, 0x8f, 0x3a, 0x96, 0xc9, 0x30, 0xed, 0x84, 0x3e, 0x66, 0x66, 0xc6, 0x30, 0xf5,
	0xa2, 0x90, 0x71, 0xaf, 0x5c, 0x32, 0x52, 0x4a, 0x38, 0x81, 0x75, 0x01, 0x37, 0x50, 0xc0, 0x8c,
	0x2e, 0xd3, 0xe8, 0x58, 0x86, 0x64, 0xd6, 0xfe, 0x1d, 0xa5, 0x4d, 0x31, 0x23, 0x19, 0x1d, 0x10,
	0x17, 0xa2, 0xb5, 0xdf, 0x24, 0x25, 0x0d, 0x4d, 0x94, 0x24, 0x84, 0x23, 0x1e, 0x92, 0x84, 0x95,
	0xbb, 0xf3, 0x7d, 0xbb, 0x7e, 0x14, 0xe2, 0x44, 0xd2, 0x7e, 0xef, 0xdb, 0x38, 0x0e, 0x71, 0x14,
	0x78, 0x47, 0xf8, 0x29, 0xea, 0x84, 0x84, 0x96, 0x80, 0x5f, 0xfa, 0x00, 0x32, 0x7b, 0xb9, 0x55,
	0xf6, 0x61, 0x16, 0x5f, 0x47, 0xd9, 0x71, 0x29, 0x10, 0x23, 0x76, 0x32, 0x94, 0x96, 0xa6, 0xbe,
	0xc9, 0x38, 0xe2, 0x59, 0x59, 0x8f, 0x1e, 0x00, 0xb8, 0x85, 0xf9, 0x23, 0x86, 0xe9, 0xc3, 0x90,
	0x71, 0x17, 0x3f, 0xcb, 0x30, 0xe3, 0x70, 0x17, 0x4c, 0xcb, 0x14, 0x5e, 0x82, 0x62, 0x5c, 0x55,
	0xea, 0xca, 0xd2, 0x64, 0xf3, 0xaf, 0x2b, 0x47, 0xfd, 0xec, 0x2c, 0x80, 0x3f, 0x7a, 0x87, 0x55,
	0x46, 0x69, 0xc8, 0x0c, 0x9f, 0xc4, 0x66, 0x57, 0x68, 0x4a, 0xf2, 0x77, 0x51, 0x8c, 0xf5, 0x8f,
	0x0a, 0x98, 0xdb, 0xc9, 0x38, 0xe2, 0x58, 0x02, 0x98, 0x4c, 0xb5, 0x08, 0x34, 0x3f, 0x63, 0x9c,
	0xc4, 0x98, 0x7a, 0x61, 0x50, 0x26, 0xaa, 0x5c, 0x39, 0xaa, 0x0b, 0xe4, 0xfa, 0x83, 0x00, 0x3e,
	0x06, 0x80, 0xa4, 0x98, 0x8a, 0xa3, 0xac, 0xaa, 0xf5, 0xca, 0x92, 0x66, 0xad, 0x18, 0xe3, 0xec,
	0x33, 0x64, 0xb6, 0x3d, 0xc9, 0x2d, 0x95, 0x7b, 0x5a, 0xf0, 0x4f, 0x30, 0x93, 0x22, 0xca, 0x43,
	0x14, 0x79, 0xc7, 0x28, 0x8c, 0x32, 0x8a, 0xab, 0x95, 0xba, 0xb2, 0xf4, 0xbd, 0xfb, 0x43, 0xb9,
	0xbc, 0x29, 0x56, 0xe1, 0x02, 0x98, 0xee, 0xa0, 0x28, 0x0c, 0x10, 0xc7, 0x1e, 0x49, 0xa2, 0xd3,
	0xea, 0xb7, 0x05, 0x6c, 0x4a, 0x2e, 0xee, 0x25, 0xd1, 0xa9, 0xfe, 0x42, 0x05, 0x3f, 0x5e, 0x4b,
	0x0a, 0xd7, 0x80, 0x96, 0xa5, 0x05, 0x31, 0xb7, 0xa4, 0x20, 0x6a, 0x56, 0x4d, 0x96, 0x2f, 0x5d,
	0x33, 0x36, 0x73, 0xd7, 0x76, 0x10, 0x3b, 0x71, 0x81, 0x80, 0xe7, 0x31, 0xdc, 0x00, 0x13, 0x3e,
	0xc5, 0x88, 0x0b, 0x13, 0x34, 0x6b, 0x79, 0x64, 0xdb, 0xdd, 0x3b, 0xd9, 0xed, 0x7b, 0xfb, 0x1b,
	0xb7, 0x24, 0xe7, 0x32, 0x42, 0xb4, 0xaa, 0xde, 0x49, 0x46, 0x90, 0x61, 0x15, 0x4c, 0x50, 0x1c,
	0x93, 0x8e, 0x38, 0xa5, 0xc9, 0x7c, 0x47, 0x7c, 0x37, 0x35, 0x30, 0xd9, 0x3d, 0x56, 0xfd, 0x8d,
	0x02, 0xe6, 0xaf, 0x19, 0xce, 0x52, 0x92, 0x30, 0x0c, 0x37, 0xc1, 0xcf, 0x43, 0x27, 0xee, 0x61,
	0x4a, 0x09, 0x2d, 0x14, 0x35, 0x0b, 0xca, 0xc2, 0x68, 0xea, 0x1b, 0x07, 0xc5, 0x5d, 0x75, 0x7f,
	0x1a, 0xf4, 0x62, 0x23, 0x87, 0xc3, 0x7d, 0xf0, 0x1d, 0xc5, 0x2c, 0x8b, 0xb8, 0xbc, 0x10, 0xff,
	0x8f, 0xbf, 0x10, 0x83, 0x35, 0xb9, 0x05, 0xdd, 0x95, 0x32, 0xfa, 0x1a, 0x98, 0xbd, 0x09, 0x90,
	0x5b, 0x7f, 0xc3, 0x38, 0x0c, 0xde, 0x71, 0xeb, 0x5d, 0x05, 0xcc, 0x48, 0xde, 0x81, 0xc8, 0x07,
	0xdf, 0x2a, 0x40, 0xeb, 0x1b, 0x2f, 0xf8, 0xdf, 0xf8, 0x0a, 0xaf, 0x4f, 0x63, 0xed, 0x36, 0x56,
	0xe9, 0xf7, 0x2f, 0x9d, 0xc1, 0x62, 0x9f, 0xbf, 0xff, 0xf0, 0x52, 0xfd, 0x07, 0x2e, 0xe7, 0xaf,
	0xd6, 0xd9, 0xc0, 0xce, 0xba, 0x1c, 0x30, 0x66, 0x36, 0x8a, 0x67, 0xac, 0xf0, 0xc9, 0x6c, 0x9c,
	0xc3, 0x4b, 0x05, 0xcc, 0x0c, 0xd9, 0x07, 0x57, 0x6f, 0x7b, 0xba, 0x72, 0xc4, 0x6b, 0xf7, 0xee,
	0xc0, 0x14, 0x77, 0x45, 0x77, 0x2f, 0x9d, 0xb9, 0xbe, 0xe7, 0xe1, 0xef, 0xde, 0xe0, 0x16, 0x6d,
	0xad, 0xe8, 0x46, 0xde, 0x56, 0xaf, 0x8f, 0xb3, 0x3e, 0xf0, 0x7a, 0xe3, 0xbc, 0xd7, 0x95, 0x1d,
	0x17, 0x19, 0x6c, 0xa5, 0x51, 0xfb, 0xf5, 0xc2, 0xa9, 0x8e, 0x7a, 0xc0, 0x9a, 0x5f, 0x14, 0xb0,
	0xe8, 0x93, 0x78, 0x6c, 0xc5, 0xcd, 0xd9, 0x21, 0xaf, 0xf7, 0xf3, 0x29, 0xde, 0x57, 0x9e, 0x6c,
	0x97, 0xcc, 0x36, 0x89, 0x50, 0xd2, 0x36, 0x08, 0x6d, 0x9b, 0x6d, 0x9c, 0x14, 0x33, 0x6e, 0xf6,
	0x72, 0x8d, 0xfe, 0x59, 0xad, 0xc9, 0xe0, 0x95, 0x5a, 0xd9, 0x72, 0x9c, 0xd7, 0x6a, 0x7d, 0x4b,
	0x08, 0x3a, 0x01, 0x33, 0x44, 0x98, 0x47, 0x87, 0x96, 0x51, 0x26, 0x66, 0x17, 0x12, 0xd2, 0x72,
	0x02, 0xd6, 0xea, 0x42, 0x5a, 0x87, 0x56, 0x4b, 0x42, 0x3e, 0xa9, 0x8b, 0x62, 0xdd, 0xb6, 0x9d,
	0x80, 0xd9, 0x76, 0x17, 0x64, 0xdb, 0x87, 0x96, 0x6d, 0x4b, 0xd8, 0xd1, 0x44, 0x51, 0xe7, 0xca,
	0xd7, 0x00, 0x00, 0x00, 0xff, 0xff, 0xac, 0x8c, 0x03, 0xd0, 0x53, 0x07, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConnInterface

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion6

// UserListServiceClient is the client API for UserListService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type UserListServiceClient interface {
	// Returns the requested user list.
	GetUserList(ctx context.Context, in *GetUserListRequest, opts ...grpc.CallOption) (*resources.UserList, error)
	// Creates or updates user lists. Operation statuses are returned.
	MutateUserLists(ctx context.Context, in *MutateUserListsRequest, opts ...grpc.CallOption) (*MutateUserListsResponse, error)
}

type userListServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewUserListServiceClient(cc grpc.ClientConnInterface) UserListServiceClient {
	return &userListServiceClient{cc}
}

func (c *userListServiceClient) GetUserList(ctx context.Context, in *GetUserListRequest, opts ...grpc.CallOption) (*resources.UserList, error) {
	out := new(resources.UserList)
	err := c.cc.Invoke(ctx, "/google.ads.googleads.v2.services.UserListService/GetUserList", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *userListServiceClient) MutateUserLists(ctx context.Context, in *MutateUserListsRequest, opts ...grpc.CallOption) (*MutateUserListsResponse, error) {
	out := new(MutateUserListsResponse)
	err := c.cc.Invoke(ctx, "/google.ads.googleads.v2.services.UserListService/MutateUserLists", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UserListServiceServer is the server API for UserListService service.
type UserListServiceServer interface {
	// Returns the requested user list.
	GetUserList(context.Context, *GetUserListRequest) (*resources.UserList, error)
	// Creates or updates user lists. Operation statuses are returned.
	MutateUserLists(context.Context, *MutateUserListsRequest) (*MutateUserListsResponse, error)
}

// UnimplementedUserListServiceServer can be embedded to have forward compatible implementations.
type UnimplementedUserListServiceServer struct {
}

func (*UnimplementedUserListServiceServer) GetUserList(ctx context.Context, req *GetUserListRequest) (*resources.UserList, error) {
	return nil, status1.Errorf(codes.Unimplemented, "method GetUserList not implemented")
}
func (*UnimplementedUserListServiceServer) MutateUserLists(ctx context.Context, req *MutateUserListsRequest) (*MutateUserListsResponse, error) {
	return nil, status1.Errorf(codes.Unimplemented, "method MutateUserLists not implemented")
}

func RegisterUserListServiceServer(s *grpc.Server, srv UserListServiceServer) {
	s.RegisterService(&_UserListService_serviceDesc, srv)
}

func _UserListService_GetUserList_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetUserListRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserListServiceServer).GetUserList(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/google.ads.googleads.v2.services.UserListService/GetUserList",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserListServiceServer).GetUserList(ctx, req.(*GetUserListRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _UserListService_MutateUserLists_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MutateUserListsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(UserListServiceServer).MutateUserLists(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/google.ads.googleads.v2.services.UserListService/MutateUserLists",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(UserListServiceServer).MutateUserLists(ctx, req.(*MutateUserListsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _UserListService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "google.ads.googleads.v2.services.UserListService",
	HandlerType: (*UserListServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetUserList",
			Handler:    _UserListService_GetUserList_Handler,
		},
		{
			MethodName: "MutateUserLists",
			Handler:    _UserListService_MutateUserLists_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "google/ads/googleads/v2/services/user_list_service.proto",
}