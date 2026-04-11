//go:build linux

package collector

import (
	"os"
	"os/user"
	"strconv"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectUser() models.UserInfo {
	info := models.UserInfo{}

	if u, err := user.Current(); err == nil {
		info.Username = u.Username
		if uid, err := strconv.ParseUint(u.Uid, 10, 32); err == nil {
			info.SID = "uid:" + strconv.FormatUint(uid, 10)
		}
	}

	// Try domain from environment
	if d := os.Getenv("DOMAIN"); d != "" {
		info.Domain = d
	}

	return info
}
