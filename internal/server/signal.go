package server

// StartSignalHandler enables runtime signal handling (noop on Windows).
func StartSignalHandler() {
    signalHandle()
}
