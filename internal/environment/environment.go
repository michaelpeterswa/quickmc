package environment

import "os"

type Environment struct {
	Project string
	Version string
	Build   string
}

func LoadEnvironment() *Environment {
	return &Environment{
		Project: os.Getenv("PAPERMC_PROJECT"),
		Version: os.Getenv("PAPERMC_VERSION"),
		Build:   os.Getenv("PAPERMC_BUILD"),
	}
}
