package main

import (
	"fmt"
	"log"

	"github.com/sisoputnfrba/tp-golang/kernel/globals"
	"github.com/sisoputnfrba/tp-golang/kernel/utils"
)

func main() {
	utils.ConfigurarLogger()

	globals.ClientConfig = utils.IniciarConfiguracion("kernel/configsKERNEL/config.json")

	Ip := globals.ClientConfig.IpCpu
	Puerto := globals.ClientConfig.PuertoCpu
	Mensaje := globals.ClientConfig.Mensaje

	fmt.Println(Ip, Puerto)

	if globals.ClientConfig == nil {
		log.Fatalf("No se pudo cargar la configuración")
	}

	utils.EnviarMensaje(Ip, Puerto, Mensaje)
}
