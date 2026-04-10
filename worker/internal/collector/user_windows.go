//go:build windows

package collector

import (
	"os"
	"os/user"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectUser() models.UserInfo {
	info := models.UserInfo{}

	if u, err := user.Current(); err == nil {
		info.Username = u.Username
		info.SID = u.Uid
	}

	if d := os.Getenv("USERDOMAIN"); d != "" {
		info.Domain = d
	}

	return info
}
