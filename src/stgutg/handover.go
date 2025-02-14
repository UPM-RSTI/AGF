package stgutg

//Functions to handle Handover Procedure

import (
	"fmt"
	"free5gclib/ngap"
	"strconv"
	"strings"
	"tglib"

	"github.com/ishidawataru/sctp"
)

// Manage HandoverRequired

func ManageHandoverRequired(conn *sctp.SCTPConn, ue *tglib.RanUeContext, targetGNBID []byte) {
	var recvMsg = make([]byte, 2048)
	fmt.Println("Entrando en HandoverRequired")

	ueSupi := strings.Split(ue.Supi, "-")[1]
	supiInt, _ := strconv.Atoi(ueSupi)
	pduId := int64(supiInt % 1e4)

	//Modificado para no requerir targetCellID
	//Modificado para requerir pduId
	sendHandoverRequired, err := tglib.GetHandoverRequired(ue.AmfUeNgapId, ue.RanUeNgapId, targetGNBID, pduId)
	fmt.Println("New AmfUeNgapId", ue.AmfUeNgapId)
	ManageError("Error in Handover Required", err)

	_, err = conn.Write(sendHandoverRequired)
	ManageError("Error in Handover Required", err)

	n, err := conn.Read(recvMsg)
	ManageError("Error in Handover Required", err)

	handoverRequestMsg, err := ngap.Decoder(recvMsg[:n])
	ManageError("Error in Handover Required", err)

	ue.AmfUeNgapId = handoverRequestMsg.InitiatingMessage.Value.HandoverRequest.ProtocolIEs.List[0].Value.AMFUENGAPID.Value
	fmt.Println("New AmfUeNgapId", ue.AmfUeNgapId)

	//Modificado para requerir pduId
	//Sacar AmfUengapId y RanId del HandoverRequest, devuelve como AmfUengapId (2) y el ue tiene 1 por lo que no matchea y devuelve error
	sendHandoverACK, err := tglib.GetHandoverRequestAcknowledge(ue.AmfUeNgapId, ue.RanUeNgapId, pduId)
	ManageError("Error in Handover ACK", err)

	_, err = conn.Write(sendHandoverACK)
	ManageError("Error in Handover ACK", err)

	n, err = conn.Read(recvMsg)
	ManageError("Error in Handover ACK", err)

	_, err = ngap.Decoder(recvMsg[:n])
	ManageError("Error in Handover ACK", err)

}

// func ManageHandoverRequestACK (conn *sctp.SCTPConn, amfUeNgapID int64, ranUeNgapID int64) {
// 	var recvMsg = make([]byte, 2048)

// 	sendHandoverACK, err := tglib.GetHandoverRequestAcknowledge(amfUeNgapID, ranUeNgapID)
// 	ManageError("Error in Handover ACK", err)

// 	_, err = conn.Write(sendHandoverACK)
// 	ManageError("Error in Handover ACK", err)

// 	n, err := conn.Read(recvMsg)
// 	ManageError("Error in Handover ACK", err)

// 	_, err = ngap.Decoder(recvMsg[:n])
// 	ManageError("Error in Handover ACK", err)
// }
