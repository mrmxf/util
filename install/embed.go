//  Copyright ©2017-2025  Mr MXF   info@mrmxf.com
//  BSD-3-Clause License           https://opensource.org/license/bsd-3-clause/

package install

import "embed"

//go:embed index.yaml releases.yaml
//go:embed recipes
var EmbedFS embed.FS
