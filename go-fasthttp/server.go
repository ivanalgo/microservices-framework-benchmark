package main

import (
    "github.com/valyala/fasthttp"
    "github.com/valyala/fasthttp/reuseport"
    "net"
    "flag"
    "runtime"
    "log"
    "os/exec"
    "os"
    "golang.org/x/sys/unix"
)

var (
    listenAddr = flag.String("listenAddr", ":8080", "Address to listen to")
    prefork    = flag.Bool("prefork", false, "use prefork")
    child      = flag.Bool("child", false, "is child proc")
    cpu        = flag.Int("cpus", runtime.NumCPU(), "cpu numbers")
    memAlloc   = flag.Bool("memAlloc", false, "allocated memory in each request")
)

func main() {
    flag.Parse()

    var err error

    s := &fasthttp.Server{
        Handler: mainHandler,
        Name:    "go",
    }
    ln := getListener()
    if err = s.Serve(ln); err != nil {
        log.Fatalf("Error when serving incoming connections: %s", err)
    }

}

func mainHandler(ctx *fasthttp.RequestCtx) {
    path := ctx.Path()
    switch string(path) {
    case "/":
        plaintextHandler(ctx)
    default:
        ctx.Error("unexpected path", fasthttp.StatusBadRequest)
    }
}

func plaintextHandler(ctx *fasthttp.RequestCtx) {
    ctx.SetContentType("text/plain")
    ctx.WriteString("Hello World!")

    if *memAlloc {
        // Add memory allocation here to simulate the realworld tasks
        size := 4096
        data, err := unix.Mmap(-1, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
        if err != nil {
            return
        }

        defer unix.Munmap(data)

        // touch memory to cause memory allocation in OS
        data[0] = 0xff
    }
}

func getListener() net.Listener {
    if !*prefork {
        runtime.GOMAXPROCS(*cpu)
        ln, err := net.Listen("tcp4", *listenAddr)
        if err != nil {
            log.Fatal(err)
        }
        return ln
    }

    if !*child {
        children := make([]*exec.Cmd, *cpu)
        for i := range children {
            children[i] = exec.Command(os.Args[0], "-prefork", "-child")
            children[i].Stdout = os.Stdout
            children[i].Stderr = os.Stderr
            if err := children[i].Start(); err != nil {
                log.Fatal(err)
            }
        }
        for _, ch := range children {
            if err := ch.Wait(); err != nil {
                log.Print(err)
            }
        }
        os.Exit(0)
        panic("unreachable")
    }

    runtime.GOMAXPROCS(1)
    ln, err := reuseport.Listen("tcp4", *listenAddr)
    if err != nil {
        log.Fatal(err)
    }
    return ln
}
