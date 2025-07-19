package dcs

type (
	HookHandler   interface{ DCSHook(*DCS) *Command }
	UnhookHandler interface{ DCSUnhook() *Command }
	PutHandler    interface{ DCSPut(uint8) *Command }

	// This aggerate methods needed for DCS handler
	Handler interface {
		HookHandler
		UnhookHandler
		PutHandler
	}
)
