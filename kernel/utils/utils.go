package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
)

/*---------------------- ESTRUCTURAS ----------------------*/

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type PCB struct {
	Pid   int
	Tid   []int
	Mutex []int
}

type TCB struct {
	Pid             int
	Tid             int
	Prioridad       int
	HilosBloqueados []int
}

type KernelRequest struct {
	Size int `json:"size"`
	PCB  PCB `json:"pcb"`
}

type TCBRequestMemory struct {
	Pid  int    `json:"pid"`
	Tid  int    `json:"tid"`
	Path string `json:"path"`
}

type IniciarProcesoResponse struct {
	Path      string `json:"path"`
	Size      int    `json:"size"`
	Prioridad int    `json:"prioridad"`
}

type TCBRequest struct {
	Pid int `json:"pid"`
	Tid int `json:"tid"`
}

type PCBRequest struct {
	Pid int `json:"pid"`
}

type CrearHiloResponse struct {
	Pid       int    `json:"pid"`
	Path      string `json:"path"`
	Prioridad int    `json:"prioridad"`
}

type CambioHilos struct {
	Pid       int `json:"pid"`
	TidActual int `json:"tidActual"`
	TidCambio int `json:"tidAEjecutar"`
}

type IOsyscall struct {
	TiempoIO int `json:"tiempoIO"`
	Pid      int `json:"pid"`
	Tid      int `json:"tid"`
}

/*-------------------- COLAS GLOBALES --------------------*/

var colaNewproceso []PCB
var colaExitproceso []PCB

var colaReadyHilo []TCB
var colaExecHilo []TCB
var colaBlockHilo []TCB
var colaExitHilo []TCB

/*-------------------- MUTEX GLOBALES --------------------*/

var mutexColaNewproceso sync.Mutex
var mutexColaExitproceso sync.Mutex

var mutexColaReadyHilo sync.Mutex
var mutexColaExecHilo sync.Mutex
var mutexColaBlockHilo sync.Mutex
var mutexColaExitHilo sync.Mutex

//var mutexProcesosActivos sync.Mutex

/*-------------------- VAR GLOBALES --------------------*/

var (
	nextPid = 1
	nextTid = 0
)

/*---------------------- CANALES ----------------------*/

var finProceso = make(chan bool)

//var procesosActivos = make(map[int]PCB) // mapa que gestiona los procesos activos y se puede acceder a travez de una clave
// en este caso seria por el pid

/*---------------------- FUNCIONES ----------------------*/
//	INICIAR CONFIGURACION Y LOGGERS

func IniciarConfiguracion(filePath string) *globals.Config {
	var config *globals.Config
	configFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
}

func ConfigurarLogger() {
	logFile, err := os.OpenFile("tp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

// INICIAR MODULO

func init() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	/*	slog.SetLogLoggerLevel(slog.LevelInfo)
		slog.SetLogLoggerLevel(slog.LevelWarn)
		slog.SetLogLoggerLevel(slog.LevelError)
	SE SETEA EL NIVEL MINIMO DE LOGS A IMPRIMIR POR CONSOLA*/

	ConfigKernel := IniciarConfiguracion("configsKERNEL/config.json")

	if ConfigKernel != nil {

		//Cuando levanto kernel se inicia un proceso ppal y luego se ejecutan syscalls?
		procesoInicial("procesoInicial", 0)

		if ConfigKernel.AlgoritmoPlanificacion == "FIFO" {
			go ejecutarHilosFIFO()
		} else if ConfigKernel.AlgoritmoPlanificacion == "PRIORIDADES" {
			// go ejecutarHilosPrioridades()
		} else if ConfigKernel.AlgoritmoPlanificacion == "COLASMULTINIVEL" {
			// go ejecutarHilosColasMultinivel()
		}
	} else {
		log.Printf("Algoritmo de planificacion no valido")
	}
}

/*---------- FUNCIONES PROCESOS ----------*/

func procesoInicial(path string, size int) {
	//CREAMOS PCB
	pcb := createPCB()
	// Verificar si se puede enviar a memoria, si hay espacio para el proceso

	if consultaEspacioAMemoria(size, pcb) {
		tcb := createTCB(pcb.Pid, 0)       // creamos hilo main
		pcb.Tid = append(pcb.Tid, tcb.Tid) // agregamos el hilo a la listas de hilos del proceso
		enviarTCBMemoria(tcb, path)
		encolarProcesoNew(pcb)
		encolarReady(tcb)
	} else {
		slog.Error("Error creando el proceso inicial")
		return
	}
}

func createPCB() PCB {
	nextPid++

	return PCB{
		Pid:   nextPid - 1,
		Tid:   []int{},
		Mutex: []int{},
	}
}

func getPCB(pid int) PCB {
	for _, pcb := range colaNewproceso {
		if pcb.Pid == pid {
			return pcb
		}
	}
	return PCB{}
}

func CrearProceso(w http.ResponseWriter, r *http.Request) {
	var proceso IniciarProcesoResponse
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&proceso)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path := proceso.Path
	size := proceso.Size
	prioridad := proceso.Prioridad

	go iniciarProceso(path, size, prioridad) // go routine para que no bloquee el hilo principal en caso de que no haya espacio en memoria para iniciar proceso. CONSULTAR!!

	w.WriteHeader(http.StatusOK)
}

func iniciarProceso(path string, size int, prioridad int) error {

	//CREAMOS PCB
	pcb := createPCB()

	// Verificar si se puede enviar a memoria, si hay espacio para el proceso
	if consultaEspacioAMemoria(size, pcb) {

		nextTid = 0
		tcb := createTCB(pcb.Pid, prioridad) // creamos hilo main
		pcb.Tid = append(pcb.Tid, tcb.Tid)   // agregamos el hilo a la listas de hilos del proceso
		enviarTCBMemoria(tcb, path)

		encolarProcesoNew(pcb)

		encolarReady(tcb)

	} else {
		slog.Warn("El tamanio del proceso es mas grande que la memoria, esperando a que finalice otro proceso ....")
		// esperar a que finalize otro proceso y volver a consultar por el espacio en memoria para inicializarlo
		<-finProceso
		iniciarProceso(path, size, prioridad)
	}
	// COMO LE AVISAMOS A CPU QUE CONTINUE CON LA PROXIMA INSTRUCCION?
	return nil
}

func FinalizarProceso(w http.ResponseWriter, r *http.Request) {
	var hilo TCBRequest
	decoder := json.NewDecoder(r.Body)

	err := decoder.Decode(&hilo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := hilo.Pid
	tid := hilo.Tid

	if tid == 0 {
		err = exitProcess(pid)
	} else {
		slog.Warn("El hilo no es el principal, no se puede ejecutar esta instruccion")
		return //Ver como hacer para que no finalice el kernel y el hilo continue con su siguiente instruccion
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func exitProcess(pid int) error {

	for _, pcb := range colaNewproceso { // lo que hace range es recoreer la lista de procesos y devuelve la posicion en la que esta y el proceso

		if pcb.Pid == pid { // si coincide el pid del proceso con el pid que se quiere finalizar

			quitarProcesoNew(pcb) // sacamos el proceso de la cola de new

			encolarProcesoExit(pcb) // agregamos el proceso a la cola de exit

			// Eliminar todos los hilos del proceso
			for i := len(colaReadyHilo) - 1; i >= 0; i-- { // recorremos la lista de hilos ready
				if colaReadyHilo[i].Pid == pid { // si el pid del hilo coincide con el pid del proceso

					quitarReady(colaReadyHilo[i]) // sacamos el hilo de la cola de ready

					encolarExit(colaReadyHilo[i]) // agregamos el hilo a la cola de exit

				}
			}

			for i := len(colaExecHilo) - 1; i >= 0; i-- { // recorremos la lista de hilos en ejecucion
				if colaExecHilo[i].Pid == pid { // si el pid del hilo coincide con el pid del proceso

					quitarExec(colaExecHilo[i]) // sacamos el hilo de la cola de ejecucion

					encolarExit(colaReadyHilo[i])

					// FALTA ENVIAR INTERRUPCION A CPU PARA QUE SAQUE EL HILO DE EJECUCION Y META A OTRO NUEVO
				}
			}

			for i := len(colaBlockHilo) - 1; i >= 0; i-- { // recorremos la lista de hilos bloqueados
				if colaBlockHilo[i].Pid == pid { // si el pid del hilo coincide con el pid del proceso

					quitarBlock(colaBlockHilo[i]) // sacamos el hilo de la cola de bloqueados

					encolarExit(colaReadyHilo[i])

				}
			}

			resp := enviarProcesoFinalizadoAMemoria(pcb)

			if resp == nil {
				//mutexProcesosActivos.Lock()
				//defer mutexProcesosActivos.Unlock()

				// eliminar el PCB del proceso terminado
				//delete(procesosActivos, pid)

				// Notificar a traves del canal
				finProceso <- true
			}
		}
	}

	return nil
}

func enviarProcesoFinalizadoAMemoria(pcb PCB) error {

	memoryRequest := PCBRequest{Pid: pcb.Pid}

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(&memoryRequest)

	if err != nil {
		slog.Error("error codificando" + err.Error())
		return err
	}

	url := fmt.Sprintf("http://%s:%d/finalizacionProceso", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("Error enviando TCB. ip: %s - puerto: %d", ip, puerto)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error en la respuesta del módulo de CPU - status_code: %d ", resp.StatusCode)
		return err
	}
	return nil
}

func consultaEspacioAMemoria(size int, pcb PCB) bool {
	var memoryRequest KernelRequest
	memoryRequest.Size = size
	memoryRequest.PCB = pcb

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(memoryRequest)

	if err != nil {
		slog.Error("Fallo el proceso: error codificando " + err.Error())
		return false
	}

	url := fmt.Sprintf("http://%s:%d/hayEspacioEnLaMemoria", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("error enviando tamanio del proceso", slog.Int("pid", pcb.Pid), slog.String("ip", ip), slog.Int("puerto", puerto))
		return false
	}
	if resp.StatusCode != http.StatusOK {
		slog.Warn("El tamaño del proceso es más grande que la memoria disponible", slog.Int("pid", pcb.Pid)) //Se hace aca, porque si lo ponemos como un else de la consulta a memoria, ante cualquier error responde esto
		return false
	}

	return true
}

func encolarProcesoNew(pcb PCB) {
	mutexColaNewproceso.Lock()
	colaNewproceso = append(colaNewproceso, pcb)
	mutexColaNewproceso.Unlock()

	log.Printf("## (<PID %d> : 0)Se crea el proceso - Estado: NEW", pcb.Pid)
}

func encolarProcesoExit(pcb PCB) {

	mutexColaExitproceso.Lock()
	colaExitproceso = append(colaExitproceso, pcb)
	mutexColaExitproceso.Unlock()

	log.Printf(" ## (<PID: %d>) finaliza el Proceso - Estado: EXIT ##", pcb.Pid)
}

func quitarProcesoNew(pcb PCB) {
	mutexColaNewproceso.Lock()
	for i, p := range colaNewproceso {
		if p.Pid == pcb.Pid {
			colaNewproceso = append(colaNewproceso[:i], colaNewproceso[i+1:]...)
		}
	}
	mutexColaNewproceso.Unlock()
}

/*---------- FUNCIONES HILOS ----------*/

func createTCB(pid int, prioridad int) TCB {
	nextTid++

	return TCB{
		Pid:             pid,
		Tid:             nextTid - 1,
		Prioridad:       prioridad,
		HilosBloqueados: []int{},
	}
}

func getTcb(pid int, tid int) TCB {
	for _, tcb := range colaReadyHilo {
		if tcb.Pid == pid && tcb.Tid == tid {
			return tcb
		}
	}
	for _, tcb := range colaExecHilo {
		if tcb.Pid == pid && tcb.Tid == tid {
			return tcb
		}
	}
	for _, tcb := range colaBlockHilo {
		if tcb.Pid == pid && tcb.Tid == tid {
			return tcb
		}
	}
	return TCB{}
}

func removeTid(tids []int, tid int) []int {
	for i, t := range tids {
		if t == tid {
			return append(tids[:i], tids[i+1:]...)
		}
	}
	return tids
}

func CrearHilo(w http.ResponseWriter, r *http.Request) {
	var hilo CrearHiloResponse
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&hilo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := hilo.Pid
	path := hilo.Path
	prioridad := hilo.Prioridad

	err = iniciarHilo(pid, path, prioridad)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func iniciarHilo(pid int, path string, prioridad int) error {

	tcb := createTCB(pid, prioridad)   // creo hilo
	enviarTCBMemoria(tcb, path)        // envio hilo a memoria con el path que le corresponde ejecutar
	pcb := getPCB(pid)                 // obtengo el proceso al que pertenece el hilo
	pcb.Tid = append(pcb.Tid, tcb.Tid) // agrego el hilo a la lista de hilos del proceso
	encolarReady(tcb)                  // agrego el hilo a la cola de ready
	// COMO LE DECIMOS A MEMORIA QUE CONTINUE CON LA PROXIMA INSTRUCCION?

	return nil
}

func FinalizarHilo(w http.ResponseWriter, r *http.Request) { //pedir a cpu que nos pase PID Y TID del hilo
	var hilo TCBRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&hilo)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := hilo.Pid
	tid := hilo.Tid

	err = exitHilo(pid, tid)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func CancelarHilo(w http.ResponseWriter, r *http.Request) {
	var hilosCancel CambioHilos
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&hilosCancel)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := hilosCancel.Pid
	//tid := hilosCancel.TidActual
	tidEliminar := hilosCancel.TidCambio

	err = exitHilo(pid, tidEliminar)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	//COMO HACEMOS PARA QUE EL HILO QUE INVOCO A LA FUNCION CONTINUE SU EJECUCION
}

func EntrarHilo(w http.ResponseWriter, r *http.Request) { //debe ser del mismo proceso

	var hilosJoin CambioHilos

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&hilosJoin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := hilosJoin.Pid
	tidActual := hilosJoin.TidActual
	tidAEjecutar := hilosJoin.TidCambio

	tcbActual := getTcb(pid, tidActual)
	tcbAEjecutar := getTcb(pid, tidAEjecutar)

	tcbAEjecutar.HilosBloqueados = append(tcbAEjecutar.HilosBloqueados, tidActual)

	quitarExec(tcbActual)
	encolarBlock(tcbActual, "PTHREAD_JOIN")
	quitarReady(tcbAEjecutar)
	encolarExec(tcbAEjecutar)

	//mandar interrupcion a cpu para que saque al hilo actual y ejecute el hilo a ejecutar
}

func exitHilo(pid int, tid int) error {

	hilo := getTcb(pid, tid)
	pcb := getPCB(pid)
	pcb.Tid = removeTid(pcb.Tid, tid)

	quitarExec(hilo) // NO ES QUE LO QUITA DE LAS TRES COLAS SINO QUE SE FIJA EN CUAL ESTA Y LO QUITA DE ESA

	quitarReady(hilo)

	quitarBlock(hilo)

	encolarExit(hilo)

	for _, tidBloqueado := range hilo.HilosBloqueados {
		// desbloquear hilos bloqueados por el hilo que finalizo
		desbloquearHilo(tidBloqueado, pid)
	}

	err := enviarHiloFinalizadoAMemoria(hilo)

	if err != nil {
		log.Printf("Error al enviar hilo finalizado a memoria, pid: %d - tid: %d", hilo.Pid, hilo.Tid)
		return err
	}

	return nil
}

// Hilos algoritmos planificacion
func ejecutarHilosFIFO() {
	var Hilo TCB
	for {
		if len(colaReadyHilo) > 0 && len(colaExecHilo) == 0 {
			Hilo = colaReadyHilo[0]
			ejecutarInstruccion(Hilo)
		}
	}
}

func ejecutarInstruccion(Hilo TCB) {
	if len(colaReadyHilo) > 0 && len(colaExecHilo) == 0 {

		quitarReady(Hilo)

		encolarExec(Hilo)

		enviarTCBCpu(Hilo) // envio el hilo a la cpu para que ejecute sus instruciones
	}
}

// Funciones de envio de TCB
func enviarTCBCpu(tcb TCB) error {

	cpuRequest := TCBRequest{}
	cpuRequest.Pid = tcb.Pid
	cpuRequest.Tid = tcb.Tid

	puerto := globals.ClientConfig.PuertoCpu
	ip := globals.ClientConfig.IpCpu

	body, err := json.Marshal(&cpuRequest)

	if err != nil {
		slog.Error("error codificando " + err.Error())
		return err
	}

	url := fmt.Sprintf("http://%s:%d/recibirTcb", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("Error enviando TCB", slog.String("ip", ip), slog.Int("puerto", puerto), slog.Any("error", err))
		return err
	}
	if resp.StatusCode != http.StatusOK {
		slog.Error("error en la respuesta del módulo de cpu:" + fmt.Sprintf("%v", resp.StatusCode))
		return err
	}
	return nil
}

func enviarTCBMemoria(tcb TCB, path string) error {

	memoryRequest := TCBRequestMemory{}
	memoryRequest.Pid = tcb.Pid
	memoryRequest.Tid = tcb.Tid
	memoryRequest.Path = path

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(&memoryRequest)

	if err != nil {
		slog.Error("error codificando " + err.Error())
		return err
	}

	url := fmt.Sprintf("http://%s:%d/recibirTcb", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("Error enviando TCB", slog.String("ip", ip), slog.Int("puerto", puerto), slog.Any("error", err)) // err contiene el error que causo que no se envie la tcb
		return err
	}
	if resp.StatusCode != http.StatusOK {
		slog.Error("Error en la respuesta del modulo de CPU", slog.Int("status_code", resp.StatusCode))
		return err
	}
	return nil
}

func enviarHiloFinalizadoAMemoria(hilo TCB) error {

	memoryRequest := TCBRequest{}
	memoryRequest.Pid = hilo.Pid
	memoryRequest.Tid = hilo.Tid

	puerto := globals.ClientConfig.PuertoMemoria
	ip := globals.ClientConfig.IpMemoria

	body, err := json.Marshal(&memoryRequest)

	if err != nil {
		slog.Error("error codificando" + err.Error())
		return err
	}

	url := fmt.Sprintf("http://%s:%d/finalizacionHilo", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))

	if err != nil {
		slog.Error("Error enviando TCB para finalizarlo. ip: %d - puerto: %s", ip, puerto)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Error en la respuesta del modulo de CPU. status_code: %d", resp.StatusCode)
		return err
	}
	return nil
}

// Funciones de estados de hilos
func desbloquearHilo(tid int, pid int) {
	for _, hilo := range colaBlockHilo {
		if hilo.Tid == tid && hilo.Pid == pid {
			quitarBlock(hilo)

			encolarReady(hilo)

			log.Printf(" ## (<PID: %d>:<TID: %d>) - Pasa de Block a Ready ##", hilo.Pid, hilo.Tid)
		}
	}
}

func encolarReady(tcb TCB) {

	mutexColaReadyHilo.Lock()
	colaReadyHilo = append(colaReadyHilo, tcb)
	mutexColaReadyHilo.Unlock()

	log.Printf("## (<PID %d>:<TID %d>) Se crea el Hilo - Estado: READY", tcb.Pid, tcb.Tid)
}

func encolarExec(tcb TCB) {
	mutexColaExecHilo.Lock()
	colaExecHilo = append(colaExecHilo, tcb)
	mutexColaExecHilo.Unlock()

	log.Printf("## (<PID %d>:<TID %d>) Se ejecuta el Hilo - Estado: EXEC", tcb.Pid, tcb.Tid)
}

func encolarBlock(tcb TCB, motivo string) {
	mutexColaBlockHilo.Lock()
	colaBlockHilo = append(colaBlockHilo, tcb)
	mutexColaBlockHilo.Unlock()

	log.Printf("(<PID: %d >:<TID: %d >) - Bloqueado por: %s", tcb.Pid, tcb.Tid, motivo)
}

func encolarExit(tcb TCB) {
	mutexColaExitHilo.Lock()
	colaExitHilo = append(colaExitHilo, tcb)
	mutexColaExitHilo.Unlock()

	log.Printf(" ## (<PID: %d>:<TID: %d>) finaliza el hilo - Estado: EXIT ##", tcb.Pid, tcb.Tid)
}

func quitarReady(tcb TCB) {
	mutexColaReadyHilo.Lock()
	for i, t := range colaReadyHilo {
		if t.Pid == tcb.Pid && t.Tid == tcb.Tid {
			colaReadyHilo = append(colaReadyHilo[:i], colaReadyHilo[i+1:]...)
		}
	}
	mutexColaReadyHilo.Unlock()
}

func quitarExec(tcb TCB) {
	mutexColaExecHilo.Lock()
	for i, t := range colaExecHilo {
		if t.Pid == tcb.Pid && t.Tid == tcb.Tid {
			colaExecHilo = append(colaExecHilo[:i], colaExecHilo[i+1:]...)
		}
	}
	mutexColaExecHilo.Unlock()
}

func quitarBlock(tcb TCB) {
	mutexColaBlockHilo.Lock()
	for i, t := range colaBlockHilo {
		if t.Pid == tcb.Pid && t.Tid == tcb.Tid {
			colaBlockHilo = append(colaBlockHilo[:i], colaBlockHilo[i+1:]...)
		}
	}
	mutexColaBlockHilo.Unlock()
}

// SYSCALL IO

func ManejarIo(w http.ResponseWriter, r *http.Request) {
	var ioSyscall IOsyscall
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&ioSyscall)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	pid := ioSyscall.Pid
	tid := ioSyscall.Tid
	tiempoIO := ioSyscall.TiempoIO

	tcb := getTcb(pid, tid)
	if tcb.Pid == 0 && tcb.Tid == 0 {
		http.Error(w, "TCB no encontrada", http.StatusNotFound)
		return
	}

	quitarExec(tcb)
	encolarBlock(tcb, "IO")

	go func() {
		// Simulate IO operation
		time.Sleep(time.Duration(tiempoIO) * time.Millisecond)
		quitarBlock(tcb)
		encolarReady(tcb)
	}()

	w.WriteHeader(http.StatusOK)
}

// MUTEX
func CrearMutex(w http.ResponseWriter, r *http.Request) {

}

func BloquearMutex(w http.ResponseWriter, r *http.Request) {

}

func LiberarMutex(w http.ResponseWriter, r *http.Request) {

}

func DumpMemory(w http.ResponseWriter, r *http.Request) {

}
