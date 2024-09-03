package globals

type Config struct {
	Puerto int `json:"puerto"`  // Puerto de escucha del servidor
	Tamanio_Memoria int `json:"tamanio_memoria"` // Tamaño de la memoria expresado en bytes
	Path_Instruccion string `json:"path_instruccion"` // Carpeta donde se encuentran los archivos de pseudocodigo
	Delay_Respuesta int `json:"delay_respuesta"` // Tiempo de espera para responder una instruccion
	IpKernel string `json:"ip_kernel"` // IP del kernel
	PuertoKernel int `json:"puerto_kernel"` // Puerto del kernel
	IpCpu string `json:"ip_cpu"` // IP de la CPU
	PuertoCpu int `json:"puerto_cpu"` // Puerto de la CPU
	IpFs string `json:"ip_fs"` // IP del filesystem
	PuertoFs int `json:"puerto_fs"` // Puerto del filesystem
	EsquemaMemoria string `json:"esquema_memoria"` // Esquema de particiones de memoria a utilizar
	AlgoritmoBusqueda string `json:"algoritmo_busqueda"` // Algoritmo de busqueda de huecos en memoria
	Particiones []Particion `json:"particiones"` // Lista ordenada con las particiones a generar en el algoritmo Particiones fijas
	Log_Level string `json:"log_level"` // Nivel de loggeo
}

type Particion struct {
	Tamanio int `json:"tamanio"` // Tamaño de la particion
}

var ClientConfig *Config