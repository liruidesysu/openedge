package native

import (
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/baidu/openedge/sdk-go/openedge"
)

type processConfigs struct {
	exec string
	pwd  string
	argv []string
	env  []string
}

func (e *nativeEngine) startProcess(name string, cfg processConfigs) (*os.Process, error) {
	p, err := os.StartProcess(
		cfg.exec,
		cfg.argv,
		&os.ProcAttr{
			Dir: cfg.pwd,
			Env: cfg.env,
			Files: []*os.File{
				os.Stdin,
				os.Stdout,
				os.Stderr,
			},
		},
	)
	if err != nil {
		e.log.WithError(err).Warnf("failed to start process (%s)", name)
		return nil, err
	}
	e.log.Infof("process (%d:%s) started", p.Pid, name)
	return p, nil
}

func (e *nativeEngine) waitProcess(p *os.Process) error {
	ps, err := p.Wait()
	if err != nil {
		e.log.WithError(err).Warnf("failed to wait process (%d)", p.Pid)
		return err
	}
	e.log.Infof("process (%d) %s", p.Pid, ps.String())
	if !ps.Success() {
		return fmt.Errorf("process exit code: %s", ps.String())
	}
	return nil
}

func (e *nativeEngine) stopProcess(p *os.Process) error {
	e.log.Infof("to stop process (%d)", p.Pid)

	err := p.Signal(syscall.SIGTERM)
	if err != nil {
		e.log.Debugf("failed to stop process (%d): %s", p.Pid, err.Error())
		return nil
	}

	done := make(chan error, 1)
	go func() {
		_, err := p.Wait()
		done <- err
	}()
	select {
	case <-time.After(e.grace):
		e.log.Warnf("timed out to wait process (%d)", p.Pid)
		err = p.Kill()
		if err != nil {
			e.log.WithError(err).Warnf("failed to kill process (%d)", p.Pid)
		}
		return fmt.Errorf("timed out to wait process (%d)", p.Pid)
	case err := <-done:
		if err != nil {
			e.log.Debugf("failed to wait process (%d): %s", p.Pid, err.Error())
		}
		return nil
	}
}

func (e *nativeEngine) statsProcess(p *os.Process) (openedge.InstanceStatus, error) {
	return openedge.InstanceStatus{}, nil
}
