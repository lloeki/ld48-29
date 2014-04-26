package main

import (
    "runtime"
    "log"
    gl "github.com/go-gl/gl"
    glfw "github.com/go-gl/glfw3"
    pa "code.google.com/p/portaudio-go/portaudio"
)

var _ = gl.Begin // TODO: remove later
var _ = glfw.Init // TODO: remove later
var _ = pa.Initialize // TODO: remove later

func errorCallback(err glfw.ErrorCode, desc string) {
    log.Printf("%v: %v\n", err, desc)
}

func main() {
    runtime.LockOSThread()

    glfw.SetErrorCallback(errorCallback)

    if !glfw.Init() {
        panic("Can't init glfw!")
    }
    defer glfw.Terminate()
}
