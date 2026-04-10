//go:build windows

package service

import (
        "fmt"
        "os"
        "path/filepath"
        "syscall"
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

        procOpenSCManager  = modadvapi32.NewProc("OpenSCManagerW")
        procCreateService  = modadvapi32.NewProc("CreateServiceW")
        procOpenService    = modadvapi32.NewProc("OpenServiceW")
        procDeleteService  = modadvapi32.NewProc("DeleteService")
        procStartService   = modadvapi32.NewProc("StartServiceW")
        procControlService = modadvapi32.NewProc("ControlServiceW")
        procCloseServiceHandle = modadvapi32.NewProc("CloseServiceHandle")
        procQueryServiceStatus = modadvapi32.NewProc("QueryServiceStatusW")
)

const (
        svcAllAccess          = 0xF01FF
        svcManagerConnect     = 1
        svcManagerCreate      = 2
        svcStart              = 0x10
        svcStop               = 0x20
        svcDelete             = 0x10000
        serviceRunning        = 4
        serviceStopped        = 1
        serviceStartPending   = 2
        serviceStopPending    = 3
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
        // Open Service Control Manager
        mgrHandle, _, err := procOpenSCManager.Call(
                0,
                0,
                uintptr(svcManagerConnect|svcManagerCreate),
        )
        if mgrHandle == 0 {
                return fmt.Errorf("failed to open Service Control Manager: %v", err)
        }
        defer procCloseServiceHandle.Call(mgrHandle)

        exePath, _ := os.Executable()
        if cfg.BinaryPath != "" {
                exePath = cfg.BinaryPath
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

        binaryPath := fmt.Sprintf(`"%s" -port %d -log-level %s`, exePath, cfg.Port, cfg.LogLevel)
        binaryPathPtr, err := syscall.UTF16PtrFromString(binaryPath)
        if err != nil {
                return fmt.Errorf("failed to encode binary path: %w", err)
        }

        svcHandle, _, err := procCreateService.Call(
                mgrHandle,
                uintptr(unsafe.Pointer(serviceName)),
                uintptr(unsafe.Pointer(displayName)),
                uintptr(svcAllAccess),
                uintptr(0x10), // SERVICE_WIN32_OWN_PROCESS
                uintptr(2),    // SERVICE_AUTO_START
                uintptr(1),    // SERVICE_ERROR_NORMAL
                uintptr(unsafe.Pointer(binaryPathPtr)),
                0, 0, 0, 0, 0,
        )
        if svcHandle == 0 {
                // Service may already exist, try opening it
                svcHandle, _, err = procOpenService.Call(
                        mgrHandle,
                        uintptr(unsafe.Pointer(serviceName)),
                        uintptr(svcAllAccess),
                )
                if svcHandle == 0 {
                        return fmt.Errorf("failed to create/open service: %v", err)
                }
                defer procCloseServiceHandle.Call(svcHandle)
                return fmt.Errorf("service already exists (use -uninstall first)")
        }
        defer procCloseServiceHandle.Call(svcHandle)

        return nil
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
        }

        // Delete the service
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

        ret, _, err := procStartService.Call(svcHandle, 0, 0)
        if ret == 0 {
                return fmt.Errorf("failed to start service: %v", err)
        }
        return nil
}

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
                return fmt.Errorf("failed to stop service: %v", err)
        }
        return nil
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

        svcHandle, _, err := procOpenService.Call(
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


