package main

func main() {
	cfg, err := ParseConfig()
	if err != nil {
		panic(err)
	}

	if cfg.Mode == "server" {
		StartServer(&cfg)
	} else {
		StartClient(&cfg)
	}
}
