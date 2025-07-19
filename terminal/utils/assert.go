package utils

func Assert(condition bool, message ...string) {
	if !condition {
		if len(message) == 1 {
			panic(message[0])
		}
		panic("failed assertion")
	}
}
