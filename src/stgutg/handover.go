package stgutg

//Functions to handle Handover Procedure

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"free5gclib/ngap/ngapType"
	"log"
	"net/http"
	"strconv"
	"strings"
	"tglib"

	"github.com/ishidawataru/sctp"
)

//VERSIÓN DE MARIO

// Payload sent to the controller in the POST /dhcptrigger
type dhcpTrigger struct {
	UeIP     string `json:"ue"`
	TargetID string `json:"gnbId"`
}

// ManageHandoverRequired -> solo envía el mensaje, la respuesta llega a la gorutine de escucha.
func ManageHandoverRequired(conn *sctp.SCTPConn, ue *tglib.RanUeContext, targetGNBID []byte) {
	ueSupi := strings.Split(ue.Supi, "-")[1]
	supiInt, _ := strconv.Atoi(ueSupi)
	pduId := int64(supiInt % 1e4)

	sendHandoverRequired, err := tglib.GetHandoverRequired(ue.AmfUeNgapId, ue.RanUeNgapId, targetGNBID, pduId)
	ManageError("error creando HandoverRequired", err)

	_, err = conn.Write(sendHandoverRequired)
	ManageError("error enviando HandoverRequired", err)

	fmt.Println("[ManageHandoverRequired] HandoverRequired enviado; respuesta vendrá a c2")
}

// ProcessHandoverMessage procesa HORequest y HOCommand y envía HandoverRequestAcknowledge
func ProcessHandoverMessage(conn *sctp.SCTPConn, msg *ngapType.NGAPPDU, ueList []*tglib.RanUeContext, supiToBip map[string][4]byte) {
	fmt.Println("[ProcessHandoverMessage] Procesando mensaje relacionado con Handover...")

	// Verificar si es un HandoverCommand
	if msg.SuccessfulOutcome != nil &&
		msg.SuccessfulOutcome.ProcedureCode.Value == ngapType.ProcedureCodeHandoverPreparation &&
		msg.SuccessfulOutcome.Value.HandoverCommand != nil {

		fmt.Println(" [ProcessHandoverMessage] HandoverCommand recibido")

		// Debug de los campos principales
		// Logs de los ids del usuario a mover
		for _, ie := range msg.SuccessfulOutcome.Value.HandoverCommand.ProtocolIEs.List {
			switch ie.Id.Value {
			case ngapType.ProtocolIEIDAMFUENGAPID:
				fmt.Printf("→ AMF-UE-NGAP-ID: %d\n", ie.Value.AMFUENGAPID.Value)
			case ngapType.ProtocolIEIDRANUENGAPID:
				fmt.Printf("→ RAN-UE-NGAP-ID: %d\n", ie.Value.RANUENGAPID.Value)
			case ngapType.ProtocolIEIDHandoverType:
				fmt.Printf("→ Handover Type: %d\n", ie.Value.HandoverType.Value)
			}
		}

		// Extrae el AMF‑UE‑NGAP‑ID
		var amfId int64
		for _, ie := range msg.SuccessfulOutcome.Value.HandoverCommand.ProtocolIEs.List {
			if ie.Id.Value == ngapType.ProtocolIEIDAMFUENGAPID {
				amfId = ie.Value.AMFUENGAPID.Value
				break
			}
		}

		// Busca el RanUeContext correspondiente
		var ue *tglib.RanUeContext
		for _, u := range ueList {
			if u.AmfUeNgapId == amfId {
				ue = u
				break
			}
		}
		if ue == nil {
			log.Printf("UE no encontrado para AMF‑UE‑NGAP‑ID %d", amfId)
			return
		}

		// Obtiene la IP del usuario usando SUPI
		ueIP, ok := getBipBySupi(ue.Supi, supiToBip)
		if !ok {
			log.Printf("No se encontró IP para SUPI %s", ue.Supi)
			return
		}

		fmt.Println("[DEBUG] Mapa SUPI → IP actualmente registrado:")
		for k, v := range supiToBip {
			fmt.Printf("  %s => %d.%d.%d.%d\n", k, v[0], v[1], v[2], v[3])
		}

		// Dispara la petición al controlador mandandole la Ip
		// recuperamos el targetGnbId que guardamos antes
		targetBytes, ok := GetTargetGnb(ue.Supi)
		if !ok {
			log.Printf("No tengo target GNB para SUPI %s", ue.Supi)
			return
		}

		// 2) Lo conviertes a string hex sin espacios
		targetHex := hex.EncodeToString(targetBytes)

		// 3) Llamas al controlador pasando IP y targetHex
		postTriggerDHCP(ueIP, targetHex)
		return
		//Al llegar el Handover Command, extraemos el AMFUENGAPID, y matcheamos el usuario con el que coincida de la lista y enviamos el supi
	}

	// Verificar que el mensaje sea un HandoverRequest
	if msg.InitiatingMessage != nil &&
		msg.InitiatingMessage.ProcedureCode.Value == ngapType.ProcedureCodeHandoverResourceAllocation &&
		msg.InitiatingMessage.Value.HandoverRequest != nil {

		hoReq := msg.InitiatingMessage.Value.HandoverRequest
		newAmfUe := hoReq.ProtocolIEs.List[0].Value.AMFUENGAPID.Value
		fmt.Printf("[ProcessHandoverMessage] Recibido HandoverRequest con AmfUeNgapId: %d\n", newAmfUe)

		// -------- Extraer el contenedor y buscar MCC y MNC --------
		container := hoReq.ProtocolIEs.List[9].Value.SourceToTargetTransparentContainer.Value

		// Buscar el patrón del PLMN (02 f8 39 en la captura, podría variar)
		// Esta implementación funciona con el free5gc porque usamos siempre el mismo mnc y mcc
		index := bytes.Index(container, []byte{0x02, 0xf8, 0x39})
		if index == -1 {
			fmt.Println("[ProcessHandoverMessage] PLMN ID no encontrado en el contenedor")
			return
		}

		plmnRaw := container[index : index+3] // Tomamos esos 3 bytes

		// MCC = D1 D2 D3
		mccDigit1 := plmnRaw[0] & 0x0F
		mccDigit2 := (plmnRaw[0] & 0xF0) >> 4
		mccDigit3 := plmnRaw[1] & 0x0F

		// MNC = D1 D2 (2 dígitos)
		mncDigit2 := (plmnRaw[2] & 0xF0) >> 4
		mncDigit1 := plmnRaw[2] & 0x0F

		// Formatear MCC y MNC como string
		mcc := fmt.Sprintf("%d%d%d", mccDigit1, mccDigit2, mccDigit3)
		mnc := fmt.Sprintf("%d%d", mncDigit1, mncDigit2)

		fmt.Printf("[ProcessHandoverMessage] MCC: %s, MNC: %s\n", mcc, mnc)

		// -------- Extraer los siguientes bytes (como parte del IMSI) --------
		// Por ejemplo, en la captura sigue "00 01", así que tomamos esos 2 bytes
		if len(container) < index+5 {
			fmt.Println("[ProcessHandoverMessage] No hay suficientes bytes para extraer parte del IMSI")
			return
		}
		subscriberBytes := container[index+3 : index+5] // Los 2 bytes siguientes (00 01)
		// Ahora los formateamos como un número con 8 dígitos, para completar hasta "00000001"
		subscriberId := fmt.Sprintf("%08d", binary.BigEndian.Uint16(subscriberBytes))
		// ---------- Crear SUPI basado en MCC y MNC ----------
		supi := fmt.Sprintf("imsi-" + mcc + mnc + subscriberId) // Aquí puedes ajustar el sufijo según quieras
		fmt.Printf("[ProcessHandoverMessage] SUPI generado: %s\n", supi)

		// ---------- Crear nuevo RanUeContext ----------

		//pduid y amfid lo extraigo
		// no hace falta crear un nuevo usuario, con crear un nuevo RanUeNgapId es suficiente
		pduId := int64(hoReq.ProtocolIEs.List[6].Value.PDUSessionResourceSetupListHOReq.List[0].PDUSessionID.Value)
		fmt.Printf("[ProcessHandoverMessage] PDU Session ID extraído: %d\n", pduId)

		// --------- Obtener nuevo RanUeNgapId ----------
		// Es posible que haya que llamar a la función NewRanUeContext
		newRanUeNgapId := int64(len(ueList) + 1) // Genera uno nuevo incremental
		fmt.Printf("[ProcessHandoverMessage] Nuevo RanUeNgapId generado: %d\n", newRanUeNgapId)

		// ----------- Crear y enviar HandoverRequestAcknowledge -----------
		sendHandoverACK, err := tglib.GetHandoverRequestAcknowledge(
			newAmfUe,
			newRanUeNgapId,
			pduId,
		)
		ManageError("Error creando HandoverRequestAcknowledge", err)

		_, err = conn.Write(sendHandoverACK)
		ManageError("Error enviando HandoverRequestAcknowledge", err)

		fmt.Println("[ProcessHandoverMessage] HandoverRequestAcknowledge enviado correctamente al AMF")

	} else {
		fmt.Println("[ProcessHandoverMessage] Mensaje recibido no es HandoverRequest. Ignorado.")
	}
}

// getBipBySupi formatea la IP ([4]byte) asociada a una SUPI
func getBipBySupi(supi string, supiToBip map[string][4]byte) (string, bool) {
	bip, ok := supiToBip[supi]
	if !ok {
		return "", false
	}
	ip := fmt.Sprintf("%d.%d.%d.%d", bip[0], bip[1], bip[2], bip[3])
	return ip, true
}

// Función que una vez que llega el HO Command notifica al controlador que el cambio de IP es posible
func postTriggerDHCP(ueIP, targetGnbId string) {
	url := "http://138.4.21.21:8080/triggerDHCP"
	body, _ := json.Marshal(dhcpTrigger{UeIP: ueIP, TargetID: targetGnbId})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Error POST /triggerDHCP: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("DHCP Trigger → %s (destino GNB=%s) status %s", ueIP, targetGnbId, resp.Status)
}
