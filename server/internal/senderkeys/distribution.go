package senderkeys

import (
	pb "github.com/qmish/FProto/server/internal/protocol/gen/fproto/v1"
)

// ToDistribution converts a SenderKey to a protobuf SenderKeyDistribution message.
func ToDistribution(groupID, senderID []byte, sk *SenderKey) *pb.SenderKeyDistribution {
	return &pb.SenderKeyDistribution{
		GroupId:    groupID,
		SenderId:   senderID,
		KeyId:      sk.KeyID,
		ChainKey:   sk.ChainKey,
		SigningKey: []byte(sk.PublicKey),
	}
}

// FromDistribution extracts the public sender key material from a distribution message.
func FromDistribution(dist *pb.SenderKeyDistribution) (keyID uint32, chainKey []byte, publicKey []byte) {
	return dist.KeyId, dist.ChainKey, dist.SigningKey
}
