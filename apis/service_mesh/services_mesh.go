package servicemesh

type ServiceMeshConfig struct {
	Name string `yaml:"name"`
	Spec `yaml:"spec"`
}

type Spec struct {
	Services []Service `yaml:"services"`
}

type Service struct {
	ServiceName string `yaml:"name"`
	Address     string `yaml:"address"`
	Port        uint32 `yaml:"port"`
}
