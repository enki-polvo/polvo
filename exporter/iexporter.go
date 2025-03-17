package exporter

type Exporter[log any] interface {
	// Getter & Setter
	Name() string
	LogChannel() chan<- *log
	// methods
	Start()
	Stop() error
}
