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

	//EnviarMensaje(globals.ClientConfig.IpKernel, globals.ClientConfig.PuertoKernel, "Hola Kernel, Soy CPU")
	//EnviarMensaje(globals.ClientConfig.IpMemoria, globals.ClientConfig.PuertoMemoria, "Hola Memoria, Soy CPU")
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

//EnviarMensaje(globals.ClientConfig.IpKernel,globals.ClientConfig.PuertoKernel,"Hola Kernel soy CPU")
//EnviarMensaje(globals.ClientConfig.IpMemoria,globals.ClientConfig.PuertoMemoria,"Hola Memoria soy CPU")
