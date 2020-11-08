// Code generated by protoc-gen-go. DO NOT EDIT.
// source: google/ads/googleads/v2/resources/bidding_strategy.proto

package resources

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
	wrappers "github.com/golang/protobuf/ptypes/wrappers"
	common "google.golang.org/genproto/googleapis/ads/googleads/v2/common"
	enums "google.golang.org/genproto/googleapis/ads/googleads/v2/enums"
	_ "google.golang.org/genproto/googleapis/api/annotations"
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

// A bidding strategy.
type BiddingStrategy struct {
	// Immutable. The resource name of the bidding strategy.
	// Bidding strategy resource names have the form:
	//
	// `customers/{customer_id}/biddingStrategies/{bidding_strategy_id}`
	ResourceName string `protobuf:"bytes,1,opt,name=resource_name,json=resourceName,proto3" json:"resource_name,omitempty"`
	// Output only. The ID of the bidding strategy.
	Id *wrappers.Int64Value `protobuf:"bytes,3,opt,name=id,proto3" json:"id,omitempty"`
	// The name of the bidding strategy.
	// All bidding strategies within an account must be named distinctly.
	//
	// The length of this string should be between 1 and 255, inclusive,
	// in UTF-8 bytes, (trimmed).
	Name *wrappers.StringValue `protobuf:"bytes,4,opt,name=name,proto3" json:"name,omitempty"`
	// Output only. The status of the bidding strategy.
	//
	// This field is read-only.
	Status enums.BiddingStrategyStatusEnum_BiddingStrategyStatus `protobuf:"varint,15,opt,name=status,proto3,enum=google.ads.googleads.v2.enums.BiddingStrategyStatusEnum_BiddingStrategyStatus" json:"status,omitempty"`
	// Output only. The type of the bidding strategy.
	// Create a bidding strategy by setting the bidding scheme.
	//
	// This field is read-only.
	Type enums.BiddingStrategyTypeEnum_BiddingStrategyType `protobuf:"varint,5,opt,name=type,proto3,enum=google.ads.googleads.v2.enums.BiddingStrategyTypeEnum_BiddingStrategyType" json:"type,omitempty"`
	// Output only. The number of campaigns attached to this bidding strategy.
	//
	// This field is read-only.
	CampaignCount *wrappers.Int64Value `protobuf:"bytes,13,opt,name=campaign_count,json=campaignCount,proto3" json:"campaign_count,omitempty"`
	// Output only. The number of non-removed campaigns attached to this bidding strategy.
	//
	// This field is read-only.
	NonRemovedCampaignCount *wrappers.Int64Value `protobuf:"bytes,14,opt,name=non_removed_campaign_count,json=nonRemovedCampaignCount,proto3" json:"non_removed_campaign_count,omitempty"`
	// The bidding scheme.
	//
	// Only one can be set.
	//
	// Types that are valid to be assigned to Scheme:
	//	*BiddingStrategy_EnhancedCpc
	//	*BiddingStrategy_PageOnePromoted
	//	*BiddingStrategy_TargetCpa
	//	*BiddingStrategy_TargetImpressionShare
	//	*BiddingStrategy_TargetOutrankShare
	//	*BiddingStrategy_TargetRoas
	//	*BiddingStrategy_TargetSpend
	Scheme               isBiddingStrategy_Scheme `protobuf_oneof:"scheme"`
	XXX_NoUnkeyedLiteral struct{}                 `json:"-"`
	XXX_unrecognized     []byte                   `json:"-"`
	XXX_sizecache        int32                    `json:"-"`
}

func (m *BiddingStrategy) Reset()         { *m = BiddingStrategy{} }
func (m *BiddingStrategy) String() string { return proto.CompactTextString(m) }
func (*BiddingStrategy) ProtoMessage()    {}
func (*BiddingStrategy) Descriptor() ([]byte, []int) {
	return fileDescriptor_3f01e34c4d3a0d3a, []int{0}
}

func (m *BiddingStrategy) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BiddingStrategy.Unmarshal(m, b)
}
func (m *BiddingStrategy) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BiddingStrategy.Marshal(b, m, deterministic)
}
func (m *BiddingStrategy) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BiddingStrategy.Merge(m, src)
}
func (m *BiddingStrategy) XXX_Size() int {
	return xxx_messageInfo_BiddingStrategy.Size(m)
}
func (m *BiddingStrategy) XXX_DiscardUnknown() {
	xxx_messageInfo_BiddingStrategy.DiscardUnknown(m)
}

var xxx_messageInfo_BiddingStrategy proto.InternalMessageInfo

func (m *BiddingStrategy) GetResourceName() string {
	if m != nil {
		return m.ResourceName
	}
	return ""
}

func (m *BiddingStrategy) GetId() *wrappers.Int64Value {
	if m != nil {
		return m.Id
	}
	return nil
}

func (m *BiddingStrategy) GetName() *wrappers.StringValue {
	if m != nil {
		return m.Name
	}
	return nil
}

func (m *BiddingStrategy) GetStatus() enums.BiddingStrategyStatusEnum_BiddingStrategyStatus {
	if m != nil {
		return m.Status
	}
	return enums.BiddingStrategyStatusEnum_UNSPECIFIED
}

func (m *BiddingStrategy) GetType() enums.BiddingStrategyTypeEnum_BiddingStrategyType {
	if m != nil {
		return m.Type
	}
	return enums.BiddingStrategyTypeEnum_UNSPECIFIED
}

func (m *BiddingStrategy) GetCampaignCount() *wrappers.Int64Value {
	if m != nil {
		return m.CampaignCount
	}
	return nil
}

func (m *BiddingStrategy) GetNonRemovedCampaignCount() *wrappers.Int64Value {
	if m != nil {
		return m.NonRemovedCampaignCount
	}
	return nil
}

type isBiddingStrategy_Scheme interface {
	isBiddingStrategy_Scheme()
}

type BiddingStrategy_EnhancedCpc struct {
	EnhancedCpc *common.EnhancedCpc `protobuf:"bytes,7,opt,name=enhanced_cpc,json=enhancedCpc,proto3,oneof"`
}

type BiddingStrategy_PageOnePromoted struct {
	PageOnePromoted *common.PageOnePromoted `protobuf:"bytes,8,opt,name=page_one_promoted,json=pageOnePromoted,proto3,oneof"`
}

type BiddingStrategy_TargetCpa struct {
	TargetCpa *common.TargetCpa `protobuf:"bytes,9,opt,name=target_cpa,json=targetCpa,proto3,oneof"`
}

type BiddingStrategy_TargetImpressionShare struct {
	TargetImpressionShare *common.TargetImpressionShare `protobuf:"bytes,48,opt,name=target_impression_share,json=targetImpressionShare,proto3,oneof"`
}

type BiddingStrategy_TargetOutrankShare struct {
	TargetOutrankShare *common.TargetOutrankShare `protobuf:"bytes,10,opt,name=target_outrank_share,json=targetOutrankShare,proto3,oneof"`
}

type BiddingStrategy_TargetRoas struct {
	TargetRoas *common.TargetRoas `protobuf:"bytes,11,opt,name=target_roas,json=targetRoas,proto3,oneof"`
}

type BiddingStrategy_TargetSpend struct {
	TargetSpend *common.TargetSpend `protobuf:"bytes,12,opt,name=target_spend,json=targetSpend,proto3,oneof"`
}

func (*BiddingStrategy_EnhancedCpc) isBiddingStrategy_Scheme() {}

func (*BiddingStrategy_PageOnePromoted) isBiddingStrategy_Scheme() {}

func (*BiddingStrategy_TargetCpa) isBiddingStrategy_Scheme() {}

func (*BiddingStrategy_TargetImpressionShare) isBiddingStrategy_Scheme() {}

func (*BiddingStrategy_TargetOutrankShare) isBiddingStrategy_Scheme() {}

func (*BiddingStrategy_TargetRoas) isBiddingStrategy_Scheme() {}

func (*BiddingStrategy_TargetSpend) isBiddingStrategy_Scheme() {}

func (m *BiddingStrategy) GetScheme() isBiddingStrategy_Scheme {
	if m != nil {
		return m.Scheme
	}
	return nil
}

func (m *BiddingStrategy) GetEnhancedCpc() *common.EnhancedCpc {
	if x, ok := m.GetScheme().(*BiddingStrategy_EnhancedCpc); ok {
		return x.EnhancedCpc
	}
	return nil
}

func (m *BiddingStrategy) GetPageOnePromoted() *common.PageOnePromoted {
	if x, ok := m.GetScheme().(*BiddingStrategy_PageOnePromoted); ok {
		return x.PageOnePromoted
	}
	return nil
}

func (m *BiddingStrategy) GetTargetCpa() *common.TargetCpa {
	if x, ok := m.GetScheme().(*BiddingStrategy_TargetCpa); ok {
		return x.TargetCpa
	}
	return nil
}

func (m *BiddingStrategy) GetTargetImpressionShare() *common.TargetImpressionShare {
	if x, ok := m.GetScheme().(*BiddingStrategy_TargetImpressionShare); ok {
		return x.TargetImpressionShare
	}
	return nil
}

func (m *BiddingStrategy) GetTargetOutrankShare() *common.TargetOutrankShare {
	if x, ok := m.GetScheme().(*BiddingStrategy_TargetOutrankShare); ok {
		return x.TargetOutrankShare
	}
	return nil
}

func (m *BiddingStrategy) GetTargetRoas() *common.TargetRoas {
	if x, ok := m.GetScheme().(*BiddingStrategy_TargetRoas); ok {
		return x.TargetRoas
	}
	return nil
}

func (m *BiddingStrategy) GetTargetSpend() *common.TargetSpend {
	if x, ok := m.GetScheme().(*BiddingStrategy_TargetSpend); ok {
		return x.TargetSpend
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*BiddingStrategy) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*BiddingStrategy_EnhancedCpc)(nil),
		(*BiddingStrategy_PageOnePromoted)(nil),
		(*BiddingStrategy_TargetCpa)(nil),
		(*BiddingStrategy_TargetImpressionShare)(nil),
		(*BiddingStrategy_TargetOutrankShare)(nil),
		(*BiddingStrategy_TargetRoas)(nil),
		(*BiddingStrategy_TargetSpend)(nil),
	}
}

func init() {
	proto.RegisterType((*BiddingStrategy)(nil), "google.ads.googleads.v2.resources.BiddingStrategy")
}

func init() {
	proto.RegisterFile("google/ads/googleads/v2/resources/bidding_strategy.proto", fileDescriptor_3f01e34c4d3a0d3a)
}

var fileDescriptor_3f01e34c4d3a0d3a = []byte{
	// 783 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x94, 0x95, 0xcb, 0x6e, 0xdb, 0x46,
	0x14, 0x86, 0x75, 0x71, 0xdc, 0x78, 0xe4, 0x0b, 0x4a, 0xa4, 0x08, 0xeb, 0x06, 0xad, 0x53, 0x20,
	0x80, 0x9a, 0x16, 0xa4, 0xc0, 0x5e, 0xd0, 0x30, 0x2b, 0x4a, 0x08, 0xac, 0x18, 0x68, 0x22, 0x50,
	0xae, 0x16, 0x85, 0x0c, 0x62, 0x44, 0x1e, 0x53, 0x84, 0xc5, 0x99, 0xc1, 0xcc, 0x50, 0x85, 0x61,
	0x78, 0xd1, 0x57, 0xe9, 0xb2, 0xbb, 0xbe, 0x46, 0x9f, 0xc2, 0x6b, 0x3f, 0x42, 0x57, 0x05, 0x87,
	0x43, 0xca, 0x96, 0xad, 0x5a, 0xde, 0x9d, 0xe1, 0xf9, 0xff, 0xef, 0x3f, 0x1c, 0x0e, 0x49, 0xf4,
	0x73, 0x4c, 0x69, 0x3c, 0x03, 0x1b, 0x47, 0xc2, 0x2e, 0xca, 0xbc, 0x9a, 0x3b, 0x36, 0x07, 0x41,
	0x33, 0x1e, 0x82, 0xb0, 0x27, 0x49, 0x14, 0x25, 0x24, 0x0e, 0x84, 0xe4, 0x58, 0x42, 0x7c, 0x6e,
	0x31, 0x4e, 0x25, 0x35, 0x5e, 0x16, 0x72, 0x0b, 0x47, 0xc2, 0xaa, 0x9c, 0xd6, 0xdc, 0xb1, 0x2a,
	0xe7, 0xfe, 0x77, 0xab, 0xe0, 0x21, 0x4d, 0x53, 0x4a, 0x4a, 0x72, 0x01, 0xdc, 0x7f, 0xbb, 0x4a,
	0x0d, 0x24, 0x4b, 0xef, 0x8e, 0x11, 0x08, 0x89, 0x65, 0x26, 0xb4, 0xf9, 0xcd, 0x23, 0xcd, 0xf2,
	0x9c, 0x81, 0xb6, 0x7e, 0x55, 0x5a, 0x59, 0x62, 0x9f, 0x26, 0x30, 0x8b, 0x82, 0x09, 0x4c, 0xf1,
	0x3c, 0xa1, 0x5c, 0x0b, 0x3e, 0xbf, 0x21, 0x28, 0x6f, 0x4e, 0xb7, 0xbe, 0xd4, 0x2d, 0xb5, 0x9a,
	0x64, 0xa7, 0xf6, 0xef, 0x1c, 0x33, 0x06, 0xbc, 0x1c, 0xeb, 0xc5, 0x0d, 0x2b, 0x26, 0x84, 0x4a,
	0x2c, 0x13, 0x4a, 0x74, 0xf7, 0xeb, 0xbf, 0x11, 0xda, 0xeb, 0x16, 0x93, 0x0d, 0xf5, 0x60, 0xc6,
	0xaf, 0x68, 0xa7, 0xcc, 0x08, 0x08, 0x4e, 0xc1, 0xac, 0x1f, 0xd4, 0xdb, 0x5b, 0xdd, 0xce, 0x95,
	0xf7, 0xe4, 0x5f, 0xef, 0x35, 0x6a, 0x2f, 0xb6, 0x5a, 0x57, 0x2c, 0x11, 0x56, 0x48, 0x53, 0x7b,
	0x09, 0xe4, 0x6f, 0x97, 0x98, 0x0f, 0x38, 0x05, 0xa3, 0x83, 0x1a, 0x49, 0x64, 0x36, 0x0f, 0xea,
	0xed, 0x96, 0xf3, 0x85, 0xb6, 0x5a, 0xe5, 0xd4, 0xd6, 0x7b, 0x22, 0x7f, 0xfa, 0x61, 0x84, 0x67,
	0x19, 0x74, 0x9b, 0x57, 0x5e, 0xd3, 0x6f, 0x24, 0x91, 0xd1, 0x41, 0x1b, 0x2a, 0x7f, 0x43, 0x79,
	0x5e, 0xdc, 0xf1, 0x0c, 0x25, 0x4f, 0x48, 0xac, 0x4c, 0xbe, 0x52, 0x1a, 0x67, 0x68, 0xb3, 0x78,
	0x26, 0xe6, 0xde, 0x41, 0xbd, 0xbd, 0xeb, 0x7c, 0xb0, 0x56, 0x1d, 0x11, 0xf5, 0x50, 0xac, 0xa5,
	0x89, 0x87, 0xca, 0xfb, 0x8e, 0x64, 0xe9, 0xfd, 0x9d, 0x62, 0x34, 0x1d, 0x61, 0x84, 0x68, 0x23,
	0x7f, 0x86, 0xe6, 0x13, 0x15, 0x75, 0xf4, 0xb8, 0xa8, 0xe3, 0x73, 0x06, 0xf7, 0x05, 0xe5, 0xd7,
	0x8b, 0x18, 0x05, 0x37, 0x0e, 0xd1, 0x6e, 0x88, 0x53, 0x86, 0x93, 0x98, 0x04, 0x21, 0xcd, 0x88,
	0x34, 0x77, 0xd6, 0xdc, 0xc1, 0x9d, 0xd2, 0xd7, 0xcb, 0x6d, 0xc6, 0x09, 0xda, 0x27, 0x94, 0x04,
	0x1c, 0x52, 0x3a, 0x87, 0x28, 0x58, 0x82, 0xee, 0xae, 0x09, 0x7d, 0x4e, 0x28, 0xf1, 0x0b, 0x44,
	0xef, 0x16, 0x7e, 0x80, 0xb6, 0x81, 0x4c, 0x31, 0x09, 0x73, 0x36, 0x0b, 0xcd, 0x4f, 0x14, 0xf0,
	0xdb, 0x95, 0x9b, 0x52, 0xbc, 0x7f, 0xd6, 0x3b, 0xed, 0xe9, 0xb1, 0xb0, 0x5f, 0xf3, 0x5b, 0xb0,
	0x58, 0x1a, 0x27, 0xe8, 0x53, 0x86, 0x63, 0x08, 0x28, 0x81, 0x80, 0x71, 0x9a, 0x52, 0x09, 0x91,
	0xf9, 0x54, 0x61, 0xed, 0x87, 0xb0, 0x03, 0x1c, 0xc3, 0x47, 0x02, 0x03, 0x6d, 0xeb, 0xd7, 0xfc,
	0x3d, 0x76, 0xfb, 0x92, 0x71, 0x84, 0x90, 0xc4, 0x3c, 0x06, 0x19, 0x84, 0x0c, 0x9b, 0x5b, 0x8a,
	0xfb, 0xcd, 0x43, 0xdc, 0x63, 0xe5, 0xe8, 0x31, 0xdc, 0xaf, 0xf9, 0x5b, 0xb2, 0x5c, 0x18, 0x14,
	0x3d, 0xd7, 0xac, 0x24, 0x65, 0x1c, 0x84, 0x48, 0x28, 0x09, 0xc4, 0x14, 0x73, 0x30, 0x3b, 0x0a,
	0xfc, 0xe3, 0x7a, 0xe0, 0xf7, 0x95, 0x7b, 0x98, 0x9b, 0xfb, 0x35, 0xff, 0x33, 0x79, 0x5f, 0xc3,
	0x38, 0x45, 0xcf, 0x74, 0x20, 0xcd, 0x24, 0xc7, 0xe4, 0x4c, 0xa7, 0x21, 0x95, 0xe6, 0xac, 0x97,
	0xf6, 0xb1, 0xb0, 0x96, 0x51, 0x86, 0xbc, 0x73, 0xd5, 0xf8, 0x05, 0xb5, 0x74, 0x0e, 0xa7, 0x58,
	0x98, 0x2d, 0x85, 0x7f, 0xbd, 0x1e, 0xde, 0xa7, 0x58, 0xf4, 0x6b, 0xbe, 0xde, 0xe5, 0x7c, 0x95,
	0x1f, 0x12, 0x8d, 0x13, 0x0c, 0x48, 0x64, 0x6e, 0xaf, 0x77, 0x48, 0x0a, 0xde, 0x30, 0xb7, 0xe4,
	0x87, 0x44, 0x2e, 0x96, 0xee, 0xf4, 0xda, 0x83, 0xf5, 0xbf, 0x48, 0xc6, 0x9b, 0x30, 0x13, 0x92,
	0xa6, 0xc0, 0x85, 0x7d, 0x51, 0x96, 0x97, 0xe5, 0xa7, 0x59, 0xab, 0x12, 0x10, 0xf6, 0xc5, 0xf2,
	0xd7, 0xfa, 0xb2, 0xfb, 0x14, 0x6d, 0x8a, 0x70, 0x0a, 0x29, 0x74, 0xff, 0x68, 0xa0, 0x57, 0x21,
	0x4d, 0xad, 0x07, 0xff, 0x3e, 0xdd, 0x67, 0x4b, 0xf9, 0x83, 0xfc, 0xb5, 0x1a, 0xd4, 0x7f, 0x3b,
	0xd2, 0xd6, 0x98, 0xce, 0x30, 0x89, 0x2d, 0xca, 0x63, 0x3b, 0x06, 0xa2, 0x5e, 0x3a, 0x7b, 0x71,
	0x07, 0xff, 0xf3, 0x47, 0x7c, 0x5b, 0x55, 0x7f, 0x36, 0x9a, 0x87, 0x9e, 0xf7, 0x57, 0xe3, 0xe5,
	0x61, 0x81, 0xf4, 0x22, 0x61, 0x15, 0x65, 0x5e, 0x8d, 0x1c, 0xcb, 0x2f, 0x95, 0xff, 0x94, 0x9a,
	0xb1, 0x17, 0x89, 0x71, 0xa5, 0x19, 0x8f, 0x9c, 0x71, 0xa5, 0xb9, 0x6e, 0xbc, 0x2a, 0x1a, 0xae,
	0xeb, 0x45, 0xc2, 0x75, 0x2b, 0x95, 0xeb, 0x8e, 0x1c, 0xd7, 0xad, 0x74, 0x93, 0x4d, 0x35, 0xec,
	0xf7, 0xff, 0x05, 0x00, 0x00, 0xff, 0xff, 0x9f, 0x64, 0x85, 0xfd, 0xbd, 0x07, 0x00, 0x00,
}