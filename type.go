package subProcess

import (
    "io"
    "sync"
    "runtime"
    "os/exec"
)

type Process struct {
    cmd         *exec.Cmd 
    shouldEnd   bool
    in          io.WriteCloser
    out         io.ReadCloser
    err         io.ReadCloser
    outbuf      []byte
    errbuf      []byte
    outMutex    sync.Mutex
    errMutex    sync.Mutex
    outCond     *sync.Cond
    errCond     *sync.Cond
    inBuffer    string
    outBuffer   []byte
    errBuffer   []byte
    outIndex    int
    errIndex    int
    inReady     chan struct{}
    outReady    bool
    errReady    bool
    eof         bool
}

func (p *Process) Wait() error {
    p.shouldEnd = true
    runtime.SetFinalizer(p, nil)
    return p.cmd.Wait()
}

func (p *Process) Send(s string) {
    p.inBuffer = s
    p.inReady <- struct{}{}
}

func (p *Process) ReadyOut() bool {
    p.outMutex.Lock()
    defer p.outMutex.Unlock()
    return p.outReady
}

func (p *Process) ReadyErr() bool {
    p.errMutex.Lock()
    defer p.errMutex.Unlock()
    return p.errReady
}

func (p *Process) RecvOut() (string, error) {
    p.outMutex.Lock()
    defer p.outMutex.Unlock()
    for !p.outReady {
        p.outCond.Wait()
    }
    if p.eof == true {
        return "", io.EOF
    }
    s := string(p.outBuffer[:p.outIndex])
    p.outIndex = 0
    p.outReady = false
    return s, nil
}

func (p *Process) RecvErr() (string, error) {
    p.errMutex.Lock()
    defer p.errMutex.Unlock()
    for !p.errReady {
        p.errCond.Wait()
    }
    if p.eof == true {
        return "", io.EOF
    }
    s := string(p.errBuffer[:p.errIndex])
    p.errIndex = 0
    p.errReady = false
    return s, nil
}
