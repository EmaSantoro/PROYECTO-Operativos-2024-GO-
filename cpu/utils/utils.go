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

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
)

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type PCB struct {
	pid   int
	base  int
	limit int
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

type BodyContexto struct {
	Pcb PCB `json:"pcb"`
	Tcb TCB `json:"tcb"`
}

type KernelExeReq struct {
	pid int `json:"pid"` // ver cuales son los keys usados en Kernel
	tid int `json:"tid"`
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

func InstructionCycle() {

	/*
		for (!exit flag || !interruption flag){
			//Fetch
		//decode
		//execute
		//check interrupt
		}
	*/

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

func Fetch(pid int, tid int, PC *int) ([]string, error) {

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

	*PC = pc + 1

	return instructions, nil

}

func Decode(instructionLine []string) (string, error) {
	if len(instructionLine) == 0 {
		err := fmt.Errorf("null intruction")
		return "nil", err
	}
	instruction := instructionLine[0]

	return instruction, nil
}

func Execute(ContextoDeEjecucion *TCB, intruction string, line []string) error {

	switch intruction {
	case "SET":
		err := Set(ContextoDeEjecucion, line[2], line[1])
		if err != nil {
			return err
		}

	case "READ_MEM":
		//funcion
	case "WRITE_MEM":
		//funcion
	case "SUM":
		//funcion
	case "SUB":
		//funcion
	case "JNZ":
		//funcion
	case "LOG":
		//funcion
	case "DUMP_MEMORY":
		//funcion
	case "IO":
		//funcion
	case "PROCESS_CREATE":
		//funcion
	case "THREAD_CREATE":
		//funcion
	case "THREAD_JOIN":
		//funcion
	case "THREAD_CANCEL":
		//funcion
	case "MUTEX_CREATE":
		//funcion
	case "MUTEX_LOCK":
		//funcion
	case "MUTEX_UNLOCK":
		//funcion
	case "THREAD_EXIT":
		//funcion
	case "PROCESS_EXIT":
		//funcion
	default:
		log.Printf("Instruccion no valida %s", intruction)
		return nil

	}
	return nil
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

	log.Printf("Cpu recibe TID : %d PID:%d del Kernel", processAndThreadIDs.pid, processAndThreadIDs.tid)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

	//contextoActual := GetContextoEjecucion(processAndThreadIDs.pid, processAndThreadIDs.tid)
	//InstructionCycle(&contextoActual)

}

//interface{} permite manejar tipos de datos desconocidos

func Set(registrosCPU *TCB, valor string, registro string) error {

	//ver si el registro en la instruccion existe

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
	valorParse, err := strconv.ParseUint(valor, 10, 32)
	if err != nil {
		log.Printf("SET error: Error al convertir valor %s al del tipo del registro %v", valor, reflect.TypeOf(campoRef))
		return err
	}
	campoRef.SetUint(valorParse)
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
