package cluster

type Config struct {
	Image             string
	ControllerImage   string
	PriorityClassName string
	BackupEnv         map[string]string
}
