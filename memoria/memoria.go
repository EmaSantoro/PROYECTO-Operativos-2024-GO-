package main

import (
	"log"
	"net/http"
	"strconv"
	"github.com/sisoputnfrba/tp-golang/memoria/globals"
	"github.com/sisoputnfrba/tp-golang/memoria/utils"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("configsMemoria/config.json")
	
	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}
	PuertoFS := globals.ClientConfig.PuertoFs
	IpFS := globals.ClientConfig.IpFs
	PuertoCpu := globals.ClientConfig.PuertoCpu
	IpCpu := globals.ClientConfig.IpCpu
	puerto := globals.ClientConfig.Puerto

	mux := http.NewServeMux() // se crea el servidor
	
	// funciones que va a manejar el servidor (Kernel , cpu y filesystem)
	//mux.HandleFunc("Endpoint", Funcion a la que responde)
	mux.HandleFunc("/mensaje", utils.RecibirMensaje)

	err := http.ListenAndServe(":"+strconv.Itoa(puerto), mux)
	if err != nil {
		panic(err)
	}

	utils.EnviarMensaje(IpCpu, PuertoCpu, "Hola CPU, Soy Memoria")
	utils.EnviarMensaje(IpFS, PuertoFS, "Hola FS, Soy Memoria")

}