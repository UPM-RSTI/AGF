package main

// #cgo CFLAGS: -pthread
// #include <signal.h>
// #include <pthread.h>
import "C"

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"free5gclib/ngap"
	"free5gclib/ngap/ngapType"
	"stgutg"
	"tglib"

	"github.com/ishidawataru/sctp"

	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

var (
	c1 = make(chan *ngapType.NGAPPDU, 200) // Mensajes normales
	c2 = make(chan *ngapType.NGAPPDU, 200) // Mensajes handover
)

var c stgutg.Conf

// listenSCTPConnection: una sola goroutine lee de conn y decide si mandar el msg a c1 o c2
func listenSCTPConnection(conn *sctp.SCTPConn, wg *sync.WaitGroup, ctx context.Context) {
	fmt.Println("Entra en listenSCTPConnection")
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("[listenSCTPConnection] Cancelado, saliendo")
			return
		default:
		}

		buffer := make([]byte, 8092)
		n, err := conn.Read(buffer)
		if err != nil {
			if err.Error() == "use of closed network connection" {
				log.Println("[listenSCTPConnection] Conexión SCTP cerrada")
				return
			}
			log.Printf("[listenSCTPConnection] Error leyendo SCTP: %v", err)
			continue
		}

		ngapMsg, err := ngap.Decoder(buffer[:n])
		if err != nil {
			log.Printf("[listenSCTPConnection] Error decodificando NGAP: %v", err)
			continue
		}

		// Decide según sea HO o no
		if isHandoverMessage(ngapMsg) {
			select {
			case c2 <- ngapMsg:
			default:
				log.Println("[listenSCTPConnection] Canal c2 lleno, descarto HO msg")
			}
		} else {
			select {
			case c1 <- ngapMsg:
			default:
				log.Println("[listenSCTPConnection] Canal c1 lleno, descarto msg normal")
			}
		}
	}
}

// isHandoverMessage: detecta si es un mensaje de handover
func isHandoverMessage(msg *ngapType.NGAPPDU) bool {
	switch msg.Present {
	case ngapType.NGAPPDUPresentInitiatingMessage:
		if msg.InitiatingMessage != nil {
			switch msg.InitiatingMessage.ProcedureCode.Value {
			case ngapType.ProcedureCodeHandoverPreparation, // HandoverRequired
				ngapType.ProcedureCodeHandoverResourceAllocation: // HandoverRequest
				return true
			}
		}
	case ngapType.NGAPPDUPresentSuccessfulOutcome:
		if msg.SuccessfulOutcome != nil {
			switch msg.SuccessfulOutcome.ProcedureCode.Value {
			case ngapType.ProcedureCodeHandoverPreparation: // HandoverCommand
				if msg.SuccessfulOutcome.Value.HandoverCommand != nil {
					return true
				}
			case ngapType.ProcedureCodeHandoverResourceAllocation:
				if msg.SuccessfulOutcome.Value.HandoverRequestAcknowledge != nil { //HandoverAck
					return true
				}
			}
		}
	}
	return false
}

// handleMessages: escucha c1 y c2 y llama a stgutg.ProcessGeneralMessage / ProcessHandoverMessage
func handleMessages(wg *sync.WaitGroup, ctx context.Context, conn *sctp.SCTPConn, ueList []*tglib.RanUeContext, teidUpfIPs map[[4]byte]stgutg.TeidUpfIp) {
	c.GetConfiguration()
	fmt.Println("Entra en handle message")
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case msg := <-c1:
			//Únicamente procesa mensajes del tipo PDUSessionResourceSetupRequest
			//TODO: Añadir mas casos para soportar los mensajes de registro de usuario
			stgutg.ProcessGeneralMessage(conn, msg, ueList, teidUpfIPs, c.Configuration.Gnb_gtp, c.Configuration.Upf_port)
		case msg := <-c2:
			stgutg.ProcessHandoverMessage(conn, msg, ueList)
		case <-ctx.Done():
			fmt.Println("[handleMessages] Cancelado, saliendo")
			return
		}
	}
}

type GnbPayload struct {
	GnbId string `json:"gnbId"`
}

// Función para registrar el AGF
func registerAGF(gnbId string) {
	url := "http://138.4.21.21:8080/AGF_registration"

	hexGnbId := hex.EncodeToString([]byte(gnbId))

	// Crear el JSON
	payload := GnbPayload{
		GnbId: hexGnbId,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Error al serializar el payload: %v", err)
	}

	// Crear la solicitud HTTP POST
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error al hacer la solicitud POST: %v", err)
	}
	defer resp.Body.Close()

	// Verificar la respuesta
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error en respuesta: %s", resp.Status)
	} else {
		log.Println("AGF registrado exitosamente")
	}
}

type UserPayload struct {
	InitialIMSI string `json:"initial_imsi"`
	GnbId       string `json:"gnb_id"`
}

func registerUser(initialIMSI string, gnbId string) {
	url := "http://138.4.21.21:8080/user_registration"

	// Convertir el GNB ID a hexadecimal, como antes
	hexGnbId := hex.EncodeToString([]byte(gnbId))

	// Crear el payload
	payload := UserPayload{
		InitialIMSI: initialIMSI,
		GnbId:       hexGnbId,
	}

	// Serializar a JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Fatalf("Error al serializar el payload de usuario: %v", err)
	}

	// Enviar POST
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatalf("Error al hacer la solicitud POST de usuario: %v", err)
	}
	defer resp.Body.Close()

	// Verificar respuesta
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error en respuesta al registrar usuario: %s", resp.Status)
	} else {
		log.Println("Usuario registrado exitosamente")
	}
}

func main() {

	log.SetOutput(os.Stdout)

	var pduList [][]byte
	// Define el contexto y WaitGroup para toda la ejecución
	ctx, cancelFunc := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	//var c stgutg.Conf
	c.GetConfiguration()

	registerAGF(c.Configuration.Gnb_id)

	fmt.Println("[MAIN] Connecting to AMF")
	conn, err := tglib.ConnectToAmf(
		c.Configuration.Amf_ngap,
		c.Configuration.Gnb_ngap,
		c.Configuration.Amf_port,
		c.Configuration.Gnbn_port,
	)
	stgutg.ManageError("Error in connection to AMF", err)
	if err != nil {
		defer conn.Close()
	}

	fmt.Println("[MAIN] Managing NG Setup")
	stgutg.ManageNGSetup(
		conn,
		c.Configuration.Gnb_id,
		c.Configuration.Initial_imsi,
		c.Configuration.Mnc,
		c.Configuration.Gnb_bitlength,
		c.Configuration.Gnb_name,
	)

	// Listas de UEs y TEID
	var ueList []*tglib.RanUeContext
	teidUpfIPs := make(map[[4]byte]stgutg.TeidUpfIp)

	// Registrar UEs (asumiendo que stgutg.RegisterUE no hace lecturas)
	// Es posible que haya que llamar de nuevo a estos métodos tras ejecutar el Handover
	/* El controlador confirma el Handover, y le pasa el imsi del usuario que se ha movido para que le registre de nuevo
	 */
	for i := 0; i < c.Configuration.UeNumber; i++ {
		//Incremento del imsi en caso de registrar varios usuarios a la vez
		imsiInt, err := strconv.Atoi(c.Configuration.Initial_imsi)
		if err != nil {
			fmt.Println(err)
		}
		imsi := strconv.Itoa(imsiInt + i)

		fmt.Println(">> Creating new UE with IMSI:", imsi)
		ue := stgutg.CreateUE(
			imsi,
			i,
			c.Configuration.K,
			c.Configuration.OPC,
			c.Configuration.OP,
		)

		fmt.Println(">> Registering UE with IMSI:", imsi)
		ue, pdu, _ := stgutg.RegisterUE(
			ue,
			c.Configuration.Mnc,
			c.Configuration.Mcc,
			conn,
		)

		registerUser(c.Configuration.Gnb_id, imsi)

		// Guardar en listas
		ueList = append(ueList, ue)
		pduList = append(pduList, pdu)

		time.Sleep(1 * time.Second) // Retraso entre registros
	}

	/* Una vez iniciada se queda en espera y trata todos los mensajes que lleguen antes de que entren en sus respectivas funciones.
	   Soporta tratamiento de los mensajes de Handover Request y Command, y envía el RequestACK de manera sequencial.
	   Una vez finalizado el Handover sigue la gorutine en espera de nuevos paquetes.
	   TODO: Soporte para paquetes de registro del usuario
	*/
	// Iniciar goroutine lectora de SCTP
	go listenSCTPConnection(conn, &wg, ctx)
	// Iniciar goroutine que maneja los mensajes
	go handleMessages(&wg, ctx, conn, ueList, teidUpfIPs)

	i := 0
	for _, pdu := range pduList {
		fmt.Println(">> Establishing PDU session for", ueList[i].Supi)

		// PDU info stored in teidUpfIPs
		stgutg.EstablishPDU(c.Configuration.SST,
			c.Configuration.SD,
			pdu,
			ueList[i],
			conn,
			c.Configuration.Gnb_gtp,
			c.Configuration.Upf_port,
			teidUpfIPs)

		i++
		time.Sleep(1 * time.Second)
	}
	fmt.Println("teidUpfIPs:", teidUpfIPs)

	// Conectar a UPF
	fmt.Println(">> Connecting to UPF")
	upfFD, err := tglib.ConnectToUpf(c.Configuration.Gnbg_port)
	stgutg.ManageError("Error in connection to UPF", err)

	fmt.Println(">> Opening traffic interfaces")
	ethSocketConn, err := tglib.NewEthSocketConn(c.Configuration.SrcIface)
	stgutg.ManageError("Error creating Ethernet socket", err)

	ipSocketConn, err := tglib.NewIPSocketConn()
	stgutg.ManageError("Error creating IP socket", err)

	// Canal para detener el programa
	stopProgram := make(chan os.Signal, 1)
	signal.Notify(stopProgram, syscall.SIGTERM, syscall.SIGINT)

	// Lanzar ListenForResponses / SendTraffic
	wg.Add(2) // Esperamos dos goroutines más
	fmt.Println(">> Listening to traffic responses")
	go stgutg.ListenForResponses(ipSocketConn, upfFD, ctx, &wg)

	fmt.Println(">> Waiting for traffic to send (Press Ctrl+C to quit)")
	utg_ul_thread_chan := make(chan stgutg.Thread)
	go stgutg.SendTraffic(upfFD, ethSocketConn, teidUpfIPs, ctx, &wg, utg_ul_thread_chan)
	utg_ul_thread := <-utg_ul_thread_chan

	// Disparar handover tras 10s (opcional)
	timer1 := time.NewTimer(10 * time.Second)
	<-timer1.C
	if len(ueList) > 0 {
		fmt.Println("Starting Handover Procedure for UE:", ueList[0].Supi)
		//Parametrizar el GNBid recibido
		stgutg.ManageHandoverRequired(conn, ueList[0], []byte{0x00, 0x01, 0x03})
	}

	// Esperar señal
	sig := <-stopProgram
	fmt.Println("\n>> Exiting program:", sig, "found")

	// Cancelamos el contexto (para cerrar goroutines)
	cancelFunc()

	// Detener captura
	// Detener captura (versión corregida)
	C.pthread_kill(
		C.ulong(utg_ul_thread.Id),
		C.int(syscall.SIGUSR1),
	)
	syscall.Shutdown(upfFD, syscall.SHUT_RD)

	// Liberar PDU
	for _, ue := range ueList {
		fmt.Println(">> Releasing PDU session for", ue.Supi)
		stgutg.ReleasePDU(
			c.Configuration.SST,
			c.Configuration.SD,
			ue,
			conn,
		)
		time.Sleep(1 * time.Second)
	}

	// Deregistrar UEs
	for _, ue := range ueList {
		fmt.Println(">> Deregistering UE", ue.Supi)
		stgutg.DeregisterUE(
			ue,
			c.Configuration.Mnc,
			conn,
		)
		time.Sleep(2 * time.Second)
	}

	time.Sleep(1 * time.Second)
	conn.Close()

	fmt.Println(">> Waiting for UTG to shut down")
	wg.Wait() // Espera a que terminen todas las goroutines

	fmt.Println(">> Closing network interfaces")
	syscall.Close(upfFD)
	syscall.Close(ethSocketConn.Fd)
	syscall.Close(ipSocketConn.Fd)

	time.Sleep(1 * time.Second)
	os.Exit(0)

}
