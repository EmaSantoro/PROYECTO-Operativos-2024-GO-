package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
)

/*---------------------- ESTRUCTURAS ----------------------*/

type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

type Paquete struct {
	ID      string `json:"ID"` //de momento es un string que indica desde donde sale el mensaje.
	Mensaje string `json:"mensaje"`
	Size    int16  `json:"size"`
	Array   []rune `json:"array"`
}

var paquete Paquete = Paquete{
	ID:      "CPU", //de momento es un string que indica desde donde sale el mensaje.
	Mensaje: "Soy CPU",
	Size:    int16(len([]rune{'H', 'o', 'l', 'a'})),
	Array:   []rune{'H', 'o', 'l', 'a'},
}

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

func init() {
	ConfigKernel := IniciarConfiguracion("configsKERNEL/config.json")
	EnviarMensaje(ConfigKernel.IpMemoria, ConfigKernel.PuertoMemoria, "Hola Memoria, Soy Kernel")
	EnviarMensaje(ConfigKernel.IpCpu, ConfigKernel.PuertoCpu, "Hola CPU, Soy Kernel")
}

func ConfigurarLogger() {
	logFile, err := os.OpenFile("tp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
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

	/*defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}*/
	log.Printf("respuesta del servidor: %s", resp.Status)
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

	log.Println("Conexion con Kernel")
	log.Printf("%+v\n", mensaje)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func EnviarPaqueteACPU() {
	ip := globals.ClientConfig.IpCpu
	puerto := globals.ClientConfig.PuertoCpu

	body, err := json.Marshal(paquete)
	if err != nil {
		log.Printf("error codificando paquete: %s", err.Error())
	}

	url := fmt.Sprintf("http://%s:%d/paquete", ip, puerto)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Printf("error enviando mensaje a ip:%s puerto:%d", ip, puerto)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}
	log.Printf("respuesta del servidor: %s", resp.Status)

}
