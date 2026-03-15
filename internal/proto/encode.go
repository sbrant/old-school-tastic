package proto

import (
	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"google.golang.org/protobuf/proto"
)

func EncodeWantConfig(nonce uint32) ([]byte, error) {
	tr := &pb.ToRadio{
		PayloadVariant: &pb.ToRadio_WantConfigId{WantConfigId: nonce},
	}
	return proto.Marshal(tr)
}

func EncodeAdminConfigRequest(from uint32, configType pb.AdminMessage_ConfigType) ([]byte, error) {
	admin := &pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_GetConfigRequest{GetConfigRequest: configType},
	}
	payload, err := proto.Marshal(admin)
	if err != nil {
		return nil, err
	}
	tr := &pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				From: from,
				To:   from,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Portnum:      pb.PortNum_ADMIN_APP,
						Payload:      payload,
						WantResponse: true,
					},
				},
			},
		},
	}
	return proto.Marshal(tr)
}

func EncodeAdminModuleConfigRequest(from uint32, configType pb.AdminMessage_ModuleConfigType) ([]byte, error) {
	admin := &pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_GetModuleConfigRequest{GetModuleConfigRequest: configType},
	}
	payload, err := proto.Marshal(admin)
	if err != nil {
		return nil, err
	}
	tr := &pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				From: from,
				To:   from,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Portnum:      pb.PortNum_ADMIN_APP,
						Payload:      payload,
						WantResponse: true,
					},
				},
			},
		},
	}
	return proto.Marshal(tr)
}

func EncodeAdminGetChannel(from uint32, channelIndex uint32) ([]byte, error) {
	admin := &pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_GetChannelRequest{GetChannelRequest: channelIndex},
	}
	payload, err := proto.Marshal(admin)
	if err != nil {
		return nil, err
	}
	tr := &pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				From:    from,
				To:      from,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Portnum:      pb.PortNum_ADMIN_APP,
						Payload:      payload,
						WantResponse: true,
					},
				},
			},
		},
	}
	return proto.Marshal(tr)
}

func EncodeAdminBeginEdit(from uint32) ([]byte, error) {
	admin := &pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_BeginEditSettings{BeginEditSettings: true},
	}
	return encodeAdminPacket(from, admin)
}

func EncodeAdminCommitEdit(from uint32) ([]byte, error) {
	admin := &pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_CommitEditSettings{CommitEditSettings: true},
	}
	return encodeAdminPacket(from, admin)
}

func encodeAdminPacket(from uint32, admin *pb.AdminMessage) ([]byte, error) {
	payload, err := proto.Marshal(admin)
	if err != nil {
		return nil, err
	}
	tr := &pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				From:    from,
				To:      from,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Portnum:      pb.PortNum_ADMIN_APP,
						Payload:      payload,
						WantResponse: true,
					},
				},
			},
		},
	}
	return proto.Marshal(tr)
}

func EncodeAdminSetChannel(from uint32, ch *pb.Channel) ([]byte, error) {
	admin := &pb.AdminMessage{
		PayloadVariant: &pb.AdminMessage_SetChannel{SetChannel: ch},
	}
	payload, err := proto.Marshal(admin)
	if err != nil {
		return nil, err
	}
	tr := &pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				From:    from,
				To:      from,
				WantAck: true,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Portnum:      pb.PortNum_ADMIN_APP,
						Payload:      payload,
						WantResponse: true,
					},
				},
			},
		},
	}
	return proto.Marshal(tr)
}

func EncodeTextMessage(from, to uint32, text string, channel uint32) ([]byte, error) {
	tr := &pb.ToRadio{
		PayloadVariant: &pb.ToRadio_Packet{
			Packet: &pb.MeshPacket{
				From:    from,
				To:      to,
				Channel: channel,
				PayloadVariant: &pb.MeshPacket_Decoded{
					Decoded: &pb.Data{
						Portnum: pb.PortNum_TEXT_MESSAGE_APP,
						Payload: []byte(text),
					},
				},
			},
		},
	}
	return proto.Marshal(tr)
}
