package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
)

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
	instruction string
	parameters  []string
}
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
		errExe, syscall := Execute(contexto, instruction)
		if errExe != nil {
			log.Printf("Error al ejecutar %v. ERROR: %v", intructionLine, errExe)
			break // es correcto que al tener un error al ejecutar la intruccion termine el ciclo
		}
		log.Printf("## TID: %d - Ejecutando: %s - Parametos : %v", contexto.tcb.tid, instruction.instruction, instruction.parameters)

		if syscall {
			log.Printf("Se termina el ciclo por syscall")
			//se debe hacer antes de chequear las interrupciones
			break
		}

	}
	// pasar el control al kernel y comunicar razon de la intrerrupcion que etsara en el error
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

type InstructionResponse struct {
	Instruction string `json:"instruction"`
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
	var instructionDecoded DecodedInstruction
	if len(instructionLine) == 0 {
		err := fmt.Errorf("null intruction")
		return instructionDecoded, err
	}

	instructionDecoded.instruction = instructionLine[0]
	instructionDecoded.parameters = instructionLine[1:]
	return instructionDecoded, nil
}

func Execute(ContextoDeEjecucion *contextoEjecucion, intruction DecodedInstruction) (error, bool) {
	var syscall bool
	tcb := &ContextoDeEjecucion.tcb
	var err error
	instructionName := intruction.instruction
	var parameters []string = intruction.parameters
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
	if err != nil {
		log.Printf("Error al ejecutar la instruccion : %v", err)
		return err, syscall
	}
	return nil, syscall
}

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

	//contextoActual := GetContextoEjecucion(processAndThreadIDs.pid, processAndThreadIDs.tid)
	//InstructionCycle(&contextoActual)

}

//interface{} permite manejar tipos de datos desconocidos

func Set(registrosCPU *TCB, valor string, registro string) error {

	//ver si el registro en la instruccion existe
	/*
		registers := reflect.ValueOf(registrosCPU)

		campoRef := registers.Elem().FieldByName(registro)

		if !campoRef.IsValid() {
			err := fmt.Errorf("SET error :registro %s no existente en la estructura", registro)
			return err
		}
		//pasar el string a unit36 ver si debe tomarse un tipo generico
		if campoRef.CanSet() {
			err := fmt.Errorf("SET error: cannot set %v", campoRef)
			return err
		}
	*/

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

}

/*
	func ConvertStringToUint32(cadena string) (uint32, error) {
		valorParse64, err := strconv.ParseUint(cadena, 10, 32)

		if err != nil {
			return 0, err
		}

		valorParse32 := uint32(valorParse64)
		return valorParse32, nil
	}
*/
func Read_Memory(context *contextoEjecucion, registroDireccion string, registroDato string) error {

	register, err := GetRegister(&context.tcb, registroDireccion)
	if err != nil {
		return err
	}
	direccionLogica := register

	direccionFisica, errT := TranslateAdress(*direccionLogica, context.pcb.base, context.pcb.limit)
	if errT != nil {
		return err // ver su solo esto basta o si hay que aca terminar el programa
	}
	log.Printf("Leer memoria con direccion fisica %d", direccionFisica)
	//leer en memoria
	return nil

}

func GetRegister(registrosCPU *TCB, registro string) (*uint32, error) {

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

func TranslateAdress(direccionLogica uint32, base uint32, limite uint32) (uint32, error) {
	direccionFisica := direccionLogica + base

	if direccionFisica > limite {
		err := fmt.Errorf("Segmentation Fault")
		return 0, err
	}

	return direccionFisica, nil
}

func Sumar(registrosCPU *TCB, registroDestino string, registroOrigen string) error {
	originRegister, finalRegister, err := obtenerOperandos(registrosCPU, registroDestino, registroOrigen)

	if err != nil {
		return err
	}

	*finalRegister += *originRegister
	return nil
}

func Restar(registrosCPU *TCB, registroDestino string, registroOrigen string) error {
	originRegister, finalRegister, err := obtenerOperandos(registrosCPU, registroDestino, registroOrigen)

	if err != nil {
		return err
	}

	*finalRegister = *finalRegister - *originRegister
	return nil
}

func obtenerOperandos(registrosCPU *TCB, registroDestino string, registroOrigen string) (*uint32, *uint32, error) {

	originRegister, errR := GetRegister(registrosCPU, registroOrigen)
	if errR != nil {
		return nil, nil, errR
	}

	register, err2 := GetRegister(registrosCPU, registroDestino)
	if err2 != nil {
		return nil, nil, err2
	}

	return originRegister, register, nil
}

func JNZ(registrosCPU *TCB, registro string, instruccion string) error {

	register, err := GetRegister(registrosCPU, registro)
	if err != nil {
		return err
	}
	instruction, errI := strconv.Atoi(instruccion)
	if errI != nil {
		return errI
	}
	if *register != 0 {
		registrosCPU.PC = uint32(instruction)
	}
	return nil

}

func Log(registrosCPU *TCB, registro string) error {

	register, err := GetRegister(registrosCPU, registro)
	if err != nil {
		return err
	}
	log.Printf("EL registro %s contiene el valor %d", registro, *register)
	return nil

}

func DumpMemory(contexto *contextoEjecucion) error {

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

func IO(contexto *contextoEjecucion, tiempo string) error {

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

func CreateProcess(contexto *contextoEjecucion, archivoInstruct string, tamArch string, prioridadTID string) error {
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

func CreateThead(contexto *contextoEjecucion, archivoInstruct string, prioridadTID string) error {
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

func JoinThead(contexto *contextoEjecucion, tid string) error {
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

func CancelThead(contexto *contextoEjecucion, tid string) error {
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

func ThreadExit(contexto *contextoEjecucion) error {

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

func ProcessExit(contexto *contextoEjecucion) error {

	err := AcualizarContextoDeEjecucion(contexto)

	if err != nil {
		log.Printf("Error al actualziar contexto de ejecucion")
		return err
	}

	errM := EnviarAModulo(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, nil, "exirProcess")
	if errM != nil {
		return errM
	}
	return nil

}
