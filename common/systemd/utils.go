/*
 * @Author: lsne
 * @Date: 2025-12-07 14:56:23
 */

package systemd

import (
	"fmt"
	"path/filepath"

	"github.com/lsne/goutils/utils/fileutil"
)

func ServiceFileExists(name string) bool {
	return fileutil.IsExists(filepath.Join(SystemdPath, fmt.Sprintf("%s.service", name)))
}
