package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
)

// var globales
var mutexInterrupt sync.Mutex
var nuevaInterrupcion Interrupt
var memoryData sync.WaitGroup
var dataFromMemory uint32 //verrr
var flagSegmentationFault bool

//DEFINICION DE TIPOS

type InstructionResponse struct {
	Instruction string `json:"instruction"`
}
type Interrupcion struct {
	Pid          int  `json:"pid"`
	Tid          int  `json:"tid"`
	Interrupcion bool `json:"interrupcion"`
}
type KernelInterrupcion struct { // ver con KERNEL
	Pid    int    `json:"pid"`
	Tid    int    `json:"tid"`
	Motivo string `json:"motivo"`
}
type Interrupt struct {
	Pid               int
	Tid               int
	Motivo            string
	flagInterrucption bool
}
type MemoryRequest struct {
	PID     int    `json:"pid"`
	TID     int    `json:"tid,omitempty"`
	Address uint32 `json:"address"`        //direccion de memoria a leer
	Size    int    `json:"size,omitempty"` //tamaño de la memoria a leer
	Data    []byte `json:"data,omitempty"` //datos a escribir o leer y los devuelvo
	Port    int    `json:"port,omitempty"` //puerto
}
type Mensaje struct {
	Mensaje string `json:"mensaje"`
}
type PCB struct {
	pid   int
	base  uint32
	limit uint32
}

type TCB struct {
	tid int
	AX  uint32
	BX  uint32
	CX  uint32
	DX  uint32
	EX  uint32
	FX  uint32
	GX  uint32
	HX  uint32
	PC  uint32
}

type contextoEjecucion struct {
	pcb PCB
	tcb TCB
}
type DecodedInstruction struct {
	instruction FuncInctruction
	parameters  []string
}
type FuncInctruction func(*contextoEjecucion, []string) error

type BodyContexto struct {
	Pcb PCB `json:"pcb"`
	Tcb TCB `json:"tcb"`
}

type ProcessCreateBody struct {
	Path     string `json:"path"`
	Size     string `json:"size"`
	Priority string `json:"prioridad"`
}
type KernelExeReq struct {
	Pid int `json:"pid"` // ver cuales son los keys usados en Kernel
	Tid int `json:"tid"`
}
type IOReq struct {
	Tiempo int `json:"tiempo"`
}

type IniciarProcesoBody struct {
	Path      string `json:"path"`
	Size      int    `json:"size"`
	Prioridad int    `json:"prioridad"`
}

type CrearHiloBody struct {
	Pid       int    `json:"pid"`
	Path      string `json:"path"`
	Prioridad int    `json:"prioridad"`
}
type EfectoHiloBody struct {
	Pid       int `json:"pid"`
	TidActual int `json:"tidActual"`
	TidCambio int `json:"tidAEjecutar"`
}

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

func init() {
	ConfigsCpu := IniciarConfiguracion("configsCPU/config.json")
	//EnviarMensaje(ConfigsCpu.IpKernel, ConfigsCpu.PuertoKernel, "Hola Kernel, Soy CPU")
	EnviarMensaje(ConfigsCpu.IpMemoria, ConfigsCpu.PuertoMemoria, "Hola Memoria, Soy CPU")
}

func ConfigurarLogger() {
	logFile, err := os.OpenFile("tp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func RecibirMensaje(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var mensaje Mensaje
	err := decoder.Decode(&mensaje)
	if err != nil {
		log.Printf("Error al decodificar mensaje: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Println("Conexion con CPU")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func EnviarMensaje(ip string, puerto int, mensajeTxt string) {
	mensaje := Mensaje{Mensaje: mensajeTxt}
	body, err := json.Marshal(mensaje)
	if err != nil {
		log.Printf("error codificando mensaje: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/mensaje", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}
	log.Printf("contexto del servidor: %s", resp.Status)
}

func RecibirPaquete(w http.ResponseWriter, r *http.Request) {
	log.Printf("entrando a func paquete")

	/*if r.Method != http.MethodGet {
		http.Error(w, "Método erroneo", http.StatusMethodNotAllowed) //detecta metodo de protocolo https
		log.Printf("error codificando mensaje: %s", err.Error())
		return
	}
	*/
	var paquete globals.Paquete
	log.Printf("creando paquete")
	if err := json.NewDecoder(r.Body).Decode(&paquete); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("ID:" + paquete.ID + "\n")
	log.Printf("Mensaje:" + paquete.Mensaje + "\n")
	log.Printf("Rune: " + string(paquete.Array) + "\n")
	log.Printf("Tamanio: %d\n", paquete.Size)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
func PedirPaquete() {
	ip := globals.ClientConfig.IpKernel
	puerto := globals.ClientConfig.PuertoKernel

	mensaje := "HOLA"
	body, _ := json.Marshal(mensaje)
	url := fmt.Sprintf("http://%s:%d/enviarPaqueteACPU", ip, puerto)
	resp, _ := http.Post(url, "application/json", bytes.NewBuffer(body))
	log.Printf("contexto del servidor: %s", resp.Status)
}

// FUNCIONES PRINCIPALES
func RecibirPIDyTID(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var processAndThreadIDs KernelExeReq
	err := decoder.Decode(&processAndThreadIDs)
	if err != nil {
		log.Printf("Error al decodificar el pedido del Kernel: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}

	log.Printf("Cpu recibe TID : %d PID:%d del Kernel", processAndThreadIDs.Pid, processAndThreadIDs.Tid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	contextoActual := GetContextoEjecucion(processAndThreadIDs.Pid, processAndThreadIDs.Tid)
	InstructionCycle(&contextoActual)

}
func GetContextoEjecucion(pid int, tid int) (context contextoEjecucion) {
	var contextoDeEjecucion contextoEjecucion
	log.Printf("PCB : %d TID : %d - Solicita Contexto de Ejecucion", pid, tid)
	url := fmt.Sprintf("http://%s:%d//obtenerContextoDeEjecucion?pid=%d&tid=%d", globals.ClientConfig.IpMemoria, globals.ClientConfig.PuertoMemoria, pid, tid)
	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("error al enviar la solicitud al módulo de memoria: %v", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("error en la respuesta del módulo de memoria: %v", response.StatusCode)
		log.Println(err)
		return
	}
	var contexto BodyContexto
	errorDecode := json.NewDecoder(response.Body).Decode(&contexto)
	if errorDecode != nil {
		log.Println("Error al decodificar el contexto de ejecucion", errorDecode)
		return
	}
	log.Printf("PCB : %d TID : %d - Solicitud Contexto de Ejecucion Exitosa", pid, tid)
	contextoDeEjecucion.pcb = contexto.Pcb
	contextoDeEjecucion.tcb = contexto.Tcb
	return contextoDeEjecucion
}

func InstructionCycle(contexto *contextoEjecucion) {

	for {

		intructionLine, err := Fetch(contexto.pcb.pid, contexto.tcb.tid, &contexto.tcb.PC)
		if err != nil {
			log.Printf("Error al buscar intruccion en el pc %d. ERROR : %v", contexto.tcb.PC, err)
			break
		}
		instruction, err2 := Decode(intructionLine)
		if err2 != nil {
			log.Printf("Error en etapa Decode. ERROR : %v", err2)
			break
		}
		errExe := Execute(contexto, instruction)
		if errExe != nil {
			log.Printf("Error al ejecutar %v. ERROR: %v", intructionLine, errExe)
		}
		log.Printf("## TID: %d - Ejecutando: %s - Parametos : %v", contexto.tcb.tid, intructionLine[0], instruction.parameters)

		flag, motivo := CheckInterrupt(*contexto)
		if flag {
			err := RealizarInterrupcion(contexto, motivo) //necesito ok de la interrucpion
			if err == nil {
				break
			}
			log.Printf("Error al ejecutar la interrupcion %v", err)
		}

	}

}

func RealizarInterrupcion(contexto *contextoEjecucion, motivo string) error {
	err := AcualizarContextoDeEjecucion(contexto)
	if err == nil {
		log.Printf("Error al actualizar contexto de ejecucion para la interrupcion")
		return err
	}
	var kernelInt KernelInterrupcion
	kernelInt.Motivo = motivo
	kernelInt.Pid = contexto.pcb.pid
	kernelInt.Tid = contexto.tcb.tid
	body, err2 := json.Marshal(kernelInt)

	if err2 != nil {
		log.Printf("Error al codificar el mensaje de la interrupcion")
		return err2
	}
	err3 := EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, bytes.NewBuffer(body), "/interrupcion") // ver con kernel
	if err3 != nil {
		return err3
	}
	return nil

}

func Fetch(pid int, tid int, PC *uint32) ([]string, error) {

	pc := *PC
	url := fmt.Sprintf("http://%s:%d//obtenerContextoDeEjecucion?pid=%d&tid=%d&pc=%d", globals.ClientConfig.IpMemoria, globals.ClientConfig.PuertoMemoria, pid, tid, pc)
	response, err := http.Get(url)
	if err != nil {
		log.Fatalf("error al enviar la solicitud al módulo de memoria: %v", err)
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("error en la respuesta del módulo de memoria: %v", response.StatusCode)
		log.Println(err)
		return nil, err
	}
	var instructionResponse InstructionResponse
	errorDecode := json.NewDecoder(response.Body).Decode(&instructionResponse)
	if errorDecode != nil {
		log.Println("Error al decodificar la instruccion", errorDecode)
		return nil, errorDecode
	}
	instructions := strings.Split(instructionResponse.Instruction, " ") //la instruccion recibida esta separada por comas, y se tomara cada una de las partes y pondra en vector de strings

	log.Printf("PID: %d TID: %d - FETCH - Program Counter: %d", pid, tid, pc)

	*PC = pc + 1 //esta bien colocarlo de esta manera ????

	return instructions, nil

}

func Decode(instructionLine []string) (DecodedInstruction, error) {
	var SetInstructions map[string]FuncInctruction = map[string]FuncInctruction{
		"SET":            Set,
		"SUM":            Sumar,
		"SUB":            Restar,
		"JNZ":            JNZ,
		"LOG":            Log,
		"DUMP_MEMORY":    DumpMemory,
		"IO":             IO,
		"PROCESS_CREATE": CreateProcess,
		"THREAD_CREATE":  CreateThead,
		"THREAD_JOIN":    JoinThead,
		"THREAD_CANCEL":  CancelThead,
		"THREAD_EXIT":    ThreadExit,
		"PROCESS_EXIT":   ProcessExit,
		"READ_MEM":       Read_Memory,
		"WRITE_MEM":      Write_Memory,
	}

	var instructionDecoded DecodedInstruction
	if len(instructionLine) == 0 {
		err := fmt.Errorf("null intruction")
		return instructionDecoded, err
	}
	functionInstruction, ok := SetInstructions[instructionLine[0]]
	if !ok {
		err := fmt.Errorf("La instruccion %s no existe", instructionLine[0])
		return instructionDecoded, err
	}
	instructionDecoded.instruction = functionInstruction
	instructionDecoded.parameters = instructionLine[1:]

	return instructionDecoded, nil
}

func Execute(ContextoDeEjecucion *contextoEjecucion, intruction DecodedInstruction) error {

	//var syscall bool
	instructionFunc := intruction.instruction
	var parameters []string = intruction.parameters

	err := instructionFunc(ContextoDeEjecucion, parameters)

	if err != nil {
		log.Printf("Error al ejecutar la instruccion : %v", err)
		return err
	}
	return nil

}

func CheckInterrupt(contexto contextoEjecucion) (bool, string) {
	if flagSegmentationFault {
		flagSegmentationFault = false
		return true, "SegmentationFault"
	}
	mutexInterrupt.Lock()
	if nuevaInterrupcion.flagInterrucption && contexto.pcb.pid == nuevaInterrupcion.Pid && contexto.tcb.tid == nuevaInterrupcion.Tid {
		nuevaInterrupcion.flagInterrucption = false
		return true, nuevaInterrupcion.Motivo
	}
	mutexInterrupt.Unlock()
	return false, ""

}

// Funciones del set de intrucciones
func Set(registrosCPU *contextoEjecucion, parameters []string) error {

	valor := parameters[1]
	registro := parameters[0]
	registers := reflect.ValueOf(registrosCPU.tcb)
	valorUint, err := strconv.ParseUint(valor, 10, 32)
	if err != nil {
		return err
	}
	err2 := ModificarValorCampo(registers, registro, uint32(valorUint))
	if err2 != nil {
		return err2
	}

	/*
		register, errR := GetRegister(registrosCPU, registro)
		if errR != nil {
			return errR
		}

		valorParse, err := strconv.ParseUint(valor, 10, 32)
		if err != nil {
			log.Printf("SET error: Error al convertir valor %s al del tipo del registro %s", valor, registro)
			return err
		}
		*register = uint32(valorParse)

		return nil
	*/
	return nil

}
func Read_Memory(context *contextoEjecucion, parameters []string) error {

	//registroDato := parameters[0]
	registroDireccion := parameters[1]
	registers := reflect.ValueOf(context.tcb)
	// obtnego el registro del destino del dato
	direccionLogica, err := ObtenerValorCampo(registers, registroDireccion)
	if err != nil {
		return err
	}
	direccionFisica, errT := TranslateAdress(direccionLogica, context.pcb.base, context.pcb.limit)
	if errT != nil {
		return errT
	}
	log.Printf("Leer memoria con direccion fisica %d", direccionFisica)
	//leer en memoria
	//VER SI SE PUEDE PEDIR Y ENVIAR POR MISMO PUERTO
	var memReq MemoryRequest
	memReq.Address = direccionFisica
	memReq.PID = context.pcb.pid
	memReq.TID = context.tcb.tid

	body, err2 := json.Marshal(memReq)

	if err2 != nil {
		return err2
	}
	err3 := EnviarAModulo(globals.ClientConfig.IpMemoria, globals.ClientConfig.PuertoMemoria, bytes.NewBuffer(body), "/ReadMemoryHandler")
	if err3 != nil {
		return err3
	}
	memoryData.Add(1)
	memoryData.Wait()

	err4 := ModificarValorCampo(registers, parameters[0], dataFromMemory)

	if err4 != nil {
		return err4
	}

	return nil

}

func RecieveDataFromMemory(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	var data uint32 // ver si esta bien con el tipo que envia memoria
	err := decoder.Decode(&data)
	if err != nil {
		log.Printf("Error al decodificar el pedido de la memorua: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	dataFromMemory = data
	memoryData.Done()

}

func Write_Memory(context *contextoEjecucion, parameters []string) error {
	//obtengo el dato
	registroDato := parameters[1]
	registers := reflect.ValueOf(context.tcb)
	dato, err := ObtenerValorCampo(registers, registroDato)
	if err != nil {
		return err
	}
	//obtengo la direccion
	registroDireccion := parameters[0]
	direccion, err2 := ObtenerValorCampo(registers, registroDireccion)
	if err2 != nil {
		return err2
	}
	direccionFisica, errTranslate := TranslateAdress(direccion, context.pcb.base, context.pcb.limit)
	if errTranslate != nil {
		return err
	}

	var memReq MemoryRequest
	memReq.Address = direccionFisica
	memReq.Data = PasarDeUintAByte(dato)
	memReq.PID = context.pcb.pid
	memReq.TID = context.tcb.tid

	body, err5 := json.Marshal(memReq)

	if err5 != nil {
		return err5
	}

	err3 := EnviarAModulo(globals.ClientConfig.IpMemoria, globals.ClientConfig.PuertoMemoria, bytes.NewBuffer(body), "/writememory")

	if err3 != nil {
		return err3
	}

	return nil
}

func PasarDeUintAByte(num uint32) []byte {
	numEnString := strconv.Itoa(int(num))

	return []byte(numEnString)
}
func TranslateAdress(direccionLogica uint32, base uint32, limite uint32) (uint32, error) {
	direccionFisica := direccionLogica + base

	if direccionFisica > limite {
		err := fmt.Errorf("Segmentation Fault")
		flagSegmentationFault = true
		return 0, err
	}

	return direccionFisica, nil
}

func Sumar(registrosCPU *contextoEjecucion, parameters []string) error {
	registroDestino := parameters[0]
	registroOrigen := parameters[1]
	registers := reflect.ValueOf(registrosCPU.tcb)
	originRegister, finalRegister, err := obtenerOperandos(registers, registroDestino, registroOrigen)

	if err != nil {
		return err
	}

	resta := finalRegister + originRegister
	err2 := ModificarValorCampo(registers, registroDestino, resta)
	if err2 != nil {
		return err2
	}
	return nil
}

func Restar(registrosCPU *contextoEjecucion, parameters []string) error {
	registroDestino := parameters[0]
	registroOrigen := parameters[1]
	registers := reflect.ValueOf(registrosCPU.tcb)
	originRegister, finalRegister, err := obtenerOperandos(registers, registroDestino, registroOrigen)

	if err != nil {
		return err
	}

	resta := finalRegister - originRegister
	err2 := ModificarValorCampo(registers, registroDestino, resta)
	if err2 != nil {
		return err2
	}
	return nil
}

func obtenerOperandos(registers reflect.Value, registroDestino string, registroOrigen string) (uint32, uint32, error) {
	/*
		originRegister, errR := GetRegister(registrosCPU, registroOrigen)
		if errR != nil {
			return nil, nil, errR
		}

		register, err2 := GetRegister(registrosCPU, registroDestino)
		if err2 != nil {
			return nil, nil, err2
		}
	*/
	// obtnego el registro del destino del dato
	valorOrigen, errOrigen := ObtenerValorCampo(registers, registroOrigen)
	if errOrigen != nil {
		return 0, 0, errOrigen
	}
	valorDestino, errDestino := ObtenerValorCampo(registers, registroDestino)
	if errDestino != nil {
		return 0, 0, errDestino
	}
	return valorOrigen, valorDestino, nil
}

func JNZ(registrosCPU *contextoEjecucion, parameters []string) error {
	instruccion := parameters[1]
	registro := parameters[0]
	registers := reflect.ValueOf(registrosCPU.tcb)
	register, err := ObtenerValorCampo(registers, registro)
	if err != nil {
		return err
	}
	instruction, errI := strconv.Atoi(instruccion)
	if errI != nil {
		return errI
	}
	if register != 0 {
		ModificarValorCampo(registers, registro, uint32(instruction))
	}
	return nil

}

func Log(registrosCPU *contextoEjecucion, parameters []string) error {
	registro := parameters[0]
	registers := reflect.ValueOf(registrosCPU.tcb)
	register, err := ObtenerValorCampo(registers, registro)
	if err != nil {
		return err
	}
	log.Printf("EL registro %s contiene el valor %d", registro, register)
	return nil

}

func DumpMemory(contexto *contextoEjecucion, parameters []string) error {

	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	errM := EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, nil, "vaciarMemoria")
	if errM != nil {
		return errM
	}
	return nil

}

func IO(contexto *contextoEjecucion, parameters []string) error {
	tiempo := parameters[0]
	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	tiempoReq, errI := strconv.Atoi(tiempo)
	if errI != nil {
		return errI
	}
	body, err := json.Marshal(IOReq{
		Tiempo: tiempoReq,
	})
	if err != nil {
		log.Printf("Error al codificar el mernsaje")
		return err
	}
	err = EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, bytes.NewBuffer(body), "manejarIo")
	if err != nil {
		return err
	}

	return nil

}

func CreateProcess(contexto *contextoEjecucion, parameters []string) error {
	archivoInstruct := parameters[0]
	tamArch := parameters[1]
	prioridadTID := parameters[2]
	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	tamArchReal, err := strconv.Atoi(tamArch)
	if err != nil {
		return err
	}
	priorityReal, err2 := strconv.Atoi(prioridadTID)
	if err2 != nil {
		return err
	}

	body, err := json.Marshal(IniciarProcesoBody{
		Path:      archivoInstruct,
		Size:      tamArchReal,
		Prioridad: priorityReal,
	})

	if err != nil {
		log.Printf("Error al codificar estructura de creacion de proceso")
		return err
	}
	err = EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, bytes.NewBuffer(body), "manejarIo")
	if err != nil {
		log.Printf("Error syscall IO : %v", err)
		return err
	}

	return nil
}

func CreateThead(contexto *contextoEjecucion, parameters []string) error {
	archivoInstruct := parameters[0]
	prioridadTID := parameters[1]
	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	priorityReal, err2 := strconv.Atoi(prioridadTID)
	if err2 != nil {
		return err
	}

	body, err := json.Marshal(CrearHiloBody{
		Path:      archivoInstruct,
		Pid:       contexto.pcb.pid,
		Prioridad: priorityReal,
	})

	if err != nil {
		log.Printf("Error al codificar estructura de creacion de hilo")
		return err
	}
	err = EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, bytes.NewBuffer(body), "crearHilo")
	if err != nil {
		log.Printf("Error syscall THREAD_CREATE : %v", err)
		return err
	}

	return nil
}

func JoinThead(contexto *contextoEjecucion, parameters []string) error {
	tid := parameters[0]
	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	tidParse, err := strconv.Atoi(tid)
	if err != nil {
		return err
	}

	body, err := json.Marshal(EfectoHiloBody{
		Pid:       contexto.pcb.pid,
		TidActual: contexto.tcb.tid,
		TidCambio: tidParse,
	})

	if err != nil {
		log.Printf("Error al codificar estructura de cambio de hilo")
		return err
	}
	err = EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, bytes.NewBuffer(body), "unirseAHilo")
	if err != nil {
		log.Printf("Error syscall THREAD_JOIN : %v", err)
		return err
	}

	return nil
}

func CancelThead(contexto *contextoEjecucion, parameters []string) error {
	tid := parameters[0]
	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	tidParse, err := strconv.Atoi(tid)
	if err != nil {
		return err
	}

	body, err := json.Marshal(EfectoHiloBody{
		Pid:       contexto.pcb.pid,
		TidActual: contexto.tcb.tid,
		TidCambio: tidParse,
	})

	if err != nil {
		log.Printf("Error al codificar estructura de cancelacion de hilo")
		return err
	}
	err = EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, bytes.NewBuffer(body), "cancelarHilo")
	if err != nil {
		log.Printf("Error syscall THREAD_CANCEL : %v", err)
		return err
	}

	return nil
}

func ThreadExit(contexto *contextoEjecucion, parameters []string) error {

	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	errM := EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, nil, "exirThread")
	if errM != nil {
		return errM
	}
	return nil

}

func ProcessExit(contexto *contextoEjecucion, parameters []string) error {

	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	errM := EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, nil, "exitProcess")
	if errM != nil {
		return errM
	}
	return nil

}

// funciones Auxiliares
// se suponen que todos los registros mantendran el tipo uint32

func AcualizarContextoDeEjecucion(contexto *contextoEjecucion) error {
	var contextoDeEjecucion BodyContexto
	contextoDeEjecucion.Pcb = contexto.pcb
	contextoDeEjecucion.Tcb = contexto.tcb
	body, err := json.Marshal(contextoDeEjecucion)
	if err != nil {
		log.Printf("Error al codificar el contexto")
		return err
	}
	errM := EnviarAModulo(globals.ClientConfig.IpMemoria, globals.ClientConfig.PuertoMemoria, bytes.NewBuffer(body), "actualizarContextoDeEjecucion")
	if errM != nil {
		return errM
	}
	return nil

}
func EnviarAModulo(ipModulo string, puertoModulo int, body io.Reader, endPoint string) error {

	url := fmt.Sprintf("http://%s:%d/%s", ipModulo, puertoModulo, endPoint)
	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		log.Printf("error enviando mensaje al End point %s - IP:%s - Puerto:%d", endPoint, ipModulo, puertoModulo)
		return err
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error al recibir la respuesta del End point %s - IP:%s - Puerto:%d", endPoint, ipModulo, puertoModulo)
		err := fmt.Errorf("%s", resp.Status)
		return err
	}
	return nil
}
func ObtenerValorCampo(estructura reflect.Value, nombreCampo string) (uint32, error) {
	campoRef := estructura.Elem().FieldByName(nombreCampo)
	if !campoRef.IsValid() {
		err := fmt.Errorf("No se encuentra el campo %s en la estructura", nombreCampo)
		return 0, err
	}
	//estamos suponiendo que jamas se podrá tener un numero de otro tipo que no sea unit32
	return (uint32(campoRef.Int())), nil

}
func ModificarValorCampo(estructura reflect.Value, nombreCampo string, nuevoValor uint32) error {
	//solo se aceptaran valores de tipo uint32
	campoRef := estructura.Elem().FieldByName(nombreCampo)
	if !campoRef.IsValid() {
		err := fmt.Errorf("No se encuentra el campo %s en la estructura", nombreCampo)
		return err
	}
	if !campoRef.CanSet() {
		err := fmt.Errorf("No se puede setear el valor del campo %s", nombreCampo)
		return err
	}
	campoRef.SetUint(uint64(nuevoValor))
	return nil
}
func RecieveInterruption(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var interrupction Interrupcion // ver si esta bien con el tipo que envia memoria
	err := decoder.Decode(&interrupction)
	if err != nil {
		log.Printf("Error al decodificar el pedido del Kerel: %s\n", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Error al decodificar mensaje"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	mutexInterrupt.Lock()
	nuevaInterrupcion.flagInterrucption = interrupction.Interrupcion
	nuevaInterrupcion.Pid = interrupction.Pid
	nuevaInterrupcion.Tid = interrupction.Tid
	mutexInterrupt.Unlock()
}

/*ng)
func GetRegister(registrosCPU *TCB, registro stri(*uint32, error) {

	switch registro {
	case "AX":
		return &registrosCPU.AX, nil
	case "BX":
		return &registrosCPU.BX, nil
	case "CX":
		return &registrosCPU.CX, nil
	case "DX":
		return &registrosCPU.DX, nil
	case "EX":
		return &registrosCPU.EX, nil
	case "FX":
		return &registrosCPU.FX, nil
	case "GX":
		return &registrosCPU.GX, nil
	case "HX":
		return &registrosCPU.HX, nil
	case "PC":
		return &registrosCPU.PC, nil
	default:
		err := fmt.Errorf("Registro %s no existente en la estructura", registro)
		return nil, err
	}
}
*/

/*
	switch instructionName {
	case "SET":
		err = Set(tcb, parameters[1], parameters[0])
	case "READ_MEM":
		//funcion
	case "WRITE_MEM":
		//funcion
	case "SUM":
		err = Sumar(tcb, parameters[0], parameters[1])
	case "SUB":
		err = Restar(tcb, parameters[0], parameters[1])
	case "JNZ":
		err = JNZ(tcb, parameters[0], parameters[1])
	case "LOG":
		err = Log(tcb, parameters[0])
	case "DUMP_MEMORY":
		syscall = true
		err = DumpMemory(ContextoDeEjecucion)
	case "IO":
		syscall = true
		err = IO(ContextoDeEjecucion, parameters[0])
	case "PROCESS_CREATE":
		syscall = true
		err = CreateProcess(ContextoDeEjecucion, parameters[0], parameters[1], parameters[2])
	case "THREAD_CREATE":
		syscall = true
		err = CreateThead(ContextoDeEjecucion, parameters[0], parameters[1])
	case "THREAD_JOIN":
		syscall = true
		err = JoinThead(ContextoDeEjecucion, parameters[0])
	case "THREAD_CANCEL":
		syscall = true
		err = CancelThead(ContextoDeEjecucion, parameters[0])
	case "MUTEX_CREATE":
		syscall = true
		//funcion
	case "MUTEX_LOCK":
		syscall = true
		//funcion
	case "MUTEX_UNLOCK":
		//funcion
	case "THREAD_EXIT":
		syscall = true
		err = ThreadExit(ContextoDeEjecucion)
	case "PROCESS_EXIT":
		syscall = true
		err = ProcessExit(ContextoDeEjecucion)
	default:
		log.Printf("Instruccion no valida %s", instructionName)
	}
*/
