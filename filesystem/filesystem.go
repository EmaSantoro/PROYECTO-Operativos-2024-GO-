package main

import (
	"log"
	"net/http"
	"strconv"
	"github.com/sisoputnfrba/tp-golang/filesystem/globals"
	"github.com/sisoputnfrba/tp-golang/filesystem/utils"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("configsFS/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}
	IpFS := globals.ClientConfig.IpMemoria
	PuertoFS := globals.ClientConfig.PuertoMemoria
	puerto := globals.ClientConfig.Puerto

	
	mux := http.NewServeMux() // se crea el servidor
	
	// funciones que va a manejar el servidor (SOLO CON MEMORIA)
	//mux.HandleFunc("Endpoint", Funcion a la que responde)
	mux.HandleFunc("/mensaje", utils.RecibirMensaje)

	err := http.ListenAndServe(":"+strconv.Itoa(puerto), mux)
	if err != nil {
		panic(err)
	}

	utils.EnviarMensaje(IpFS, PuertoFS, "Hola Memoria, Soy FS")
}