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

type HandoverRequest struct {
	Supi  string
	GnbId string
}

var (
	c1 = make(chan *ngapType.NGAPPDU, 200) // Mensajes normales
	c2 = make(chan *ngapType.NGAPPDU, 200) // Mensajes handover
	//handoverTrigger = make(chan bool)                   // Canal para recibir el trigger del Handover
	// canal ahora envía la SUPI concreta
	//handoverTrigger = make(chan string, 1)
	handoverTrigger = make(chan HandoverRequest, 1) //cambiamos el canal para que reciba una estructura en lugar del supi solo
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
func handleMessages(wg *sync.WaitGroup, ctx context.Context, conn *sctp.SCTPConn, ueList []*tglib.RanUeContext, teidUpfIPs map[[4]byte]stgutg.TeidUpfIp, supiToBip map[string][4]byte) {
	c.GetConfiguration()
	fmt.Println("Entra en handle message")
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case msg := <-c1:
			//Únicamente procesa mensajes del tipo PDUSessionResourceSetupRequest
			//TODO: Añadir mas casos para soportar los mensajes de registro de usuario
			stgutg.ProcessGeneralMessage(conn, msg, ueList, teidUpfIPs, supiToBip, c.Configuration.Gnb_gtp, c.Configuration.Upf_port)
		case msg := <-c2:
			stgutg.ProcessHandoverMessage(conn, msg, ueList, supiToBip)
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
	//InitialIMSI string `json:"initial_imsi"`
	Supi  string `json:"supi"` // añadimos el supi
	IMSI  string `json:"imsi"`
	GnbId string `json:"gnb_id"`
}

// Modificación de la petición de registro de un UE, para que el AGF envíe también el SUPI para identificación del usuario
func registerUser(supi string, imsi string, gnbId string) {
	url := "http://138.4.21.21:8080/user_registration"

	// Convertir el GNB ID a hexadecimal, como antes
	hexGnbId := hex.EncodeToString([]byte(gnbId))

	// Crear el payload
	payload := UserPayload{
		Supi:  supi, //se añade el supi
		IMSI:  imsi,
		GnbId: hexGnbId,
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

// Función que recibe la petición para disparar el Handover (cuando reciba el HO Command del Core y se lo notifiquemos al controlador)
// Esta función actualmente recibe el gnbId, lo valida y enviaba a handoverTrigger el valor de true
/*func handleHandoverTrigger(w http.ResponseWriter, r *http.Request) {
	// Leer el cuerpo como JSON
	var payload struct {
		GnbId string `json:"gnbId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if payload.GnbId == "" {
		http.Error(w, "Falta 'gnbId' en el JSON", http.StatusBadRequest)
		return
	}

	// Enviar señal para iniciar el Handover
	handoverTrigger <- true

	// Responder al controlador
	fmt.Fprintf(w, "Handover solicitado para GNBId: %s", payload.GnbId)
}
*/

// Función que inicia el handover (stg-utg.ManageHandoverRequired)
func handleHandoverTrigger(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Supi  string `json:"supi"`
		GnbId string `json:"gnbId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.Supi == "" || payload.GnbId == "" {
		http.Error(w, "JSON inválido: falta supi o gnbId", http.StatusBadRequest)
		return
	}
	handoverTrigger <- HandoverRequest{Supi: payload.Supi, GnbId: payload.GnbId}
	fmt.Fprintf(w, "HO solicitado para SUPI %s a GNB %s", payload.Supi, payload.GnbId)
}

// Función para manejar el procedimiento de Handover
/*func startHandoverProcedure(ueList []*tglib.RanUeContext, conn *sctp.SCTPConn) {
	timer1 := time.NewTimer(10 * time.Second)
	<-timer1.C
	if len(ueList) > 0 {
		fmt.Println("Starting Handover Procedure for UE:", ueList[0].Supi)
		stgutg.ManageHandoverRequired(conn, ueList[0], []byte{0x00, 0x01, 0x02})
	}
}
*/

// MAIN
func main() {

	log.SetOutput(os.Stdout)
	// Iniciar el servidor HTTP para escuchar el trigger
	http.HandleFunc("/triggerHandover", handleHandoverTrigger)
	go http.ListenAndServe("0.0.0.0:8082", nil)

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
	//asociación de supi con bip
	var supiToBip = make(map[string][4]byte)

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

		//se cambia el register para que incluya el supi
		registerUser(ue.Supi, imsi, c.Configuration.Gnb_id)

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
	go handleMessages(&wg, ctx, conn, ueList, teidUpfIPs, supiToBip)

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

	/*
		go func() {
			for {
				<-handoverTrigger
				startHandoverProcedure(ueList, conn) // Ejecutar el Handover
			}
		}()
	*/
	// Disparar handover tras 10s (opcional)
	/*timer1 := time.NewTimer(10 * time.Second)
	<-timer1.C
	if len(ueList) > 0 {
		fmt.Println("Starting Handover Procedure for UE:", ueList[0].Supi)
		//Parametrizar el GNBid recibido
		stgutg.ManageHandoverRequired(conn, ueList[0], []byte{0x00, 0x01, 0x03})
	}
	*/
	go func() {
		for {
			req := <-handoverTrigger // ahora recibimos un struct con SUPI y GNB
			log.Printf("Trigger recibido: SUPI=%s, GNB destino=%s", req.Supi, req.GnbId)
			var ue *tglib.RanUeContext

			log.Println("Contenido actual de ueList en este AGF:")
			for _, u := range ueList {
				log.Printf(" - SUPI registrado: %s", u.Supi)
			}
			for _, u := range ueList {
				if u.Supi == req.Supi {
					ue = u
					break
				}
			}
			if ue == nil {
				log.Printf("SUPI %s no encontrado en este AGF", req.Supi)
				continue
			}
			log.Printf("Iniciando HO para UE %s (SUPI %s) hacia GNB %s", ue.Supi, req.Supi, req.GnbId)
			// Usar el gnbId real como []byte (decodificar hex si viene así)
			targetGnbIdBytes, err := hex.DecodeString(req.GnbId)
			if err != nil {
				log.Printf("Error al decodificar GNB ID: %v", err)
				continue
			}
			log.Printf(" Ejecutando ManageHandoverRequired desde AGF con GNB ID: %s", c.Configuration.Gnb_id)
			log.Printf("Target GNB (bytes): % X", targetGnbIdBytes)
			stgutg.ManageHandoverRequired(conn, ue, targetGnbIdBytes)
		}
	}()

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
