package globals

type Config struct {
	Puerto int `json:"puerto"`  // Puerto de escucha del servidor
	IpMemoria string `json:"ipMemoria"` // IP de la memoria
	PuertoMemoria int `json:"puertoMemoria"` // Puerto de la memoria
	Mount_dir string `json:"mount_dir"` // path donde se encuentran los archivos fs
	Block_size int `json:"block_size"` // tamaño de bloque de los archivos fs
	Block_count int `json:"block_count"` // cantidad de bloques de los archivos fs
	Block_access_delay  int `json:"block_access_delay"` // espera luego del acceso a un bloque
	Log_level string `json:"log_level"` // nivel de log
}
var ClientConfig *Config