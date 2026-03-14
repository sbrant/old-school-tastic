package proto

import (
	"fmt"

	pb "buf.build/gen/go/meshtastic/protobufs/protocolbuffers/go/meshtastic"
	"google.golang.org/protobuf/proto"
)

type Packet struct {
	FromRadio *pb.FromRadio
	Mesh      *pb.MeshPacket
	Portnum   pb.PortNum
	Payload   any
	Raw       []byte
}

func DecodeFromRadio(raw []byte) (*Packet, error) {
	fr := &pb.FromRadio{}
	if err := proto.Unmarshal(raw, fr); err != nil {
		return nil, fmt.Errorf("unmarshal FromRadio: %w", err)
	}

	pkt := &Packet{
		FromRadio: fr,
		Raw:       raw,
	}

	switch v := fr.PayloadVariant.(type) {
	case *pb.FromRadio_Packet:
		pkt.Mesh = v.Packet
		if d := v.Packet.GetDecoded(); d != nil {
			pkt.Portnum = d.GetPortnum()
			pkt.Payload = decodePayload(d.GetPortnum(), d.GetPayload())
		}
	}

	return pkt, nil
}

func decodePayload(portnum pb.PortNum, data []byte) any {
	switch portnum {
	case pb.PortNum_TEXT_MESSAGE_APP:
		return string(data)

	case pb.PortNum_POSITION_APP:
		pos := &pb.Position{}
		if err := proto.Unmarshal(data, pos); err == nil {
			return pos
		}

	case pb.PortNum_NODEINFO_APP:
		user := &pb.User{}
		if err := proto.Unmarshal(data, user); err == nil {
			return user
		}

	case pb.PortNum_TELEMETRY_APP:
		tel := &pb.Telemetry{}
		if err := proto.Unmarshal(data, tel); err == nil {
			return tel
		}

	case pb.PortNum_ROUTING_APP:
		routing := &pb.Routing{}
		if err := proto.Unmarshal(data, routing); err == nil {
			return routing
		}

	case pb.PortNum_TRACEROUTE_APP:
		route := &pb.RouteDiscovery{}
		if err := proto.Unmarshal(data, route); err == nil {
			return route
		}

	case pb.PortNum_ADMIN_APP:
		admin := &pb.AdminMessage{}
		if err := proto.Unmarshal(data, admin); err == nil {
			return admin
		}

	case pb.PortNum_WAYPOINT_APP:
		wp := &pb.Waypoint{}
		if err := proto.Unmarshal(data, wp); err == nil {
			return wp
		}

	case pb.PortNum_NEIGHBORINFO_APP:
		ni := &pb.NeighborInfo{}
		if err := proto.Unmarshal(data, ni); err == nil {
			return ni
		}

	case pb.PortNum_STORE_FORWARD_APP:
		sf := &pb.StoreAndForward{}
		if err := proto.Unmarshal(data, sf); err == nil {
			return sf
		}
	}

	return data
}

func PortnumName(p pb.PortNum) string {
	if name, ok := pb.PortNum_name[int32(p)]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", p)
}

func NodeIDStr(num uint32) string {
	return fmt.Sprintf("!%08x", num)
}
