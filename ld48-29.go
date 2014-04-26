package main

import (
    "runtime"
    "time"
    "math"
    "log"
    "io"
    "errors"
    "os"
    "syscall"
    "image"
    "image/png"
    gl "github.com/go-gl/gl"
    glfw "github.com/go-gl/glfw3"
    pa "code.google.com/p/portaudio-go/portaudio"
)

var _ = pa.Initialize // TODO: remove later


// iterate faster

func rerun() (err error) {
    log.Println("rerun")
    gopath := os.Getenv("GOPATH")
    env := []string{"GOPATH=" + gopath}
    args := []string{"go", "run", "ld48-29.go"}
    err = syscall.Exec("/usr/local/bin/go", args, env)
    log.Fatal(err)

    return
}

func reexec() {
    err := rerun()
    if err != nil { panic(err) }
}

// glfw callbacks

func onError(err glfw.ErrorCode, desc string) {
    log.Printf("%v: %v\n", err, desc)
}

func onKey(window *glfw.Window, k glfw.Key, s int, action glfw.Action, mods glfw.ModifierKey) {
    if action != glfw.Press {
        return
    }

    switch glfw.Key(k) {
    case glfw.KeyR:
        if mods & glfw.ModSuper != 0 {
            reexec()
        }
    case glfw.KeyEscape:
        window.SetShouldClose(true)
    default:
        return
    }
}


// utils

func readTexture(r io.Reader) (texId gl.Texture, err error) {
    img, err := png.Decode(r)
    if err != nil {
        return gl.Texture(0), err
    }

    rgba, ok := img.(*image.NRGBA)
    if !ok {
        return gl.Texture(0), errors.New("not an NRGBA image")
    }

    texId = gl.GenTexture()
    texId.Bind(gl.TEXTURE_2D)
    gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
    gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)

    w, h := rgba.Bounds().Dx(), rgba.Bounds().Dy()
    raw := make([]byte, w*h*4)
    raw_stride := w * 4
    if raw_stride != rgba.Stride {
        return gl.Texture(0), errors.New("incompatible stride")
    }

    dst := len(raw) - raw_stride
    for src := 0; src < len(rgba.Pix); src += rgba.Stride {
        copy(raw[dst:dst+raw_stride], rgba.Pix[src:src+rgba.Stride])
        dst -= raw_stride
    }

    lod := 0
    border := 0
    gl.TexImage2D(gl.TEXTURE_2D, lod, gl.RGBA, w, h, border, gl.RGBA, gl.UNSIGNED_BYTE, raw)

    return
}

func spriteQuad(x int, y int, w int, h int) {
    scaledSpriteQuad(x, y, w, h, 1.0)
}

func scaledSpriteQuad(x int, y int, w int, h int, scale float32) {
    // TODO: set elsewhere (spritesheet property)
    size := 256 // spritesheet size
    unit :=  16 // sprite unit size

    // sprite to absolute pixel coords
    // origin is as tex: bottom-left
    x1 :=      x * unit
    y1 :=      y * unit
    x2 := x1 + w * unit
    y2 := y1 + h * unit

    // abs pixel to relative tex coords
    rx1 := float32(x1) / float32(size)
    rx2 := float32(x2) / float32(size)
    ry1 := float32(y1) / float32(size)
    ry2 := float32(y2) / float32(size)

    // scale sprite
    qsize := 1.0 * scale

    // relative model coords
    // sprite-centered origin, half each way
    qx1 := -qsize * float32(unit * w) / 2.0
    qy1 := -qsize * float32(unit * h) / 2.0
    qx2 :=  qsize * float32(unit * w) / 2.0
    qy2 :=  qsize * float32(unit * h) / 2.0

    // draw sprite quad
    gl.MatrixMode(gl.MODELVIEW)
    gl.Begin(gl.QUADS)
    gl.Normal3f(0, 0, 1)
    gl.TexCoord2f(rx1, ry1)
    gl.Vertex3f(qx1, qy1, 1.0)
    gl.TexCoord2f(rx2, ry1)
    gl.Vertex3f(qx2, qy1, 1.0)
    gl.TexCoord2f(rx2, ry2)
    gl.Vertex3f(qx2, qy2, 1.0)
    gl.TexCoord2f(rx1, ry2)
    gl.Vertex3f(qx1, qy2, 1.0)
    gl.End()
}

func drawSprite(texture gl.Texture, x float64, y float64, list uint) {
    gl.LoadIdentity()
    texture.Bind(gl.TEXTURE_2D)
    gl.Translatef(float32(x), float32(y), 0)
    gl.CallList(list)
}

// main

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

    textures, lists := setup()
    defer destroy(textures)

    for !window.ShouldClose() {
        render(textures, lists)
        window.SwapBuffers()
        glfw.PollEvents()
    }
}


// renderer

func setup() (textures map[string]gl.Texture, lists map[string]uint) {
    gl.Enable(gl.TEXTURE_2D)
    gl.Enable(gl.DEPTH_TEST)
    gl.Enable(gl.LIGHTING)
    gl.Enable(gl.CULL_FACE)

    gl.ClearColor(0.0, 0.0, 0.5, 0)
    gl.ClearDepth(1)
    gl.DepthFunc(gl.LEQUAL)

    textures = map[string]gl.Texture{}
    lists = map[string]uint{}

    img, err := os.Open("spritesheet.png")
    if err != nil { log.Panic(err) }
    defer img.Close()

    spriteSheet, err := readTexture(img)
    if err != nil { log.Panic(err) }
    textures["sprites"] = spriteSheet

    quad := gl.GenLists(1)
    gl.NewList(quad, gl.COMPILE)
    spriteQuad(0, 0, 2, 2)
    gl.EndList()

    lists["test"] = quad

    return
}

func destroy(textures map[string]gl.Texture) {
    for _, texture := range textures {
        texture.Delete()
    }
}

func render(textures map[string]gl.Texture, lists map[string]uint) {
    // start afresh
    gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

    // set viewport
    width := float32(640.0)
    height := float32(480.0)
    gl.Viewport(0, 0, int(width)*2, int(height)*2) // times 2 because HiDPI
    gl.MatrixMode(gl.PROJECTION)
    gl.LoadIdentity()
    gl.Ortho(0, float64(width), 0, float64(height), -1.0, 1.0)

    gl.MatrixMode(gl.MODELVIEW);
    gl.LoadIdentity()

    // lighten things
    ambient    := []float32{0.5, 0.5, 0.5, 1}
    diffuse    := []float32{1, 1, 1, 1}
    lightpos   := []float32{-5, 5, 10, 0}
    gl.Lightfv(gl.LIGHT0, gl.AMBIENT, ambient)
    gl.Lightfv(gl.LIGHT0, gl.DIFFUSE, diffuse)
    gl.Lightfv(gl.LIGHT0, gl.POSITION, lightpos)
    gl.Enable(gl.LIGHT0)

    gl.Disable(gl.TEXTURE_2D)
    gl.Disable(gl.LIGHTING)
    gl.Begin(gl.TRIANGLES)
    gl.Color3f(1.0, 0.0, 0.0)
    gl.Vertex3f(0, 0, 0.0)
    gl.Color3f(0.0, 1.0, 0.0)
    gl.Vertex3f(width, 0, 0.0)
    gl.Color3f(0.0, 0.0, 1.0)
    gl.Vertex3f(0.0, height, 0.0)
    gl.End()
    gl.Begin(gl.TRIANGLES)
    gl.Color3f(1.0, 0.0, 0.0)
    gl.Vertex3f(width, height, 0.0)
    gl.Color3f(0.0, 0.0, 1.0)
    gl.Vertex3f(0.0, height, 0.0)
    gl.Color3f(0.0, 1.0, 0.0)
    gl.Vertex3f(width, 0, 0.0)
    gl.End()
    gl.Enable(gl.TEXTURE_2D)
    gl.Enable(gl.LIGHTING)

    drawSprite(textures["sprites"],   0,   0, lists["test"])
    drawSprite(textures["sprites"], 320,   0, lists["test"])
    drawSprite(textures["sprites"], 640,   0, lists["test"])
    drawSprite(textures["sprites"], 320, 240, lists["test"])
    drawSprite(textures["sprites"], 320, 480, lists["test"])
    drawSprite(textures["sprites"],   0, 240, lists["test"])
    drawSprite(textures["sprites"],   0, 480, lists["test"])
    drawSprite(textures["sprites"], 640, 240, lists["test"])
    drawSprite(textures["sprites"], 640, 480, lists["test"])

    t := float64(time.Now().UnixNano()) / math.Pow(10, 9)

    x := (math.Sin(2*math.Pi*t/60) + 1)/2 * float64(width)
    y := (math.Cos(2*math.Pi*t/60) + 1)/2 * float64(height)
    drawSprite(textures["sprites"], x, y, lists["test"])

    x = (math.Sin(10*2*math.Pi*t/60) + 1)/2 * float64(width)
    y = (math.Cos(10*2*math.Pi*t/60) + 1)/2 * float64(height)
    drawSprite(textures["sprites"], x, y, lists["test"])

    x = (math.Sin(60*2*math.Pi*t/60) + 1)/2 * float64(width)
    y = (math.Cos(60*2*math.Pi*t/60) + 1)/2 * float64(height)
    drawSprite(textures["sprites"], x, y, lists["test"])
}
