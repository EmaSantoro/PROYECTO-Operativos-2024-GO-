package main

import (
	"log"
	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	"github.com/sisoputnfrba/tp-golang/kernel/utils"
	"net/http"
	"strconv"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("kernel/configsKERNEL/config.json")
	
	if globals.ClientConfig == nil {
			log.Fatalf("No se pudo cargar la configuración")
		}
	
	puerto := globals.ClientConfig.Puerto
	IpCpu := globals.ClientConfig.IpCpu
	PuertoCpu := globals.ClientConfig.PuertoCpu
	IpMemoria := globals.ClientConfig.IpMemoria
	PuertoMemoria := globals.ClientConfig.PuertoMemoria

	utils.EnviarMensaje(IpCpu, PuertoCpu, "Hola Cpu,  Soy Kernel")
	utils.EnviarMensaje(IpMemoria, PuertoMemoria, "Hola Memoria,  Soy Kernel")

	mux := http.NewServeMux()
	// funciones que va a manejar el servidor (Kernel y Memoria)
	//mux.HandleFunc("Endpoint", Funcion a la que responde)
	mux.HandleFunc("/mensaje", utils.RecibirMensaje)


	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)
	/* if err != nil {
		panic(err)
	} */


}
