package main

import (
	"log"
	"net/http"
	"strconv"
	"github.com/sisoputnfrba/tp-golang/cpu/globals"
	"github.com/sisoputnfrba/tp-golang/cpu/utils"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("configsCpu/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}
	IpMemoria := globals.ClientConfig.IpMemoria
	PuertoMemoria := globals.ClientConfig.PuertoMemoria
	puerto := globals.ClientConfig.Puerto

	

	mux := http.NewServeMux()
	// funciones que va a manejar el servidor (Kernel y Memoria)
	//mux.HandleFunc("Endpoint", Funcion a la que responde)
	mux.HandleFunc("/mensaje", utils.RecibirMensaje)

	err := http.ListenAndServe(":"+strconv.Itoa(puerto), mux)
	if err != nil {
		panic(err)
	}

	utils.EnviarMensaje(IpMemoria, PuertoMemoria, "Hola Memoria, Soy CPU")

}
