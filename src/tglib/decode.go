package tglib

import (
	"free5gclib/nas"
	"free5gclib/ngap/ngapType"
)

func GetNasPdu(ue *RanUeContext, msg *ngapType.DownlinkNASTransport) (m *nas.Message) {
	for _, ie := range msg.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDNASPDU {
			pkg := []byte(ie.Value.NASPDU.Value)
			m, err := NASDecode(ue, nas.GetSecurityHeaderType(pkg), pkg)
			if err != nil {
				return nil
			}
			return m
		}
	}
	return nil
}

// Not needed
func GetAMFUENGAPID(msg *ngapType.HandoverRequest) (amfuengapid int64) {
	for _, ie := range msg.ProtocolIEs.List {
		if ie.Id.Value == ngapType.ProtocolIEIDAMFUENGAPID {
			return ie.Value.AMFUENGAPID.Value
		}
	}
	return 0
}
