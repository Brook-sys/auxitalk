package supervisor

// Example:
//
// sup := supervisor.NewSupervisor(supervisor.ProcessOptions{
//     MaxRestarts: 3,
// })
//
// sup.Register(supervisor.ProcessSpec{
//     ID: "whatsapp",
//     Command: "/path/to/plugin-whatsapp",
// })
//
// sup.Start(context.Background(), "whatsapp")
//
// sup.Call(context.Background(), "whatsapp", "plugin.health", nil)
//
// sup.Stop("whatsapp")
