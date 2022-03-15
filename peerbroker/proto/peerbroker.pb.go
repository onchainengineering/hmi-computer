// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.26.0
// 	protoc        v3.6.1
// source: peerbroker/proto/peerbroker.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type WebRTCSessionDescription struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	SdpType int32  `protobuf:"varint,1,opt,name=sdp_type,json=sdpType,proto3" json:"sdp_type,omitempty"`
	Sdp     string `protobuf:"bytes,2,opt,name=sdp,proto3" json:"sdp,omitempty"`
}

func (x *WebRTCSessionDescription) Reset() {
	*x = WebRTCSessionDescription{}
	if protoimpl.UnsafeEnabled {
		mi := &file_peerbroker_proto_peerbroker_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *WebRTCSessionDescription) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*WebRTCSessionDescription) ProtoMessage() {}

func (x *WebRTCSessionDescription) ProtoReflect() protoreflect.Message {
	mi := &file_peerbroker_proto_peerbroker_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use WebRTCSessionDescription.ProtoReflect.Descriptor instead.
func (*WebRTCSessionDescription) Descriptor() ([]byte, []int) {
	return file_peerbroker_proto_peerbroker_proto_rawDescGZIP(), []int{0}
}

func (x *WebRTCSessionDescription) GetSdpType() int32 {
	if x != nil {
		return x.SdpType
	}
	return 0
}

func (x *WebRTCSessionDescription) GetSdp() string {
	if x != nil {
		return x.Sdp
	}
	return ""
}

type Exchange struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Message:
	//	*Exchange_Sdp
	//	*Exchange_IceCandidate
	Message isExchange_Message `protobuf_oneof:"message"`
}

func (x *Exchange) Reset() {
	*x = Exchange{}
	if protoimpl.UnsafeEnabled {
		mi := &file_peerbroker_proto_peerbroker_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Exchange) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Exchange) ProtoMessage() {}

func (x *Exchange) ProtoReflect() protoreflect.Message {
	mi := &file_peerbroker_proto_peerbroker_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Exchange.ProtoReflect.Descriptor instead.
func (*Exchange) Descriptor() ([]byte, []int) {
	return file_peerbroker_proto_peerbroker_proto_rawDescGZIP(), []int{1}
}

func (m *Exchange) GetMessage() isExchange_Message {
	if m != nil {
		return m.Message
	}
	return nil
}

func (x *Exchange) GetSdp() *WebRTCSessionDescription {
	if x, ok := x.GetMessage().(*Exchange_Sdp); ok {
		return x.Sdp
	}
	return nil
}

func (x *Exchange) GetIceCandidate() string {
	if x, ok := x.GetMessage().(*Exchange_IceCandidate); ok {
		return x.IceCandidate
	}
	return ""
}

type isExchange_Message interface {
	isExchange_Message()
}

type Exchange_Sdp struct {
	Sdp *WebRTCSessionDescription `protobuf:"bytes,1,opt,name=sdp,proto3,oneof"`
}

type Exchange_IceCandidate struct {
	IceCandidate string `protobuf:"bytes,2,opt,name=ice_candidate,json=iceCandidate,proto3,oneof"`
}

func (*Exchange_Sdp) isExchange_Message() {}

func (*Exchange_IceCandidate) isExchange_Message() {}

var File_peerbroker_proto_peerbroker_proto protoreflect.FileDescriptor

var file_peerbroker_proto_peerbroker_proto_rawDesc = []byte{
	0x0a, 0x21, 0x70, 0x65, 0x65, 0x72, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x2f, 0x70, 0x65, 0x65, 0x72, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x70, 0x65, 0x65, 0x72, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x22,
	0x47, 0x0a, 0x18, 0x57, 0x65, 0x62, 0x52, 0x54, 0x43, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e,
	0x44, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x19, 0x0a, 0x08, 0x73,
	0x64, 0x70, 0x5f, 0x74, 0x79, 0x70, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x73,
	0x64, 0x70, 0x54, 0x79, 0x70, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x64, 0x70, 0x18, 0x02, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x03, 0x73, 0x64, 0x70, 0x22, 0x76, 0x0a, 0x08, 0x45, 0x78, 0x63, 0x68,
	0x61, 0x6e, 0x67, 0x65, 0x12, 0x38, 0x0a, 0x03, 0x73, 0x64, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x24, 0x2e, 0x70, 0x65, 0x65, 0x72, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x57,
	0x65, 0x62, 0x52, 0x54, 0x43, 0x53, 0x65, 0x73, 0x73, 0x69, 0x6f, 0x6e, 0x44, 0x65, 0x73, 0x63,
	0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x48, 0x00, 0x52, 0x03, 0x73, 0x64, 0x70, 0x12, 0x25,
	0x0a, 0x0d, 0x69, 0x63, 0x65, 0x5f, 0x63, 0x61, 0x6e, 0x64, 0x69, 0x64, 0x61, 0x74, 0x65, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x0c, 0x69, 0x63, 0x65, 0x43, 0x61, 0x6e, 0x64,
	0x69, 0x64, 0x61, 0x74, 0x65, 0x42, 0x09, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x32, 0x53, 0x0a, 0x0a, 0x50, 0x65, 0x65, 0x72, 0x42, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x12, 0x45,
	0x0a, 0x13, 0x4e, 0x65, 0x67, 0x6f, 0x74, 0x69, 0x61, 0x74, 0x65, 0x43, 0x6f, 0x6e, 0x6e, 0x65,
	0x63, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x14, 0x2e, 0x70, 0x65, 0x65, 0x72, 0x62, 0x72, 0x6f, 0x6b,
	0x65, 0x72, 0x2e, 0x45, 0x78, 0x63, 0x68, 0x61, 0x6e, 0x67, 0x65, 0x1a, 0x14, 0x2e, 0x70, 0x65,
	0x65, 0x72, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2e, 0x45, 0x78, 0x63, 0x68, 0x61, 0x6e, 0x67,
	0x65, 0x28, 0x01, 0x30, 0x01, 0x42, 0x29, 0x5a, 0x27, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x63, 0x6f, 0x64, 0x65, 0x72, 0x2f, 0x63, 0x6f, 0x64, 0x65, 0x72, 0x2f,
	0x70, 0x65, 0x65, 0x72, 0x62, 0x72, 0x6f, 0x6b, 0x65, 0x72, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_peerbroker_proto_peerbroker_proto_rawDescOnce sync.Once
	file_peerbroker_proto_peerbroker_proto_rawDescData = file_peerbroker_proto_peerbroker_proto_rawDesc
)

func file_peerbroker_proto_peerbroker_proto_rawDescGZIP() []byte {
	file_peerbroker_proto_peerbroker_proto_rawDescOnce.Do(func() {
		file_peerbroker_proto_peerbroker_proto_rawDescData = protoimpl.X.CompressGZIP(file_peerbroker_proto_peerbroker_proto_rawDescData)
	})
	return file_peerbroker_proto_peerbroker_proto_rawDescData
}

var file_peerbroker_proto_peerbroker_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_peerbroker_proto_peerbroker_proto_goTypes = []interface{}{
	(*WebRTCSessionDescription)(nil), // 0: peerbroker.WebRTCSessionDescription
	(*Exchange)(nil),                 // 1: peerbroker.Exchange
}
var file_peerbroker_proto_peerbroker_proto_depIdxs = []int32{
	0, // 0: peerbroker.Exchange.sdp:type_name -> peerbroker.WebRTCSessionDescription
	1, // 1: peerbroker.PeerBroker.NegotiateConnection:input_type -> peerbroker.Exchange
	1, // 2: peerbroker.PeerBroker.NegotiateConnection:output_type -> peerbroker.Exchange
	2, // [2:3] is the sub-list for method output_type
	1, // [1:2] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_peerbroker_proto_peerbroker_proto_init() }
func file_peerbroker_proto_peerbroker_proto_init() {
	if File_peerbroker_proto_peerbroker_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_peerbroker_proto_peerbroker_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*WebRTCSessionDescription); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_peerbroker_proto_peerbroker_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Exchange); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_peerbroker_proto_peerbroker_proto_msgTypes[1].OneofWrappers = []interface{}{
		(*Exchange_Sdp)(nil),
		(*Exchange_IceCandidate)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_peerbroker_proto_peerbroker_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_peerbroker_proto_peerbroker_proto_goTypes,
		DependencyIndexes: file_peerbroker_proto_peerbroker_proto_depIdxs,
		MessageInfos:      file_peerbroker_proto_peerbroker_proto_msgTypes,
	}.Build()
	File_peerbroker_proto_peerbroker_proto = out.File
	file_peerbroker_proto_peerbroker_proto_rawDesc = nil
	file_peerbroker_proto_peerbroker_proto_goTypes = nil
	file_peerbroker_proto_peerbroker_proto_depIdxs = nil
}
