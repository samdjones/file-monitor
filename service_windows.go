//go:build windows

package main

import "golang.org/x/sys/windows/svc"

type fileMonitorService struct {
	run func()
}

func (s *fileMonitorService) Execute(_ []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {
	status <- svc.Status{State: svc.StartPending}
	go s.run()
	status <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	for c := range r {
		if c.Cmd == svc.Stop || c.Cmd == svc.Shutdown {
			break
		}
	}
	status <- svc.Status{State: svc.StopPending}
	return false, 0
}

func isService() bool {
	ok, err := svc.IsWindowsService()
	return err == nil && ok
}

func runService(runFn func()) error {
	// Service name in the dispatch table doesn't need to match for
	// SERVICE_WIN32_OWN_PROCESS services (Windows allows any name).
	return svc.Run("", &fileMonitorService{run: runFn})
}
