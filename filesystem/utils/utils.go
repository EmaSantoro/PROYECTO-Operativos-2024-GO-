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
var ConfigFS *globals.Config

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
		iniciarArchivos()
	}
}

func iniciarArchivos() {

	pathFS := ConfigFS.Mount_dir

	archMap := validarArchivo(pathFS, "bitmap.dat")
	log.Println(archMap)

	archBloque := validarArchivo(pathFS, "bloques.dat")
	log.Println(archBloque)
}

func validarArchivo(path string, nombreArchivo string) *os.File {
	_, err := os.Stat(nombreArchivo)

	if os.IsNotExist(err) {
		crearArchivo(path, nombreArchivo)
	}
	return abrirArchivo(nombreArchivo)
}

func crearArchivo(path string, nombreArchivo string) {
	nombre := path + "/" + nombreArchivo
	archivo, err := os.Create(nombre)
	if err != nil {
		log.Fatalf("Error al crear el archivo '%s': %v", path, err)
	}
	defer archivo.Close()
}

func abrirArchivo(nombre string) *os.File {
	archivo, err := os.OpenFile(nombre, os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}
	return archivo
}

//Bloques dat almacena los datos de bloques, y bitmap almacena los bloques que estan ocupados
