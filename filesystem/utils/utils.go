package utils

import (
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/sisoputnfrba/tp-golang/filesystem/globals"
)

/*---------------------- ESTRUCTURAS ----------------------*/
type Mensaje struct {
	Mensaje string `json:"mensaje"`
}

/*-------------------- VAR GLOBALES --------------------*/

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

func ConfigurarLogger() {
	logFile, err := os.OpenFile("tp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		panic(err)
	}
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func init() {
	ConfigFs := IniciarConfiguracion("configsFS/config.json")

	//Al iniciar el modulo se debera validar que existan los archivos bitmap.dat y bloques.dat. En caso que no existan se deberan crear. Caso contrario se deberan tomar los ya existentes.
	if ConfigFs != nil {
			archMap := validarArchivo("bitmap.dat")
			log.Println(archMap)

			archBloque := validarArchivo("bloques.dat")
			log.Println(archBloque)
		}
}

func validarArchivo(nombre string) *os.File {
		_, err := os.Stat(nombre)
		if os.IsNotExist(err) {
			
			return crearArchivo(nombre)
		}
		return archivo(nombre)
}

func crearArchivo(nombre string)*os.File{ 
	archivo, err := os.Create(nombre)
	if err != nil {
		log.Fatal(err)
	}
	return archivo
}

func archivo(nombre string)*os.File{
	archivo, err := os.OpenFile(nombre, os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	return archivo
}

