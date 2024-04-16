package subProcess

import (
    "io"
    "sync"
    "os/exec"
    "runtime"
)

func NewProcess(bufSize int, dir string, name string, arg ...string) (*Process, error) {
    var err error
    p := &Process{}
    p.shouldEnd = false
    cmd := exec.Command(name, arg...)
    p.cmd = cmd
    p.cmd.Dir = dir
    if p.in, err = p.cmd.StdinPipe(); err != nil {
        return nil, err
    }
    if p.out, err = p.cmd.StdoutPipe(); err != nil {
        return nil, err
    }
    if p.err, err = p.cmd.StderrPipe(); err != nil {
        return nil, err
    }
    if err = cmd.Start(); err != nil {
        return nil, err
    }
    p.inReady = make(chan struct{})
    p.outReady = false
    p.errReady = false
    p.outbuf = make([]byte, bufSize)
    p.errbuf = make([]byte, bufSize)
    p.outBuffer = make([]byte, bufSize)
    p.errBuffer = make([]byte, bufSize)
    p.outIndex = 0
    p.errIndex = 0
    p.outCond = sync.NewCond(&p.outMutex)
    p.errCond = sync.NewCond(&p.errMutex)
    p.eof = false
    runtime.SetFinalizer(p, func(p *Process) {
        p.Wait()
    })
    
    // fill buffers in goroutins
    go p.handleIn()
    go p.handleOut()
    go p.handleErr()

    return p, nil
}

func (p *Process) handleIn() {
    for !p.shouldEnd {
        <- p.inReady
        p.in.Write([]byte(p.inBuffer))
    }
}

func (p *Process) handleOut() {
    for !p.shouldEnd {
        n, err := p.out.Read(p.outbuf)
        p.outMutex.Lock()
        if err == io.EOF {
            p.eof = true
        }
        length := p.outIndex + n
        capcity := cap(p.outBuffer)
        if length > capcity {
            temp := p.outBuffer
            p.outBuffer = make([]byte, length)
            for i, v := range temp {
                p.outBuffer[i] = v
            }
        }
        for i, j := p.outIndex, 0; j < n; i, j = i+1, j+1 {
            p.outBuffer[i] = p.outbuf[j]
        }
        p.outIndex += n
        p.outReady = true
        p.outCond.Signal()
        p.outMutex.Unlock()
    }
}

func (p *Process) handleErr() {
    for !p.shouldEnd {
        n, err := p.err.Read(p.errbuf)
        p.errMutex.Lock()
        if err == io.EOF {
            p.eof = true
        }
        length := p.errIndex + n
        capcity := cap(p.errBuffer)
        if length > capcity {
            temp := p.errBuffer
            p.errBuffer = make([]byte, length)
            for i, v := range temp {
                p.errBuffer[i] = v
            }
        }
        for i, j := p.errIndex, 0; j < n; i, j = i+1, j+1 {
            p.errBuffer[i] = p.errbuf[j]
        }
        p.errIndex += n
        p.errReady = true
        p.errCond.Signal()
        p.errMutex.Unlock()
    }
}
