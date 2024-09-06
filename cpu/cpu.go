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

	globals.ClientConfig = utils.IniciarConfiguracion("configsCPU/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}

	//IpMemoria := globals.ClientConfig.IpMemoria
	//PuertoMemoria := globals.ClientConfig.PuertoMemoria
	puerto := globals.ClientConfig.Puerto
	//IpKernel := globals.ClientConfig.IpKernel
	//PuertoKernel := globals.ClientConfig.PuertoKernel

	mux := http.NewServeMux()
	mux.HandleFunc("/mensaje", utils.RecibirMensaje)
	mux.HandleFunc("/paquete", utils.RecibirPaquete)

	http.ListenAndServe(":"+strconv.Itoa(puerto), mux)
	/* if err != nil {
		panic(err)
	} */
}
