package supervisor

// Exemplo de uso do supervisor:
//
// supervisor := supervisor.NewSupervisor()
//
// cmd := exec.Command("./plugin-whatsapp")
// supervisor.Register("whatsapp", cmd)
//
// supervisor.Start(context.Background(), "whatsapp")
//
// // depois...
// supervisor.Stop("whatsapp")
