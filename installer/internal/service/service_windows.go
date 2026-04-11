//go:build windows

package service

import (
        "fmt"
        "os"
        "path/filepath"
        "syscall"
        "time"
        "unsafe"
)

type scmManager struct {
        name string
}

func newPlatformManager(name string) (Manager, error) {
        if err := validateName(name); err != nil {
                return nil, err
        }
        return &scmManager{name: name}, nil
}

var (
        modadvapi32 = syscall.NewLazyDLL("advapi32.dll")

        procOpenSCManager       = modadvapi32.NewProc("OpenSCManagerW")
        procCreateService       = modadvapi32.NewProc("CreateServiceW")
        procOpenService         = modadvapi32.NewProc("OpenServiceW")
        procDeleteService       = modadvapi32.NewProc("DeleteServiceW")
        procStartService        = modadvapi32.NewProc("StartServiceW")
        procControlService      = modadvapi32.NewProc("ControlServiceW")
        procCloseServiceHandle  = modadvapi32.NewProc("CloseServiceHandle")
        procQueryServiceStatus  = modadvapi32.NewProc("QueryServiceStatusW")
)

const (
        svcAllAccess        = 0xF01FF
        svcManagerConnect   = 1
        svcManagerCreate    = 2
        svcStart            = 0x10
        svcStop             = 0x20
        svcDelete           = 0x10000
        serviceRunning      = 4
        serviceStopped      = 1
        serviceStartPending = 2
        serviceStopPending  = 3
)

type serviceStatus struct {
        ServiceType             uint32
        CurrentState            uint32
        ControlsAccepted        uint32
        Win32ExitCode           uint32
        ServiceSpecificExitCode uint32
        CheckPoint              uint32
        WaitHint                uint32
}

func (s *scmManager) Install(cfg Config) error {
        mgrHandle, _, err := procOpenSCManager.Call(0, 0, uintptr(svcManagerConnect|svcManagerCreate))
        if mgrHandle == 0 {
                return fmt.Errorf("failed to open Service Control Manager: %v", err)
        }
        defer procCloseServiceHandle.Call(mgrHandle)

        exePath := cfg.BinaryPath
        if exePath == "" {
                return fmt.Errorf("binary path is empty")
        }
        exePath = filepath.Clean(exePath)

        serviceName, err := syscall.UTF16PtrFromString(s.name)
        if err != nil {
                return fmt.Errorf("failed to encode service name: %w", err)
        }

        displayName, err := syscall.UTF16PtrFromString("System Worker - System monitoring HTTPS service")
        if err != nil {
                return fmt.Errorf("failed to encode display name: %w", err)
        }

        // binaryPath includes the exe path and all command-line arguments.
        // Each path containing spaces MUST be quoted.
        binaryPath := fmt.Sprintf(`"%s" -port %d -log-level %s -cert-dir "%s\cert" -log-dir "%s\logs"`,
                exePath, cfg.Port, cfg.LogLevel, cfg.DataDir, cfg.DataDir)
        binaryPathPtr, err := syscall.UTF16PtrFromString(binaryPath)
        if err != nil {
                return fmt.Errorf("failed to encode binary path: %w", err)
        }

        // Try to create the service
        svcHandle, _, createErr := procCreateService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                uintptr(unsafe.Pointer(displayName)),
                uintptr(svcAllAccess),
                uintptr(0x10), // SERVICE_WIN32_OWN_PROCESS
                uintptr(2),   // SERVICE_AUTO_START
                uintptr(1),   // SERVICE_ERROR_NORMAL
                uintptr(unsafe.Pointer(binaryPathPtr)),
                0, 0, 0, 0, 0,
        )
        if svcHandle != 0 {
                procCloseServiceHandle.Call(svcHandle)
                return nil
        }

        // Service already exists — stop, delete, wait, then recreate
        fmt.Printf("       Service already exists, recreating...\n")
        s.stopAndDeleteService()

        for i := 0; i < 10; i++ {
                time.Sleep(500 * time.Millisecond)
                if !s.IsInstalled() {
                        break
                }
        }

        svcHandle, _, createErr = procCreateService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                uintptr(unsafe.Pointer(displayName)),
                uintptr(svcAllAccess),
                uintptr(0x10), // SERVICE_WIN32_OWN_PROCESS
                uintptr(2),   // SERVICE_AUTO_START
                uintptr(1),   // SERVICE_ERROR_NORMAL
                uintptr(unsafe.Pointer(binaryPathPtr)),
                0, 0, 0, 0, 0,
        )
        if svcHandle == 0 {
                return fmt.Errorf("failed to create service after removing old one: %v", createErr)
        }
        procCloseServiceHandle.Call(svcHandle)
        return nil
}

// stopAndDeleteService stops the existing service and waits for it to fully stop,
// then deletes it from the SCM.
func (s *scmManager) stopAndDeleteService() {
        mgrHandle, _, _ := procOpenSCManager.Call(0, 0, uintptr(svcManagerConnect))
        if mgrHandle == 0 {
                return
        }
        defer procCloseServiceHandle.Call(mgrHandle)

        serviceName, _ := syscall.UTF16PtrFromString(s.name)

        svcHandle, _, _ := procOpenService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                uintptr(svcStop|svcDelete),
        )
        if svcHandle == 0 {
                return
        }
        defer procCloseServiceHandle.Call(svcHandle)

        // Try to stop the service
        var status serviceStatus
        procControlService.Call(svcHandle, svcStop, uintptr(unsafe.Pointer(&status)))

        // Wait up to 10 seconds for the service to fully stop
        for i := 0; i < 20; i++ {
                time.Sleep(500 * time.Millisecond)
                procQueryServiceStatus.Call(svcHandle, uintptr(unsafe.Pointer(&status)))
                if status.CurrentState == serviceStopped {
                        break
                }
        }

        procDeleteService.Call(svcHandle)
}

func (s *scmManager) Uninstall() error {
        mgrHandle, _, err := procOpenSCManager.Call(0, 0, uintptr(svcManagerConnect))
        if mgrHandle == 0 {
                return fmt.Errorf("failed to open Service Control Manager: %v", err)
        }
        defer procCloseServiceHandle.Call(mgrHandle)

        serviceName, err := syscall.UTF16PtrFromString(s.name)
        if err != nil {
                return err
        }

        svcHandle, _, err := procOpenService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                uintptr(svcStop|svcDelete),
        )
        if svcHandle == 0 {
                return fmt.Errorf("service '%s' not found", s.name)
        }
        defer procCloseServiceHandle.Call(svcHandle)

        // Stop the service if running
        var status serviceStatus
        procQueryServiceStatus.Call(svcHandle, uintptr(unsafe.Pointer(&status)))
        if status.CurrentState != serviceStopped {
                procControlService.Call(svcHandle, svcStop, uintptr(unsafe.Pointer(&status)))
                // Wait up to 10 seconds
                for i := 0; i < 20; i++ {
                        time.Sleep(500 * time.Millisecond)
                        procQueryServiceStatus.Call(svcHandle, uintptr(unsafe.Pointer(&status)))
                        if status.CurrentState == serviceStopped {
                                break
                        }
                }
        }

        ret, _, err := procDeleteService.Call(svcHandle)
        if ret == 0 {
                return fmt.Errorf("failed to delete service: %v", err)
        }
        return nil
}

func (s *scmManager) Start() error {
        mgrHandle, _, err := procOpenSCManager.Call(0, 0, uintptr(svcManagerConnect))
        if mgrHandle == 0 {
                return fmt.Errorf("failed to open Service Control Manager: %v", err)
        }
        defer procCloseServiceHandle.Call(mgrHandle)

        serviceName, err := syscall.UTF16PtrFromString(s.name)
        if err != nil {
                return err
        }

        svcHandle, _, err := procOpenService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                uintptr(svcStart),
        )
        if svcHandle == 0 {
                return fmt.Errorf("failed to open service: %v", err)
        }
        defer procCloseServiceHandle.Call(svcHandle)

        // Verify the binary exists before trying to start
        var status serviceStatus
        procQueryServiceStatus.Call(svcHandle, uintptr(unsafe.Pointer(&status)))
        if status.CurrentState == serviceRunning {
                return nil // already running
        }

        ret, _, err := procStartService.Call(svcHandle, 0, 0)
        if ret == 0 {
                return fmt.Errorf("failed to start service: %v", err)
        }
        return nil
}

// Stop stops the service and waits for it to fully stop.
func (s *scmManager) Stop() error {
        mgrHandle, _, err := procOpenSCManager.Call(0, 0, uintptr(svcManagerConnect))
        if mgrHandle == 0 {
                return fmt.Errorf("failed to open Service Control Manager: %v", err)
        }
        defer procCloseServiceHandle.Call(mgrHandle)

        serviceName, err := syscall.UTF16PtrFromString(s.name)
        if err != nil {
                return err
        }

        svcHandle, _, err := procOpenService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                uintptr(svcStop),
        )
        if svcHandle == 0 {
                return fmt.Errorf("failed to open service: %v", err)
        }
        defer procCloseServiceHandle.Call(svcHandle)

        var status serviceStatus
        ret, _, err := procControlService.Call(svcHandle, svcStop, uintptr(unsafe.Pointer(&status)))
        if ret == 0 {
                return fmt.Errorf("failed to send stop signal: %v", err)
        }

        // Poll until the service is fully stopped (max 10 seconds)
        for i := 0; i < 20; i++ {
                time.Sleep(500 * time.Millisecond)
                procQueryServiceStatus.Call(svcHandle, uintptr(unsafe.Pointer(&status)))
                if status.CurrentState == serviceStopped {
                        return nil
                }
        }
        return fmt.Errorf("service did not stop within 10 seconds (state=%d)", status.CurrentState)
}

func (s *scmManager) Status() (string, error) {
        mgrHandle, _, err := procOpenSCManager.Call(0, 0, uintptr(svcManagerConnect))
        if mgrHandle == 0 {
                return "unknown", fmt.Errorf("failed to open SCM: %v", err)
        }
        defer procCloseServiceHandle.Call(mgrHandle)

        serviceName, err := syscall.UTF16PtrFromString(s.name)
        if err != nil {
                return "unknown", err
        }

        svcHandle, _, _ := procOpenService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                0,
        )
        if svcHandle == 0 {
                return "not installed", nil
        }
        defer procCloseServiceHandle.Call(svcHandle)

        var status serviceStatus
        procQueryServiceStatus.Call(svcHandle, uintptr(unsafe.Pointer(&status)))

        switch status.CurrentState {
        case serviceRunning:
                return "running", nil
        case serviceStopped:
                return "stopped", nil
        case serviceStartPending:
                return "starting", nil
        case serviceStopPending:
                return "stopping", nil
        default:
                return fmt.Sprintf("state %d", status.CurrentState), nil
        }
}

func (s *scmManager) IsInstalled() bool {
        mgrHandle, _, _ := procOpenSCManager.Call(0, 0, uintptr(svcManagerConnect))
        if mgrHandle == 0 {
                return false
        }
        defer procCloseServiceHandle.Call(mgrHandle)

        serviceName, _ := syscall.UTF16PtrFromString(s.name)
        svcHandle, _, _ := procOpenService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                0,
        )
        if svcHandle == 0 {
                return false
        }
        procCloseServiceHandle.Call(svcHandle)
        return true
}
