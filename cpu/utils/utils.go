package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"os"

	"github.com/sisoputnfrba/tp-golang/cpu/globals"
)

type Mensaje struct {
	Mensaje string `json:"mensaje"`
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
	log.Printf("respuesta del servidor: %s", resp.Status)
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
	log.Printf("respuesta del servidor: %s", resp.Status)
}


