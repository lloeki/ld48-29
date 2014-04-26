package main

import (
    "runtime"
    "log"
    gl "github.com/go-gl/gl"
    glfw "github.com/go-gl/glfw3"
    pa "code.google.com/p/portaudio-go/portaudio"
)

var _ = pa.Initialize // TODO: remove later

func onError(err glfw.ErrorCode, desc string) {
    log.Printf("%v: %v\n", err, desc)
}

func onKey(window *glfw.Window, k glfw.Key, s int, action glfw.Action, mods glfw.ModifierKey) {
    if action != glfw.Press {
        return
    }

    switch glfw.Key(k) {
    case glfw.KeyEscape:
        window.SetShouldClose(true)
    default:
        return
    }
}

func main() {
    runtime.LockOSThread()

    glfw.SetErrorCallback(onError)

    if !glfw.Init() {
        panic("Can't init glfw!")
    }
    defer glfw.Terminate()

    window, err := glfw.CreateWindow(640, 480, "LD48-29", nil, nil)
    if err != nil {
        log.Panic(err)
    }

    window.SetKeyCallback(onKey)

    window.MakeContextCurrent()

    setup()
    defer destroy()

    for !window.ShouldClose() {
        render()
        window.SwapBuffers()
        glfw.PollEvents()
    }
}

func setup() {
    gl.Enable(gl.TEXTURE_2D)
    gl.Enable(gl.DEPTH_TEST)
    gl.Enable(gl.LIGHTING)
    gl.Enable(gl.CULL_FACE)

    gl.ClearColor(0.0, 0.0, 0.5, 0)
    gl.ClearDepth(1)
    gl.DepthFunc(gl.LEQUAL)
}

func destroy() {
    // TODO: release objects
}

func render() {
    gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
    gl.MatrixMode(gl.PROJECTION)
    gl.LoadIdentity()
    gl.Frustum(-1, 1, -1, 1, 1.0, 10.0)
}
