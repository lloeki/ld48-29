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

const (
    INPUT_UP    = 0
    INPUT_DOWN  = 1
    INPUT_LEFT  = 2
    INPUT_RIGHT = 3
)

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

func onKey(input chan int, window *glfw.Window, k glfw.Key, s int, action glfw.Action, mods glfw.ModifierKey) {
    if action != glfw.Press{
        return
    }

    switch glfw.Key(k) {
    case glfw.KeyR:
        if mods & glfw.ModSuper != 0 {
            reexec()
        }
    case glfw.KeyUp:
        input <- INPUT_UP
    case glfw.KeyDown:
        input <- INPUT_DOWN
    case glfw.KeyLeft:
        input <- INPUT_LEFT
    case glfw.KeyRight:
        input <- INPUT_RIGHT
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

func drawSprite(texture gl.Texture, x float64, y float64, a float64, s float64, list uint) {
    deg := math.Mod(360 * float64(a) / (2 * math.Pi), 360.0)
    gl.LoadIdentity()
    texture.Bind(gl.TEXTURE_2D)
    gl.Translatef(float32(x), float32(y), 0)
    gl.Rotatef(float32(deg), 0.0, 0.0, 1.0);
    gl.Scalef(float32(s), float32(s), 1.0)
    gl.CallList(list)
}

func makeSprite(x int, y int, w int, h int) (quad uint) {
    quad = gl.GenLists(1)
    gl.NewList(quad, gl.COMPILE)
    spriteQuad(x, y, w, h)
    gl.EndList()

    return
}

func drawTile(texture gl.Texture, x int, y int, list uint) {
    sx := float64(16 * x + 16 / 2)
    sy := float64(16 * y + 16 / 2)
    drawSprite(texture, sx, sy, 0, 1, list)
}

func drawWaterTile(x int, y int, t float64) {
    waveHeight := 0.0
    wavePhase := -1 + x % 2 * 2

    if (t > 0) {
        waveHeight = float64(wavePhase) * math.Sin(t)
    }

    qx1, qy1 := 16 * float32(x), 16 * float32(y)
    qx2, qy2 := qx1 + 16, qy1 + 16

    gl.Disable(gl.TEXTURE_2D)
    gl.Disable(gl.LIGHTING)
    gl.MatrixMode(gl.MODELVIEW)
    gl.LoadIdentity()
    gl.Color4f(0.0, 0.0, 0.5, 0.3)

    gl.Begin(gl.QUADS)
    gl.Normal3f(0.0, 0.0, 1.0)
    gl.Vertex3f(qx1, qy1, 1.0)
    gl.Vertex3f(qx2, qy1, 1.0)
    gl.Vertex3f(qx2, qy2 + float32(waveHeight), 1.0)
    gl.Vertex3f(qx1, qy2 + float32(waveHeight), 1.0)
    gl.End()

    gl.Enable(gl.LIGHTING)
    gl.Enable(gl.TEXTURE_2D)
}

// main

func main() {
    done := make(chan int)
    input := make(chan int, 64)

    go renderer(done, input)
    go func() {
        for {
            in := <- input
            log.Printf("input %d", in)
        }
    }()

    <-done
}


// renderer

var mouseX float64
var mouseY float64
var mouseVisible bool

var ww int
var wh int

func renderer(done chan int, input chan int) {
    runtime.LockOSThread()

    glfw.SetErrorCallback(onError)

    if !glfw.Init() {
        panic("Can't init glfw!")
    }
    defer glfw.Terminate()

    glfw.WindowHint(glfw.Resizable, 0)

    ww = 640*2
    wh = 480*2

    window, err := glfw.CreateWindow(ww, wh, "LD48-29", nil, nil)
    if err != nil {
        log.Panic(err)
    }

    window.SetInputMode(glfw.Cursor, glfw.CursorHidden)

    onKeyClosure := func (window *glfw.Window, k glfw.Key, s int, action glfw.Action, mods glfw.ModifierKey) {
        onKey(input, window, k, s, action, mods)
    }

    onMouseClosure := func (window *glfw.Window, x float64, y float64) {
        rx := float64(ww) / (vx2 - vx1)
        ry := float64(wh) / (vy2 - vy1)
        mouseX, mouseY = x/rx - vx1, vy2 - (y/ry - vy1)
        mouseVisible = mouseX < vx2 && mouseX >= 0 && mouseY < vy2 && mouseY >= 0
    }

    window.SetKeyCallback(onKeyClosure)
    window.SetCursorPositionCallback(onMouseClosure)

    window.MakeContextCurrent()

    textures, lists := setup()
    defer destroy(textures)

    for !window.ShouldClose() {
        render(textures, lists)
        window.SwapBuffers()
        glfw.PollEvents()
    }

    done <- 1
}

func setup() (textures map[string]gl.Texture, lists map[string]uint) {
    gl.Enable(gl.TEXTURE_2D)
    gl.Enable(gl.DEPTH_TEST)
    gl.Enable(gl.LIGHTING)
    gl.Enable(gl.CULL_FACE)
    gl.Enable(gl.BLEND)

    gl.ClearColor(0.4, 0.8, 0.95, 0)
    gl.ClearDepth(1)
    gl.DepthFunc(gl.LEQUAL)
    gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

    textures = map[string]gl.Texture{}
    lists = map[string]uint{}

    // load spritesheet and make sprites

    img, err := os.Open("spritesheet.png")
    if err != nil { log.Panic(err) }
    defer img.Close()

    spriteSheet, err := readTexture(img)
    if err != nil { log.Panic(err) }
    textures["sprites"] = spriteSheet

    lists["test"] = makeSprite(0, 0, 2, 2)
    lists["cursor"] = makeSprite(3, 2, 1, 1)
    lists["cloud1"] = makeSprite(0, 2, 3, 2)
    lists["cloud2"] = makeSprite(0, 4, 2, 2)
    lists["cloud3"] = makeSprite(2, 4, 2, 2)
    lists["stonewall"] = makeSprite(2, 0, 1, 1)
    lists["stonewallright"] = makeSprite(3, 0, 1, 1)
    lists["stonewalltopright"] = makeSprite(3, 1, 1, 1)
    lists["stonewalltop"] = makeSprite(2, 1, 1, 1)
    lists["stonewallleft"] = makeSprite(4, 0, 1, 1)
    lists["stonewalltopleft"] = makeSprite(4, 1, 1, 1)

    return
}

func destroy(textures map[string]gl.Texture) {
    for _, texture := range textures {
        texture.Delete()
    }
}

var vx1 float64
var vy1 float64
var vx2 float64
var vy2 float64

func render(textures map[string]gl.Texture, lists map[string]uint) {
    // start afresh
    gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

    // set viewport
    width := float32(ww)
    height := float32(wh)
    density := 2 // times 2 because HiDPI
    gl.Viewport(0, 0, int(width)*density, int(height)*density)

    vw := 320
    vh := 240

    // set projection
    gl.MatrixMode(gl.PROJECTION)
    gl.LoadIdentity()
    vx1, vy1 = 0, 0
    vx2, vy2 = float64(vw), float64(vh)
    gl.Ortho(vx1, vx2, vy1, vy2, -1.0, 1.0)

    gl.MatrixMode(gl.MODELVIEW);
    gl.LoadIdentity()

    // lighten things
    ambient    := []float32{1, 1, 1, 1}
    gl.Lightfv(gl.LIGHT0, gl.AMBIENT, ambient)
    gl.Enable(gl.LIGHT0)

    // time source

    t := float64(time.Now().UnixNano()) / math.Pow(10, 9)

    // clouds

    fy := 2 * math.Pi * t / 60
    drawSprite(textures["sprites"], 200, 200+8*math.Sin(fy/1.3), 0, 2.0, lists["cloud1"])

    // wall tiles

    for j := 0; j < 9; j++ {
        drawTile(textures["sprites"], 3, j, lists["stonewallright"])
    }
    drawTile(textures["sprites"], 3, 9, lists["stonewalltopright"])

    for i := 0; i < 3; i++ {
        for j:= 0; j < 9; j++ {
            drawTile(textures["sprites"], i, j, lists["stonewall"])
        }
        drawTile(textures["sprites"], i, 9, lists["stonewalltop"])
    }

    for j := 0; j < 6; j++ {
        drawTile(textures["sprites"], 19 - 2, j, lists["stonewallleft"])
    }
    drawTile(textures["sprites"], 19 - 2, 6, lists["stonewalltopleft"])

    for i := 0; i < 2; i++ {
        for j:= 0; j < 6; j++ {
            drawTile(textures["sprites"], 19 - i, j, lists["stonewall"])
        }
        drawTile(textures["sprites"], 19 - i, 6, lists["stonewalltop"])
    }

    // water
    for i := 0; i < 320 / 16; i++ {
        for j := 0; j < 3; j++ {
            wt := 0.0
            if j == 2 {
                wt = t
            }
            drawWaterTile(i, j, wt)
        }
    }

    // mouse pointer

    if mouseVisible {
        drawSprite(textures["sprites"], mouseX, mouseY, 0, 1.0, lists["cursor"])
    }
}
