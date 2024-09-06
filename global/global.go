package global

type Paquete struct {
	ID      string `json:"ID"` //de momento es un string que indica desde donde sale el mensaje.
	Mensaje string `json:"mensaje"`
	Size    int16  `json:"size"`
	Array   []rune `json:"array"`
}
