package main

import (
	"log"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	"github.com/sisoputnfrba/tp-golang/kernel/utils"
)

func main() {
	utils.ConfigurarLogger()
	log.Println("Hola soy un log")

	globals.ClientConfig = utils.IniciarConfiguracion("kernel/configsKERNEL/config.json")

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}

	utils.EnviarMensaje(globals.ClientConfig.IpCpu, globals.ClientConfig.PuertoCpu, "Conexion exitosa desde kernel")
}
