package peerwg

import (
	"bytes"
	"strconv"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"inet.af/netaddr"
	"tailscale.com/types/key"
)

const peerMessageSeparator byte = '\n'

// WireguardPeerMessage is a message received from a wireguard peer, indicating
// it would like to connect.
type WireguardPeerMessage struct {
	// Recipient is the uuid of the agent that the message was intended for.
	Recipient uuid.UUID `json:"recipient"`
	// Disco is the disco public key of the peer.
	Disco key.DiscoPublic `json:"disco"`
	// Public is the public key of the peer.
	Public key.NodePublic `json:"public"`
	// IPv6 is the IPv6 address of the peer.
	IPv6 netaddr.IP `json:"ipv6"`
}

// WireguardPeerMessageRecipientHint parses the first part of a serialized
// WireguardPeerMessage to quickly determine if the message is meant for the
// provided agentID.
func WireguardPeerMessageRecipientHint(agentID []byte, msg []byte) (bool, error) {
	idx := bytes.Index(msg, []byte{peerMessageSeparator})
	if idx == -1 {
		return false, xerrors.Errorf("invalid peer message, no separator")
	}

	return bytes.Equal(agentID, msg[:idx]), nil
}

func (pm *WireguardPeerMessage) UnmarshalText(text []byte) error {
	sp := bytes.Split(text, []byte{peerMessageSeparator})
	if len(sp) != 4 {
		return xerrors.Errorf("expected 4 parts, got %d", len(sp))
	}

	err := pm.Recipient.UnmarshalText(sp[0])
	if err != nil {
		return xerrors.Errorf("parse recipient: %w", err)
	}

	err = pm.Disco.UnmarshalText(sp[1])
	if err != nil {
		return xerrors.Errorf("parse disco: %w", err)
	}

	err = pm.Public.UnmarshalText(sp[2])
	if err != nil {
		return xerrors.Errorf("parse public: %w", err)
	}

	pm.IPv6, err = netaddr.ParseIP(string(sp[3]))
	if err != nil {
		return xerrors.Errorf("parse ipv6: %w", err)
	}

	return nil
}

func (pm WireguardPeerMessage) MarshalText() ([]byte, error) {
	const expectedLen = 223
	var buf bytes.Buffer
	buf.Grow(expectedLen)

	recp, _ := pm.Recipient.MarshalText()
	_, _ = buf.Write(recp)
	_ = buf.WriteByte(peerMessageSeparator)

	disco, _ := pm.Disco.MarshalText()
	_, _ = buf.Write(disco)
	_ = buf.WriteByte(peerMessageSeparator)

	pub, _ := pm.Public.MarshalText()
	_, _ = buf.Write(pub)
	_ = buf.WriteByte(peerMessageSeparator)

	ipv6 := pm.IPv6.StringExpanded()
	_, _ = buf.WriteString(ipv6)

	// Ensure we're always allocating exactly enough.
	if buf.Len() != expectedLen {
		panic("buffer length mismatch: want 221, got " + strconv.Itoa(buf.Len()))
	}
	return buf.Bytes(), nil
}
