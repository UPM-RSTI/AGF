package stgutg

// PDU
// Functions to manage the PDU sessions, including establishment, modification
// and release.
// Version: 0.9
// Date: 9/6/21

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"syscall"
	"time"

	"free5gclib/nas"
	"free5gclib/nas/nasMessage"
	"free5gclib/nas/nasTestpacket"
	"free5gclib/ngap/ngapType"
	"free5gclib/openapi/models"
	"tglib"

	"github.com/ishidawataru/sctp"
)

// 3GPP TS 24.501 8.3.2.1
// If the length of the element is not fixed it is represented with a negative number:
// -1 and -2 for elements whose length is encoded with 1 and 2 bytes respectively
var PDUSessionEstablishmentAcceptOptionalElementsLength = map[byte]int{
	0x59: 2,
	0x29: -1,
	0x56: 2,
	0x22: -1,
	0x75: -2,
	0x78: -2,
	0x79: -2,
	0x7B: -2,
	0x25: -1,
	0x17: -1,
	0x18: 4,
	0x77: -2,
	0x66: -1,
	0x1F: 3,
}

// 3GPP TS 24.501 8.3.2.1
// These optional element have a total length of 1 byte and their ID's
// are coded with octets 4 to 7 of that byte
var PDUSessionEstablishmentAcceptOptionalElementsHalfByte = []byte{
	0x80,
	0xC0,
}

// EstablishPDU
// Function that establishes a new PDU session for a given UE.
// It requres a previously generated UE and an active SCTP connection with an AMF.
// Se ha eliminado la lectura directa de la respuesta (conn.Read).
// Si necesitas procesar la respuesta, hazlo en la goroutine de lectura y/o handleMessages.
func EstablishPDU(
	sst int32,
	sd string,
	pdu []byte,
	ue *tglib.RanUeContext,
	conn *sctp.SCTPConn,
	gnb_gtp string,
	upf_port int,
	teidUpfIPs map[[4]byte]TeidUpfIp,
) {
	sNssai := models.Snssai{
		Sst: sst,
		Sd:  sd,
	}

	ueSupi := strings.Split(ue.Supi, "-")[1]
	supiInt, _ := strconv.Atoi(ueSupi)
	pduId := int64(supiInt % 1e4) //TODO: Check if it works with large SUPI numbers

	// Construir NAS PDU SessionEstablishmentRequest
	pdu = nasTestpacket.GetUlNasTransport_PduSessionEstablishmentRequest(
		uint8(pduId),
		nasMessage.ULNASTransportRequestTypeInitialRequest,
		"internet",
		&sNssai,
	)

	// Encriptar/cifrar NAS
	pdu, err := tglib.EncodeNasPduWithSecurity(
		ue,
		pdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered,
		true,
		false,
	)
	ManageError("Error establishing PDU", err)

	// Empaquetar en Uplink NAS
	sendMsg, err := tglib.GetUplinkNASTransport(
		ue.AmfUeNgapId,
		ue.RanUeNgapId,
		pdu,
	)
	ManageError("Error establishing PDU", err)

	// Enviar por SCTP
	_, err = conn.Write(sendMsg)
	ManageError("Error establishing PDU", err)

	// ───────────────────────────────────────────────────────────────────────────────
	// Antes se hacía:
	//    n, err := conn.Read(recvMsg)
	//    msg, err := ngap.Decoder(recvMsg[:n])
	//    ... extraer bip, teid, etc.
	// Se elimina la lectura directa para evitar colisión con la goroutine central.
	//
	// Igualmente, aquí enviamos la PDUSessionResourceSetupResponse tras decodificar
	// TODO: si no has recibido la info de bip/teid, no puedes completarlo aquí,
	//       lo ideal es enviarla en la función de "Process" en handleMessages.

	sendMsg, err = tglib.GetPDUSessionResourceSetupResponse(
		ue.AmfUeNgapId,
		ue.RanUeNgapId,
		pduId, // The function from tglib has been changed to include pduId parameter (see Packet.go and Build.go)
		gnb_gtp,
	)
	ManageError("Error establishing PDU", err)

	_, err = conn.Write(sendMsg)
	ManageError("Error establishing PDU", err)

	fmt.Println("[EstablishPDU] Mensaje de SetupRequest enviado, sin leer respuesta aquí.")
}

// ReleasePDU
// Function that removes a previously established PDU session.
// Se ha eliminado la lectura directa de la respuesta. Se asume que la respuesta
// la capturará la goroutine central (c1).
func ReleasePDU(
	sst int32,
	sd string,
	ue *tglib.RanUeContext,
	conn *sctp.SCTPConn,
) []byte {
	sNssai := models.Snssai{Sst: sst, Sd: sd}

	ueSupi := strings.Split(ue.Supi, "-")[1]
	supiInt, err := strconv.Atoi(ueSupi)
	ManageError("Error releasing PDU", err)

	pduId := int64(supiInt % 1e4)

	// 1) Enviar PduSessionReleaseRequest
	pdu := nasTestpacket.GetUlNasTransport_PduSessionReleaseRequest(uint8(pduId))
	pdu, err = tglib.EncodeNasPduWithSecurity(
		ue,
		pdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered,
		true,
		false,
	)
	ManageError("Error releasing PDU", err)

	sendMsg, err := tglib.GetUplinkNASTransport(
		ue.AmfUeNgapId,
		ue.RanUeNgapId,
		pdu,
	)
	ManageError("Error releasing PDU", err)

	_, err = conn.Write(sendMsg)
	ManageError("Error releasing PDU", err)
	time.Sleep(100 * time.Millisecond)

	// 2) Enviar PDUSessionResourceReleaseResponse
	sendMsg, err = tglib.GetPDUSessionResourceReleaseResponse(
		ue.AmfUeNgapId,
		ue.RanUeNgapId,
		pduId,
	)
	ManageError("Error releasing PDU", err)

	_, err = conn.Write(sendMsg)
	ManageError("Error releasing PDU", err)
	time.Sleep(10 * time.Millisecond)

	// 3) Enviar PduSessionReleaseComplete
	pdu = nasTestpacket.GetUlNasTransport_PduSessionReleaseComplete(
		uint8(pduId),
		nasMessage.ULNASTransportRequestTypeInitialRequest,
		"internet",
		&sNssai,
	)
	pdu, err = tglib.EncodeNasPduWithSecurity(
		ue,
		pdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered,
		true,
		false,
	)
	ManageError("Error releasing PDU", err)

	sendMsg, err = tglib.GetUplinkNASTransport(
		ue.AmfUeNgapId,
		ue.RanUeNgapId,
		pdu,
	)
	ManageError("Error releasing PDU", err)

	_, err = conn.Write(sendMsg)
	ManageError("Error releasing PDU", err)

	time.Sleep(1 * time.Second)

	// Antes se hacía una lectura de respuesta aquí, eliminada.
	return pdu
}

// ModifyPDU
// Se ha eliminado también la lectura directa. El resto de lógica se mantiene.
func ModifyPDU(
	sst int32,
	sd string,
	ue *tglib.RanUeContext,
	conn *sctp.SCTPConn,
) []byte {
	sNssai := models.Snssai{Sst: sst, Sd: sd}

	ueSupi := strings.Split(ue.Supi, "-")[1]
	supiInt, err := strconv.Atoi(ueSupi)
	ManageError("Error modifying PDU", err)

	pduId := int64(supiInt % 1e4) //TODO: Check if it works with large SUPI numbers

	pdu := nasTestpacket.GetUlNasTransport_PduSessionModificationRequest(
		uint8(pduId),
		nasMessage.ULNASTransportRequestTypeExistingPduSession,
		"internet",
		&sNssai,
	)
	pdu, err = tglib.EncodeNasPduWithSecurity(
		ue,
		pdu,
		nas.SecurityHeaderTypeIntegrityProtectedAndCiphered,
		true,
		false,
	)
	ManageError("Error modifying PDU", err)

	sendMsg, err := tglib.GetUplinkNASTransport(
		ue.AmfUeNgapId,
		ue.RanUeNgapId,
		pdu,
	)
	ManageError("Error modifying PDU", err)

	_, err = conn.Write(sendMsg)
	ManageError("Error modifying PDU", err)

	// Se elimina la lectura directa de la respuesta:
	//   n, err := conn.Read(recvMsg)
	//   ...
	// Deja que la goroutine central lea y reenvíe el mensaje a tu handler.

	fmt.Println("[ModifyPDU] Petición de modificación enviada (no se lee respuesta aquí).")
	return pdu
}

// DecodePDUSessionResourceSetupRequestTransfer
// Function that extracts UPF IP address and TEID from a given PDUSessionResourceSetupRequestTransfer
func DecodePDUSessionResourceSetupRequestTransfer(PDUSessionResourceSetupRequestTransfer []byte) ([]byte, []byte) {
	var bteid []byte
	var bupfip []byte

	offset := 3 //  We skip number of protocolIEs as we are only interested in the first or second one
	for offset < len(PDUSessionResourceSetupRequestTransfer) {

		if int(binary.BigEndian.Uint16(PDUSessionResourceSetupRequestTransfer[offset:offset+2])) != 139 {
			offset += 3 + int(PDUSessionResourceSetupRequestTransfer[offset+3]) + 1
		} else {
			offset += 3

			UPTrasportLayerInfoLength := int(PDUSessionResourceSetupRequestTransfer[offset])
			offset += 1

			UPTransportLayerInfo := PDUSessionResourceSetupRequestTransfer[offset : offset+UPTrasportLayerInfoLength]

			bteid = UPTransportLayerInfo[UPTrasportLayerInfoLength-4:]
			bupfip = UPTransportLayerInfo[UPTrasportLayerInfoLength-8 : UPTrasportLayerInfoLength-4]
			break
		}
	}

	return bteid, bupfip
}

// DecodePDUSessionNASPDU
// Function that extracts UE IP address from a given PDUSessionNASPDU message
func DecodePDUSessionNASPDU(PDUSessionNASPDU []byte) [4]byte {
	var bip [4]byte

	plainNAS5GSMessage := PDUSessionNASPDU[7:]

	payloadContainerLength := binary.BigEndian.Uint16(plainNAS5GSMessage[4:6])
	payloadContainerPlainNAS5GSMessage := plainNAS5GSMessage[6 : 6+payloadContainerLength]

	QoSRulesLength := binary.BigEndian.Uint16(payloadContainerPlainNAS5GSMessage[5:7])

	opElements := payloadContainerPlainNAS5GSMessage[5+2+QoSRulesLength+7:]
	length := len(opElements)
	index := 0

outerloop:
	for index < length {
		opElementID := opElements[index]
		if opElementID == 0x29 { // PDU Address
			bip = ([4]byte)(opElements[index+3 : index+7])
			index += 7
			break outerloop
		}

		// half-byte
		for _, id := range PDUSessionEstablishmentAcceptOptionalElementsHalfByte {
			if opElementID&0xF0 == id {
				index += 1
				continue outerloop
			}
		}

		opElementLength := PDUSessionEstablishmentAcceptOptionalElementsLength[opElementID]

		if opElementLength > 0 {
			index += opElementLength
		} else if opElementLength == -1 { // 1 byte length
			opElementLength = int(opElements[index+1])
			index += 1 + 1 + opElementLength
		} else if opElementLength == -2 { // 2 bytes length
			opElementLength = int(binary.BigEndian.Uint16(opElements[index+1 : index+3]))
			index += 1 + 2 + opElementLength
		}
	}

	return bip
}

// Función que procesa el NGAP PDUSessionResourceSetupRequest que antes se leía dentro de EstablishPDU.
// Extrae bip, teid, etc. y registra en el map teidUpfIPs.
func ProcessPDUSessionResourceSetupRequest(
	conn *sctp.SCTPConn,
	msg *ngapType.NGAPPDU,
	ue *tglib.RanUeContext,
	teidUpfIPs map[[4]byte]TeidUpfIp,
	gnb_gtp string,
	upf_port int,
) {
	fmt.Println("[ProcessPDUSessionResourceSetupRequest] procesando la Request")

	// Extraer la info
	pduSessionSetupList := msg.InitiatingMessage.Value.
		PDUSessionResourceSetupRequest.ProtocolIEs.List[2].
		Value.PDUSessionResourceSetupListSUReq.List[0]

	bip := DecodePDUSessionNASPDU(pduSessionSetupList.PDUSessionNASPDU.Value)
	fmt.Println("bip:", bip)

	bteid, bupfip := DecodePDUSessionResourceSetupRequestTransfer(
		pduSessionSetupList.PDUSessionResourceSetupRequestTransfer,
	)

	teid := binary.BigEndian.Uint32(bteid)
	fmt.Println("teid:", teid)

	// Ajusta el puerto GTP si es 2152, etc.
	upfAddr := syscall.SockaddrInet4{Addr: ([4]byte)(bupfip), Port: upf_port}
	fmt.Println("upfaddr:", upfAddr)

	// Guardar en el mapa
	teidUpfIPs[bip] = TeidUpfIp{teid, &upfAddr}

	fmt.Println("[ProcessPDUSessionResourceSetupRequest] TEID e IP asignados al UE.")

	ueSupi := strings.Split(ue.Supi, "-")[1]
	supiInt, _ := strconv.Atoi(ueSupi)
	pduId := int64(supiInt % 1e4)

	sendMsg, err := tglib.GetPDUSessionResourceSetupResponse(
		ue.AmfUeNgapId,
		ue.RanUeNgapId,
		pduId,
		gnb_gtp, // gnb_gtp
	)
	ManageError("Error creando PDUSessionResourceSetupResponse", err)

	_, err = conn.Write(sendMsg)
	ManageError("Error enviando PDUSessionResourceSetupResponse", err)

	fmt.Println("[ProcessPDUSessionResourceSetupRequest] completado. bip/teid asignados.")
}

// Método para procesar el PDUSessionResourceSetupRequest
func ProcessGeneralMessage(
	conn *sctp.SCTPConn,
	msg *ngapType.NGAPPDU,
	ueList []*tglib.RanUeContext,
	teidUpfIPs map[[4]byte]TeidUpfIp,
	gnb_gtp string,
	upf_port int,
) {
	// Identificar si es PDU Session Setup, etc.
	if msg.InitiatingMessage != nil &&
		msg.InitiatingMessage.Value.PDUSessionResourceSetupRequest != nil {
		// Buscar UE en ueList
		// En tu caso, extráelo de RANUENGAPID o similar

		for i := 0; i < len(ueList); i++ {
			ProcessPDUSessionResourceSetupRequest(conn, msg, ueList[i], teidUpfIPs, gnb_gtp, upf_port)
			time.Sleep(1 * time.Second)
		}
	} else {
		fmt.Println("[ProcessGeneralMessage] Mensaje NGAP no handleado:", msg)
	}
}

// Función auxiliar en desuso
func printHex(bytearray []byte) {
	for _, byteelement := range bytearray {
		fmt.Printf("%02X ", byteelement)
	}
	fmt.Print("\n\n")
}
